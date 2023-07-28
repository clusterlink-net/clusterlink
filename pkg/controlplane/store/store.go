package store

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
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
	Connections           map[string]string
	LocalServiceEndpoints map[string]string
	LocalPortMap          map[int]bool
	RemoteServiceMap      map[string][]string
	MyEventManager        event.EventManager
}

type MbgInfo struct {
	Id                string
	Ip                string
	Cport             ServicePort
	DataPortRange     ServicePort
	MaxPorts          int
	CaFile            string
	CertificateFile   string
	KeyFile           string
	Dataplane         string
	DataplaneEndpoint string
}

type Mbgctl struct {
	Id string
	Ip string
}

type RemoteService struct {
	Id          string
	MbgId       string
	MbgIp       string
	Description string
}

type LocalService struct {
	Id           string
	Ip           string
	Port         string
	Description  string
	PeersExposed []string //ToDo not uniqe
}

type ServicePort struct {
	Local    string
	External string
}

var s = mbgState{MyInfo: MbgInfo{},
	MbgctlArr:        make(map[string]Mbgctl),
	MbgArr:           make(map[string]MbgInfo),
	InactiveMbgArr:   make(map[string]MbgInfo),
	MyServices:       make(map[string]LocalService),
	RemoteServices:   make(map[string][]RemoteService),
	LocalPortMap:     make(map[int]bool),
	Connections:      make(map[string]string),
	RemoteServiceMap: make(map[string][]string),
	MyEventManager:   event.EventManager{},
}
var stopCh = make(map[string]chan bool)

const (
	ProjectFolder = "/.gw/"
	LogFile       = "gw.log"
	DBFile        = "gwApp"
)

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

func GetConnectionArr() map[string]string {
	return s.Connections
}
func GetMbgctlArr() map[string]Mbgctl {
	return s.MbgctlArr
}

