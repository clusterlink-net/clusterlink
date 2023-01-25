/*
Name: protocol package
Desc: protocol package contain all struct for communicate (Request and Response)
	  with the MBG
*/
package protocol

//AddPeer
type PeerRequest struct {
	Id    string
	Ip    string
	Cport string
}

//Service
type ServiceRequest struct {
	Id    string
	Ip    string
	MbgID string
}

//Hello
type HelloResponse struct {
	Status string
}

//Expose
type ExposeRequest struct {
	Id    string
	Ip    string
	MbgID string
}

//Connect
type ConnectRequest struct {
	Id     string
	IdDest string
	Policy string
	MbgID  string
}

type ConnectReply struct {
	Message     string
	ConnectType string
	ConnectDest string
}

//Disconnect
type DisconnectRequest struct {
	Id     string
	IdDest string
	Policy string
}
