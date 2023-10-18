// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
)

const (
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
	direction     eventmanager.Direction
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
	c.direction = eventmanager.Incoming
}

// Run client object
func (c *TCPForwarder) RunTCPForwarder(direction eventmanager.Direction) (int, int, time.Time, time.Time, error) {
	c.startTstamp = time.Now()
	clog.Infof("*** Start TCP Forwarder ***")
	clog.Infof("[%v] Start listen: %v  send to : %v \n", c.Name, c.Listener, c.Target)
	c.direction = direction
	err := c.acceptLoop(direction)
	return c.incomingBytes, c.outgoingBytes, c.startTstamp, time.Now(), err
}

// Start listen loop and pass data to destination according to controlFrame
func (c *TCPForwarder) acceptLoop(direction eventmanager.Direction) error {
	if c.SeverConn != nil {
		if err := c.dispatch(c.SeverConn, direction); err != nil {
			clog.Errorln("failed to dispatch server connection:", err) // TODO: close connection
		}
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
			clog.Info("[", c.Name, "]: accept connection", ac.LocalAddr().String(), "->", ac.RemoteAddr().String())
			if err != nil {
				clog.Errorln("Error:", err)
				return err
			}
			if err = c.dispatch(ac, direction); err != nil {
				clog.Errorln("failed to dispatch:", err)
			}
		}
	}
	return nil
}

// Connect to client and call ioLoop function
func (c *TCPForwarder) dispatch(ac net.Conn, direction eventmanager.Direction) error {
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
func (c *TCPForwarder) ioLoop(cl, server net.Conn, direction eventmanager.Direction) error {
	defer cl.Close()
	defer server.Close()

	clog.Debug("[Cient] listen to:", cl.LocalAddr().String(), "in port:", cl.RemoteAddr().String())
	clog.Debug("[Cient] send data to:", server.RemoteAddr().String(), "from port:", server.LocalAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	// TODO: handle errors in connection dispatch/handling
	if direction == eventmanager.Incoming {
		go func() { _ = c.clientToServer(done, cl, server, eventmanager.Incoming) }()
		go func() { _ = c.serverToClient(done, cl, server, eventmanager.Outgoing) }()
	} else {
		go func() { _ = c.clientToServer(done, cl, server, eventmanager.Outgoing) }()
		go func() { _ = c.serverToClient(done, cl, server, eventmanager.Incoming) }()
	}

	done.Wait()

	return nil
}

// Copy data from client to server and send control frame
func (c *TCPForwarder) clientToServer(wg *sync.WaitGroup, cl, server net.Conn, direction eventmanager.Direction) error {
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
		if direction == eventmanager.Incoming {
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
	}

	return err
}

// Copy data from server to client
func (c *TCPForwarder) serverToClient(wg *sync.WaitGroup, cl, server net.Conn, direction eventmanager.Direction) error {
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
		if direction == eventmanager.Incoming {
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
	}

	return err
}

// CloseConnection close the connections to all net.Conn
func (c *TCPForwarder) CloseConnection() {
	if c.SeverConn != nil {
		c.SeverConn.Close()
	}
	if c.ClientConn != nil {
		c.ClientConn.Close()
	}
}

// CloseConnectionSignal - trigger close connection signal
func (c *TCPForwarder) CloseConnectionSignal() {
	c.CloseConn <- true
}
