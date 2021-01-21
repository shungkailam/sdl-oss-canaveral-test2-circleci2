package api

import (
	"cloudservices/common/policy"
)

var (
	// https://docs.google.com/document/d/1Mw-OlxEPEWGp3GHC35jQc0YS-5NiwJ0PHUtSRnTUuKs/edit?ts=5f591713#heading=h.gkrlcvnj6iwd

	// Policies are defined at the path level. Components in the path form nodes in a tree root at the first component node.
	// The policy at the matching lower node (farthest from root) overrides policies found at matching preceding components nodes.
	// It behaves like a longest prefix matching with components as the prefixes
	eventPathPolicies = policy.Policies{
		// Default for events, policies at longer matching components override these
		{Name: InfraEventAudience, Path: "/serviceDomain:.*"},
		{Name: ProjectEventAudience, Path: "/serviceDomain:.*/project:.*"},

		// Overrides for specific event paths
		{Name: InfraProjectEventAudience, Path: "/serviceDomain:.*/project:.*/service:.*/instance:.*/binding:.*/status"},
		{Name: InfraProjectEventAudience, Path: "/serviceDomain:.*/project:.*/service:.*/instance:.*/status"},

		// Prometheus specific overrides
		{Name: InfraProjectEventAudience, Path: "/serviceDomain:.*/project:.*/service:Prometheus/instance:.*/alertmanager/endpoint/health/status"},
		{Name: InfraProjectEventAudience, Path: "/serviceDomain:.*/project:.*/service:Prometheus/instance:.*/prometheus/endpoint/health/status"},
		{Name: InfraEventAudience, Path: "/serviceDomain:.*/project:.*/service:Prometheus/instance:.*/servicemonitor/deployment/status"},
	}
)
