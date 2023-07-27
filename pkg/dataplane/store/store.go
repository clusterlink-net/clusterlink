package store

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os/user"
	"path"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Store struct {
	Id               string
	CertAuthority    string
	Cert             string
	Key              string
	Dataplane        string
	DataPortRange    string
	ControlPlaneAddr string
	PortMap          map[int]bool
	Connections      map[string]string
	dataMutex        sync.Mutex
}

var stopCh = make(map[string]chan bool)

const (
	ConnExist = "connection already setup"
	MinPort   = 5000
	MaxPort   = 10000
)

// Set store parameters
func NewStore(s *Store) *Store {
	sObj := &Store{
		Id:               s.Id,
		CertAuthority:    s.CertAuthority,
		Cert:             s.Cert,
		Key:              s.Key,
		Dataplane:        s.Dataplane,
		DataPortRange:    s.DataPortRange,
		ControlPlaneAddr: s.GetProtocolPrefix() + "controlplane:443",
		Connections:      make(map[string]string),
		PortMap:          make(map[int]bool),
		dataMutex:        sync.Mutex{},
	}

	sObj.SaveState()
	return sObj
}

// Return data-plane id
func (s *Store) GetMyId() string {
	return s.Id
}

// Return data-plane certificate
func (s *Store) GetCerts() (string, string, string) {
	return s.CertAuthority, s.Cert, s.Key
}

// Return data-plane type TCP/MTLS
func (s *Store) GetDataplane() string {
	return s.Dataplane
}

// Return controlplane endpoint
func (s *Store) GetControlPlaneAddr() string {
	return s.ControlPlaneAddr
}

// Return import service local port
func (s *Store) GetSvcPort(id string) string {
	return s.Connections[id]
}

// Gets an available free port to use per connection
func (s *Store) GetFreePorts(connectionID string) (string, error) {
	if port, ok := s.Connections[connectionID]; ok {
		if _, okStop := stopCh[connectionID]; !okStop { //Create stop channel for case is not exist link in MBG restore
			stopCh[connectionID] = make(chan bool)
		}
		return port, fmt.Errorf(ConnExist)
	}
	rand.NewSource(time.Now().UnixNano())
	if len(s.Connections) == (MaxPort - MinPort) {
		return "", fmt.Errorf("all ports taken up, Try again after sometime")
	}
	timeout := time.After(10 * time.Second)
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return "", fmt.Errorf("all ports taken up, Try again after sometime")
		default:
			random := rand.Intn(MaxPort - MinPort)
			port := MinPort + random
			if !s.PortMap[port] {
				s.PortMap[port] = true
				s.Connections[connectionID] = ":" + strconv.Itoa(port)
				stopCh[connectionID] = make(chan bool)
				s.SaveState()
				return s.Connections[connectionID], nil

			}
		}
	}
}

// Frees up used ports by a connection
func (s *Store) FreeUpPorts(connectionID string) {
	log.Infof("Start to FreeUpPorts for service: %s", connectionID)
	port, _ := s.Connections[connectionID]
	lval, _ := strconv.Atoi(port[1:])
	stopCh[connectionID] <- true
	delete(s.PortMap, lval)
	delete(s.Connections, connectionID)
	//SaveState()
}

// Send stop channel signal to stop import service listener
func (s *Store) WaitServiceStopCh(connectionID, servicePort string) {
	if _, ok := s.Connections[connectionID]; ok {
		<-stopCh[connectionID]
		log.Infof("Receive signal to close service %v: with port %v\n", connectionID, servicePort)
	}
}

// Choose the correct prefix of url according to the data-plane type
func (s *Store) GetProtocolPrefix() string {
	if s.Dataplane == "mtls" {
		return "https://"
	} else {
		return "http://"
	}
}

// Local Http Client contains a shorter timeout delays for connection and used for local connection inside the cluster
func (s *Store) GetLocalHttpClient() http.Client {
	return s.getHttpClient(time.Second * 3)
}

// Remote Http Client contains a long timeout delays for connection and is used for connection cross clusters
func (s *Store) GetRemoteHttpClient() http.Client {
	return s.getHttpClient(time.Minute * 3)
}

// Create HTTP client for mTLS or TCP connection
func (s *Store) getHttpClient(timeout time.Duration) http.Client {
	if s.Dataplane == "mtls" {
		cert, err := ioutil.ReadFile(s.CertAuthority)
		if err != nil {
			log.Fatalf("could not open certificate file: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)

		certificate, err := tls.LoadX509KeyPair(s.Cert, s.Key)
		if err != nil {
			log.Fatalf("could not load certificate: %v", err)
		}

		client := http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{certificate},
					ServerName:   s.Id,
				},
			},
		}
		return client
	} else {
		return http.Client{}
	}
}

/** Database **/
const (
	ProjectFolder = "/.gw/"
	DBFile        = "store.json"
)

// Get folder to save all files
func configPath() string {
	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ProjectFolder, DBFile)

}

// Save dataplane store for debug use
func (s *Store) SaveState() {
	s.dataMutex.Lock()
	jsonC, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		log.Errorf("Unable to write json file Error: %v", err)
		s.dataMutex.Unlock()
		return
	}
	ioutil.WriteFile(configPath(), jsonC, 0644) // os.ModeAppend)
	s.dataMutex.Unlock()
}
