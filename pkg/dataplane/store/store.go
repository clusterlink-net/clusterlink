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

package store

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/utils/netutils"
)

type Store struct {
	ID               string
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
func NewStore(s *Store, controlplane string) *Store {
	sObj := &Store{
		ID:               s.ID,
		CertAuthority:    s.CertAuthority,
		Cert:             s.Cert,
		Key:              s.Key,
		Dataplane:        s.Dataplane,
		DataPortRange:    s.DataPortRange,
		ControlPlaneAddr: s.GetProtocolPrefix() + controlplane,
		Connections:      make(map[string]string),
		PortMap:          make(map[int]bool),
		dataMutex:        sync.Mutex{},
	}

	sObj.SaveState()
	return sObj
}

// Return data-plane id
func (s *Store) GetMyID() string {
	return s.ID
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
		if _, okStop := stopCh[connectionID]; !okStop { // Create stop channel for case is not exist link in MBG restore
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
			random := rand.Intn(MaxPort - MinPort) //nolint:gosec // G404: Use of weak random is fine for random port selection
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
	port := s.Connections[connectionID]
	lval, _ := strconv.Atoi(port[1:])
	stopCh[connectionID] <- true
	delete(s.PortMap, lval)
	delete(s.Connections, connectionID)
	// SaveState()
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
	}

	return "http://"
}

// Local HTTP Client contains a shorter timeout delays for connection and used for local connection inside the cluster
func (s *Store) GetLocalHTTPClient() http.Client {
	return s.getHTTPClient(time.Second * 3)
}

// Remote HTTP Client contains a long timeout delays for connection and is used for connection cross clusters
func (s *Store) GetRemoteHTTPClient() http.Client {
	return s.getHTTPClient(time.Minute * 3)
}

// Create HTTP client for mTLS or TCP connection
func (s *Store) getHTTPClient(timeout time.Duration) http.Client {
	if s.Dataplane == "mtls" {
		cert, err := os.ReadFile(s.CertAuthority)
		if err != nil {
			log.Fatalf("could not open certificate file: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)

		certificate, err := tls.LoadX509KeyPair(s.Cert, s.Key)
		if err != nil {
			log.Fatalf("could not load certificate: %v", err)
		}

		tlsConfig := netutils.ConfigureSafeTLSConfig()
		tlsConfig.RootCAs = caCertPool
		tlsConfig.Certificates = []tls.Certificate{certificate}
		tlsConfig.ServerName = s.ID

		client := http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
		return client
	}

	return http.Client{}
}

/** Database **/
const (
	ProjectFolder = "/.gw/"
	DBFile        = "store.json"
)

// Get folder to save all files
func configPath() string {
	usr, _ := user.Current() // set cfg file in home directory
	return path.Join(usr.HomeDir, ProjectFolder, DBFile)

}

// Save dataplane store for debug use
func (s *Store) SaveState() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	jsonC, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		log.Errorf("Unable to write json file Error: %v", err)
	} else if err = os.WriteFile(configPath(), jsonC, 0600); err != nil {
		log.Errorf("failed to write configuration file %s: %v", configPath(), err)
	}
}
