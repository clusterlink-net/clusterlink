/**********************************************************/
/* Package client contain function that run for
/* mbg client that can run inside the host, destination
/* and mbg
/**********************************************************/
package mbgDataplane

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "mbgDataplane/TCPForwarder")

var (
	maxDataBufferSize = 64 * 1024
)

type MbgTcpForwarder struct {
	Listener  string
	Target    string
	Name      string
	CloseConn chan bool
}

//Init client fields
func (c *MbgTcpForwarder) InitTcpForwarder(listener, target, name string) {
	c.Listener = listener
	c.Target = target
	c.Name = name
}

//Run client object
func (c *MbgTcpForwarder) RunTcpForwarder() {
	log.Infof("*** Start TCP Forwarder ***")
	log.Infof("Start listen: %v  send to : %v \n", c.Listener, c.Target)

	err := c.acceptLoop()
	fmt.Println("Error:", err)
}

//Start listen loop and pass data to destination according to controlFrame
func (c *MbgTcpForwarder) acceptLoop() error {
	// open listener
	acceptor, err := net.Listen("tcp", c.Listener)
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		log.Debugf("[client]: accept connetion", ac.LocalAddr().String(), "->", ac.RemoteAddr().String())
		if err != nil {
			return err
		}
		go c.dispatch(ac)
	}
}

//Connect to client and call ioLoop function
func (c *MbgTcpForwarder) dispatch(ac net.Conn) error {
	log.Debugf("[client]: before dial TCP", c.Target)
	nodeConn, err := net.Dial("tcp", c.Target)
	log.Debugf("[client]: after dial TCP", c.Target)
	if err != nil {
		return err
	}
	return c.ioLoop(ac, nodeConn)
}

//Transfer data from server to client and back
func (c *MbgTcpForwarder) ioLoop(cl, mbg net.Conn) error {
	defer cl.Close()
	defer mbg.Close()

	log.Debugf("[Cient] listen to:", cl.LocalAddr().String(), "in port:", cl.RemoteAddr().String())
	log.Debugf("[Cient] send data to:", mbg.RemoteAddr().String(), "from port:", mbg.LocalAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go c.clientToServer(done, cl, mbg)
	go c.serverToClient(done, cl, mbg)

	done.Wait()

	return nil
}

//Copy data from client to server and send control frame
func (c *MbgTcpForwarder) clientToServer(wg *sync.WaitGroup, cl, mbg net.Conn) error {
	defer wg.Done()
	var err error
	bufData := make([]byte, maxDataBufferSize)

	for {
		numBytes, err := cl.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				log.Infof("[clientToServer]: Read error %v\n", err)
			}

			break
		}
		// Another point to apply policies
		_, err = mbg.Write(bufData[:numBytes])
		if err != nil {
			log.Infof("[clientToServer]: Write error %v\n", err)
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
func (c *MbgTcpForwarder) serverToClient(wg *sync.WaitGroup, cl, mbg net.Conn) error {
	defer wg.Done()

	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := mbg.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				log.Infof("[serverToClient]: Read error %v\n", err)
			}
			break
		}
		// Another point to apply policies
		_, err = cl.Write(bufData[:numBytes])
		if err != nil {
			log.Infof("[serverToClient]: Write error %v\n", err)
			break
		}
	}
	return err
}

func (c *MbgTcpForwarder) waitToCloseSignal(wg *sync.WaitGroup) {
	defer wg.Done()
	<-c.CloseConn
	//cl.Close() ,mbg.Close()- TBD -check if need to close also the internal connections
	log.Infof("[%v] Receive signal to close connection\n", c.Name)
}

func (c *MbgTcpForwarder) CloseConnection() {
	c.CloseConn <- true
}
