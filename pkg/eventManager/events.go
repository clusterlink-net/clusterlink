package eventManager

const (
	Incoming = iota
	Outgoing
)

const Wildcard = "*"

const (
	Allow int = iota
	Deny
	AllowAll
	AllowPartial
)

const (
	NewConnectionRequest = "NewConnectionRequest"
	AddPeerRequest       = "AddPeerRequest"
	NewRemoteService     = "NewRemoteService"
	ExposeRequest        = "ExposeRequest"
)

type ConnectionRequestAttr struct {
	SrcService string
	DstService string
	Direction  int
	OtherMbg   string //Optional: Would not be set if its an outgoing connection
}

type ConnectionRequestResp struct {
	Action    int
	TargetMbg string
	BitRate   int // Mbps
}

type NewRemoteServiceAttr struct {
	Service string
	Mbg     string
}

type NewRemoteServiceResp struct {
	Action int
}

type ExposeRequestAttr struct {
	Service string
}

type ExposeRequestResp struct {
	Action     int
	TargetMbgs []string
}

type AddPeerAttr struct {
	PeerMbg string
}

type AddPeerResp struct {
	Action int
}

type ServiceListRequestAttr struct {
	SrcMbg string
}

type ServiceListRequestResp struct {
	Action   int
	Services []string
}

type ServiceRequestAttr struct {
	SrcMbg string
}

type ServiceRequestResp struct {
	Action int
}
