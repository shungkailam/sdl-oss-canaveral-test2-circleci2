package apitesthelper

import (
	"bytes"
	accountapi "cloudservices/account/api"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/cfssl"
	"cloudservices/cloudmgmt/config"
	"cloudservices/cloudmgmt/event"
	cmRouter "cloudservices/cloudmgmt/router"
	"cloudservices/cloudmgmt/websocket"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/model"
	"cloudservices/common/service"
	eventapi "cloudservices/event/api"
	operatorapi "cloudservices/operator/api"
	operatorconfig "cloudservices/operator/config"
	tenantpoolapi "cloudservices/tenantpool/api"
	tenantpoolconfig "cloudservices/tenantpool/config"
	tenantpoolmodel "cloudservices/tenantpool/model"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

// Default values. Some of the values are overidden in StartServices
var (
	UserPassword      = "P@ssw0rd"
	TestServer        = "test.ntnxsherlock.com"
	TestPort          = 443
	TestSecure        = true
	RESTServer        = "https://test.ntnxsherlock.com"
	UseRealHTTPServer = false
)

// const (
// 	UserPassword          = "foo"
// 	TestServer            = "localhost"
// 	TestPort              = 8080
// 	TestSecure            = false
// 	RESTServer            = "http://localhost:8080"
// )

// OAuth2 related variables
const IDP_BASE_URL = "https://idp-dev.nutanix.com"
const MY_NUTANIX_URL = "https://demo-my.nutanix.com"

var (
	OAuth2Config = &oauth2.Config{
		ClientID:     "r5OVZbVdam1haHpfKUQoLmiUEtga",
		ClientSecret: "BBfnauBPs44A_T0mvmUN1a9UyPMa",
		RedirectURL:  "https://my.ntnxsherlock.com/auth/oauth",
		Scopes: []string{
			"openid",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth2/authorize", IDP_BASE_URL),
			TokenURL: fmt.Sprintf("%s/oauth2/token", IDP_BASE_URL),
		},
	}
	keyService crypto.KeyService
	serverLock sync.Mutex
	httpRouter *httprouter.Router
)

// StartServicesConfig has all the configs to start the services
// The services can be started in go-routines
type StartServicesConfig struct {
	StartPort       int
	EdgeProvisioner tenantpoolmodel.EdgeProvisioner
}

func init() {
	keyService = crypto.NewKeyService(*config.Cfg.AWSRegion, *config.Cfg.JWTSecret, *config.Cfg.AWSKMSKey, *config.Cfg.UseKMS)
	// Needs parse before logging!
	// flag.Parse()
	// make lock duration shorter to reduce test sleep time
	*config.Cfg.LoginLockDurationSeconds = 5
}

func changeServersToLocalhost(port int) {
	RESTServer = fmt.Sprintf("%s:%d", "http://localhost", port)
	TestServer = "localhost"
	TestPort = port
	TestSecure = false
}

func checkPortInUse(port int) bool {
	return checkHostPortInUse("", port)
}

func checkHostPortInUse(host string, port int) bool {
	conn, _ := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 2*time.Second)
	if conn != nil {
		conn.Close()
		return true
	}
	return false
}

func getRandomPort(host string) (int, error) {
	maxN := big.NewInt(65535 - 1024 + 1)
	for i := 0; i < 5; i++ {
		nBig, err := rand.Int(rand.Reader, maxN)
		if err != nil {
			return 0, err
		}
		//Min port - 1024 ,Max port - 65535
		port := int(nBig.Int64()) + 1024
		if checkHostPortInUse(host, port) {
			time.Sleep(time.Second)
			continue
		}
		return port, nil
	}
	return 0, errors.New("Retry exceeded")
}

func getHostIPAddr() (string, error) {
	return "", nil

}

