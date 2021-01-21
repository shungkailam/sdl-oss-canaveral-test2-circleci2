package api

import (
	"cloudservices/cloudmgmt/cfssl"
	"cloudservices/cloudmgmt/config"
	"cloudservices/cloudmgmt/util"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/utils"
	"strings"

	texttmpl "text/template"

	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/yaml"

	voyagerClient "github.com/appscode/voyager/client/clientset/versioned"

	voyagerv1beta1 "github.com/appscode/voyager/apis/voyager/v1beta1"

	"github.com/go-redis/redis"
	"github.com/golang/glog"
)

const (
	voyagerIngressResourceName = "cloudmgmt-wstun-ingress"
	wstunSvcName               = "wstun-svc"
	wstunDuration              = 30 * time.Minute
	bufSize                    = 2048
)

var wst = utils.NewWstunUtil()

type setupSSHTunnelingOptions struct {
	doc                 model.WstunRequest
	skipValidation      bool
	duration            time.Duration
	setupBasicAuth      bool
	username            string
	password            string
	disableRewriteRules bool
	dns                 string
	headers             string
	publicKey           string // set this to indicate update to extend session
}
type teardownSSHTunnelingOptions struct {
	doc            model.WstunTeardownRequest
	skipValidation bool
	duration       time.Duration
	setupBasicAuth bool
}

// allow some code path to be disabled to simplify testing
func isDisableK8S() bool {
	return os.Getenv("DISABLE_K8S") == "1"
}
func isDisableSSHKeyGen() bool {
	return os.Getenv("DISABLE_SSH_KEYGEN") == "1"
}
func isDisableSSHProfileCheck() bool {
	return os.Getenv("DISABLE_SSH_PROFILE_CHECK") == "1"
}

func getSSHDNS() string {
	base := *config.Cfg.ProxyUrlBase
	i := len("https://")
	return base[i:]
}

func cleanUpSSHTunneling(context context.Context, redisClient *redis.Client) {
	ports := wst.ClearExpiredPorts(redisClient)
	if len(ports) != 0 {
		if isDisableK8S() {
			return
		}
		removeWstunServicePorts(context, ports)
		removeVoyagerSSHPorts(context, ports)
	}
}

