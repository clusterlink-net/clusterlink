/**********************************************************/
/* TCP forwarder create bi directional TCP forwarding from client
/* to server
/**********************************************************/
package dataplane

import (
	"io"
	"net"
	"sync"
	"time"

	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
)

var (
	maxDataBufferSize = 64 * 1024
)

type TCPForwarder struct {
	Listener      string
	Target        string
	Name          string
	SeverConn     net.Conn // getting server handle in case of http connect
	ClientConn    net.Conn // getting client handle in case of http connect
	CloseConn     chan bool
	incomingBytes int
	outgoingBytes int
	direction     eventManager.Direction
	startTstamp   time.Time
}

// Init client fields
func (c *TCPForwarder) Init(listener, target, name string) {
	c.Listener = listener
	c.Target = target
	c.Name = name
}

func (c *TCPForwarder) SetServerConnection(server net.Conn) {
	c.SeverConn = server
}

func (c *TCPForwarder) SetClientConnection(client net.Conn) {
	c.ClientConn = client
	c.direction = eventManager.Incoming
}

// Run client object
func (c *TCPForwarder) RunTcpForwarder(direction eventManager.Direction) (int, int, time.Time, time.Time, error) {
	c.startTstamp = time.Now()
	clog.Infof("*** Start TCP Forwarder ***")
	clog.Infof("[%v] Start listen: %v  send to : %v \n", c.Name, c.Listener, c.Target)
	c.direction = direction
	err := c.acceptLoop(direction)
	return c.incomingBytes, c.outgoingBytes, c.startTstamp, time.Now(), err
}

// Start listen loop and pass data to destination according to controlFrame
func (c *TCPForwarder) acceptLoop(direction eventManager.Direction) error {
	if c.SeverConn != nil {
		c.dispatch(c.SeverConn, direction)
	} else {
		// open listener
		acceptor, err := net.Listen("tcp", c.Listener)
		if err != nil {
			clog.Errorln("Error:", err)
			return err
		}
		// loop until signalled to stop
		for {
			ac, err := acceptor.Accept()
			clog.Info("[", c.Name, "]: accept connetion", ac.LocalAddr().String(), "->", ac.RemoteAddr().String())
			if err != nil {
				clog.Errorln("Error:", err)
				return err
			}
			c.dispatch(ac, direction)
		}
	}
	return nil
}

// Connect to client and call ioLoop function
func (c *TCPForwarder) dispatch(ac net.Conn, direction eventManager.Direction) error {
	var nodeConn net.Conn
	if c.ClientConn == nil {
		var err error
		clog.Info("[", c.Name, "]: before dial TCP", c.Target)
		nodeConn, err = net.Dial("tcp", c.Target)
		clog.Info("[", c.Name, "]: after dial TCP", c.Target)
		if err != nil {
			return err
		}
	} else {
		nodeConn = c.ClientConn
	}
	return c.ioLoop(ac, nodeConn, direction)
}

// Transfer data from server to client and back
func (c *TCPForwarder) ioLoop(cl, server net.Conn, direction eventManager.Direction) error {
	defer cl.Close()
	defer server.Close()

	clog.Debug("[Cient] listen to:", cl.LocalAddr().String(), "in port:", cl.RemoteAddr().String())
	clog.Debug("[Cient] send data to:", server.RemoteAddr().String(), "from port:", server.LocalAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	if direction == eventManager.Incoming {
		go c.clientToServer(done, cl, server, eventManager.Incoming)
		go c.serverToClient(done, cl, server, eventManager.Outgoing)
	} else {
		go c.clientToServer(done, cl, server, eventManager.Outgoing)
		go c.serverToClient(done, cl, server, eventManager.Incoming)
	}

	done.Wait()

	return nil
}

// Copy data from client to server and send control frame
func (c *TCPForwarder) clientToServer(wg *sync.WaitGroup, cl, server net.Conn, direction eventManager.Direction) error {
	defer wg.Done()
	var err error
	bufData := make([]byte, maxDataBufferSize)

	for {
		var numBytes int

		numBytes, err = cl.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil // Ignore EOF error
			} else {
				clog.Infof("[clientToServer]: Read error %v\n", err)
			}

			break
		}
		if direction == eventManager.Incoming {
			c.incomingBytes += numBytes
		} else {
			c.outgoingBytes += numBytes
		}
		// Another point to apply policies
		_, err = server.Write(bufData[:numBytes])
		if err != nil {
			clog.Infof("[clientToServer]: Write error %v\n", err)
			break
		}
	}
	c.CloseConnection()
	if err == io.EOF {
		return nil
	} else {
		return err
	}

}

// Copy data from server to client
func (c *TCPForwarder) serverToClient(wg *sync.WaitGroup, cl, server net.Conn, direction eventManager.Direction) error {
	defer wg.Done()

	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		var numBytes int

		numBytes, err = server.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil // Ignore EOF error
			} else {
				clog.Infof("[serverToClient]: Read error %v\n", err)
			}
			break
		}
		if direction == eventManager.Incoming {
			c.incomingBytes += numBytes
		} else {
			c.outgoingBytes += numBytes
		}
		// Another point to apply policies
		_, err = cl.Write(bufData[:numBytes])
		if err != nil {
			clog.Infof("[serverToClient]: Write error %v\n", err)
			break
		}
	}
	c.CloseConnection()
	if err == io.EOF {
		return nil
	} else {
		return err
	}
}

// Close connections fo all net.Conn
func (c *TCPForwarder) CloseConnection() {
	if c.SeverConn != nil {
		c.SeverConn.Close()
	}
	if c.ClientConn != nil {
		c.ClientConn.Close()
	}

}

// Trigger close connection signal
func (c *TCPForwarder) CloseConnectionSignal() {
	c.CloseConn <- true
}
