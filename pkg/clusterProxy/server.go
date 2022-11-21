/**********************************************************/
/* Package server contain function that run for
/* mbg server that run inside the mbg
/**********************************************************/
package clusterProxy

import (
	"fmt"
	"io"
	"net"
	"sync"
)


type ProxyServer struct {
	Listener      string
	ServiceTarget string
}

//Init server fields
func (s *ProxyServer) InitServer(listener, target string) {
	s.Listener = listener
	s.ServiceTarget = target

}

//Run server object
func (s *ProxyServer) RunServer() {
	fmt.Println("********** Start Server ************")
	fmt.Printf("Strart listen: %v send to: %v \n", s.Listener, s.ServiceTarget)

	err := s.acceptLoop() // missing channel for signal handler
	fmt.Println("Error:", err)
}

//Start listen to client
func (s *ProxyServer) acceptLoop() error {
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

//get client data and controlFrame and connect to service/destination
func (s *ProxyServer) dispatch(c net.Conn, mbg string) error {

	fmt.Println("[server] before dial to:", s.ServiceTarget)
	nodeConn, err := net.Dial("tcp", s.ServiceTarget)
	fmt.Println("[server] after dial to:", s.ServiceTarget)
	if err != nil {
		return err
	}
	return s.ioLoop(c, nodeConn)
}

//Transfer data from server to client and back
func (s *ProxyServer) ioLoop(cl, mbg net.Conn) error {
	defer cl.Close()
	defer mbg.Close()

	fmt.Println("[server] listen to:", cl.LocalAddr().String(), "in port:", cl.RemoteAddr().String())
	fmt.Println("[server] send data to:", mbg.RemoteAddr().String(), "from port:", mbg.LocalAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go s.clientToServer(done, cl, mbg)
	go s.serverToClient(done, cl, mbg)

	done.Wait()

	return nil
}

//Copy data from client to server
func (s *ProxyServer) clientToServer(wg *sync.WaitGroup, cl, mbg net.Conn) error {

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

		_, err = mbg.Write(bufData[:numBytes])
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
func (s *ProxyServer) serverToClient(wg *sync.WaitGroup, cl, mbg net.Conn) error {
	defer wg.Done()

	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := mbg.Read(bufData)
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
// 	 	 read(mbg, payload, frame.Len) // might require multiple reads and need a timeout deadline set
//	     send(cl, payload)
//    }
// }