// Setup ssh tunneling to Edge
func (dbAPI *dbObjectModelAPI) SetupSSHTunneling(context context.Context, doc model.WstunRequest, callback func(context.Context, interface{}) error) (model.WstunPayload, error) {
	options := setupSSHTunnelingOptions{
		doc:                 doc,
		skipValidation:      false,
		duration:            wstunDuration,
		setupBasicAuth:      false,
		disableRewriteRules: false,
		dns:                 "",
		headers:             "",
		publicKey:           "",
	}
	return dbAPI.setupSSHTunneling(context, options, callback)
}
func (dbAPI *dbObjectModelAPI) setupSSHTunneling(context context.Context, options setupSSHTunnelingOptions, callback func(context.Context, interface{}) error) (model.WstunPayload, error) {
	doc := options.doc
	skipValidation := options.skipValidation
	duration := options.duration

	resp := model.WstunPayload{WstunRequest: doc}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID
	resp.TenantID = tenantID

	// must be infra admin
	if !skipValidation && !auth.IsInfraAdminRole(authContext) {
		return resp, errcode.NewBadRequestError("permission")
	}

	tenant, err := dbAPI.GetTenant(context, tenantID)
	if err != nil {
		return resp, err
	}
	disableSSHProfileCheck := isDisableSSHProfileCheck()
	// tenant profile must have SSH enabled
	if !skipValidation && !disableSSHProfileCheck && (tenant.Profile == nil || !tenant.Profile.EnableSSH) {
		return resp, errcode.NewBadRequestError("SSH Disabled")
	}

	// make sure edge (service domain) belong to tenant
	svcDomain, err := dbAPI.GetServiceDomain(context, doc.ServiceDomainID)
	if err != nil {
		return resp, errcode.NewBadRequestError("ServiceDomain")
	}
	if !skipValidation && !disableSSHProfileCheck && (svcDomain.Profile == nil || !svcDomain.Profile.EnableSSH) {
		return resp, errcode.NewBadRequestError("SSH Disabled for Service Domain")
	}

	// only require edge connection for prod
	// otherwise would be difficult to test this API
	if dbAPI.prod {
		// make sure edge is connected
		if !IsEdgeConnected(tenantID, doc.ServiceDomainID) {
			return resp, errcode.NewBadRequestError("ServiceDomain/offline")
		}
	}

	redisClient := dbAPI.GetRedisClient()
	portAllocated := false
	tunnelingAdded := false
	var port int32
	defer func() {
		if err != nil {
			if tunnelingAdded {
				removeSSHTunnelingPort(context, port, options.setupBasicAuth)
			}
			if portAllocated {
				wst.ReleasePort(redisClient, model.WstunTeardownRequestInternal{
					TenantID:        tenantID,
					ServiceDomainID: doc.ServiceDomainID,
					PublicKey:       resp.PublicKey,
					Endpoint:        doc.Endpoint,
				}, duration)
			}
		} else {
			// use setup call as opportunity to do clean up
			go cleanUpSSHTunneling(context, redisClient)
		}
	}()
	if !isDisableSSHKeyGen() {
		if options.publicKey != "" {
			// update case, reuse public key
			resp.PublicKey = options.publicKey
			resp.PrivateKey = ""
		} else {
			keyPair, err := cfssl.GenerateKeyPair()
			if err != nil {
				glog.Warningf(base.PrefixRequestID(context, "failed to generate ssh key pair: %s\n"), err.Error())
				return resp, err
			}
			resp.PublicKey = keyPair.PublicKey
			resp.PrivateKey = keyPair.PrivateKey
		}
	}
	// allocate a port
	// resp.PublicKey must be set
	err = wst.AllocatePort(redisClient, &resp, duration)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(context, "failed to allocate ssh port for edge: %s\n"), doc.ServiceDomainID)
		return resp, errcode.NewInternalError("port")
	}
	portAllocated = true

	port = int32(resp.Port)
	// skipValidation is set only for http service proxy
	// in this case we always want to setup voyager rule,
	// regardless of the profile.AllowCliSSH flag
	err = addSSHTunnelingPort(context, port, !disableSSHProfileCheck && (skipValidation || tenant.Profile.AllowCliSSH), options)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(context, "failed to add ssh port %d to wstun-svc or voyager ingress rule\n"), port)
		return resp, err
	}
	tunnelingAdded = true

	// invoke and wait for
	if callback != nil {
		err = callback(context, &resp)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(context, "failed to send ssh setup message to edge for tenant: %s, edge: %s\n"), tenantID, doc.ServiceDomainID)
		}
		if !dbAPI.prod {
			// ignore callback err if not prod
			err = nil
		}
	}

	if err == nil && doc.Endpoint != "" {
		resp.URL = fmt.Sprintf("%s/%s", *config.Cfg.ProxyUrlBase, doc.GetProxyEndpointPath())
	}

	return resp, err
}

func newVoyagerClientSet() (*voyagerClient.Clientset, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// create the clientset for voyager
	return voyagerClient.NewForConfig(restConfig)
}

// add port to wstun-svc and voyager ingress rules
func addSSHTunnelingPort(context context.Context, port int32, allowCliSSH bool, options setupSSHTunnelingOptions) (err error) {
	if isDisableK8S() {
		return nil
	}
	err = addWstunServicePort(context, port, options)
	if err != nil {
		return
	}
	// only setup voyager port-forwarding if cli access to SSH is allowed
	if allowCliSSH {
		err = addVoyagerSSHPort(context, port, options)
		if err != nil {
			removeWstunServicePort(context, port)
		}
	}
	return
}

