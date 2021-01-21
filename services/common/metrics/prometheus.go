package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DBConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_connection_count",
			Help: "Number of DB connections.",
		},
		[]string{"hostname"},
	)

	WebSocketConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "websocket_connection_count",
			Help: "Number of websocket connections.",
		},
		[]string{"hostname", "tenant_id", "edge_id"},
	)

	WebSocketMessageCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_message_count",
			Help: "Number of websocket messages.",
		},
		[]string{"hostname", "message_name", "tenant_id", "edge_id"},
	)

	// count of websocket message going through redis
	WebSocketFederatedMessageCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_federated_message_count",
			Help: "Number of federated websocket messages.",
		},
		[]string{"hostname", "message_name", "tenant_id", "edge_id"},
	)

	RESTAPITime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "rest_api_time_seconds",
			Help: "Response time in seconds of REST API calls.",
		},
		[]string{"hostname", "method", "path"},
	)
	GRPCCallCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_call_count",
			Help: "Number of GRPC calls.",
		},
		[]string{"hostname", "method"},
	)
	CreatingTrialTenants = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trial_tenants_creating_count",
			Help: "Number of creating trial tenants.",
		},
		[]string{"registration"},
	)
	AssignedTrialTenants = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trial_tenants_assigned_count",
			Help: "Number of assigned trial tenants.",
		},
		[]string{"registration"},
	)
	AvailableTrialTenants = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trial_tenants_available_count",
			Help: "Number of available trial tenants.",
		},
		[]string{"registration"},
	)
	FailedTrialTenants = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trial_tenants_failed_count",
			Help: "Number of failed trial tenants.",
		},
		[]string{"registration"},
	)
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(DBConnections)
	prometheus.MustRegister(WebSocketConnections)
	prometheus.MustRegister(WebSocketMessageCount)
	prometheus.MustRegister(WebSocketFederatedMessageCount)
	prometheus.MustRegister(RESTAPITime)
	prometheus.MustRegister(GRPCCallCount)
	prometheus.MustRegister(CreatingTrialTenants)
	prometheus.MustRegister(AssignedTrialTenants)
	prometheus.MustRegister(AvailableTrialTenants)
	prometheus.MustRegister(FailedTrialTenants)
}