func GetDataplane() string {
	return s.MyInfo.Dataplane
}
func GetDataplaneEndpoint() string {
	return s.MyInfo.DataplaneEndpoint
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

func GetEventManager() *event.EventManager {
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
	s.MyInfo.DataplaneEndpoint = "dataplane:443"
	log = logrus.WithField("component", s.MyInfo.Id)
	CreateProjectfolder()
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

func SetConnection(service, port string) {
	s.Connections[service] = port
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

func RemoveMbg(mbg string) {
	mbgArrMutex.Lock()
	delete(s.MbgArr, mbg)
	delete(s.InactiveMbgArr, mbg)
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

// Return Function fields
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

func LookupLocalService(label, ip string) (LocalService, error) {
	// Need to look up the label to find local service
	// If label isnt found, Check for IP.
	// If we cant find the service, we get the "service id" as a wildcard
	// which is sent to the policy engine to decide.
	localSvc, err := LookupLocalServiceFromLabel(label)
	if err != nil {
		log.Infof("Unable to find id local service for label: %v, error: %v", label, err)
		localSvc, err = LookupLocalServiceFromIP(ip)
		if err != nil {
			log.Infof("Unable to find id local service for ip: %v, error: %v", ip, err)
		}
	}
	return localSvc, err
}
func LookupLocalServiceFromLabel(label string) (LocalService, error) {
	for _, service := range s.MyServices {
		// Compare Service Labels
		if service.Id == label {
			return service, nil
		}
	}
	// If the local app/service is not defined, we send the name as a "wildcard"
	return LocalService{Id: "*", Ip: "", Description: ""}, errors.New("unable to find local service")
}

func LookupLocalServiceFromIP(network string) (LocalService, error) {
	serviceNetwork := strings.Split(network, ":")
	for _, service := range s.MyServices {
		// Compare Service IPs
		if strings.Split(service.Ip, ":")[0] == serviceNetwork[0] {
			return service, nil
		}
	}
	// If the local app/service is not defined, we send the name as a "wildcard"
	return LocalService{Id: "*", Ip: "", Description: ""}, errors.New("unable to find local service")
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
	if _, ok := s.MbgArr[id]; ok {
		mbgArrMutex.RLock()
		mbgI := s.MbgArr[id]
		mbgArrMutex.RUnlock()
		return mbgI.Ip + mbgI.Cport.External
	} else {
		log.Errorf("Peer(%s) does not exist", id)
		return ""
	}

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

func IsMbgInactivePeer(id string) bool {
	mbgArrMutex.RLock()
	_, ok := s.InactiveMbgArr[id]
	mbgArrMutex.RUnlock()
	return ok
}

func IsServiceLocal(id string) bool {
	_, exist := s.MyServices[id]
	return exist
}

func AddMbgNbr(id, ip, cport string) {
	mbgArrMutex.Lock()
	if _, ok := s.MbgArr[id]; ok {
		log.Infof("Neighbor already added %s", id)
		mbgArrMutex.Unlock()
		return
	}
	s.MbgArr[id] = MbgInfo{Id: id, Ip: ip, Cport: ServicePort{External: ":" + cport, Local: ""}}
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

// Frees up used ports by a connection
func FreeUpPorts(connectionID string) {
	log.Infof("Start to FreeUpPorts for service: %s", connectionID)
	port, _ := s.Connections[connectionID]
	lval, _ := strconv.Atoi(port[1:])
	stopCh[connectionID] <- true
	delete(s.LocalPortMap, lval)
	delete(s.Connections, connectionID)
	SaveState()
}

func AddLocalService(id, ip string, port uint16) {
	if _, ok := s.MyServices[id]; ok {
		log.Infof("Local Service already added %s", id) //Allow overwrite service
	}
	s.MyServices[id] = LocalService{Id: id, Ip: ip, Port: strconv.Itoa(int(port))}
	log.Infof("Adding local service: %s", id)
	PrintState()
	SaveState()
}

func AddPeerLocalService(id, peer string) {
	if val, ok := s.MyServices[id]; ok {
		_, exist := exists(val.PeersExposed, peer)
		if !exist {
			val.PeersExposed = append(val.PeersExposed, peer)
			s.MyServices[id] = val
			SaveState()
		} else {
			log.Warnf("Peer: %s already exposed", peer)
		}
	}
}

func DelPeerLocalService(id, peer string) {
	if val, ok := s.MyServices[id]; ok {
		index, exist := exists(val.PeersExposed, peer)
		if !exist {
			return
		}
		val.PeersExposed = append((val.PeersExposed)[:index], (val.PeersExposed)[index+1:]...)
		s.MyServices[id] = val
	}
	SaveState()
}

func DelLocalService(id string) {
	if _, ok := s.MyServices[id]; ok {
		delete(s.MyServices, id)
		log.Infof("Delete local service: %s", id)
		PrintState()
		SaveState()
		return
	} else {
		log.Errorf("Local Service %s doesn't exist", id)
	}
}

func exists(slice []string, entry string) (int, bool) {
	for i, e := range slice {
		if e == entry {
			return i, true
		}
	}
	return -1, false
}

func CreateImportService(importId string) {
	if _, ok := s.RemoteServices[importId]; ok {
		log.Infof("Import service:[%v] already exist", importId)

	} else {
		s.RemoteServiceMap[importId] = []string{}
		s.RemoteServices[importId] = []RemoteService{}
		log.Infof("Create import service:[%v] ", importId)

	}

	PrintState()
	SaveState()
}

func AddRemoteService(id, ip, description, MbgId string) {
	svc := RemoteService{Id: id, MbgId: MbgId, MbgIp: ip, Description: description}
	if mbgs, ok := s.RemoteServiceMap[id]; ok {
		_, exist := exists(mbgs, MbgId)
		if !exist {
			s.RemoteServiceMap[id] = append(mbgs, MbgId)
			s.RemoteServices[id] = append(s.RemoteServices[id], svc)
		}
	} else {
		s.RemoteServiceMap[id] = []string{MbgId}
		s.RemoteServices[id] = []RemoteService{svc}
	}
	log.Infof("Adding remote service: [%v]", s.RemoteServiceMap[id])
	PrintState()
	SaveState()
}

func DelRemoteService(id, mbg string) {
	if _, ok := s.RemoteServices[id]; ok {
		if mbg == "" { //delete service for all MBgs
			delete(s.RemoteServices, id)
			delete(s.RemoteServiceMap, id)
			FreeUpPorts(id)
			log.Infof("Delete Remote service: %s", id)
			GetEventManager().RaiseRemoveRemoteServiceEvent(eventManager.RemoveRemoteServiceAttr{Service: id, Mbg: mbg})

			PrintState()
		} else {
			RemoveMbgFromService(id, mbg, s.RemoteServiceMap[id])
		}
	} else {
		log.Errorf("Remote Service %s doesn't exist", id)
	}
}

func RemoveMbgFromServiceMap(mbg string) {
	for svc, mbgs := range s.RemoteServiceMap {
		RemoveMbgFromService(svc, mbg, mbgs)
	}
}

func RemoveMbgFromService(svcId, mbg string, mbgs []string) {
	//Remove from service map
	index, exist := exists(mbgs, mbg)
	if !exist {
		return
	}
	s.RemoteServiceMap[svcId] = append((mbgs)[:index], (mbgs)[index+1:]...)
	log.Infof("MBG removed from remote service %v->[%+v]", svcId, s.RemoteServiceMap[svcId])
	//Remove from service array
	for idx, reSvc := range s.RemoteServices[svcId] {
		if reSvc.MbgId == mbg {
			s.RemoteServices[svcId] = append((s.RemoteServices[svcId])[:idx], (s.RemoteServices[svcId])[idx+1:]...)
			break
		}
	}
	log.Infof("MBG service %v len %v", svcId, len(s.RemoteServiceMap[svcId]))
	if len(s.RemoteServiceMap[svcId]) == 0 {
		delete(s.RemoteServices, svcId)
		delete(s.RemoteServiceMap, svcId)
		FreeUpPorts(svcId)
		GetEventManager().RaiseRemoveRemoteServiceEvent(eventManager.RemoveRemoteServiceAttr{Service: svcId, Mbg: ""}) //remove the service
	} else { //remove specific mbg from the mbg
		GetEventManager().RaiseRemoveRemoteServiceEvent(eventManager.RemoveRemoteServiceAttr{Service: svcId, Mbg: mbg})
	}
	SaveState()
}

func (s *LocalService) GetIpAndPort() string {
	//Support only DNS target
	target := s.Ip + ":" + s.Port
	return target
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
		services = services + se.Id + ", "
	}
	log.Infof("Myservices: %v", services)
	log.Infof("Remoteservices: %v", s.RemoteServiceMap)
	log.Infof("Connections: %v", s.Connections)
	log.Infof("****************************")
}

func CreateProjectfolder() string {
	usr, _ := user.Current()
	fol := path.Join(usr.HomeDir, ProjectFolder)
	//Create folder
	err := os.MkdirAll(fol, 0755)
	if err != nil {
		log.Println(err)
	}
	return fol
}

/** Database **/
func configPath() string {
	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ProjectFolder, DBFile)

}

func SaveState() {
	dataMutex.Lock()
	jsonC, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		log.Errorf("Unable to write json file Error: %v", err)
		dataMutex.Unlock()
		return
	}
	ioutil.WriteFile(configPath(), jsonC, 0644) // os.ModeAppend)
	dataMutex.Unlock()
}

func readState() mbgState {
	dataMutex.Lock()
	data, _ := ioutil.ReadFile(configPath())
	var state mbgState
	json.Unmarshal(data, &state)
	//Don't change part of the Fields
	state.MyEventManager.HttpClient = s.MyEventManager.HttpClient
	dataMutex.Unlock()
	return state
}
