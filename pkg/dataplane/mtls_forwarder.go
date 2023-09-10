/**********************************************************/
/* mTLS Forwader : This is created per service-pair connections.
/**********************************************************/
// Workflow of mTLS forwarder usage
// After Expose of a service at MBG 1 run the following APIs :
//    1) StartLocalService for the exported service at other remote application (for e.g. App 2)
//    2) When LocalService receives an accepted connection from APP 2, Do a Connect API to APP 1
//    3) MBG1 starts a StartReceiverService with the necessary details such as endpoint, and sends it as Connect Response

package dataplane

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"time"

	"github.com/go-chi/chi"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/utils/netutils"
)

type MTLSForwarder struct {
	Name            string
	connectionToken string
	Connection      net.Conn
	MTLSConnection  net.Conn
	ChiRouter       *chi.Mux
	incomingBytes   int
	outgoingBytes   int
	startTstamp     time.Time
}

const (
	ConnectionURL = "/connectionData/"
)

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(_, _ string) (net.Conn, error) {
	return cd.c, nil
}

// Start MTLS Forwarder on a specific MTLS target
// targetIPPort in the format of <ip:port>
// connect is set to true on a client side
func (m *MTLSForwarder) StartMTLSForwarderClient(targetIPPort, name, certca, certificate, key, serverName string, endpointConn net.Conn) (int, int, time.Time, time.Time, error) {
	clog.Infof("Starting to initialize MTLS Forwarder for MBG Dataplane at %s", ConnectionURL+m.Name)
	m.startTstamp = time.Now()
	connectMbg := "https://" + targetIPPort + ConnectionURL + name
	m.connectionToken = name
	m.Connection = endpointConn
	m.Name = name

	// Create TCP connection with TLS handshake
	TLSClientConfig := m.CreateTLSConfig(certca, certificate, key, serverName)
	MTLSConn, err := tls.Dial("tcp", targetIPPort, TLSClientConfig)
	if err != nil {
		clog.Infof("Error in connecting.. %+v", err)
		m.CloseConnection()
		return 0, 0, m.startTstamp, time.Now(), err
	}

	TLSConnectClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: TLSClientConfig,
			DialTLS:         connDialer{MTLSConn}.Dial,
		},
	}
	req, err := http.NewRequest(http.MethodGet, connectMbg, nil)
	if err != nil {
		clog.Infof("Failed to create new request %v", err)
		m.CloseConnection()
		return 0, 0, m.startTstamp, time.Now(), err
	}
	resp, err := TLSConnectClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		clog.Infof("Error in TLS Connection %v", err)
		m.CloseConnection()
		return 0, 0, m.startTstamp, time.Now(), err
	}

	m.MTLSConnection = MTLSConn
	clog.Infof("mtlS Connection Established Resp:%s(%d) to Target: %s", resp.Status, resp.StatusCode, connectMbg)
	clog.Infof("Starting mTLS Forwarder client for MBG Dataplane at %s  to target %s with certs(%s,%s)", ConnectionURL+m.Name, targetIPPort, certificate, key)

	go func() { // From forwarder to other MBG
		err = m.MTLSDispatch(event.Outgoing)
		clog.Infof("failed to dispatch outgoing connection: %v", err)
	}()

	if err = m.dispatch(event.Incoming); err != nil { // From source to forwarder
		clog.Infof("failed to dispatch incoming connection: %v", err)
	}
	return m.incomingBytes, m.outgoingBytes, m.startTstamp, time.Now(), err
}

// Start mtls Forwarder server on a specific mtls target
// Register handling function (for hijack the connection) and start dispatch to destination
func (m *MTLSForwarder) StartMTLSForwarderServer(targetIPPort, name, certificate, key string, endpointConn net.Conn) (int, int, time.Time, time.Time, error) {
	clog.Infof("Starting to initialize mTLS Forwarder for MBG Dataplane at %s", ConnectionURL+m.Name)
	m.startTstamp = time.Now()
	// Register function for handling the dataplane traffic
	clog.Infof("Register new handle func to address =%s", ConnectionURL+name)
	m.ChiRouter.Get(ConnectionURL+name, m.ConnectHandler)

	connectMbg := "https://" + targetIPPort + ConnectionURL + name

	clog.Infof("Connect MBG Target =%s", connectMbg)
	m.Connection = endpointConn
	m.Name = name

	if err := m.dispatch(event.Outgoing); err != nil {
		clog.Infof("failed to dispatch outgoing connection: %+v", err)
		return m.incomingBytes, m.outgoingBytes, m.startTstamp, time.Now(), err
	}

	clog.Infof("Starting mTLS forwarder server at /connectionData/%s to target %s with certs(%s,%s)", m.Name, targetIPPort, certificate, key)
	return m.incomingBytes, m.outgoingBytes, m.startTstamp, time.Now(), nil
}

