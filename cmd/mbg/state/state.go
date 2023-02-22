package state

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/eventManager"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

var log = logrus.WithField("component", s.MyInfo.Id)
var dataMutex sync.Mutex
var mbgArrMutex sync.RWMutex
var ChiRouter *chi.Mux = chi.NewRouter()

type mbgState struct {
	MyInfo                MbgInfo
	MbgctlArr             map[string]Mbgctl
	MbgArr                map[string]MbgInfo
	InactiveMbgArr        map[string]MbgInfo
	MyServices            map[string]LocalService
	RemoteServices        map[string][]RemoteService
	Connections           map[string]ServicePort
	LocalServiceEndpoints map[string]string
	LocalPortMap          map[int]bool
	ExternalPortMap       map[int]bool
	RemoteServiceMap      map[string][]string
	MyEventManager        eventManager.MbgEventManager
}

type MbgInfo struct {
	Id              string
	Ip              string
	Cport           ServicePort
	DataPortRange   ServicePort
	MaxPorts        int
	CaFile          string
	CertificateFile string
	KeyFile         string
	Dataplane       string
}

type Mbgctl struct {
	Id string
	Ip string
}

type RemoteService struct {
	Service service.Service
	MbgId   string // For now to identify a service to a MBG
}

type LocalService struct {
	Service service.Service
}

type ServicePort struct {
	Local    string
	External string
}

var s = mbgState{MyInfo: MbgInfo{},
	MbgctlArr:             make(map[string]Mbgctl),
	MbgArr:                make(map[string]MbgInfo),
	InactiveMbgArr:        make(map[string]MbgInfo),
	MyServices:            make(map[string]LocalService),
	RemoteServices:        make(map[string][]RemoteService),
	Connections:           make(map[string]ServicePort),
	LocalServiceEndpoints: make(map[string]string),
	LocalPortMap:          make(map[int]bool),
	ExternalPortMap:       make(map[int]bool),
	RemoteServiceMap:      make(map[string][]string),
	MyEventManager:        eventManager.MbgEventManager{},
}

func GetMyIp() string {
	return s.MyInfo.Ip
}

func GetMyId() string {
	return s.MyInfo.Id
}

func GetMyCport() ServicePort {
	return s.MyInfo.Cport
}

func GetMyInfo() MbgInfo {
	return s.MyInfo
}

func GetMbgList() []string {
	mList := []string{}
	// Copied list is returned to avoid the caller iterate on the original map
	// due to potential panic when iteration and map update happen simulataneously
	mbgArrMutex.RLock()
	for m, _ := range s.MbgArr {
		mList = append(mList, m)
	}
	mbgArrMutex.RUnlock()
	return mList
}

func GetConnectionArr() map[string]ServicePort {
	return s.Connections
}
func GetMbgctlArr() map[string]Mbgctl {
	return s.MbgctlArr
}

func GetDataplane() string {
	return s.MyInfo.Dataplane
}
func GetChiRouter() (r *chi.Mux) {
	return ChiRouter
}

func GetLocalServicesArr() map[string]LocalService {
	return s.MyServices
}

func GetRemoteServicesArr() map[string][]RemoteService {
	return s.RemoteServices
}

func GetEventManager() *eventManager.MbgEventManager {
	return &s.MyEventManager
}

func SetState(id, ip, cportLocal, cportExternal, localDataPortRange, externalDataPortRange, caFile, certificate, key, dataplane string) {
	s.MyInfo.Id = id
	s.MyInfo.Ip = ip
	s.MyInfo.Cport.Local = ":" + cportLocal
	s.MyInfo.Cport.External = ":" + cportExternal
	s.MyInfo.DataPortRange.Local = localDataPortRange
	s.MyInfo.DataPortRange.External = externalDataPortRange
	s.MyInfo.MaxPorts = 1000 // TODO
	s.MyInfo.CaFile = caFile
	s.MyInfo.CertificateFile = certificate
	s.MyInfo.KeyFile = key
	s.MyInfo.Dataplane = dataplane
	log = logrus.WithField("component", s.MyInfo.Id)
	SaveState()
}

