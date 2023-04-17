/**********************************************************/
/* Package client contain function that run for
/* mbg client that can run inside the host, destination
/* and mbg
/**********************************************************/
package clusterProxy

import (
	"fmt"
	"io"
	"net"
	"sync"
)

var (
	maxDataBufferSize = 64 * 1024
)

type ProxyClient struct {
	Listener  string
	Target    string
	Name      string
	CloseConn chan bool
}

// Init client fields
func (c *ProxyClient) InitClient(listener, target, name string) {
	c.Listener = listener
	c.Target = target
	c.Name = name
	c.CloseConn = make(chan bool, 2)
}

// Run client object
func (c *ProxyClient) RunClient(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("[%v] start connection : listen: %v  send to : %v \n", c.Name, c.Listener, c.Target)

	done := &sync.WaitGroup{}
	done.Add(1)

	go c.acceptLoop()
	go c.waitToCloseSignal(done)

	done.Wait()
	fmt.Printf("[%v] Close connection \n", c.Name)
}

// Start listen loop and pass data to destination according to controlFrame
func (c *ProxyClient) acceptLoop() {
	// open listener
	acceptor, err := net.Listen("tcp", c.Listener)
	if err != nil {
		fmt.Printf("[%v] Error: %v\n", c.Name, err)
	}
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		fmt.Printf("[%v]: accept connection %v -> %v\n", c.Name, ac.LocalAddr().String(), ac.RemoteAddr().String())
		if err != nil {
			fmt.Println("Error:", err)
		}
		go c.dispatch(ac)
	}
}

// Connect to client and call ioLoop function
func (c *ProxyClient) dispatch(ac net.Conn) error {
	fmt.Printf("[%v]: before dial TCP %v\n", c.Name, c.Target)
	nodeConn, err := net.Dial("tcp", c.Target)
	fmt.Printf("[%v]: after dial TCP %v\n", c.Name, c.Target)
	if err != nil {
		return err
	}
	return c.ioLoop(ac, nodeConn)
}

// Transfer data from server to client and back
func (c *ProxyClient) ioLoop(cl, mbg net.Conn) error {
	defer cl.Close()
	defer mbg.Close()

	fmt.Printf("[%v] listen to: %v in port: %v \n", c.Name, cl.LocalAddr().String(), cl.RemoteAddr().String())
	fmt.Printf("[%v] send data to: %v from port: %v\n", c.Name, mbg.RemoteAddr().String(), mbg.LocalAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go c.clientToServer(done, cl, mbg)
	go c.serverToClient(done, cl, mbg)

	done.Wait()
	fmt.Printf("[%v] Connection close \n", c.Name)
	return nil
}

// Copy data from client to server and send control frame
func (c *ProxyClient) clientToServer(wg *sync.WaitGroup, cl, mbg net.Conn) error {
	defer wg.Done()
	var err error
	bufData := make([]byte, maxDataBufferSize)

	for {
		numBytes, err := cl.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[%v][clientToServer]: Read error %v\n", c.Name, err)
			}

			break
		}

		_, err = mbg.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[%v][clientToServer]: Write error %v\n", c.Name, err)
			break
		}
	}
	if err == io.EOF {
		return nil
	} else {
		return err
	}

}

// Copy data from server to client
func (c *ProxyClient) serverToClient(wg *sync.WaitGroup, cl, mbg net.Conn) error {
	defer wg.Done()

	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := mbg.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[%v][serverToClient]: Read error %v\n", c.Name, err)
			}
			break
		}
		_, err = cl.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[%v][serverToClient]: Write error %v\n", c.Name, err)
			break
		}
	}
	return err
}

func (c *ProxyClient) waitToCloseSignal(wg *sync.WaitGroup) {
	defer wg.Done()
	<-c.CloseConn
	//cl.Close() ,mbg.Close()- TBD -check if need to close also the internal connections
	fmt.Printf("[%v] Receive signal to close connection\n", c.Name)
}

func (c *ProxyClient) CloseConnection() {
	c.CloseConn <- true
}
