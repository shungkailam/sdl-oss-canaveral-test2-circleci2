package ssh

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"

	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/base"
	"cloudservices/common/crypto"

	netx "cloudservices/common/net"

	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

const (
	longWait        = 30 * time.Minute
	readBufferSize  = 1024
	writeBufferSize = 1024
)

type sshConnectionInfo struct {
	ServiceDomainID string
	Token           string
	Port            int
	PrivateKey      string
}

// message format:
// SSH-SETUP|<service domain id>|<token>|<port>|<private key>
func parseWsMsg(msg string) (sshConnectionInfo, error) {
	fmt.Printf("parseWsMsg: %s\n", msg)
	info := sshConnectionInfo{}
	ss := strings.Split(msg, "|")
	if len(ss) != 5 || ss[0] != "SSH-SETUP" {
		return info, fmt.Errorf("parseWsMsg: bad ws message %s", msg)
	}
	port, err := strconv.Atoi(ss[3])
	if err != nil {
		return info, err
	}
	info.ServiceDomainID = ss[1]
	info.Token = ss[2]
	info.Port = port
	info.PrivateKey = ss[4]
	return info, nil
}

func ConfigureWSSSHService(dbAPI api.ObjectModelAPI, router *httprouter.Router, redisClient *redis.Client) {
	handler := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		glog.Infoln("WS> {")

		uc, err := netx.NewConn(w, r, nil, readBufferSize, writeBufferSize)
		if err != nil {
			glog.Errorf("WS> Error> upgrade: %v\n", err)
			return
		}

		defer func() {
			if err != nil {
				uc.Close()
			}
		}()

		uc.SetWriteDeadline(time.Now().Add(longWait))
		uc.SetReadDeadline(time.Now().Add(longWait))

		// read first message from websocket
		mt, message, err := uc.GetWebsocketConn().ReadMessage()
		if err != nil {
			glog.Errorf("WS> Error> read: %v\n", err)
			return
		}
		if mt != websocket.TextMessage {
			err = fmt.Errorf("Unexpected websocket message")
			glog.Errorln("WS> Warning> unexpected non-text first message, bail")
			return
		}
		tm := string(message)
		info, err := parseWsMsg(tm)
		if err != nil {
			glog.Errorf("WS> Error> parseWsMsg: %v\n", err)
			return
		}

		// TODO: add validation of token, check ssh session, etc.
		claims, err := crypto.VerifyJWT(info.Token)
		if err != nil {
			glog.Errorf("WS> Error> failed to verify auth token: %v\n", err)
			return
		}
		role, ok := claims["specialRole"].(string)
		if !ok {
			glog.Errorf("WS> Error> failed to get specialRole from auth token: %v\n", claims)
			return
		}
		if role != "admin" {
			glog.Errorf("WS> Error> infra admin role required for ssh access: %v\n", claims)
			return
		}

		tenantID, ok := claims["tenantId"].(string)
		if !ok {
			glog.Errorf("WS> Error> failed to get tenantID from auth token: %v\n", claims)
			return
		}
		// verify edge belong to the tenant
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims:   claims,
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		_, err = dbAPI.GetServiceDomain(ctx, info.ServiceDomainID)
		if err != nil {
			glog.Errorf("WS> Error> failed to get service domain with ID %s: %v\n", info.ServiceDomainID, err)
			return
		}

		client, err := crypto.SetupSSH(*config.Cfg.SSHUser, info.PrivateKey, *config.Cfg.WstunHost, info.Port)
		if err != nil {
			glog.Errorf("WS> Error> setup ssh failed: %v\n", err)
			return
		}

		// Each ClientConn can support multiple interactive sessions,
		// represented by a Session.
		session, err := client.NewSession()
		if err != nil {
			glog.Errorf("WS> Error> client.NewSession: %v\n", err)
			return
		}

		defer func() {
			if err != nil {
				glog.Errorf("WS> Error> %v\n", err)
				session.Close()
			}
		}()

		err = crypto.RequestPtyForSSH(session)
		if err != nil {
			return
		}

		stdin, stdout, stderr, err := crypto.GetStdIOE(session)
		if err != nil {
			return
		}

		err = session.Shell()
		if err != nil {
			return
		}

		go crypto.PipeSSHSessionToWS(session, uc, stdin, stdout, stderr)
	}
	router.Handle("GET", "/ssh", handler)
}
