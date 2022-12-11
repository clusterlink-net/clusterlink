/*
Name: protocol package
Desc: protocol package contain all struct for communicate (Request and Response)
	  with the MBG
*/
package protocol

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
