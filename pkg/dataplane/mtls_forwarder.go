/**********************************************************/
/* mTLS Forwader : This is created per service-pair connections.
/**********************************************************/
// Workflow of mTLS forwarder usage
// After Expose of a service at MBG 1 run the following APIs :
//    1) StartLocalService for the exported service at other remote application (for e.g. App 2)
//    2) When LocalService receives an accepted connection from APP 2, Do an Connect API to APP 1
//    3) MBG1 starts a StartReceiverService with the necessary details such as endpoint, and sends it as Connect Response

package dataplane

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"strings"
	"time"

	"github.com/go-chi/chi"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
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
	ConnectionUrl = "/connectionData/"
)

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(network, addr string) (net.Conn, error) {
	return cd.c, nil
}

// Start MTLS Forwarder on a specific MTLS target
// targetIPPort in the format of <ip:port>
// connect is set to true on a client side
func (m *MTLSForwarder) StartMTLSForwarderClient(targetIPPort, name, certca, certificate, key, ServerName string, endpointConn net.Conn) (int, int, time.Time, time.Time, error) {
	clog.Infof("Starting to initialize MTLS Forwarder for MBG Dataplane at %s", ConnectionUrl+m.Name)
	m.startTstamp = time.Now()
	connectMbg := "https://" + targetIPPort + ConnectionUrl + name
	m.connectionToken = name
	m.Connection = endpointConn
	m.Name = name

	//Create TCP connection with TLS handshake
	TLSClientConfig := m.CreateTlsConfig(certca, certificate, key, ServerName)
	MTLS_conn, err := tls.Dial("tcp", targetIPPort, TLSClientConfig)
	if err != nil {
		clog.Infof("Error in connecting.. %+v", err)
		m.CloseConnection()
		return 0, 0, m.startTstamp, time.Now(), err
	}

	//clog.Debugln("mTLS Debug Check:", m.certDebg(targetIPPort, name, tlsConfig))

	TlsConnectClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: TLSClientConfig,
			DialTLS:         connDialer{MTLS_conn}.Dial,
		},
	}
	req, err := http.NewRequest(http.MethodGet, connectMbg, nil)
	if err != nil {
		clog.Infof("Failed to create new request %v", err)
		m.CloseConnection()
		return 0, 0, m.startTstamp, time.Now(), err
	}
	resp, err := TlsConnectClient.Do(req)
	if err != nil {
		clog.Infof("Error in Tls Connection %v", err)
		m.CloseConnection()
		return 0, 0, m.startTstamp, time.Now(), err
	}

	m.MTLSConnection = MTLS_conn
	clog.Infof("mtlS Connection Established Resp:%s(%d) to Target: %s", resp.Status, resp.StatusCode, connectMbg)
	clog.Infof("Starting mTLS Forwarder client for MBG Dataplane at %s  to target %s with certs(%s,%s)", ConnectionUrl+m.Name, targetIPPort, certificate, key)
	//From forwarder to other MBG
	go m.MTLSDispatch(event.Outgoing)
	//From source to forwarder
	m.dispatch(event.Incoming)

	return m.incomingBytes, m.outgoingBytes, m.startTstamp, time.Now(), nil
}

// Start mtls Forwarder server on a specific mtls target
// Register handling function (for hijack the connection) and start dispatch to destination
func (m *MTLSForwarder) StartMTLSForwarderServer(targetIPPort, name, certca, certificate, key string, endpointConn net.Conn) (int, int, time.Time, time.Time, error) {
	clog.Infof("Starting to initialize mTLS Forwarder for MBG Dataplane at %s", ConnectionUrl+m.Name)
	m.startTstamp = time.Now()
	// Register function for handling the dataplane traffic
	clog.Infof("Register new handle func to address =%s", ConnectionUrl+name)
	m.ChiRouter.Get(ConnectionUrl+name, m.ConnectHandler)

	connectMbg := "https://" + targetIPPort + ConnectionUrl + name

	clog.Infof("Connect MBG Target =%s", connectMbg)
	m.Connection = endpointConn
	m.Name = name

	m.dispatch(event.Outgoing)
	clog.Infof("Starting mTLS Forwarder server for MBG Dataplane at /connectionData/%s  to target %s with certs(%s,%s)", m.Name, targetIPPort, certificate, key)
	return m.incomingBytes, m.outgoingBytes, m.startTstamp, time.Now(), nil

}