// StartServices starts the services locally.
// go test runs test packages unless -p 1 is set.
// go test -parallel is for parallelizing tests in a package.
// Care must be taken not to call from the same process more than once.
func StartServices(servicesConfig *StartServicesConfig) {
	serverLock.Lock()
	defer serverLock.Unlock()
	if checkPortInUse(servicesConfig.StartPort) {
		panic(fmt.Errorf("%d is already in use", servicesConfig.StartPort))
	}
	changeServersToLocalhost(servicesConfig.StartPort)
	hostIP := "localhost"
	// e.g DOCKER_HOST=tcp://10.192.4.253:2376 set by circleci
	dockerHost := base.GetEnvWithDefault("DOCKER_HOST", "localhost")
	if strings.HasPrefix(dockerHost, "tcp://") {
		URL, _ := url.Parse(dockerHost)
		dockerHost = URL.Hostname()
	}
	grpcStartPort := servicesConfig.StartPort + 1
	accountServiceConfig := &service.ServiceConfig{Host: "localhost", Port: grpcStartPort + int(service.AccountService)}
	eventServiceConfig := &service.ServiceConfig{Host: "localhost", Port: grpcStartPort + int(service.EventService)}
	operatorServiceConfig := &service.ServiceConfig{Host: "localhost", Port: grpcStartPort + int(service.OperatorService)}
	tenantPoolServiceConfig := &service.ServiceConfig{Host: "localhost", Port: grpcStartPort + int(service.TenantPoolService)}
	// REST port for Operator service
	operatorconfig.Cfg.Port = base.IntPtr(operatorServiceConfig.Port + 1000)
	// AI service is running inside a container and exposing host port.
	// If you run the tests inside a container(ex: canaveral) we need to use host ip to connect to the service.
	// Note: Since it is using host port, during canavaeral build,there will be port conflicts,
	// so randomize the port number.
	//Min port - 1024 ,Max port - 65535
	aiServiceHostPort, err := getRandomPort(hostIP)
	if err != nil {
		panic(err)
	}
	fmt.Println("AI Service: selecting port", aiServiceHostPort)
	aiServiceConfig := &service.ServiceConfig{Host: hostIP, Port: aiServiceHostPort}
	service.OverrideServiceConfig(service.AccountService, accountServiceConfig)
	service.OverrideServiceConfig(service.EventService, eventServiceConfig)
	service.OverrideServiceConfig(service.OperatorService, operatorServiceConfig)
	service.OverrideServiceConfig(service.TenantPoolService, tenantPoolServiceConfig)
	service.OverrideServiceConfig(service.AIService, aiServiceConfig)
	// Start ai service
	go func() {
		// Container is run in the dockerHost
		aiDockerHostPort := fmt.Sprintf("%s:%d:8500", dockerHost, aiServiceConfig.Port)
		aiServiceDockerTag := "ai:build"
		cmd := exec.Command("docker", "run", "-p", aiDockerHostPort, aiServiceDockerTag)
		fmt.Printf("Starting AI server on %s:%d\n", dockerHost, aiServiceConfig.Port)
		err := cmd.Run()
		if err != nil {
			panic(err)
		}
	}()

	if dockerHost != "localhost" {
		go func() {
			// set up ssh tunnel to the container in remote-docker (same as docker host) from localhost
			tunnelParam := fmt.Sprintf("%d:%s:%d", aiServiceConfig.Port, dockerHost, aiServiceConfig.Port)
			fmt.Printf("Setting up SSH tunnel %s\n", tunnelParam)
			cmd := exec.Command("ssh", "-4", "-N", "-L", tunnelParam, "remote-docker")
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		}()
	}

	// Start account service
	go func() {
		server, err := accountapi.NewAPIServer()
		if err == nil {
			defer server.Close()
			fmt.Println("Starting account server on ", accountServiceConfig.Port)
			err = service.StartServer(accountServiceConfig.Port, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
				server.Register(gServer)
				return gServer.Serve(listener)
			})
		}
		if err != nil {
			panic(err)
		}
	}()

	// Start event service
	go func() {
		server, err := eventapi.NewAPIServer()
		if err == nil {
			defer server.Close()
			fmt.Println("Starting event server on ", eventServiceConfig.Port)
			err = service.StartServer(eventServiceConfig.Port, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
				server.Register(gServer)
				return gServer.Serve(listener)
			})
		}
		if err != nil {
			panic(err)
		}
	}()

	// Start operator service
	go func() {
		server := operatorapi.NewAPIServer()
		server.RegisterAPIHandlers()
		// Start the http server
		fmt.Println("Starting operator server on ", operatorServiceConfig.Port)
		go server.StartServer()

		panic(service.StartServer(operatorServiceConfig.Port, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
			server.Register(gServer)
			return gServer.Serve(listener)
		}))
	}()

	if servicesConfig.EdgeProvisioner != nil {
		// Start tenantpool service
		go func() {
			// Override default config
			tenantpoolconfig.Cfg.TenantPoolScanDelay = base.DurationPtr(time.Second * 3)
			server := tenantpoolapi.NewAPIServerEx(servicesConfig.EdgeProvisioner)
			defer server.Close()
			fmt.Println("Starting tenantpool server on ", tenantPoolServiceConfig.Port)
			err := service.StartServer(tenantPoolServiceConfig.Port, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
				server.Register(gServer)
				return gServer.Serve(listener)
			})
			if err != nil {
				panic(err)
			}
		}()
	}

	// Start cloudmgmt REST service
	go func() {
		api.InitGlobals()
		cfssl.InitCfssl(*config.Cfg.CfsslProtocol, *config.Cfg.CfsslHost, *config.Cfg.CfsslPort)

		dbAPI, err := api.NewObjectModelAPI()
		if err == nil {
			defer dbAPI.Close()
			err = cmRouter.RegisterOAuthHandlers(dbAPI, config.Cfg.RedirectURLs.Values())
			if err != nil {
				panic(err)
			}
			event.RegisterEventListeners(dbAPI)
			router := httprouter.New()
			httpRouter = router
			msgSvc := websocket.ConfigureWSMessagingService(dbAPI, router, nil)

			var redisClient *redis.Client
			// // if running redis locally, can uncomment the following to test
			// redisClient = redis.NewClient(&redis.Options{
			// 	Addr:     "localhost:6379",
			// 	Password: "", // no password set
			// 	DB:       0,  // use default DB
			// })

			cmRouter.ConfigureRouter(dbAPI, router, redisClient, msgSvc, *config.Cfg.ContentDir)
			fmt.Println("Starting REST server on ", servicesConfig.StartPort)
			// need this even if UseRealHTTPServer = false
			err = http.ListenAndServe(fmt.Sprintf(":%d", servicesConfig.StartPort), router)
		}
		if err != nil {
			panic(err)
		}
	}()

	// Wait for the services to start up
	time.Sleep(time.Second * 5)
}