// remove port from wstun-svc and voyager ingress rules
func removeSSHTunnelingPort(context context.Context, port int32, setupBasicAuth bool) error {
	glog.V(5).Infof(base.PrefixRequestID(context, "removeSSHTunnelingPort: port=%d\n"), port)
	if isDisableK8S() {
		return nil
	}
	// try to remove both regardless of error
	err1 := removeWstunServicePort(context, port)
	err2 := removeVoyagerSSHPort(context, port)
	if err1 != nil {
		return err1
	}
	return err2
}

func makeHeaders(context context.Context, headers string) []string {
	headers = strings.TrimSpace(headers)
	if headers == "" {
		return nil
	}
	nvMap := make(map[string]string)
	err := json.Unmarshal([]byte(headers), &nvMap)
	if err != nil {
		// warn and ignore
		glog.Warningf(base.PrefixRequestID(context, "Failed to parse headers [%s], ignored: %s\n"), headers, err)
		return nil
	}
	headerList := []string{}
	for n, v := range nvMap {
		tn := strings.TrimSpace(n)
		tv := strings.TrimSpace(v)
		if tn != "" && tv != "" {
			headerList = append(headerList, fmt.Sprintf("%s %s", tn, tv))
		}
	}
	if len(headerList) == 0 {
		return nil
	}
	return headerList
}

