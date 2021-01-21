package kubeprovider

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

const testOutput = `    nodes:
- address: 1.2.3.4
  role:
  - controlplane
  - etcd
  - worker
  user: admin
- address: 1.2.3.4
  role:
  - etcd
  - worker
  user: admin
- address: 7.8.9.10
  role:
  - controlplane
  user: admin
ssh_key_path: /home/admin/.ssh/id_rsa
ssh_agent_auth: false
ignore_docker_version: false
kubernetes_version: v1.14.1-rancher1-1
cluster_name: rkeTester
`

func TestRKEProvider(t *testing.T) {
	rkeProvider := NewRKEProvider("rkeTester")
	rkeProvider.AddNode("1.2.3.4", "admin", []string{"controlplane", "etcd", "worker"})
	rkeProvider.AddNode("1.2.3.4", "admin", []string{"etcd", "worker"})
	rkeProvider.AddNode("7.8.9.10", "admin", []string{"controlplane"})
	rkeProvider.config.Version = "v1.14.1-rancher1-1"
	conf, err := rkeProvider.GetConf()
	require.NoError(t, err)
	if strings.TrimSpace(conf) != strings.TrimSpace(testOutput) {
		t.Fatalf("\n%s does not match expected\n%s", conf, testOutput)
	}
}
