/**********************************************************/
/* mTLS Forwader : This is created per service-pair connections.
/**********************************************************/
// Generate Certificates
// openssl req -newkey rsa:2048   -new -nodes -x509   -days 3650   -out ~/mtls/tcnode7_cert.pem   -keyout ~/mtls/tcnode7_key.pem   -subj "/C=US/ST=California/L=mbg/O=ibm/OU=dev/CN=tcnode7" -addext "subjectAltName = IP:10.20.20.2"
// openssl req -newkey rsa:2048   -new -nodes -x509   -days 3650   -out ~/mtls/tcnode6_cert.pem   -keyout ~/mtls/tcnode6_key.pem   -subj "/C=US/ST=California/L=mbg/O=ibm/OU=dev/CN=tcnode6" -addext "subjectAltName = IP:10.20.20.1"

// Workflow of mTLS forwarder usage
// After Expose of a service at Cluster 1 run the following APIs :
//    1) StartClusterService for the exported service at other remote Clusters (for e.g. Cluster 2)
//    2) When ClusterService receives an accepted connection from Cluster 2, Do an Connect API to Cluster 1
//    3) Cluster1 starts a StartReceiverService with the necessary details such as endpoint, and sends it as Connect Response

package mbgDataplane

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

type MbgMtlsForwarder struct {
	Name           string
	Connection     net.Conn
	mtlsConnection net.Conn
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(network, addr string) (net.Conn, error) {
	return cd.c, nil
}

var mlog = logrus.WithField("component", "mbgDataplane/mTLSForwarder")

// Start mtls Forwarder on a specific mtls target
// targetIPPort in the format of <ip:port>
// connect is set to true on a client side
func (m *MbgMtlsForwarder) StartmTlsForwarder(targetIPPort, name, certificate, key string, endpointConn net.Conn, connect bool) {
	mlog.Infof("Starting to initialize mTLS Forwarder for MBG Dataplane at /mbgData/%s", m.Name)
	// Register function for handling the dataplane traffic
	http.HandleFunc("/mbgData/"+name, m.mbgConnectHandler)

	connectMbg := "https://" + targetIPPort + "/mbgData/" + name

	mlog.Infof("Connect MBG Target =%s", connectMbg)
	m.Connection = endpointConn
	m.Name = name
	if connect {
		// Read the key pair to create certificate
		cert, err := tls.LoadX509KeyPair(certificate, key)
		if err != nil {
			mlog.Fatalf("LoadX509KeyPair -%v \ncertificate: %v \nkey:%v", err, certificate, key)
		}

		// Create a CA certificate pool and add cert.pem to it
		caCert, err := ioutil.ReadFile(certificate)
		if err != nil {
			mlog.Fatalf("ReadFile certificate %v :%v", certificate, err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		TLSClientConfig := &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{cert},
		}
		mtls_conn, err := tls.Dial("tcp", targetIPPort, TLSClientConfig)
		if err != nil {
			mlog.Infof("Error in connecting.. %+v", err)
		}
		TlsConnectClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{cert},
				},
				DialTLS: connDialer{mtls_conn}.Dial,
			},
		}
		req, err := http.NewRequest(http.MethodGet, connectMbg, nil)
		if err != nil {
			mlog.Infof("Failed to create new request %v", err)
		}
		resp, err := TlsConnectClient.Do(req)
		if err != nil {
			mlog.Infof("Error in Tls Connection %v", err)
		}

		m.mtlsConnection = mtls_conn
		mlog.Infof("mtlS Connection Established RespCode(%d)", resp.StatusCode)

		go m.mtlsDispatch()
	}
	go m.dispatch()
	mlog.Infof("Starting mTLS Forwarder for MBG Dataplane at /mbgData/%s  to target %s with certs(%s,%s)", m.Name, m.TargetMbg, certificate, key)

}

func (m *MbgMtlsForwarder) mbgConnectHandler(w http.ResponseWriter, r *http.Request) {
	mlog.Infof("Received mbgConnect (%s) from %s", m.Name, r.RemoteAddr)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	//Hijack the connection
	conn, _, err := hj.Hijack()
	if err != nil {
		mlog.Infof("Hijacking failed %v", err)

	}
	conn.Write([]byte{})
	fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")

	mlog.Infof("Connection Hijacked  %v->%v", conn.RemoteAddr().String(), conn.LocalAddr().String())

	m.mtlsConnection = conn
	mlog.Infof("Starting to dispatch mtls Connection")
	go m.mtlsDispatch()
}

func (m *MbgMtlsForwarder) mtlsDispatch() error {
	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := m.mtlsConnection.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				mlog.Infof("Read error %v\n", err)
			}
			break
		}
		m.Connection.Write(bufData[:numBytes])
	}
	mlog.Infof("Initiating end of mtls connection(%s)", m.Name)
	m.CloseConnection()
	if err == io.EOF {
		return nil
	} else {
		return err
	}
}

func (m *MbgMtlsForwarder) dispatch() error {
	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := m.Connection.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				mlog.Infof("Read error %v\n", err)
			}
			break
		}
		m.mtlsConnection.Write(bufData[:numBytes])
	}
	mlog.Infof("Initiating end of connection(%s)", m.Name)
	m.CloseConnection()
	if err == io.EOF {
		return nil
	} else {
		return err
	}
}

func (m *MbgMtlsForwarder) CloseConnection() {
	m.Connection.Close()
	m.mtlsConnection.Close()
}

func CloseMtlsServer(ip string) {
	// Create a Server instance to listen on port 8443 with the TLS config
	server := &http.Server{
		Addr: ip,
	}
	server.Shutdown(context.Background())
}
func StartMtlsServer(ip, certificate, key string) {
	// Create the TLS Config with the CA pool and enable Client certificate validation
	caCert, err := ioutil.ReadFile(certificate)
	if err != nil {
		mlog.Fatal(err)
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
		Addr:      ip,
		TLSConfig: tlsConfig,
	}

	mlog.Infof("Starting mTLS Server for MBG Dataplane/Controlplane")

	// Listen to HTTPS connections with the server certificate and wait
	mlog.Fatal(server.ListenAndServeTLS(certificate, key))
}