// Hijack the http connection and use it as TCP connection
func (m *MTLSForwarder) ConnectHandler(w http.ResponseWriter, r *http.Request) {
	clog.Infof("Received Connect (%s) from %s\n", m.Name, r.RemoteAddr)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	// Hijack the connection
	conn, _, err := hj.Hijack()
	if err != nil {
		clog.Infof("Hijacking failed %v\n", err)
		return
	}

	if err = conn.SetDeadline(time.Time{}); err != nil {
		clog.Infof("failed to clear deadlines on connection: %v", err)
		return
	}

	if _, err := conn.Write([]byte{}); err != nil {
		clog.Infof("failed to write on connection: %v", err)
		_ = conn.Close() // close the connection ignoring errors
		return
	}

	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")
	clog.Infof("Connection Hijacked  %v->%v", conn.RemoteAddr().String(), conn.LocalAddr().String())

	m.MTLSConnection = conn
	clog.Infof("Starting to dispatch MTLS Connection")
	go func() {
		if err := m.MTLSDispatch(event.Incoming); err != nil {
			clog.Infof("failed to dispatch incoming connection: %v", err)
		}
	}()
}

// Dispatch from server to client side
func (m *MTLSForwarder) MTLSDispatch(direction event.Direction) error {
	bufData := make([]byte, maxDataBufferSize)
	var err error

	for {
		numBytes, err := m.MTLSConnection.Read(bufData)
		if err != nil {
			if err != io.EOF { // don't log EOF
				clog.Infof("MTLSDispatch: Read error %v\n", err)
			}
			break
		}

		_, err = m.Connection.Write(bufData[:numBytes]) // TODO: track actually written byte count?
		if err != nil {
			if err != io.EOF { // don't log EOF
				clog.Infof("MTLSDispatch: Write error %v\n", err)
			}
			break
		}

		if direction == event.Incoming {
			m.incomingBytes += numBytes
		} else {
			m.outgoingBytes += numBytes
		}
	}

	clog.Infof("Initiating end of MTLS connection(%s)", m.Name)
	m.CloseConnection()
	if err == io.EOF {
		return nil
	}

	return err
}

// Dispatch from client to server side
func (m *MTLSForwarder) dispatch(direction event.Direction) error {
	bufData := make([]byte, maxDataBufferSize)
	var err error

	for {
		var numBytes int

		numBytes, err = m.Connection.Read(bufData)
		if err != nil {
			if err != io.EOF {
				clog.Errorf("Dispatch: Read error %v  connection: (local:%s Remote:%s)->,(local: %s Remote%s) ", err,
					m.Connection.LocalAddr(), m.Connection.RemoteAddr(), m.MTLSConnection.LocalAddr(), m.MTLSConnection.RemoteAddr())
			}
			break
		}

		if m.MTLSConnection == nil {
			clog.Info("Start Waiting for MTLSConnection") // start infinite loop
			for m.MTLSConnection == nil {
				time.Sleep(time.Microsecond)
			}
			clog.Info("Finish Waiting for MTLSConnection ") // Finish infinite loop
		}

		_, err = m.MTLSConnection.Write(bufData[:numBytes])
		if err != nil {
			clog.Errorf("Dispatch: Write error %v  connection: (local:%s Remote:%s)->,(local: %s Remote%s) ", err,
				m.Connection.LocalAddr(), m.Connection.RemoteAddr(), m.MTLSConnection.LocalAddr(), m.MTLSConnection.RemoteAddr())
			break
		}

		if direction == event.Incoming {
			m.incomingBytes += numBytes
		} else {
			m.outgoingBytes += numBytes
		}
	}

	clog.Infof("Initiating end of connection(%s)", m.Name)
	m.CloseConnection()
	if err == io.EOF {
		return nil
	}

	return err
}

// CloseConnection close all net connections
func (m *MTLSForwarder) CloseConnection() {
	if m.Connection != nil {
		m.Connection.Close()
	}
	if m.MTLSConnection != nil {
		m.MTLSConnection.Close()
	}
}

// Get certca, certificate, key  and create tls config
func (m *MTLSForwarder) CreateTLSConfig(certca, certificate, key, serverName string) *tls.Config {
	// Read the key pair to create certificate
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		clog.Fatalf("LoadX509KeyPair -%v \ncertificate: %v \nkey:%v", err, certificate, key)
	}

	// Create a CA certificate pool and add ca to it
	caCert, err := os.ReadFile(certca)
	if err != nil {
		clog.Fatalf("ReadFile certificate %v :%v", certca, err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := netutils.ConfigureSafeTLSConfig()
	tlsConfig.RootCAs = caCertPool
	tlsConfig.Certificates = []tls.Certificate{cert}
	tlsConfig.ServerName = serverName
	return tlsConfig
}