// ClientDo if UseRealHTTPServer is true, then call netClient.Do, else call
// httpRouter.ServeHTTP directly.
func ClientDo(netClient *http.Client, req *http.Request) (*http.Response, error) {
	if UseRealHTTPServer {
		return netClient.Do(req)
	}
	rr := httptest.NewRecorder()
	httpRouter.ServeHTTP(rr, req)
	return rr.Result(), nil
}

// StructToReader converts an object into io.Reader object
func StructToReader(obj interface{}) (io.Reader, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// NewHTTPRequest if UseRealHTTPServer is true, then call http.NewRequest, else
// call httptest.NetRequest
func NewHTTPRequest(method, url string, body io.Reader) (*http.Request, error) {
	if UseRealHTTPServer {
		return http.NewRequest(method, url, body)
	}
	return httptest.NewRequest(method, url, body), nil
}

// NewHTTPRequestFromStruct creates the HTTP request object from the struct object
func NewHTTPRequestFromStruct(method, url string, body interface{}) (*http.Request, error) {
	var r io.Reader
	var err error
	if body != nil {
		r, err = StructToReader(body)
		if err != nil {
			return nil, err
		}
	}
	return NewHTTPRequest(method, url, r)
}

// ResponseWriter is used for getting the response from io.Writer
type ResponseWriter struct {
	buffer *bytes.Buffer
}

// NewResponseWriter returns an instance of ResponseWriter
func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{buffer: &bytes.Buffer{}}
}

// Reset resets the buffer to accept new writes
func (writer *ResponseWriter) Reset() {
	writer.buffer.Reset()
}

