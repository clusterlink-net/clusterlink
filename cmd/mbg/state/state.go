package state

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type mbgState struct {
	MyInfo                MbgInfo
	ClusterArr            map[string]LocalCluster
	MbgArr                map[string]MbgInfo
	MyServices            map[string]LocalService
	RemoteServices        map[string]RemoteService
	Connections           map[string]ClusterPort
	LocalServiceEndpoints map[string]string
	LocalPortMap          map[int]bool
	ExternalPortMap       map[int]bool
}

type MbgInfo struct {
	Id              string
	Ip              string
	Cport           ClusterPort
	DataPortRange   ClusterPort
	MaxPorts        int
	CertificateFile string
	KeyFile         string
	CertData        []byte
	KeyData         []byte
}

type LocalCluster struct {
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

type ClusterPort struct {
	Local    string
	External string
}

var s = mbgState{MyInfo: MbgInfo{},
	ClusterArr:      make(map[string]LocalCluster),
	MbgArr:          make(map[string]MbgInfo),
	MyServices:      make(map[string]LocalService),
	RemoteServices:  make(map[string]RemoteService),
	Connections:     make(map[string]ClusterPort),
	LocalPortMap:    make(map[int]bool),
	ExternalPortMap: make(map[int]bool)}

func GetMyIp() string {
	return s.MyInfo.Ip
}

func GetMyId() string {
	return s.MyInfo.Id
}

func GetMyCport() ClusterPort {
	return s.MyInfo.Cport
}

func GetMyInfo() MbgInfo {
	return s.MyInfo
}

func GetMbgArr() map[string]MbgInfo {
	return s.MbgArr
}

func GetConnectionArr() map[string]ClusterPort {
	return s.Connections
}
func GetLocalClusterArr() map[string]LocalCluster {
	return s.ClusterArr
}

func SetState(id, ip, cportLocal, cportExternal, localDataPortRange, externalDataPortRange, certificate, key string) {
	s.MyInfo.Id = id
	s.MyInfo.Ip = ip
	s.MyInfo.Cport.Local = cportLocal
	s.MyInfo.Cport.External = cportExternal
	s.MyInfo.DataPortRange.Local = localDataPortRange
	s.MyInfo.DataPortRange.External = externalDataPortRange
	s.MyInfo.MaxPorts = 1000 // TODO
	s.MyInfo.CertificateFile = certificate
	s.MyInfo.KeyFile = key
	var err error
	s.MyInfo.CertData, err = os.ReadFile(certificate)
	if err != nil {
		log.Fatal(err)
	}
	s.MyInfo.KeyData, err = os.ReadFile(key)
	if err != nil {
		log.Fatal(err)
	}
	SaveState()
}

func SetLocalCluster(id, ip string) {
	log.Info(s)
	s.ClusterArr[id] = LocalCluster{Id: id, Ip: ip}
	SaveState()
}

func UpdateState() {
	s = readState()
}

//Return Function fields
func GetLocalService(id string) LocalService {
	val, ok := s.MyServices[id]
	if !ok {
		log.Fatalf("Service %v is not exist", id)
	}
	return val
}

func GetRemoteService(id string) RemoteService {
	val, ok := s.RemoteServices[id]
	if !ok {
		log.Fatalf("Service %v is not exist", id)
	}
	return val

}

func GetServiceMbgIp(Ip string) string {
	svcIp := strings.Split(Ip, ":")[0]
	for _, m := range s.MbgArr {
		if m.Ip == svcIp {
			mbgIp := m.Ip + ":" + m.Cport.External
			return mbgIp
		}
	}
	log.Infof("[MBG %v]Error Service %v is not defined", s.MyInfo.Id, Ip)
	s.Print()
	return ""
}
func IsServiceLocal(id string) bool {
	_, exist := s.MyServices[id]
	return exist
}

func AddMbgNbr(id, ip, cport, certFile, keyFile string) {
	s.MbgArr[id] = MbgInfo{Id: id, Ip: ip, Cport: ClusterPort{External: cport, Local: ""}, CertificateFile: certFile, KeyFile: keyFile}
	log.Infof("[MBG %v] add MBG neighbors array %v", s.MyInfo.Id, s.MbgArr[id])
	s.Print()
	SaveState()
}

func UpdateMbgCerts(id, certFile, keyFile string) {
	mbgInfo := s.MbgArr[id]
	mbgInfo.CertificateFile = certFile
	mbgInfo.KeyFile = keyFile
}

// Gets an available free port to use per connection
func GetFreePorts(connectionID string) (ClusterPort, error) {
	if port, ok := s.Connections[connectionID]; ok {
		return port, fmt.Errorf("connection already setup")
	}
	rand.NewSource(time.Now().UnixNano())
	if len(s.Connections) == s.MyInfo.MaxPorts {
		return ClusterPort{}, fmt.Errorf("all Ports taken up, Try again after sometimes")
	}
	lval, _ := strconv.Atoi(s.MyInfo.DataPortRange.Local)
	eval, _ := strconv.Atoi(s.MyInfo.DataPortRange.External)
	timeout := time.After(10 * time.Second)
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return ClusterPort{}, fmt.Errorf("all Ports taken up, Try again after sometimes")
		default:
			random := rand.Intn(s.MyInfo.MaxPorts)
			localPort := lval + random
			externalPort := eval + random
			if !s.LocalPortMap[localPort] {
				log.Infof("[MBG %v] Free Local Port available at %v", s.MyInfo.Id, localPort)
				if !s.ExternalPortMap[externalPort] {
					log.Infof("[MBG %v] Free External Port available at %v", s.MyInfo.Id, externalPort)
					s.LocalPortMap[localPort] = true
					s.ExternalPortMap[externalPort] = true
					myPort := ClusterPort{Local: strconv.Itoa(localPort), External: strconv.Itoa(externalPort)}
					s.Connections[connectionID] = myPort
					SaveState()
					return myPort, nil
				}
			}
		}
	}
}

