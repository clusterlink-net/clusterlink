/**********************************************************/
/* Package server contain function that run for
/* service node server that run inside the service node
/**********************************************************/
package server

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.ibm.com/ei-agent/pkg/client"
	"github.ibm.com/ei-agent/pkg/setupFrame"
)

var (
	maxDataBufferSize = 64 * 1024
)

type SnServer struct {
	Listener      string
	ServiceTarget string
	SnMode        bool
	SnClient      *client.SnClient
}

//Init server fields
func (s *SnServer) SrverInit(listener, servicenode string, snmode bool, client *client.SnClient) {
	s.Listener = listener
	s.ServiceTarget = servicenode
	s.SnMode = snmode
	s.SnClient = client
}

//Run server object
func (s *SnServer) RunSrver() {
	fmt.Println("********** Start Server ************")
	fmt.Printf("Strart listen: %v send to: %v \n", s.Listener, s.ServiceTarget)

	err := s.acceptLoop() // missing channel for signal handler
	fmt.Println("Error:", err)
}

//Start listen to client
func (s *SnServer) acceptLoop() error {
	// open listener
	acceptor, err := net.Listen("tcp", s.Listener)
	fmt.Println("[server] acceptLoop : before accept for ip", acceptor.Addr())
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		c, err := acceptor.Accept()
		fmt.Println("[server] acceptLoop : get accept")

		if err != nil {
			return err
		}
		go s.dispatch(c, s.ServiceTarget)
	}
}

//get client data and setupFrame and connect to service/destination
func (s *SnServer) dispatch(c net.Conn, servicenode string) error {
	//choose which sevice to pass
	setupPacket := setupFrame.GetSetupPacket(c)
	if s.SnMode { //For service node update the target
		if setupPacket.Service.Name == "Forward" {
			s.ServiceTarget = s.SnClient.Listener
			s.SnClient.Target = setupPacket.DestIp + ":" + setupPacket.DestPort
		} else if setupPacket.Service.Name == "TCP-split" {
			s.ServiceTarget = setupPacket.Service.Ip
			s.SnClient.Target = setupPacket.DestIp + ":" + setupPacket.DestPort
		} else {
			s.ServiceTarget = setupPacket.DestIp + ":" + setupPacket.DestPort
		}
	}
	fmt.Println("[server] before dial to:", s.ServiceTarget)
	nodeConn, err := net.Dial("tcp", s.ServiceTarget)
	fmt.Println("[server] after dial to:", s.ServiceTarget)
	if err != nil {
		return err
	}
	return s.ioLoop(c, nodeConn)
}

//Transfer data from server to client and back
func (s *SnServer) ioLoop(cl, sn net.Conn) error {
	defer cl.Close()
	defer sn.Close()

	fmt.Println("[server] listen to:", cl.LocalAddr().String(), "in port:", cl.RemoteAddr().String())
	fmt.Println("[server] send data to:", sn.RemoteAddr().String(), "from port:", sn.LocalAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go s.clientToServer(done, cl, sn)
	go s.serverToClient(done, cl, sn)

	done.Wait()

	return nil
}

//Copy data from client to server
func (s *SnServer) clientToServer(wg *sync.WaitGroup, cl, sn net.Conn) error {

	defer wg.Done()
	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := cl.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[clientToServer]: Read error %v\n", err)
			}

			break
		}

		_, err = sn.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[clientToServer]: Write error %v\n", err)
			break
		}
	}
	if err == io.EOF {
		return nil
	} else {
		return err
	}

}

//Copy data from server to client
func (s *SnServer) serverToClient(wg *sync.WaitGroup, cl, sn net.Conn) error {
	defer wg.Done()

	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := sn.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[serverToClient]: Read error %v\n", err)
			}
			break
		}
		_, err = cl.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[serverToClient]: Write error %v\n", err)
			break
		}
	}
	return err
}

//bufSetup := make([]byte, maxSetupBufferSize)

// allocate 4B frame-buffer and 64KB payload buffer
// forever {
//    read 4B into frame-buffer
//    if frame.Type == control { // not expected yet, except for error returns from SN
// 	     read and process control frame
//    } else {
// 	 	 read(sn, payload, frame.Len) // might require multiple reads and need a timeout deadline set
//	     send(cl, payload)
//    }
// }
