package eventManager

import "time"

type Direction int

const (
	Incoming Direction = iota
	Outgoing
)

func (d Direction) String() string {
	return [...]string{"Incoming", "Outgoing"}[d]
}

type Action int

const (
	Allow Action = iota
	Deny
	AllowAll
	AllowPartial
)

type ConnectionState int

const (
	Ongoing ConnectionState = iota
	Complete
	Denied
	PeerDenied
)

func (a Action) String() string {
	return [...]string{"Allow", "Deny", "AllowAll", "AllowPartial"}[a]
}

const Wildcard = "*"

const (
	NewConnectionRequest = "NewConnectionRequest"
	ConnectionStatus     = "ConnectionStatus"
	AddPeerRequest       = "AddPeerRequest"
	NewRemoteService     = "NewRemoteService"
	ExposeRequest        = "ExposeRequest"
	RemovePeerRequest    = "RemovePeerRequest"
	RemoveRemoteService  = "RemoveRemoteService"
)

type ConnectionRequestAttr struct {
	SrcService string
	DstService string
	Direction  Direction
	OtherMbg   string //Optional: Would not be set if its an outgoing connection
}

type ConnectionRequestResp struct {
	Action    Action
	TargetMbg string
	BitRate   int // Mbps
}

type ConnectionStatusAttr struct {
	ConnectionId    string // Unique ID to track a connection from start to end within the gateway
	SrcService      string // Source application/service initiating the connection
	DstService      string // Destination application/service receiving the connection
	IncomingBytes   int
	OutgoingBytes   int
	DestinationPeer string // The peer where the destination/source service is located depending on the Direction
	StartTstamp     time.Time
	LastTstamp      time.Time
	Direction       Direction // Incoming/Outgoing
	State           ConnectionState
}

type NewRemoteServiceAttr struct {
	Service string
	Mbg     string
}

type RemoveRemoteServiceAttr struct {
	Service string
	Mbg     string
}

type NewRemoteServiceResp struct {
	Action Action
}

type ExposeRequestAttr struct {
	Service string
}

type ExposeRequestResp struct {
	Action     Action
	TargetMbgs []string
}

type AddPeerAttr struct {
	PeerMbg string
}

type AddPeerResp struct {
	Action Action
}

type RemovePeerAttr struct {
	PeerMbg string
}

type ServiceListRequestAttr struct {
	SrcMbg string
}

type ServiceListRequestResp struct {
	Action   Action
	Services []string
}

type ServiceRequestAttr struct {
	SrcMbg string
}

type ServiceRequestResp struct {
	Action Action
}