func SetMbgctl(id, ip string) {
	log.Info(s)
	s.MbgctlArr[id] = Mbgctl{Id: id, Ip: ip}
	SaveState()
}

func SetChiRouter(r *chi.Mux) {
	ChiRouter = r
}

func UpdateState() {
	s = readState()
	log = logrus.WithField("component", s.MyInfo.Id)
}

func restorePeer(id string) {
	peerResp, err := s.MyEventManager.RaiseAddPeerEvent(eventManager.AddPeerAttr{PeerMbg: id})
	if err != nil {
		log.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == eventManager.Deny {
		log.Infof("Denying add peer(%s) due to policy", id)
		RemoveMbgNbr(id)
		return
	}
}
func RestoreMbg() {
	// Getting a copy of MBG List since, there is a chance of MBG array modified downstream
	for _, mbg := range GetMbgList() {
		restorePeer(mbg)
	}
	// For now, we do expose of local services only when prompted by management and not by default
}

func InactivateMbg(mbg string) {
	mbgArrMutex.Lock()
	mbgI := s.MbgArr[mbg]
	s.InactiveMbgArr[mbg] = mbgI
	delete(s.MbgArr, mbg)
	mbgArrMutex.Unlock()
	RemoveMbgFromServiceMap(mbg)
	PrintState()
	SaveState()
}

func ActivateMbg(mbgId string) {
	log.Infof("Activating MBG %s", mbgId)
	peerResp, err := s.MyEventManager.RaiseAddPeerEvent(eventManager.AddPeerAttr{PeerMbg: mbgId})
	if err != nil {
		log.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == eventManager.Deny {
		log.Infof("Denying add peer(%s) due to policy", mbgId)
		return
	}
	mbgArrMutex.Lock()
	mbgI, ok := s.InactiveMbgArr[mbgId]
	if !ok {
		// Ignore
		mbgArrMutex.Unlock()
		return
	}
	s.MbgArr[mbgId] = mbgI
	delete(s.InactiveMbgArr, mbgId)
	mbgArrMutex.Unlock()

	PrintState()
	SaveState()
}

//Return Function fields
func GetLocalService(id string) LocalService {
	val, ok := s.MyServices[id]
	if !ok {
		log.Errorf("Service %v does not exist", id)
	}
	return val
}

func GetRemoteService(id string) []RemoteService {
	val, ok := s.RemoteServices[id]
	if !ok {
		log.Errorf("Service %v does not exist", id)
	}
	return val

}

func LookupLocalService(network string) (LocalService, error) {

	serviceNetwork := strings.Split(network, ":")
	for _, service := range s.MyServices {
		// Compare Service IPs
		if strings.Split(service.Service.Ip, ":")[0] == serviceNetwork[0] {
			return service, nil
		}
	}
	return LocalService{}, errors.New("unable to find local service")
}
func GetServiceMbgIp(Ip string) string {
	svcIp := strings.Split(Ip, ":")[0]
	mbgArrMutex.RLock()
	for _, m := range s.MbgArr {
		if m.Ip == svcIp {
			mbgIp := m.Ip + m.Cport.External
			return mbgIp
		}
	}
	mbgArrMutex.RUnlock()
	log.Errorf("Service %v is not defined", Ip)
	PrintState()
	return ""
}

func GetMbgTarget(id string) string {
	mbgArrMutex.RLock()
	mbgI := s.MbgArr[id]
	mbgArrMutex.RUnlock()
	return mbgI.Ip + mbgI.Cport.External
}

func GetMbgTargetPair(id string) (string, string) {
	mbgArrMutex.RLock()
	mbgI := s.MbgArr[id]
	mbgArrMutex.RUnlock()
	return mbgI.Ip, mbgI.Cport.External
}

func IsMbgPeer(id string) bool {
	mbgArrMutex.RLock()
	_, ok := s.MbgArr[id]
	mbgArrMutex.RUnlock()
	return ok
}

func GetMyMbgCerts() (string, string, string) {
	return s.MyInfo.CaFile, s.MyInfo.CertificateFile, s.MyInfo.KeyFile
}

func IsServiceLocal(id string) bool {
	_, exist := s.MyServices[id]
	return exist
}

func AddMbgNbr(id, ip, cport string) {
	mbgArrMutex.Lock()
	if _, ok := s.MbgArr[id]; ok {
		log.Infof("Neighbor already added %s", id)
		return
	}
	s.MbgArr[id] = MbgInfo{Id: id, Ip: ip, Cport: ServicePort{External: cport, Local: ""}}
	mbgArrMutex.Unlock()

	PrintState()
	SaveState()
}

func RemoveMbgNbr(id string) {
	mbgArrMutex.Lock()
	delete(s.MbgArr, id)
	mbgArrMutex.Unlock()
	SaveState()
}

// Gets an available free port to use per connection
func GetFreePorts(connectionID string) (ServicePort, error) {
	if port, ok := s.Connections[connectionID]; ok {
		return port, fmt.Errorf("connection already setup")
	}
	rand.NewSource(time.Now().UnixNano())
	if len(s.Connections) == s.MyInfo.MaxPorts {
		return ServicePort{}, fmt.Errorf("all ports taken up, Try again after sometime")
	}
	lval, _ := strconv.Atoi(s.MyInfo.DataPortRange.Local)
	eval, _ := strconv.Atoi(s.MyInfo.DataPortRange.External)
	timeout := time.After(10 * time.Second)
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return ServicePort{}, fmt.Errorf("all ports taken up, Try again after sometime")
		default:
			random := rand.Intn(s.MyInfo.MaxPorts)
			localPort := lval + random
			externalPort := eval + random
			if !s.LocalPortMap[localPort] {
				if !s.ExternalPortMap[externalPort] {
					s.LocalPortMap[localPort] = true
					s.ExternalPortMap[externalPort] = true
					myPort := ServicePort{Local: ":" + strconv.Itoa(localPort), External: ":" + strconv.Itoa(externalPort)}
					s.Connections[connectionID] = myPort
					SaveState()
					return myPort, nil
				}
			}
		}
	}
}

