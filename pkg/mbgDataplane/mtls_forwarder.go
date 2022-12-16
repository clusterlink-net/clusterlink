/**********************************************************/
/* mTLS Forwader : This is created per service-pair connections.
/**********************************************************/
// Generate Certificates
// openssl req -newkey rsa:2048   -new -nodes -x509   -days 3650   -out ~/mtls/tcnode7_cert.pem   -keyout ~/mtls/tcnode7_key.pem   -subj "/C=US/ST=California/L=mbg/O=ibm/OU=dev/CN=tcnode7" -addext "subjectAltName = IP:10.20.20.2"
// openssl req -newkey rsa:2048   -new -nodes -x509   -days 3650   -out ~/mtls/tcnode6_cert.pem   -keyout ~/mtls/tcnode6_key.pem   -subj "/C=US/ST=California/L=mbg/O=ibm/OU=dev/CN=tcnode6" -addext "subjectAltName = IP:10.20.20.1"

package mbgDataplane

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	maxDataBufferSize = 64 * 1024
)

type mTlsForwarder struct {
	TargetMbg  string
	Name       string
	TlsClient  *http.Client
	Connection net.Conn
	CloseConn  chan bool
}

var mlog = logrus.WithField("component", "mbgDataplane/mTLSForwarder")

//Init client fields
func (m *mTlsForwarder) InitmTlsForwarder(target, name, certificate, key string) {
	m.TargetMbg = target + "/" + name
	m.Name = name
	// Read the key pair to create certificate
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		log.Fatal(err)
	}

	// Create a CA certificate pool and add cert.pem to it
	caCert, err := ioutil.ReadFile(certificate)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create a HTTPS client and supply the created CA pool and certificate
	m.TlsClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}
	// Register function for handling the dataplane traffic
	http.HandleFunc("/mbgData/"+m.Name, m.mbgDataHandler)
	mlog.Infof("Starting mTLS Forwarder for MBG Dataplane at /mbgData/%s", m.Name)

}

func StartMtlsServer(mbgIP, certificate, key string) {
	// Create the TLS Config with the CA pool and enable Client certificate validation

	caCert, err := ioutil.ReadFile(certificate)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	tlsConfig.BuildNameToCertificate()

	// Create a Server instance to listen on port 8443 with the TLS config
	server := &http.Server{
		Addr:      mbgIP + ":8443",
		TLSConfig: tlsConfig,
	}

	mlog.Infof("Starting mTLS Server for MBG Dataplane/Controlplane")
	// Listen to HTTPS connections with the server certificate and wait
	log.Fatal(server.ListenAndServeTLS(certificate, key))
}

func (m *mTlsForwarder) mbgDataHandler(mbgResp http.ResponseWriter, mbgR *http.Request) {
	// Read the response body
	defer mbgR.Body.Close()
	mbgData, err := ioutil.ReadAll(mbgR.Body)
	if err != nil {
		log.Fatal(err)
	}
	// Send to the active TCP Connection
	if m.Connection != nil {
		//mlog.Infof("mbgDataHandler: Received %s, and sending to cluster %s\n", mbgData, m.Connection.LocalAddr().String())
		_, err = m.Connection.Write(mbgData)
		if err != nil {
			mlog.Infof("mbgDataHandler: Write error %v\n", err)
		}
	} else {
		mlog.Errorf("Received message before active connection")
	}
	mbgResp.WriteHeader(200)
}

//Connect to client and call ioLoop function
func (m *mTlsForwarder) dispatch(ac net.Conn) error {
	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := ac.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				mlog.Infof("Read error %v\n", err)
			}
			break
		}
		//mlog.Infof("Got a message %s from Cluster and sending to MBG", bufData[:numBytes])
		m.TlsClient.Post(m.TargetMbg, "text/plain", bytes.NewBuffer(bufData[:numBytes]))
	}
	if err == io.EOF {
		return nil
	} else {
		return err
	}
}

func (m *mTlsForwarder) setSocketConnection(ac net.Conn) {
	m.Connection = ac
}

func (m *mTlsForwarder) CloseConnection() {
	m.Connection.Close()
}

// Start a Cluster Service which is a proxy for  remote service
// It receives connections from local service and performs Connect API
// and sets up an mTLS forwarding to the remote service upon accepted (policy checks, etc)
func StartClusterService(serviceName, clusterServicePort, targetMbg, certificate, key string) error {
	mlog.Infof("Waiting for connection at %s", clusterServicePort)
	acceptor, err := net.Listen("tcp", clusterServicePort)
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		mlog.Infof("Accept connection %s->%s ", ac.LocalAddr().String(), ac.RemoteAddr().String())
		if err != nil {
			return err
		}
		// Ideally do a control plane connect API, Policy checks, and then create a mTLS forwarder
		// RemoteEndPoint has to be in the connect Request/Response
		var mtlsForward mTlsForwarder
		var remoteEndPoint string
		mtlsForward.InitmTlsForwarder(targetMbg, remoteEndPoint, certificate, key)
		mtlsForward.setSocketConnection(ac)
		go mtlsForward.dispatch(ac)
	}
}

func testsendClusterData(clusterIn string, data []byte) {
	nodeConn, err := net.Dial("tcp", clusterIn)
	if err != nil {
		log.Fatalf("Failed to connect to socket %+v", err)
	}
	go testrecvClusterData(nodeConn)
	for {
		nodeConn.Write(data)
		time.Sleep(1 * time.Second)
	}
}

func testrecvClusterData(conn net.Conn) {
	bufData := make([]byte, maxDataBufferSize)
	for {
		numBytes, err := conn.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				mlog.Infof("Read error %v\n", err)
			}
			break
		}
		mlog.Infof("Received %s in Socket Connection", bufData[:numBytes])
	}
}

////// Tests - TODO: Move this to tests
//run in tcnode6
// func tcnode6() {
// 	go StartMtlsServer("10.20.20.1", "/home/pravein/mtls/tcnode6_cert.pem", "/home/pravein/mtls/tcnode6_key.pem")
// 	go StartClusterService("testService1", ":9000", "https://10.20.20.2:8443/mbgData", "/home/pravein/mtls/tcnode7_cert.pem", "/home/pravein/mtls/tcnode7_key.pem")

// 	time.Sleep(1 * time.Second)
// 	testsendClusterData(":9000", []byte("I am tcnode6-test1 cluster"))
// }

// //run in tcnode7
// func tcnode7() {
// 	go StartMtlsServer("10.20.20.2", "/home/pravein/mtls/tcnode7_cert.pem", "/home/pravein/mtls/tcnode7_key.pem")
// 	go StartClusterService("testService2", ":9000", "https://10.20.20.1:8443/mbgData", "/home/pravein/mtls/tcnode6_cert.pem", "/home/pravein/mtls/tcnode6_key.pem")

// 	time.Sleep(1 * time.Second)
// 	testsendClusterData(":9000", []byte("I am tcnode7-test1 cluster"))
// }

// func main() {
// 	tcnode7()
// }