// Gets an available free port to be used within the cluster for a remote service endpoint
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
				log.Infof("[MBG %v] Free Local Port available at %v", s.MyInfo.Id, localPort)
				s.LocalPortMap[localPort] = true
				myPort := strconv.Itoa(localPort)
				s.LocalServiceEndpoints[serviceName] = ":" + myPort
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

func AddLocalService(id, ip, domain string) {
	s.MyServices[id] = LocalService{Service: service.Service{id, ip, domain}}
	log.Infof("[MBG %v] addd service %v", s.MyInfo.Id, service.GetService(id))
	s.Print()
	SaveState()
}

func AddRemoteService(id, ip, domain, MbgId string) {
	s.RemoteServices[id] = RemoteService{Service: service.Service{id, ip, domain}, MbgId: MbgId}
	log.Infof("[MBG %v] addd service %v", s.MyInfo.Id, service.GetService(id))
	s.Print()
	SaveState()
}

func (s *mbgState) Print() {
	log.Infof("[MBG %v]: Id: %v Ip: %v Cport %v", s.MyInfo.Id, s.MyInfo.Id, s.MyInfo.Ip, s.MyInfo.Cport)
	log.Infof("[MBG %v]: MBG neighbors : %v", s.MyInfo.Id, s.MbgArr)
	log.Infof("[MBG %v]: Myservices: %v", s.MyInfo.Id, s.MyServices)
	log.Infof("[MBG %v]: Remoteservices: %v", s.MyInfo.Id, s.RemoteServices)
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
	log.Infof("Update MBG state")
	jsonC, _ := json.MarshalIndent(s, "", "\t")
	log.Debugf("state save in", configPath())
	ioutil.WriteFile(configPath(), jsonC, 0644) // os.ModeAppend)
}

func readState() mbgState {
	data, _ := ioutil.ReadFile(configPath())
	var s mbgState
	json.Unmarshal(data, &s)
	return s
}
