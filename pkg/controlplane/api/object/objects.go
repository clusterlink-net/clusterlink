package apiObject

import (
	"time"

	"github.ibm.com/mbg-agent/pkg/controlplane/eventmanager"
)

// ImportReply return dataplane information (port) about new import endpoint
type ImportReply struct {
	Id   string
	Port string
}

// HeartBeats
type HeartBeat struct {
	Id string
}

// ConnectRequest
type ConnectRequest struct {
	Id     string
	IdDest string
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
	SrcIp  string
	DestIp string
	DestId string
}

// New connection to import service reply
type NewImportConnParmaReply struct {
	Action string
	Target string
	SrcId  string
	ConnId string
}

// New connection to export service struct request
type NewExportConnParmaReq struct {
	SrcId   string
	SrcGwId string
	DestId  string
}

// New connection to import service struct reply
type NewExportConnParmaReply struct {
	Action          string
	SrcGwEndpoint   string
	DestSvcEndpoint string
	ConnId          string
}

// Connection Status
type ConnectionStatus struct {
	ConnectionId  string
	GlobalId      string // To be used to trace a flow across gateways
	IncomingBytes int
	OutgoingBytes int
	StartTstamp   time.Time
	LastTstamp    time.Time
	Direction     eventmanager.Direction
	State         eventmanager.ConnectionState
}