// update voyager ingress resource rules
func addVoyagerSSHPort(context context.Context, port int32, options setupSSHTunnelingOptions) (err error) {
	doc := options.doc
	setupBasicAuth := options.setupBasicAuth
	clientsetVoyager, err := newVoyagerClientSet()
	if err != nil {
		return
	}
	// create ingress instance for pod namespace
	ingressInstance := clientsetVoyager.VoyagerV1beta1().Ingresses(util.GetPodNamespace())

	// get our ingress resource by name
	ing, err := ingressInstance.Get(voyagerIngressResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	portExist := false
	host := getSSHDNS()
	// skip if rule already setup
	for _, rule := range ing.Spec.Rules {
		if doc.Endpoint != "" {
			if http := rule.HTTP; http != nil {
				for _, p := range http.Paths {
					if int32(p.Backend.IngressBackend.ServicePort.IntValue()) == port {
						portExist = true
						break
					}
				}
			}
		} else {
			if tcp := rule.TCP; tcp != nil && int32(tcp.Port.IntValue()) == port {
				portExist = true
			}
		}
	}
	if portExist {
		return
	}

	// add a rule
	if doc.Endpoint != "" {
		// if have endpoint, only add http rule
		// form path from svc domain id and endpoint
		pathVal := doc.GetProxyEndpointPath()
		path := "/" + pathVal
		rewriteRules := []string{}
		if !options.disableRewriteRules {
			rewriteRule := fmt.Sprintf(`^([^\ ]*\ /)%s[/]?(.*)     \1\2`, pathVal)
			rewriteRules = append(rewriteRules, rewriteRule)
		}
		svcName := wstunSvcName
		if setupBasicAuth {
			svcName = fmt.Sprintf("%s-%d", wstunSvcName, port)
		}
		// override headers
		headerRules := makeHeaders(context, options.headers)
		rule := voyagerv1beta1.IngressRule{
			Host: host,
			IngressRuleValue: voyagerv1beta1.IngressRuleValue{
				TCP: nil,
				HTTP: &voyagerv1beta1.HTTPIngressRuleValue{
					Port: intstr.IntOrString{Type: 0, IntVal: 80},
					Paths: []voyagerv1beta1.HTTPIngressPath{
						{
							Path: path,
							Backend: voyagerv1beta1.HTTPIngressBackend{
								IngressBackend: voyagerv1beta1.IngressBackend{
									ServiceName: svcName,
									ServicePort: intstr.IntOrString{Type: 0, IntVal: port},
								},
								RewriteRules: rewriteRules,
								HeaderRules:  headerRules,
							},
						},
					},
				},
			},
		}
		ing.Spec.Rules = append(ing.Spec.Rules, rule)
		// if dns != "", also add HTTP rule for it
		// TODO: update route53 to add this
		if options.dns != "" {
			rule = voyagerv1beta1.IngressRule{
				Host: options.dns,
				IngressRuleValue: voyagerv1beta1.IngressRuleValue{
					TCP: nil,
					HTTP: &voyagerv1beta1.HTTPIngressRuleValue{
						Port: intstr.IntOrString{Type: 0, IntVal: 80},
						Paths: []voyagerv1beta1.HTTPIngressPath{
							{
								Path: "/",
								Backend: voyagerv1beta1.HTTPIngressBackend{
									IngressBackend: voyagerv1beta1.IngressBackend{
										ServiceName: svcName,
										ServicePort: intstr.IntOrString{Type: 0, IntVal: port},
									},
									RewriteRules: nil,
									HeaderRules:  headerRules,
								},
							},
						},
					},
				},
			}
			ing.Spec.Rules = append(ing.Spec.Rules, rule)
			// add route53 alias
			if err := utils.UpsertAliasRecord(context, host, options.dns); err != nil {
				// warn and ignore
				glog.Warningf(base.PrefixRequestID(context, "UpsertAliasRecord: failed %s\n"), err)
			}
		}
	} else {
		// no endpoint, add TCP rule
		rule := voyagerv1beta1.IngressRule{
			Host: host,
			IngressRuleValue: voyagerv1beta1.IngressRuleValue{
				HTTP: nil,
				TCP: &voyagerv1beta1.TCPIngressRuleValue{
					Port: intstr.IntOrString{Type: 0, IntVal: port},
					Backend: voyagerv1beta1.IngressBackend{
						ServiceName: wstunSvcName,
						ServicePort: intstr.IntOrString{Type: 0, IntVal: port},
					},
				},
			},
		}
		ing.Spec.Rules = append(ing.Spec.Rules, rule)
	}
	_, err = ingressInstance.Update(ing)
	return
}
func removeVoyagerSSHPort(context context.Context, port int32) (err error) {
	clientsetVoyager, err := newVoyagerClientSet()
	if err != nil {
		return
	}
	// create ingress instance for pod namespace
	ingressInstance := clientsetVoyager.VoyagerV1beta1().Ingresses(util.GetPodNamespace())

	// get our ingress resource by name
	ing, err := ingressInstance.Get(voyagerIngressResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	host := getSSHDNS()
	rules := ing.Spec.Rules
	newRules := []voyagerv1beta1.IngressRule{}
	for _, rule := range rules {
		tcp := rule.TCP
		http := rule.HTTP
		if tcp != nil {
			if int32(tcp.Port.IntValue()) != port {
				newRules = append(newRules, rule)
			}
		} else if http != nil {
			match := false
			for _, p := range http.Paths {
				if int32(p.Backend.ServicePort.IntValue()) == port {
					match = true
					break
				}
			}
			if !match {
				newRules = append(newRules, rule)
			} else if rule.Host != host {
				// delete route53 alias
				if err := utils.DeleteAliasRecord(context, host, rule.Host); err != nil {
					// warn and ignore
					glog.Warningf(base.PrefixRequestID(context, "DeleteAliasRecord: failed %s\n"), err)
				}
			}
		}
	}
	if len(rules) == len(newRules) {
		return
	}
	ing.Spec.Rules = newRules
	_, err = ingressInstance.Update(ing)
	return
}

func removeVoyagerSSHPorts(context context.Context, ports []int32) (err error) {
	clientsetVoyager, err := newVoyagerClientSet()
	if err != nil {
		return
	}
	// create ingress instance for pod namespace
	ingressInstance := clientsetVoyager.VoyagerV1beta1().Ingresses(util.GetPodNamespace())

	// get our ingress resource by name
	ing, err := ingressInstance.Get(voyagerIngressResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	rules := ing.Spec.Rules
	var rulesToKeep []voyagerv1beta1.IngressRule
	for _, rule := range rules {
		keep := true
		if rule.TCP != nil {
			port := int32(rule.TCP.Port.IntValue())
			for _, pt := range ports {
				if pt == port {
					keep = false
					break
				}
			}
		}
		if keep {
			rulesToKeep = append(rulesToKeep, rule)
		}
	}
	if len(rules) != len(rulesToKeep) {
		ing.Spec.Rules = rulesToKeep
		_, err = ingressInstance.Update(ing)
	}
	return
}

type portData struct {
	Port int32
}

func createBasicAuthSecret(clientset *kubernetes.Clientset, nameSpace, username, password string, port int32) (err error) {
	// always delete first
	deleteBasicAuthSecret(clientset, nameSpace, port)

	secret := apiv1.Secret{}
	authData := []byte(fmt.Sprintf("%s::%s\n", username, password))
	encodedAuthData := base64.StdEncoding.EncodeToString(authData)
	secretYaml := fmt.Sprintf(`kind: Secret
apiVersion: v1
metadata:
  name: wstun-secret-%d
type: Opaque
data:
  auth: %s
`, port, encodedAuthData)
	err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(secretYaml), bufSize).Decode(&secret)
	if err != nil {
		return
	}
	servicesClient := clientset.CoreV1().Secrets(nameSpace)
	_, err = servicesClient.Create(&secret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return
		}
		err = nil
	}
	return
}
func deleteBasicAuthSecret(clientset *kubernetes.Clientset, nameSpace string, port int32) (err error) {
	secretName := fmt.Sprintf("wstun-secret-%d", port)
	servicesClient := clientset.CoreV1().Secrets(nameSpace)
	err = servicesClient.Delete(secretName, nil)
	return
}

func renderTemplate(yaml string, data interface{}) (string, error) {
	t := texttmpl.Must(texttmpl.New("template").Parse(yaml))
	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
func createBasicAuthService(clientset *kubernetes.Clientset, nameSpace string, port int32) (err error) {
	// always delete first
	deleteBasicAuthService(clientset, nameSpace, port)

	svc := apiv1.Service{}
	svcYaml := `kind: Service
apiVersion: v1
metadata:
  name: wstun-svc-{{.Port}}
  annotations:
    ingress.appscode.com/auth-type: basic
    ingress.appscode.com/auth-realm: HttpServiceProxy
    ingress.appscode.com/auth-secret: wstun-secret-{{.Port}}
spec:
  selector:
    app: wstun-server
  ports:
  - name: ssh-port-{{.Port}}
    port: {{.Port}}
    protocol: TCP
    targetPort: {{.Port}}
`
	svcdata := portData{
		Port: port,
	}
	renderedSvcYaml, err := renderTemplate(string(svcYaml), svcdata)
	if err != nil {
		return
	}
	err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(renderedSvcYaml), bufSize).Decode(&svc)
	if err != nil {
		return
	}
	servicesClient := clientset.CoreV1().Services(nameSpace)
	_, err = servicesClient.Create(&svc)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return
		}
		err = nil
	}
	return
}
func deleteBasicAuthService(clientset *kubernetes.Clientset, nameSpace string, port int32) (err error) {
	svcName := fmt.Sprintf("wstun-svc-%d", port)
	servicesClient := clientset.CoreV1().Services(nameSpace)
	err = servicesClient.Delete(svcName, nil)
	return
}

