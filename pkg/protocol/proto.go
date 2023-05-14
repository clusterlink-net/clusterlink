/*
Name: protocol package
Desc: protocol package contain all struct for communicate (Request and Response)

	with the MBG
*/
package protocol

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

// Disconnect
type DisconnectRequest struct {
	Id     string
	IdDest string
	Policy string
}
