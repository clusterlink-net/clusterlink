/**********************************************************/
/* Package client contain function that run for
/* mbg client that can run inside the host, destination
/* and mbg
/**********************************************************/
package client

import (
	"fmt"
	"io"
	"net"
	"sync"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

var (
	maxDataBufferSize = 64 * 1024
)

type MbgClient struct {
	Listener       string
	Target         string
	SetupFrameFlag bool
	setupGwMode    bool
	hostService    service.Service
	destService    service.Service
}

//Init client fields
func (c *MbgClient) InitClient(listener, target string) {
	c.Listener = listener
	c.Target = target
}

//Run client object
func (c *MbgClient) RunClient() {
	fmt.Println("*** Start Client ***")
	fmt.Printf("Strart listen: %v  send to : %v \n", c.Listener, c.Target)

	err := c.acceptLoop()
	fmt.Println("Error:", err)
}

//Start listen loop and pass data to destination according to controlFrame
func (c *MbgClient) acceptLoop() error {
	// open listener
	acceptor, err := net.Listen("tcp", c.Listener)
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		fmt.Println("[client]: accept connetion", ac.LocalAddr().String(), "->", ac.RemoteAddr().String())
		if err != nil {
			return err
		}
		go c.dispatch(ac)
	}
}

//Connect to client and call ioLoop function
func (c *MbgClient) dispatch(ac net.Conn) error {
	fmt.Println("[client]: before dial TCP", c.Target)
	nodeConn, err := net.Dial("tcp", c.Target)
	fmt.Println("[client]: after dial TCP", c.Target)
	if err != nil {
		return err
	}
	return c.ioLoop(ac, nodeConn)
}

//Transfer data from server to client and back
func (c *MbgClient) ioLoop(cl, mbg net.Conn) error {
	defer cl.Close()
	defer mbg.Close()

	fmt.Println("[Cient] listen to:", cl.LocalAddr().String(), "in port:", cl.RemoteAddr().String())
	fmt.Println("[Cient] send data to:", mbg.RemoteAddr().String(), "from port:", mbg.LocalAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go c.clientToServer(done, cl, mbg)
	go c.serverToClient(done, cl, mbg)

	done.Wait()

	return nil
}

//Copy data from client to server and send control frame
func (c *MbgClient) clientToServer(wg *sync.WaitGroup, cl, mbg net.Conn) error {
	defer wg.Done()
	var err error
	bufData := make([]byte, maxDataBufferSize)

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
func (c *MbgClient) serverToClient(wg *sync.WaitGroup, cl, mbg net.Conn) error {
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
