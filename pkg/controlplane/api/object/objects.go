package apiobject

import (
	"time"

	"github.com/clusterlink-org/clusterlink/pkg/controlplane/eventmanager"
)

// ImportReply return dataplane information (port) about new import endpoint
type ImportReply struct {
	ID   string
	Port string
}

// HeartBeats
type HeartBeat struct {
	ID string
}

// ConnectRequest
type ConnectRequest struct {
	ID     string
	IDDest string
	Policy string
	MbgID  string
}

type ConnectReply struct {
	Connect     bool
	ConnectType string
	ConnectDest string
}

// New connection to import service request
type NewImportConnParmaReq struct {
	SrcIP  string
	DestIP string
	DestID string
}

// New connection to import service reply
type NewImportConnParmaReply struct {
	Action string
	Target string
	SrcID  string
	ConnID string
}

// New connection to export service struct request
type NewExportConnParmaReq struct {
	SrcID   string
	SrcGwID string
	DestID  string
}

// New connection to import service struct reply
type NewExportConnParmaReply struct {
	Action          string
	SrcGwEndpoint   string
	DestSvcEndpoint string
	ConnID          string
}

// Connection Status
type ConnectionStatus struct {
	ConnectionID  string
	GlobalID      string // To be used to trace a flow across gateways
	IncomingBytes int
	OutgoingBytes int
	StartTstamp   time.Time
	LastTstamp    time.Time
	Direction     eventmanager.Direction
	State         eventmanager.ConnectionState
}