func (writer *ResponseWriter) Write(p []byte) (n int, err error) {
	return writer.buffer.Write(p)
}

// GetBody gets the response body
func (writer *ResponseWriter) GetBody(obj interface{}) error {
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return errors.New("Pointer expected")
	}
	return json.NewDecoder(writer.buffer).Decode(obj)
}

func GenTenantToken() (string, error) {
	token, err := keyService.GenTenantToken()
	if err != nil {
		return "", err
	}
	return token.EncryptedToken, nil
}

func CreateTenant(t *testing.T, dbAPI api.ObjectModelAPI, name string) model.Tenant {
	tenantID := base.GetUUID()
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	tenantToken, err := GenTenantToken()
	if err != nil {
		t.Fatal(err)
	}
	// Create tenant object
	doc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	// create tenant
	resp, err := dbAPI.CreateTenant(ctx, &doc, nil)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("create tenant successful, %s", resp)
	return doc
}

func CreateUser(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, role string) model.User {
	var userID = base.GetUUID()
	var userName = "John Doe - " + userID
	var userEmail = userID + "@test.com"

	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	user := model.User{
		BaseModel: model.BaseModel{
			ID:       userID,
			TenantID: tenantID,
			Version:  0,
		},
		Name:     userName,
		Email:    userEmail,
		Password: UserPassword,
		Role:     role,
	}
	// create user
	resp, err := dbAPI.CreateUser(ctx, &user, nil)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("create user successful, %s", resp)

	if userID != resp.(model.CreateDocumentResponse).ID {
		t.Fatal("user id mismatch")
	}
	return user
}

func GetHTTPClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func GetOAuthCode(t *testing.T, user, password string) (string, error) {
	client := GetHTTPClient()
	form := url.Values{}
	form.Add("response_type", "code")
	form.Add("client_id", OAuth2Config.ClientID)
	form.Add("redirect_uri", OAuth2Config.RedirectURL)
	form.Add("scope", "openid")
	payload := form.Encode()
	req, err := http.NewRequest("POST", OAuth2Config.Endpoint.AuthURL, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.SetBasicAuth(user, password)
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error getting code. Error: %s", err.Error())
	}
	header := response.Header
	location := header.Get("Location")
	exp, err := regexp.Compile("code=[a-zA-Z0-9\\-]+")
	codeParam := string(exp.Find([]byte(location)))
	code := strings.Replace(codeParam, "code=", "", -1)
	return code, nil
}

// GenerateIDWithPrefix generates an ID with the prefix if specified
func GenerateIDWithPrefix(prefix string) string {
	id := base.GetUUID()
	prefix = strings.TrimSpace(prefix)
	prefixLen := len(prefix)
	if prefixLen > 0 && prefixLen < len(id) {
		id = prefix + id[prefixLen:]
	}
	return id
}

// SyncWait is used for verifying API callbacks
type SyncWait struct {
	ch chan bool
	t  *testing.T
}

// NewSyncWait creates a sync wait with timeout
func NewSyncWait(t *testing.T) *SyncWait {
	return &SyncWait{t: t, ch: make(chan bool, 1)}
}

// Done is called to send done signal
func (sw *SyncWait) Done() {
	sw.ch <- true
}

// WaitWithTimeout waits for 3 seconds max
func (sw *SyncWait) WaitWithTimeout() {
	select {
	case <-sw.ch:
		return
	case <-time.After(3 * time.Second):
		sw.t.Fatal("timed out waiting")
	}
}

func ExtractJWTClaims(t *testing.T, token string) jwt.MapClaims {
	// It is a JWT token
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatal(fmt.Sprintf("Invalid token for %s", token))
	}
	claims := jwt.MapClaims{}
	payloadBytes, err := jwt.DecodeSegment(parts[1])
	if err != nil {
		t.Fatal(fmt.Sprintf("Invalid payload for token %s", token))
	}
	if err = json.Unmarshal(payloadBytes, &claims); err != nil {
		t.Fatal(fmt.Sprintf("Invalid payload for token %s. Error: %s", token, err.Error()))
	}
	return claims
}
