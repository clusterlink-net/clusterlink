/**********************************************************/
/* Package server contain function that run for
/* mbg server that run inside the mbg
/**********************************************************/
package mbgDataplane

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type MbgServer struct {
	Listener      string
	ServiceTarget string
	Name          string
}

//Init server fields
func (s *MbgServer) InitServer(listener, target, name string) {
	s.Listener = listener
	s.ServiceTarget = target
	s.Name = name

}

//Run server object
func (s *MbgServer) RunServer() {
	fmt.Printf("[%v] Start listen: %v send to: %v \n", s.Name, s.Listener, s.ServiceTarget)

	err := s.acceptLoop() // missing channel for signal handler
	fmt.Printf("[%v] Error: %v", s.Name, err)
}

//Start listen to client
func (s *MbgServer) acceptLoop() error {
	// open listener
	acceptor, err := net.Listen("tcp", s.Listener)
	fmt.Printf("[%v] acceptLoop : before accept for ip %v \n", s.Name, acceptor.Addr())
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		c, err := acceptor.Accept()
		fmt.Printf("[%v] acceptLoop : get accept \n", s.Name)

		if err != nil {
			return err
		}
		go s.dispatch(c, s.ServiceTarget)
	}
}

//get client data and controlFrame and connect to service/destination
func (s *MbgServer) dispatch(c net.Conn, mbg string) error {

	fmt.Printf("[%v] before dial to: %v \n", s.Name, s.ServiceTarget)
	nodeConn, err := net.Dial("tcp", s.ServiceTarget)
	fmt.Printf("[%v] after dial to: %v \n", s.Name, s.ServiceTarget)
	if err != nil {
		return err
	}
	return s.ioLoop(c, nodeConn)
}

//Transfer data from server to client and back
func (s *MbgServer) ioLoop(cl, mbg net.Conn) error {
	defer cl.Close()
	defer mbg.Close()

	fmt.Printf("[%v] listen to: %v in port: %v\n", s.Name, cl.LocalAddr().String(), cl.RemoteAddr().String())
	fmt.Printf("[%v] send data to: %v from port: %v \n", s.Name, mbg.RemoteAddr().String(), mbg.LocalAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go s.clientToServer(done, cl, mbg)
	go s.serverToClient(done, cl, mbg)

	done.Wait()

	return nil
}

//Copy data from client to server
func (s *MbgServer) clientToServer(wg *sync.WaitGroup, cl, mbg net.Conn) error {

	defer wg.Done()
	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := cl.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[%v][clientToServer]: Read error %v\n", s.Name, err)
			}

			break
		}

		_, err = mbg.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[%v][clientToServer]: Write error %v\n", s.Name, err)
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
func (s *MbgServer) serverToClient(wg *sync.WaitGroup, cl, mbg net.Conn) error {
	defer wg.Done()

	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := mbg.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[%v][serverToClient]: Read error %v\n", s.Name, err)
			}
			break
		}
		_, err = cl.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[%v][serverToClient]: Write error %v\n", s.Name, err)
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
// 	 	 read(mbg, payload, frame.Len) // might require multiple reads and need a timeout deadline set
//	     send(cl, payload)
//    }
// }