// Gets an available free port to be used within the MBG for a remote service endpoint
func GetFreeLocalPort(serviceName string) (string, error) {
	if port, ok := s.LocalServiceEndpoints[serviceName]; ok {
		return port, fmt.Errorf("connection already setup")
	}
	rand.NewSource(time.Now().UnixNano())
	if len(s.LocalServiceEndpoints) == s.MyInfo.MaxPorts {
		return "", fmt.Errorf("all ports taken up, Try again after sometimes")
	}
	lval, _ := strconv.Atoi(s.MyInfo.DataPortRange.Local)
	timeout := time.After(10 * time.Second)
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return "", fmt.Errorf("all Ports taken up, Try again after sometimes")
		default:
			random := rand.Intn(s.MyInfo.MaxPorts)
			localPort := lval + random
			if !s.LocalPortMap[localPort] {
				s.LocalPortMap[localPort] = true
				myPort := ":" + strconv.Itoa(localPort)
				s.LocalServiceEndpoints[serviceName] = myPort
				SaveState()
				return myPort, nil
			}
		}
	}
}

// Frees up used ports by a connection
func FreeUpPorts(connectionID string) {
	port, _ := s.Connections[connectionID]
	lval, _ := strconv.Atoi(port.Local)
	eval, _ := strconv.Atoi(port.External)
	delete(s.LocalPortMap, lval)
	delete(s.ExternalPortMap, eval)
	delete(s.Connections, connectionID)
}

