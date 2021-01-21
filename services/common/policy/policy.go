// Policy manager with policies stored at nodes in a tree.
// Polices at lower nodes (farthest from the root) override the ones matched higher up in the tree
// such that the matching is like a longest prefix match
// https://docs.google.com/document/d/1Mw-OlxEPEWGp3GHC35jQc0YS-5NiwJ0PHUtSRnTUuKs/edit?ts=5f591713#heading=h.gkrlcvnj6iwd

package policy

import (
	"cloudservices/common/errcode"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/golang/glog"
)

// Node is a node for an n-ary tree to represent each component in a policy path
type Node struct {
	ID         string           `json:"id"`
	IDRegex    *regexp.Regexp   `json:"-"`
	PolicyName string           `json:"policyName,omitempty"`
	ChildNodes map[string]*Node `json:"childNodes,omitempty"`
}

// Manager manages policies
type Manager struct {
	mutex *sync.RWMutex
	root  *Node
}

// NewManager returns a policy manager
func NewManager() *Manager {
	return &Manager{
		mutex: &sync.RWMutex{},
		root:  &Node{ID: "Root"},
	}
}

// Policy is the policy for JSON representation
type Policy struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// Policies ...
type Policies []*Policy

// PathComponents returns the path components in the policy
func (policy *Policy) PathComponents() []string {
	tokens := strings.Split(policy.Path, "/")
	pathComponents := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		pathComponents = append(pathComponents, token)
	}
	return pathComponents
}

// PolicyName returns the policy name
func (policy *Policy) PolicyName() string {
	return strings.TrimSpace(policy.Name)
}

// SetPolicy sets policy for given path components from the node.
// If the path components parameter is empty, the policyName is set for the node
func (node *Node) SetPolicy(pathComponents []string, policyName string) error {
	if policyName == "" {
		return errcode.NewBadRequestError("policyName")
	}
	if len(pathComponents) == 0 {
		node.PolicyName = policyName
		return nil
	}
	currPathComponent := pathComponents[0]
	if node.ChildNodes == nil {
		node.ChildNodes = map[string]*Node{}
	}
	var childNode *Node
	if len(node.ChildNodes) > 0 {
		if cNode, ok := node.ChildNodes[currPathComponent]; ok {
			childNode = cNode
		}
	}
	if childNode == nil {
		childNode = &Node{ID: currPathComponent}
		// Regex detection. * must be used for regex
		if strings.Contains(currPathComponent, "*") {
			pathRegexp, err := regexp.Compile(currPathComponent)
			if err != nil {
				return errcode.NewBadRequestExError("policyPath", fmt.Sprintf("Error in compiling regex %s. Error: %s", currPathComponent, err.Error()))
			}
			childNode.IDRegex = pathRegexp
		}
		node.ChildNodes[childNode.ID] = childNode
	}
	return childNode.SetPolicy(pathComponents[1:], policyName)
}

// GetPolicy returns policy for the path components
func (node *Node) GetPolicy(pathComponents []string) (string, error) {
	var policyName string
	if len(pathComponents) == 0 {
		return policyName, nil
	}
	currPathComponent := pathComponents[0]
	// No match from this node onwards
	if node.ID != currPathComponent && (node.IDRegex == nil || !node.IDRegex.MatchString(currPathComponent)) {
		return policyName, nil
	}
	// Get the policy from the current node
	policyName = node.PolicyName
	if len(node.ChildNodes) == 0 || len(pathComponents) == 1 {
		return policyName, nil
	}

	// Consider child nodes with the remaining path components
	pathComponents = pathComponents[1:]
	nextPathComponent := pathComponents[0]
	// Give preference to direct match
	childNode, ok := node.ChildNodes[nextPathComponent]
	if ok {
		lowerPolicyName, err := childNode.GetPolicy(pathComponents)
		if err != nil {
			return "", err
		}
		if lowerPolicyName != "" {
			policyName = lowerPolicyName
		}
	} else {
		for _, childNode := range node.ChildNodes {
			lowerPolicyName, err := childNode.GetPolicy(pathComponents)
			if err != nil {
				return "", err
			}
			if lowerPolicyName != "" {
				// Override the current policy name with the policy found at inner/lower/farther node
				policyName = lowerPolicyName
				break
			}
		}
	}
	return policyName, nil
}

// LoadPolicies loads from a policy file
func (mgr *Manager) LoadPolicies(policies Policies) error {
	root := &Node{ID: "Root"}
	for _, policy := range policies {
		policyName := policy.PolicyName()
		if policyName == "" {
			continue
		}
		pathComponents := policy.PathComponents()
		if len(pathComponents) == 0 {
			continue
		}
		err := root.SetPolicy(pathComponents, policyName)
		if err != nil {
			glog.Errorf("Error in setting policy %s for %+v. Error: %s", policyName, pathComponents, err.Error())
			return err
		}
	}
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	mgr.root = root
	return nil
}

// SetPolicy sets policies. The path looks like
// <node>/
func (mgr *Manager) SetPolicy(policy Policy) error {
	policyName := policy.PolicyName()
	if policyName == "" {
		return errcode.NewBadRequestError("policyName")
	}
	pathComponents := policy.PathComponents()
	if len(pathComponents) == 0 {
		return errcode.NewBadRequestError("pathComponents")
	}
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	return mgr.root.SetPolicy(pathComponents, policyName)
}

// GetPolicy returns the policy associated with the path components
func (mgr *Manager) GetPolicy(policyPath string) (*Policy, error) {
	policy := &Policy{Path: policyPath}
	policyName, err := mgr.GetPolicyName(policy.PathComponents())
	if err != nil {
		return nil, err
	}
	policy.Name = policyName
	return policy, nil
}

// GetPolicyName returns the policy name for the given path components
func (mgr *Manager) GetPolicyName(pathComponents []string) (string, error) {
	policyName := ""
	if len(pathComponents) == 0 {
		return policyName, errcode.NewBadRequestError("pathComponents")
	}
	currPathComponent := pathComponents[0]
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	childNodes := mgr.root.ChildNodes
	if len(childNodes) == 0 {
		return policyName, nil
	}
	var err error
	// Give preference to direct match
	childNode, ok := childNodes[currPathComponent]
	if ok {
		policyName, err = childNode.GetPolicy(pathComponents)
		if err != nil {
			return "", err
		}
	} else {
		for _, childNode := range childNodes {
			policyName, err = childNode.GetPolicy(pathComponents)
			if err != nil {
				return policyName, err
			}
			if policyName != "" {
				break
			}
		}
	}
	return policyName, nil
}

// DumpPolicies writes out the policies for debugging
func (mgr *Manager) DumpPolicies() {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	ba, _ := json.MarshalIndent(mgr.root, " ", " ")
	glog.Infof(string(ba))
}
