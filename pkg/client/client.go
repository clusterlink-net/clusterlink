package client

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.ibm.com/ei-agent/pkg/setupFrame"
)

var (
	maxDataBufferSize = 64 * 1024
)

type SnClient struct {
	Listener       string
	Target         string
	SetupFrameFlag bool
	AppDestPort    string
	AppDestIp      string
	ServiceType    string
}

func (c *SnClient) InitClient(listener, target string, setupFrameFlag bool, appDestPort, appDestIp, serviceType string) {
	c.Listener = listener
	c.Target = target
	c.SetupFrameFlag = setupFrameFlag
	c.AppDestPort = appDestPort
	c.AppDestIp = appDestIp
	c.ServiceType = serviceType

}

func (c *SnClient) RunClient() {
	fmt.Println("********** Start Client ************")
	fmt.Printf("Strart client listen: %v  send to server: %v \n", c.Listener, c.Target)

	err := c.acceptLoop()
	fmt.Println("Error:", err)
}

func (c *SnClient) acceptLoop() error {
	// open listener
	acceptor, err := net.Listen("tcp", c.Listener)
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		if err != nil {
			return err
		}
		go c.dispatch(ac)
	}
}

func (c *SnClient) dispatch(ac net.Conn) error {
	fmt.Println("[ClientDispatch] Target is", c.Target)
	nodeConn, err := net.Dial("tcp", c.Target)
	if err != nil {
		return err
	}
	return c.ioLoop(ac, nodeConn)
}

func (c *SnClient) ioLoop(cl, sn net.Conn) error {
	defer cl.Close()
	defer sn.Close()

	fmt.Println("Cient", cl.RemoteAddr().String(), "->", cl.LocalAddr().String())
	fmt.Println("Server", sn.LocalAddr().String(), "->", sn.RemoteAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go c.clientToServer(done, cl, sn)
	go c.serverToClient(done, cl, sn)

	done.Wait()

	return nil
}

func (c *SnClient) clientToServer(wg *sync.WaitGroup, cl, sn net.Conn) error {

	defer wg.Done()
	var err error

	if c.SetupFrameFlag {
		setupFrame.SendFrame(cl, sn, c.AppDestIp, c.AppDestPort, c.ServiceType) //Need to check performance impact
		fmt.Printf("[clientToServer]: Finish send SetupFrame \n")
	}
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

func (c *SnClient) serverToClient(wg *sync.WaitGroup, cl, sn net.Conn) error {
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
