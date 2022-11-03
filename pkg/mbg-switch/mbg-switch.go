/**********************************************************/
/* Package server contain function that run for
/* mbg server that run inside the mbg
/**********************************************************/
package mbgSwitch

import (
	"fmt"

	"github.ibm.com/mbg-agent/pkg/server"
)

type MbgSwitch struct {
	server *server.MbgServer
}

//Init server fields
func (s *MbgSwitch) InitMbgSwitch(listener, target string) {
	s.server = new(server.MbgServer)

	s.server.InitServer(listener, target)
}

//Run server object
func (s *MbgSwitch) RunMbgSwitch() {
	fmt.Println("********** Start MBG switch ************")
	s.server.RunServer()
}