// Hijack the http connection and use it as TCP connection
func (m *MTLSForwarder) ConnectHandler(w http.ResponseWriter, r *http.Request) {
	clog.Infof("Received Connect (%s) from %s", m.Name, r.RemoteAddr)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	//Hijack the connection
	conn, _, err := hj.Hijack()
	if err != nil {
		clog.Infof("Hijacking failed %v", err)
		return
	}
	conn.Write([]byte{})
	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")

	clog.Infof("Connection Hijacked  %v->%v", conn.RemoteAddr().String(), conn.LocalAddr().String())

	m.MTLSConnection = conn
	clog.Infof("Starting to dispatch MTLS Connection")
	go m.MTLSDispatch(event.Incoming)
}

// Dispatch from server to client side
func (m *MTLSForwarder) MTLSDispatch(direction event.Direction) error {
	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := m.MTLSConnection.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				clog.Infof("MTLSDispatch: Read error %v\n", err)
			}
			break
		}
		m.Connection.Write(bufData[:numBytes])
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
	} else {
		return err
	}
}

// Dispatch from client to server side
func (m *MTLSForwarder) dispatch(direction event.Direction) error {
	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := m.Connection.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				clog.Errorf("Dispatch: Read error %v  connection: (local:%s Remote:%s)->,(local: %s Remote%s) ", err,
					m.Connection.LocalAddr(), m.Connection.RemoteAddr(), m.MTLSConnection.LocalAddr(), m.MTLSConnection.RemoteAddr())

			}
			break
		}
		if m.MTLSConnection == nil {
			clog.Info("Start Waiting for MTLSConnection") //start infinite loop
			for m.MTLSConnection == nil {
				time.Sleep(time.Microsecond)
			}
			clog.Info("Finish Waiting for MTLSConnection ") //Finish infinite loop
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
	} else {
		return err
	}
}

// Close all net connections
func (m *MTLSForwarder) CloseConnection() {
	if m.Connection != nil {
		m.Connection.Close()
	}
	if m.MTLSConnection != nil {
		m.MTLSConnection.Close()
	}

}

// Close MTLS server
func CloseMTLSServer(ip string) {
	// Create a Server instance to listen on port 443 with the TLS config
	server := &http.Server{
		Addr: ip,
	}
	server.Shutdown(context.Background())
}

// Get certca, certificate, key  and create tls config
func (m *MTLSForwarder) CreateTlsConfig(certca, certificate, key, ServerName string) *tls.Config {
	// Read the key pair to create certificate
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		clog.Fatalf("LoadX509KeyPair -%v \ncertificate: %v \nkey:%v", err, certificate, key)
	}

	// Create a CA certificate pool and add ca to it
	caCert, err := ioutil.ReadFile(certca)
	if err != nil {
		clog.Fatalf("ReadFile certificate %v :%v", certca, err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	TLSConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
		ServerName:   ServerName,
	}
	return TLSConfig
}

// method for debug only -use to debug MTLS connection
func (m *MTLSForwarder) certDebg(target, name string, tlsConfig *tls.Config) string {
	clog.Infof("Starting tls debug to addr %v name %v", target, name)
	conn, err := tls.Dial("tcp", target, tlsConfig)
	if err != nil {
		panic("Server doesn't support SSL certificate err: " + err.Error())
	}
	ip := strings.Split(target, ":")[0]
	err = conn.VerifyHostname(ip)
	if err != nil {
		panic("Hostname doesn't match with certificate: " + err.Error())
	}
	expiry := conn.ConnectionState().PeerCertificates[0].NotAfter
	clog.Infof("Issuer: %s\nExpiry: %v\n", conn.ConnectionState().PeerCertificates[0].Issuer, expiry.Format(time.RFC850))
	conn.Close()
	return "Debug succeed"
}
