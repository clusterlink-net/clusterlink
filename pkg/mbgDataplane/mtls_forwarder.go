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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

type MbgMtlsForwarder struct {
	TargetMbg  string
	Name       string
	TlsClient  *http.Client
	Connection net.Conn
	CloseConn  chan bool
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(network, addr string) (net.Conn, error) {
	return cd.c, nil
}

var mlog = logrus.WithField("component", "mbgDataplane/mTLSForwarder")

//Init client fields
func (m *MbgMtlsForwarder) InitmTlsForwarder(targetIPPort, name, certificate, key string, connect bool) {
	mlog.Infof("Starting to initialize mTLS Forwarder for MBG Dataplane at /mbgData/%s", m.Name)

	m.TargetMbg = "https://" + targetIPPort + "/mbgData/" + name
	connectMbg := "https://" + targetIPPort + "/mbgDataConnect"
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

	// TlsClient for the POST Method
	m.TlsClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	// Trying out Connect Method
	if connect {
		// Reference : https://tip.golang.org/src/net/http/transport_test.go

		// Create a HTTPS client and supply the created CA pool and certificate
		// Below Snippets tries different methods :

		// Method 1:

		// Outcome :
		// Encountered following error:
		// INFO   [2023-01-02 10:47:03] Starting to initialize mTLS Forwarder for MBG Dataplane at /mbgData/  component=mbgDataplane/mTLSForwarder
		// INFO   [2023-01-02 10:47:03] Connect resp:  400                            component=mbgDataplane/TCPForwarder

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

		// Method 2:

		// OutCome:
		// Encounter following error :
		// INFO   [2023-01-02 10:20:57] Starting to initialize mTLS Forwarder for MBG Dataplane at /mbgData/  component=mbgDataplane/mTLSForwarder
		// INFO   [2023-01-02 10:20:57] Successfully dialed TLS 10.20.20.2:40698      component=mbgDataplane/TCPForwarder
		// INFO   [2023-01-02 10:20:57] Error in Tls Connection Connect "https://10.20.20.1:8443/mbgDataConnect": http: server gave HTTP response to HTTPS client  component=mbgDataplane/mTLSForwarder
		// panic: runtime error: invalid memory address or nil pointer dereference
		// [signal SIGSEGV: segmentation violation code=0x1 addr=0x10 pc=0x7178e5]

		// TlsConnectClient := &http.Client{
		// 	Transport: &http.Transport{
		// 		TLSClientConfig: &tls.Config{
		// 			RootCAs:      caCertPool,
		// 			Certificates: []tls.Certificate{cert},
		// 		},
		// 		Dial: func(network, addr string) (net.Conn, error) {
		// 			TLSClientConfig := &tls.Config{
		// 				RootCAs:      caCertPool,
		// 				Certificates: []tls.Certificate{cert},
		// 			}

		// 			conn, err := tls.Dial("tcp", targetIPPort, TLSClientConfig)
		// 			if err != nil {
		// 				return nil, err
		// 			}
		// 			log.Infof("Successfully dialed TLS %v", conn.LocalAddr().String())
		// 			return conn, nil
		// 		},
		// 	},
		// }

		req, err := http.NewRequest(http.MethodConnect, connectMbg, bytes.NewBuffer([]byte("jsonData")))
		if err != nil {
			mlog.Infof("Failed to create new request %v", err)
		}
		resp, err := TlsConnectClient.Do(req)
		if err != nil {
			mlog.Infof("Error in Tls Connection %v", err)
		}
		log.Println("Connect resp: ", resp.StatusCode)
	}
	// Register function for handling the dataplane traffic
	http.HandleFunc("/mbgData/"+m.Name, m.mbgDataHandler)
	mlog.Infof("Starting mTLS Forwarder for MBG Dataplane at /mbgData/%s", m.Name)
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
		Addr:      ip,
		TLSConfig: tlsConfig,
	}

	mlog.Infof("Starting mTLS Server for MBG Dataplane/Controlplane")
	http.HandleFunc("/mbgDataConnect", mbgConnectHandler)

	// Listen to HTTPS connections with the server certificate and wait
	log.Fatal(server.ListenAndServeTLS(certificate, key))
}

func mbgConnectHandler(w http.ResponseWriter, r *http.Request) {
	//Phrase struct from request
	log.Infof("Received HTTP connect to service:")

	var c protocol.ConnectRequest
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Connect control plane logic
	//Check if we can hijack connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	//Write response
	//Hijack the connection
	conn, _, err := hj.Hijack()
	//connection logic
	w.WriteHeader(http.StatusOK)

	log.Info("Connection Hijacked  %s->%s", conn.RemoteAddr().String(), conn.RemoteAddr().String())
}

func (m *MbgMtlsForwarder) mbgDataHandler(mbgResp http.ResponseWriter, mbgR *http.Request) {
	// Read the response body
	defer mbgR.Body.Close()
	mbgData, err := ioutil.ReadAll(mbgR.Body)
	if err != nil {
		log.Fatal(err)
	}
	// Send to the active TCP Connection
	if m.Connection != nil {
		_, err = m.Connection.Write(mbgData)
		if err != nil {
			mlog.Infof("mbgDataHandler: Write error %v\n", err)
		}
	} else {
		mlog.Errorf("Received message before active connection")
	}
	mbgResp.WriteHeader(http.StatusOK)
}

//Connect to client and call ioLoop function
func (m *MbgMtlsForwarder) dispatch(ac net.Conn) error {
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
		m.TlsClient.Post(m.TargetMbg, "application/octet-stream", bytes.NewBuffer(bufData[:numBytes]))
	}
	if err == io.EOF {
		return nil
	} else {
		return err
	}
}

func (m *MbgMtlsForwarder) setSocketConnection(ac net.Conn) {
	m.Connection = ac
}

func (m *MbgMtlsForwarder) waitToCloseSignal(wg *sync.WaitGroup) {
	defer wg.Done()
	<-m.CloseConn
	//cl.Close() ,mbg.Close()- TBD -check if need to close also the internal connections
	mlog.Infof("[%v] Receive signal to close connection\n", m.Name)
}

func (m *MbgMtlsForwarder) CloseConnection() {
	m.CloseConn <- true
	m.Connection.Close()
}
