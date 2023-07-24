package apiObject

import (
	"time"

	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
)

// AddPeer
type PeerRequest struct {
	Id    string
	Ip    string
	Cport string
}

// Remove Peer
type PeerRemoveRequest struct {
	Id        string
	Propagate bool
}

// Service Add
type ServiceRequest struct {
	Id          string
	Ip          string
	Port        string
	Description string
	MbgID       string
}
type ServiceReply struct { //Retrun open port for remote service
	Id   string
	Port string
}

// Service Delete
type ServiceDeleteRequest struct {
	Id   string
	Peer string
}

// Hello
type HelloResponse struct {
	Status string
}

// Hello - HeartBeats
type HeartBeat struct {
	Id string
}

// Expose
type ExposeRequest struct {
	Id          string
	Ip          string
	Description string
	MbgID       string
}

// Service Binding request
type BindingRequest struct {
	Id        string
	Port      int
	Name      string
	Namespace string
	MbgApp    string
}

// Connect
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
	Direction     eventManager.Direction
	State         eventManager.ConnectionState
}