// note: this function requires RBAC permission to be configured for the pod to work
func addWstunServicePort(context context.Context, port int32, options setupSSHTunnelingOptions) (err error) {
	// use k8s API to add port to wstun svc
	var clientset *kubernetes.Clientset
	var wstunSvc *apiv1.Service
	clientset, err = util.NewK8sClientSet()
	if err != nil {
		return
	}
	nameSpace := util.GetPodNamespace()
	servicesClient := clientset.CoreV1().Services(nameSpace)

	if options.setupBasicAuth {
		// use a separate service + secrets for basic auth
		err = createBasicAuthSecret(clientset, nameSpace, options.username, options.password, port)
		if err != nil {
			return
		}
		err = createBasicAuthService(clientset, nameSpace, port)
		if err != nil {
			_ = deleteBasicAuthSecret(clientset, nameSpace, port)
			return
		}
	} else {
		wstunSvc, err = servicesClient.Get("wstun-svc", metav1.GetOptions{})
		if err != nil {
			return
		}
		portName := fmt.Sprintf("ssh-port-%d", port)
		portExist := false
		ports := wstunSvc.Spec.Ports
		for _, pt := range ports {
			if pt.Name == portName {
				portExist = true
				break
			}
		}
		if !portExist {
			// let's update the service to add the port
			wstunSvc.Spec.Ports = append(wstunSvc.Spec.Ports, apiv1.ServicePort{
				Name:       portName,
				Protocol:   "TCP",
				Port:       port,
				TargetPort: intstr.IntOrString{Type: 0, IntVal: port},
			})
			wstunSvc, err = servicesClient.Update(wstunSvc)
			if err != nil {
				return
			}
		}
	}
	return
}

