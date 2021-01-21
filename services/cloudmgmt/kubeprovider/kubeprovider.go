package kubeprovider

import (
	"gopkg.in/yaml.v2"
)

const (
	defaultRKESSHKeyPath = "/home/admin/.ssh/id_rsa"
)

// Kubeprovider interface is to be implemented.
type Kubeprovider interface {
	//Name string
	// TODO: revisit definition to make it provider agnostic
	AddNode(address string, user string, roles []string) error
	GetConf() (string, error)
}

// RKENode are the node representation in Rancher
type RKENode struct {
	Address string   `yaml:"address"`
	User    string   `yaml:"user"`
	Role    []string `yaml:"role"`
}

// RKENetwork is the network representation in Rancher
type RKENetwork struct {
	Plugin string `yaml:"plugin"`
}

// RKEProvider implements Kubeprovider for rancher clusters.
type RKEProvider struct {
	config RancherKubernetesEngineConfig
}

// NewRKEProvider return a RKEProvider
func NewRKEProvider(name string) *RKEProvider {
	rkeProvider := &RKEProvider{}
	rkeProvider.config.ClusterName = name
	rkeProvider.config.SSHKeyPath = defaultRKESSHKeyPath
	return rkeProvider
}

// AddNode  is used to add nodes to the RKE conf file
func (rkeProvider *RKEProvider) AddNode(address string, user string, roles []string) error {
	rNode := RKEConfigNode{Address: address, User: user, Role: roles}
	rkeProvider.config.Nodes = append(rkeProvider.config.Nodes, rNode)
	return nil
}

// GetConf  is used to print the rke con
func (rkeProvider *RKEProvider) GetConf() (string, error) {
	conf, err := yaml.Marshal(&rkeProvider.config)
	if err != nil {
		return "", err
	}
	return string(conf), err
}
