package eventManager

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

func (a Action) String() string {
	return [...]string{"Allow", "Deny", "AllowAll", "AllowPartial"}[a]
}

const Wildcard = "*"

const (
	NewConnectionRequest = "NewConnectionRequest"
	AddPeerRequest       = "AddPeerRequest"
	NewRemoteService     = "NewRemoteService"
	ExposeRequest        = "ExposeRequest"
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

type NewRemoteServiceAttr struct {
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
