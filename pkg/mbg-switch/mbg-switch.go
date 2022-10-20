/**********************************************************/
/* Package server contain function that run for
/* mbg server that run inside the mbg
/**********************************************************/
package mbgSwitch

import (
	"fmt"

	"github.ibm.com/mbg-agent/pkg/client"
	"github.ibm.com/mbg-agent/pkg/server"
)

type MbgSwitch struct {
	server *server.MbgServer
}

//Init server fields
func (s *MbgSwitch) InitMbgSwitch(listener, mbg string, mbgmode bool, client *client.MbgClient) {
	s.server = new(server.MbgServer)

	s.server.InitServer(listener, mbg, mbgmode, client)
}

//Run server object
func (s *MbgSwitch) RunMbgSwitch() {
	fmt.Println("********** Start MBG switch ************")
	s.server.RunServer()
}
