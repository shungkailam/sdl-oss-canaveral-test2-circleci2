package api

import (
	"cloudservices/common/base"
	"context"
	"net/http"

	"github.com/graarh/golang-socketio"
	"github.com/julienschmidt/httprouter"
)

// ResponseBase websocket message response format
type ResponseBase struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

// ObjectRequest message sent in websocket create/update notification
type ObjectRequest struct {
	RequestID string      `json:"requestId"`
	TenantID  string      `json:"tenantId"`
	Doc       interface{} `json:"doc"`
}

// DeleteRequest message sent in websocket delete notification
// swagger:model DeleteRequest
type DeleteRequest struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	ID string `json:"id"`
}

// WSMessagingService interface for Websocket messaging service
type WSMessagingService interface {
	// broadcast message to all edges for a tenant
	BroadcastMessage(ctx context.Context, tenantID string, msgName string, msg interface{})
	// send message to the given edge via Ack
	SendMessage(ctx context.Context, origin string, tenantID string, edgeID string, msgName string, msg interface{}) (string, error)
	// send message to the given edge via Ack. Wait for response synchronously before returning.
	SendMessageSync(ctx context.Context, origin string, tenantID string, edgeID string, msgName string, msg interface{}) (string, error)
	// send message to the given edge via Emit
	EmitMessage(ctx context.Context, origin string, tenantID string, edgeID string, msgName string, msg interface{}) error
	// get channel for the given edge
	GetChannel(tenantID string, edgeID string) *gosocketio.Channel
	// set channel for the given edge
	SetChannel(tenantID string, edgeID string, c *gosocketio.Channel)
	// remove the channel
	RemoveChannel(c *gosocketio.Channel)
	// get all connected edge ids for the given tenant
	GetConnectedEdgeIDs(tenantID string) []string
	// create httprouter handle for websocket
	MakeWebSocketHandler() httprouter.Handle
	// initialize edge related websocket handler
	InitEdge()
	// initialize sensor related websocket handler
	InitSensor()
	// initialize support log related websocket handler
	InitLog()
	// initialize application related websocket handler
	InitApplication()
	// initialize MLModel related websocket handler
	InitMLModel()
	// resync state to redis
	ResyncState()
	// whether the edge is connected via websocket to some cloudmgmt
	IsConnectedEdge(tenantID string, edgeID string) bool
	// whether the edges are connected via websocket to some cloudmgmt
	GetEdgeConnections(tenantID string, edgeIDs ...string) map[string]bool
	// proxy http call to edge over websocket
	SendHTTPRequest(ctx context.Context, tenantID string, edgeID string, req *http.Request, url string) (*http.Response, error)
}

var wsMsgService WSMessagingService

// SetWebsocketService sets the websocket instance
func SetWebsocketService(msgService WSMessagingService) {
	wsMsgService = msgService
}

// IsEdgeConnected returns the connection status of the edge.
func IsEdgeConnected(tenantID string, edgeID string) bool {
	if base.IsDemoTenantEdge(tenantID, edgeID) {
		return true
	}
	return wsMsgService != nil && wsMsgService.IsConnectedEdge(tenantID, edgeID)
}

// GetEdgeConnections returns the edge connections
func GetEdgeConnections(tenantID string, edgeIDs ...string) map[string]bool {
	connectionFlags := map[string]bool{}
	realEdgeIDs := make([]string, 0, len(edgeIDs))
	for _, edgeID := range edgeIDs {
		if base.IsDemoTenantEdge(tenantID, edgeID) {
			connectionFlags[edgeID] = true
		} else {
			realEdgeIDs = append(realEdgeIDs, edgeID)
		}
	}
	if wsMsgService != nil && len(realEdgeIDs) > 0 {
		realEdgeConnections := wsMsgService.GetEdgeConnections(tenantID, realEdgeIDs...)
		for key, value := range realEdgeConnections {
			connectionFlags[key] = value
		}
	}
	return connectionFlags
}
