/*
Name: protocol package
Desc: protocol package contain all struct for communicate (Request and Response)
	  with the MBG
*/
package protocol

//AddService
type AddServiceRequest struct {
	Id     string
	Ip     string
	Domain string
}

//Expose
type ExposeRequest struct {
	Id     string
	Ip     string
	Domain string
	MbgID  string
}

//Hello
type HelloRequest struct {
	Id    string
	Ip    string
	Cport string
}

type HelloResponse struct {
	Status string
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