// note: this function requires RBAC permission to be configured for the pod to work
func removeWstunServicePort(context context.Context, port int32) (err error) {
	glog.V(5).Infof(base.PrefixRequestID(context, "removeWstunServicePort: port=%d\n"), port)
	// use k8s API to remove port from wstun svc
	var clientset *kubernetes.Clientset
	var wstunSvc *apiv1.Service
	clientset, err = util.NewK8sClientSet()
	if err != nil {
		return
	}
	nameSpace := util.GetPodNamespace()

	// always attempt to delete basic auth resources
	err2 := deleteBasicAuthService(clientset, nameSpace, port)
	if err2 != nil {
		glog.Warningf(base.PrefixRequestID(context, "removeWstunServicePort: failed to delete service=%s\n"), err2)
	}
	err2 = deleteBasicAuthSecret(clientset, nameSpace, port)
	if err2 != nil {
		glog.Warningf(base.PrefixRequestID(context, "removeWstunServicePort: failed to delete secret=%s\n"), err2)
	}

	servicesClient := clientset.CoreV1().Services(nameSpace)
	wstunSvc, err = servicesClient.Get("wstun-svc", metav1.GetOptions{})
	if err != nil {
		return
	}
	portName := fmt.Sprintf("ssh-port-%d", port)
	ports := wstunSvc.Spec.Ports
	idx := -1
	for i, pt := range ports {
		if pt.Name == portName {
			idx = i
			break
		}
	}
	if idx != -1 {
		// let's update the service to remove the port
		ports[idx] = ports[len(ports)-1]
		wstunSvc.Spec.Ports = ports[:len(ports)-1]
		wstunSvc, err = servicesClient.Update(wstunSvc)
		if err != nil {
			return
		}
	}
	return
}

// note: this function requires RBAC permission to be configured for the pod to work
func removeWstunServicePorts(context context.Context, ports []int32) (err error) {
	// use k8s API to remove ports from wstun svc
	var clientset *kubernetes.Clientset
	var wstunSvc *apiv1.Service
	clientset, err = util.NewK8sClientSet()
	if err != nil {
		return
	}
	nameSpace := util.GetPodNamespace()
	servicesClient := clientset.CoreV1().Services(nameSpace)
	wstunSvc, err = servicesClient.Get("wstun-svc", metav1.GetOptions{})
	if err != nil {
		return
	}
	// remove all ports with matching name
	rePortName := regexp.MustCompile(`ssh-port-(\d+)`)
	specPorts := wstunSvc.Spec.Ports
	var portsToKeep []apiv1.ServicePort
	for _, sp := range specPorts {
		keep := true
		m := rePortName.FindStringSubmatch(sp.Name)
		if len(m) == 2 {
			// format match
			ipt, _ := strconv.Atoi(m[1])
			port := int32(ipt)
			for _, pt := range ports {
				if pt == port {
					// port match
					keep = false
					break
				}
			}
		}
		if keep {
			portsToKeep = append(portsToKeep, sp)
		}
	}
	if len(portsToKeep) != len(specPorts) {
		// let's update the service to remove the ports
		wstunSvc.Spec.Ports = portsToKeep
		wstunSvc, err = servicesClient.Update(wstunSvc)
		if err != nil {
			return
		}
	}
	return
}

