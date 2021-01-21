package crypto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	netx "cloudservices/common/net"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// References:
// https://blog.ralch.com/tutorial/golang-ssh-connection/

const (
	wsMsgCmd    = "cmd"
	wsMsgResize = "resize"
)

type wsMsg struct {
	Type string `json:"type"`
	Cmd  string `json:"cmd"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// Conn wraps a net.Conn, and sets a deadline for every read
// and write operation.
type Conn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (c *Conn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

// SSHDialTimeout similar to dial but with timeout apply to dial and read / write
// It also sends keepalive packets every 2 seconds to keep idle connection alive
func SSHDialTimeout(network, addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	conn, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}

	timeoutConn := &Conn{conn, timeout, timeout}
	c, chans, reqs, err := ssh.NewClientConn(timeoutConn, addr, config)
	if err != nil {
		return nil, err
	}
	client := ssh.NewClient(c, chans, reqs)

	// this sends keepalive packets every 2 seconds
	// there's no useful response from these, so we can just abort if there's an error
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for range t.C {
			_, _, err := client.Conn.SendRequest("keepalive@golang.org", true, nil)
			if err != nil {
				return
			}
		}
	}()
	return client, nil
}

// SetupSSH sets up ssh using the given info
func SetupSSH(user, privateKey, host string, port int) (*ssh.Client, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	return SSHDialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), config, 15*time.Second)
}

// RequestPtyForSSH request xterm for ssh session with default settings
func RequestPtyForSSH(session *ssh.Session) (err error) {
	modes := ssh.TerminalModes{
		ssh.ECHOE:         1,
		ssh.ECHOK:         1,
		ssh.ECHOKE:        1,
		ssh.ECHOCTL:       1,
		ssh.PENDIN:        1,
		ssh.CS8:           1,
		ssh.PARENB:        1,
		ssh.TTY_OP_ISPEED: 38400, // input speed = 38.4kbaud
		ssh.TTY_OP_OSPEED: 38400, // output speed = 38.4kbaud
	}
	return session.RequestPty("xterm", 40, 80, modes)
}

// GetStdIOE gets stdin, stdout, stderr from the ssh session
func GetStdIOE(session *ssh.Session) (stdin io.WriteCloser, stdout io.Reader, stderr io.Reader, err error) {
	stdin, err = session.StdinPipe()
	if err != nil {
		return
	}
	stdout, err = session.StdoutPipe()
	if err != nil {
		return
	}
	stderr, err = session.StderrPipe()
	return
}

// PipeSSHSessionToWS pipe the ssh stdin, stdout, stderr to the given websocket connection
// This function will wait till connection is done.
func PipeSSHSessionToWS(session *ssh.Session, uc *netx.Conn, stdin io.WriteCloser, stdout, stderr io.Reader) {
	var waitgroup sync.WaitGroup
	waitgroup.Add(1)
	go func() {
		// can't simply do io.Copy here since we want
		// to handle ws resize message in addition to data
		// io.Copy(stdin, uc)
		// waitgroup.Done()
		wsConn := uc.GetWebsocketConn()
		for {
			//read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				glog.Errorf("WS> Error> read message: %v\n", err)
				break
			}
			//unmashal bytes into struct
			msgObj := wsMsg{}
			if err := json.Unmarshal(wsData, &msgObj); err != nil {
				glog.Errorf("WS> Error> unmarshal: %v\n", err)
				break
			}
			switch msgObj.Type {
			case wsMsgResize:
				// handle xterm.js size change
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := session.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						glog.Errorf("WS> Error> WindowChange: %v\n", err)
						break
					}
				}
			case wsMsgCmd:
				// handle xterm.js stdin
				decodeBytes, err := base64.StdEncoding.DecodeString(msgObj.Cmd)
				if err != nil {
					glog.Errorf("WS> Error> decode string: %v\n", err)
					break
				}
				if _, err := stdin.Write(decodeBytes); err != nil {
					glog.Errorf("WS> Error> write failed: %v\n", err)
					break
				}
			}
		}
		waitgroup.Done()
	}()
	// go io.Copy(uc, stdout)
	// go io.Copy(uc, stderr)
	go func() {
		var err error
		cout := chanFromReader(stdout)
		cerr := chanFromReader(stderr)
		pingTicker := time.NewTicker(15 * time.Second)
		defer func() {
			pingTicker.Stop()
		}()
		wsConn := uc.GetWebsocketConn()
		for {
			select {
			case oData := <-cout:
				if oData == nil {
					return
				}
				_, err = uc.Write(oData)
				if err != nil {
					return
				}
			case eData := <-cerr:
				if eData == nil {
					return
				}
				_, err = uc.Write(eData)
				if err != nil {
					return
				}
			case <-pingTicker.C:
				err = wsConn.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					return
				}
			}
		}
	}()
	waitgroup.Wait()
	glog.Infoln("Closing ssh session and ws connection")
	session.Close()
	uc.Close()
}

func chanFromReader(r io.Reader) chan []byte {
	c := make(chan []byte)

	go func() {
		for {
			b := make([]byte, 1024)
			n, err := r.Read(b)
			if n > 0 {
				c <- b[:n]
			}
			if err != nil {
				c <- nil
				break
			}
		}
	}()

	return c
}