func AddLocalService(id, ip, description string) {
	if _, ok := s.MyServices[id]; ok {
		log.Infof("Local Service already added %s", id)
		return
	}
	s.MyServices[id] = LocalService{Service: service.Service{Id: id, Ip: ip, Description: description}}
	log.Infof("Adding local service: %s", id)
	PrintState()
	SaveState()
}

func AddRemoteService(id, ip, description, MbgId string) {
	svc := RemoteService{Service: service.Service{Id: id, Ip: ip, Description: description}, MbgId: MbgId}
	if mbgs, ok := s.RemoteServiceMap[id]; ok {
		s.RemoteServiceMap[id] = append(mbgs, MbgId) //TODO- check uniqueness
		s.RemoteServices[id] = append(s.RemoteServices[id], svc)
	} else {
		s.RemoteServiceMap[id] = []string{MbgId}
		s.RemoteServices[id] = []RemoteService{svc}
	}
	log.Infof("Adding remote service: [%v]", s.RemoteServiceMap[id])
	PrintState()
	SaveState()
}

func RemoveMbgFromServiceMap(mbg string) {
	for svc, mbgs := range s.RemoteServiceMap {
		index := -1
		for i, mbgVal := range mbgs {
			if mbg == mbgVal {
				index = i
				break
			}
		}
		if index == -1 {
			continue
		}
		s.RemoteServiceMap[svc] = append((mbgs)[:index], (mbgs)[index+1:]...)
		log.Infof("MBG removed from remote service %v->[%+v]", svc, s.RemoteServiceMap[svc])
	}
}

func GetAddrStart() string {
	if s.MyInfo.Dataplane == "mtls" {
		return "https://"
	} else {
		return "http://"
	}
}
func GetHttpClient() http.Client {
	if s.MyInfo.Dataplane == "mtls" {
		cert, err := ioutil.ReadFile(s.MyInfo.CaFile)
		if err != nil {
			log.Fatalf("could not open certificate file: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)

		certificate, err := tls.LoadX509KeyPair(s.MyInfo.CertificateFile, s.MyInfo.KeyFile)
		if err != nil {
			log.Fatalf("could not load certificate: %v", err)
		}

		client := http.Client{
			Timeout: time.Minute * 3,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{certificate},
					ServerName:   s.MyInfo.Id,
				},
			},
		}
		return client
	} else {
		return http.Client{}
	}
}

func PrintState() {
	log.Infof("****** MBG State ********")
	log.Infof("ID: %v IP: %v%v", s.MyInfo.Id, s.MyInfo.Ip, s.MyInfo.Cport)
	nb := ""
	inb := ""
	services := ""
	mbgArrMutex.RLock()
	for _, n := range s.MbgArr {
		nb = nb + n.Id + " "
	}
	mbgArrMutex.RUnlock()

	log.Infof("MBG neighbors : %s", nb)
	for _, n := range s.InactiveMbgArr {
		inb = inb + n.Id + ", "
	}
	log.Infof("Inactive MBG neighbors : %s", inb)
	for _, se := range s.MyServices {
		services = services + se.Service.Id + ", "
	}
	log.Infof("Myservices: %v", services)
	log.Infof("Remoteservices: %v", s.RemoteServiceMap)
	log.Infof("****************************")
}

/// Json code ////
func configPath() string {
	cfgFile := ".mbgApp"
	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, cfgFile)

	//set cfg file in the git
	//_, filename, _, _ := runtime.Caller(1)
	//return path.Join(path.Dir(filename), cfgFile)

}

func SaveState() {
	dataMutex.Lock()
	jsonC, _ := json.MarshalIndent(s, "", "\t")
	ioutil.WriteFile(configPath(), jsonC, 0644) // os.ModeAppend)
	dataMutex.Unlock()
}

func readState() mbgState {
	data, _ := ioutil.ReadFile(configPath())
	var s mbgState
	json.Unmarshal(data, &s)
	return s
}