// Setup ssh tunneling to Edge, write response to writer
func (dbAPI *dbObjectModelAPI) SetupSSHTunnelingW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	doc := model.WstunRequest{}
	err := base.Decode(&r, &doc)
	if err != nil {
		return errcode.NewMalformedBadRequestError("body")
	}
	resp, err := dbAPI.SetupSSHTunneling(context, doc, callback)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp)
}

// Teardown ssh tunneling to Edge
func (dbAPI *dbObjectModelAPI) TeardownSSHTunneling(context context.Context, doc model.WstunTeardownRequest, callback func(context.Context, interface{}) error) error {
	options := teardownSSHTunnelingOptions{
		doc:            doc,
		skipValidation: false,
		duration:       wstunDuration,
		setupBasicAuth: false,
	}
	return dbAPI.teardownSSHTunneling(context, options, callback)
}
func (dbAPI *dbObjectModelAPI) teardownSSHTunneling(context context.Context, options teardownSSHTunnelingOptions, callback func(context.Context, interface{}) error) error {
	glog.V(5).Infof(base.PrefixRequestID(context, "teardownSSHTunneling: options=%+v\n"), options)
	doc := options.doc
	skipValidation := options.skipValidation
	d := options.duration
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}

	// ensure tenant id match auth
	tenantID := authContext.TenantID

	// must be infra admin
	if !skipValidation && !auth.IsInfraAdminRole(authContext) {
		return errcode.NewBadRequestError("permission")
	}

	// make sure service domain belong to tenant
	_, err = dbAPI.GetServiceDomain(context, doc.ServiceDomainID)
	if err != nil {
		return errcode.NewBadRequestError("svcDomain")
	}

	// only require edge connection for prod
	// otherwise would be difficult to test this API
	if dbAPI.prod {
		// make sure edge is connected
		if !IsEdgeConnected(tenantID, doc.ServiceDomainID) {
			return errcode.NewBadRequestError("edge/offline")
		}
	}

	port, err := wst.ReleasePort(dbAPI.GetRedisClient(), model.WstunTeardownRequestInternal{
		TenantID:        tenantID,
		ServiceDomainID: doc.ServiceDomainID,
		PublicKey:       doc.PublicKey,
		Endpoint:        doc.Endpoint,
	}, d)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(context, "failed to remove ssh port for edge: %s, error:%s\n"), doc.ServiceDomainID, err.Error())
	}

	glog.V(5).Infof(base.PrefixRequestID(context, "teardownSSHTunneling: release port returned port: %d\n"), port)

	if port != -1 {
		err = removeSSHTunnelingPort(context, port, options.setupBasicAuth)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(context, "failed to remove ssh port %d from wstun-svc or voyager ingress, error: %s\n"), port, err.Error())
		}
	}

	// invoke and wait for callback
	if callback != nil {
		err = callback(context, &doc)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(context, "failed to send ssh teardown message to edge for edge: %s\n"), doc.ServiceDomainID)
		}
		if !dbAPI.prod {
			// ignore callback err if not prod
			err = nil
		}
	}

	return err
}

// Teardown ssh tunneling to Edge, write response to writer
func (dbAPI *dbObjectModelAPI) TeardownSSHTunnelingW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	doc := model.WstunTeardownRequest{}
	err := base.Decode(&r, &doc)
	if err != nil {
		return errcode.NewMalformedBadRequestError("body")
	}
	return dbAPI.TeardownSSHTunneling(context, doc, callback)
}
