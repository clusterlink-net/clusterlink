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
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
)

type MbgMtlsForwarder struct {
	TargetMbg  string
	Name       string
	TlsClient  *http.Client
	Connection net.Conn
	CloseConn  chan bool
}

var mlog = logrus.WithField("component", "mbgDataplane/mTLSForwarder")

//Init client fields
func (m *MbgMtlsForwarder) InitmTlsForwarder(target, name, certificate, key string) {
	mlog.Infof("Starting to initialize mTLS Forwarder for MBG Dataplane at /mbgData/%s", m.Name)

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

func CloseMtlsServer(mbgIP string) {
	// Create a Server instance to listen on port 8443 with the TLS config
	server := &http.Server{
		Addr: mbgIP + ":8443",
	}
	server.Shutdown(context.Background())
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

// Start a Cluster Service which is a proxy for remote service
// It receives connections from local service and performs Connect API
// and sets up an mTLS forwarding to the remote service upon accepted (policy checks, etc)
func StartClusterService(serviceId, clusterServicePort, targetMbg, certificate, key string) error {
	acceptor, err := net.Listen("tcp", clusterServicePort)
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		state.UpdateState()
		mlog.Infof("Receiving Outgoing connection %s->%s ", ac.RemoteAddr().String(), ac.LocalAddr().String())
		if err != nil {
			return err
		}

		// Ideally do a control plane connect API, Policy checks, and then create a mTLS forwarder
		// RemoteEndPoint has to be in the connect Request/Response

		localSvc, err := state.LookupLocalService(ac.RemoteAddr().String())
		if err != nil {
			log.Infof("Denying Outgoing connection%v", err)
			ac.Close()
			continue
		}
		log.Infof("[MBG %v] Accepting Outgoing Connect request from service: %v to service: %v", state.GetMyId(), localSvc.Service.Id, serviceId)

		destSvc := state.GetRemoteService(serviceId)
		mbgIP := state.GetServiceMbgIp(destSvc.Service.Ip)
		//Send connection request to other MBG
		connectType, connectDest, err := ConnectReq(localSvc.Service.Id, serviceId, "forward", mbgIP)
		if err != nil && err.Error() != "Connection already setup!" {
			log.Infof("[MBG %v] Send connect failure to Cluster = %v ", state.GetMyId(), err.Error())
			ac.Close()
			continue
		}
		log.Infof("[MBG %v] Using %s for  %s/%s to connect to Service-%v", state.GetMyId(), connectType, targetMbg, connectDest, destSvc.Service.Id)

		var mtlsForward MbgMtlsForwarder
		mtlsForward.InitmTlsForwarder(targetMbg, connectDest, certificate, key)

		mtlsForward.setSocketConnection(ac)
		go mtlsForward.dispatch(ac)
	}
}

// Receiver service is run at the cluster of the server service which receives connection from a remote service
func StartReceiverService(clusterServicePort, targetMbg, remoteEndPoint, certificate, key string) error {
	conn, err := net.Dial("tcp", clusterServicePort)
	if err != nil {
		return err
	}
	mlog.Infof("Receiver Connection at %s, %s", conn.LocalAddr().String(), remoteEndPoint)
	var mtlsForward MbgMtlsForwarder
	mtlsForward.InitmTlsForwarder(targetMbg, remoteEndPoint, certificate, key)
	mtlsForward.setSocketConnection(conn)
	go mtlsForward.dispatch(conn)
	return nil
}
