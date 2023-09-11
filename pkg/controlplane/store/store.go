package store

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
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

	event "github.ibm.com/mbg-agent/pkg/controlplane/eventmanager"
	"github.ibm.com/mbg-agent/pkg/utils/netutils"
)

var log = logrus.WithField("component", s.MyInfo.ID)
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
	ID                string
	IP                string
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
	ID string
	IP string
}

type RemoteService struct {
	ID          string
	MbgID       string
	MbgIP       string
	Description string
}

type LocalService struct {
	ID           string
	IP           string
	Port         string
	Description  string
	PeersExposed []string // TODO: not unique
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

const (
	ProjectFolder = "/.gw/"
	LogFile       = "gw.log"
	DBFile        = "gwApp"
	k8s           = "k8s"
	vm            = "vm"
)

func GetMyIP() string {
	return s.MyInfo.IP
}

func GetMyID() string {
	return s.MyInfo.ID
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
	for m := range s.MbgArr {
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
	s.MyInfo.ID = id
	s.MyInfo.IP = ip
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
	log = logrus.WithField("component", s.MyInfo.ID)
	CreateProjectfolder()
	SaveState()
}

func SetMbgctl(id, ip string) {
	log.Info(s)
	s.MbgctlArr[id] = Mbgctl{ID: id, IP: ip}
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
	log = logrus.WithField("component", s.MyInfo.ID)
}

func restorePeer(id string) {
	peerResp, err := s.MyEventManager.RaiseAddPeerEvent(event.AddPeerAttr{PeerMbg: id})
	if err != nil {
		log.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == event.Deny {
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

func ActivateMbg(mbgID string) {
	log.Infof("Activating MBG %s", mbgID)
	peerResp, err := s.MyEventManager.RaiseAddPeerEvent(event.AddPeerAttr{PeerMbg: mbgID})
	if err != nil {
		log.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == event.Deny {
		log.Infof("Denying add peer(%s) due to policy", mbgID)
		return
	}
	mbgArrMutex.Lock()
	mbgI, ok := s.InactiveMbgArr[mbgID]
	if !ok {
		// Ignore
		mbgArrMutex.Unlock()
		return
	}
	s.MbgArr[mbgID] = mbgI
	delete(s.InactiveMbgArr, mbgID)
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
		if service.ID == label {
			return service, nil
		}
	}
	// If the local app/service is not defined, we send the name as a "wildcard"
	return LocalService{ID: "*", IP: "", Description: ""}, errors.New("unable to find local service")
}

func LookupLocalServiceFromIP(network string) (LocalService, error) {
	serviceNetwork := strings.Split(network, ":")
	for _, service := range s.MyServices {
		// Compare Service IPs
		if strings.Split(service.IP, ":")[0] == serviceNetwork[0] {
			return service, nil
		}
	}
	// If the local app/service is not defined, we send the name as a "wildcard"
	return LocalService{ID: "*", IP: "", Description: ""}, errors.New("unable to find local service")
}

func GetServiceMbgIP(ip string) string {
	svcIP := strings.Split(ip, ":")[0]
	mbgArrMutex.RLock()
	for _, m := range s.MbgArr {
		if m.IP == svcIP {
			mbgIP := m.IP + m.Cport.External
			return mbgIP
		}
	}
	mbgArrMutex.RUnlock()
	log.Errorf("Service %v is not defined", ip)
	PrintState()
	return ""
}

func GetMbgTarget(id string) string {
	if _, ok := s.MbgArr[id]; ok {
		mbgArrMutex.RLock()
		mbgI := s.MbgArr[id]
		mbgArrMutex.RUnlock()
		return mbgI.IP + mbgI.Cport.External
	} else {
		log.Errorf("Peer(%s) does not exist", id)
		return ""
	}

}

func GetMbgTargetPair(id string) (string, string) {
	mbgArrMutex.RLock()
	mbgI := s.MbgArr[id]
	mbgArrMutex.RUnlock()
	return mbgI.IP, mbgI.Cport.External
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
	s.MbgArr[id] = MbgInfo{ID: id, IP: ip, Cport: ServicePort{External: ":" + cport, Local: ""}}
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
	delete(s.Connections, connectionID)
	SaveState()
}

func AddLocalService(id, ip string, port uint16) {
	if _, ok := s.MyServices[id]; ok {
		log.Infof("Local Service already added %s", id) // Allow overwrite service
	}
	s.MyServices[id] = LocalService{ID: id, IP: ip, Port: strconv.Itoa(int(port))}
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

func CreateImportService(importID string) {
	if _, ok := s.RemoteServices[importID]; ok {
		log.Infof("Import service:[%v] already exist", importID)

	} else {
		s.RemoteServiceMap[importID] = []string{}
		s.RemoteServices[importID] = []RemoteService{}
		log.Infof("Create import service:[%v] ", importID)

	}

	PrintState()
	SaveState()
}

func AddRemoteService(id, ip, description, mbgID string) {
	svc := RemoteService{ID: id, MbgID: mbgID, MbgIP: ip, Description: description}
	if mbgs, ok := s.RemoteServiceMap[id]; ok {
		_, exist := exists(mbgs, mbgID)
		if !exist {
			s.RemoteServiceMap[id] = append(mbgs, mbgID)
			s.RemoteServices[id] = append(s.RemoteServices[id], svc)
		}
	} else {
		s.RemoteServiceMap[id] = []string{mbgID}
		s.RemoteServices[id] = []RemoteService{svc}
	}
	log.Infof("Adding remote service: [%v]", s.RemoteServiceMap[id])
	PrintState()
	SaveState()
}

func DelRemoteService(id, mbg string) {
	if _, ok := s.RemoteServices[id]; ok {
		if mbg == "" { // delete service for all MBgs
			delete(s.RemoteServices, id)
			delete(s.RemoteServiceMap, id)
			FreeUpPorts(id)
			log.Infof("Delete Remote service: %s", id)
			if err := GetEventManager().RaiseRemoveRemoteServiceEvent(
				event.RemoveRemoteServiceAttr{
					Service: id,
					Mbg:     mbg,
				}); err != nil {
				log.Errorf("failed to raise remote service removal event for %s: %+v", id, err)
			}

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

func RemoveMbgFromService(svcID, mbg string, mbgs []string) {
	// Remove from service map
	index, exist := exists(mbgs, mbg)
	if !exist {
		return
	}
	s.RemoteServiceMap[svcID] = append((mbgs)[:index], (mbgs)[index+1:]...)
	log.Infof("MBG removed from remote service %v->[%+v]", svcID, s.RemoteServiceMap[svcID])
	// Remove from service array
	for idx, reSvc := range s.RemoteServices[svcID] {
		if reSvc.MbgID == mbg {
			s.RemoteServices[svcID] = append((s.RemoteServices[svcID])[:idx], (s.RemoteServices[svcID])[idx+1:]...)
			break
		}
	}
	log.Infof("MBG service %v provided by %d peers", svcID, len(s.RemoteServiceMap[svcID]))
	if len(s.RemoteServiceMap[svcID]) == 0 {
		delete(s.RemoteServices, svcID)
		delete(s.RemoteServiceMap, svcID)
		FreeUpPorts(svcID)
		// TODO: handle the error?
		_ = GetEventManager().RaiseRemoveRemoteServiceEvent(event.RemoveRemoteServiceAttr{Service: svcID, Mbg: ""}) // remove the service
	} else { // remove specific mbg from the mbg
		// TODO: handle the error?
		_ = GetEventManager().RaiseRemoveRemoteServiceEvent(event.RemoveRemoteServiceAttr{Service: svcID, Mbg: mbg})
	}
	SaveState()
}

func (s *LocalService) GetIPAndPort() string {
	// Support only DNS target
	target := s.IP + ":" + s.Port
	return target
}

func GetAddrStart() string {
	if s.MyInfo.Dataplane == "mtls" {
		return "https://"
	} else {
		return "http://"
	}
}

func GetHTTPClient() http.Client {
	if s.MyInfo.Dataplane == "mtls" {
		cert, err := os.ReadFile(s.MyInfo.CaFile)
		if err != nil {
			log.Fatalf("could not open certificate file: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)

		certificate, err := tls.LoadX509KeyPair(s.MyInfo.CertificateFile, s.MyInfo.KeyFile)
		if err != nil {
			log.Fatalf("could not load certificate: %v", err)
		}

		tlsConfig := netutils.ConfigureSafeTLSConfig()
		tlsConfig.RootCAs = caCertPool
		tlsConfig.Certificates = []tls.Certificate{certificate}
		tlsConfig.ServerName = s.MyInfo.ID

		client := http.Client{
			Timeout: 3 * time.Minute,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
		return client
	} else {
		return http.Client{}
	}
}

func PrintState() {
	log.Infof("****** MBG State ********")
	log.Infof("ID: %v IP: %v%v", s.MyInfo.ID, s.MyInfo.IP, s.MyInfo.Cport)
	nb := ""
	inb := ""
	services := ""
	mbgArrMutex.RLock()
	for _, n := range s.MbgArr {
		nb = nb + n.ID + " "
	}
	mbgArrMutex.RUnlock()

	log.Infof("MBG neighbors : %s", nb)
	for _, n := range s.InactiveMbgArr {
		inb = inb + n.ID + ", "
	}
	log.Infof("Inactive MBG neighbors : %s", inb)
	for _, se := range s.MyServices {
		services = services + se.ID + ", "
	}
	log.Infof("Myservices: %v", services)
	log.Infof("Remoteservices: %v", s.RemoteServiceMap)
	log.Infof("Connections: %v", s.Connections)
	log.Infof("****************************")
}

func CreateProjectfolder() string {
	usr, _ := user.Current()
	fol := path.Join(usr.HomeDir, ProjectFolder)
	// Create folder
	err := os.MkdirAll(fol, 0755)
	if err != nil {
		log.Println(err)
	}
	return fol
}

/** Database **/
func configPath() string {
	// set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ProjectFolder, DBFile)

}

func SaveState() {
	dataMutex.Lock()
	defer dataMutex.Unlock()
	jsonC, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		log.Errorf("Unable to write json file Error: %v", err)
		return
	}
	if err = os.WriteFile(configPath(), jsonC, 0600); err != nil {
		log.Errorf("unable to write config file %s: %v", configPath(), err)
	}
}

func readState() mbgState {
	dataMutex.Lock()
	defer dataMutex.Unlock()

	data, err := os.ReadFile(configPath())
	if err != nil {
		return mbgState{}
	}

	var state mbgState
	if err = json.Unmarshal(data, &state); err != nil {
		return mbgState{}
	}
	// Don't change part of the Fields
	state.MyEventManager.HTTPClient = s.MyEventManager.HTTPClient
	return state
}
