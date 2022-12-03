package state

import (
	"encoding/json"
	"io/ioutil"
	"os/user"
	"path"
	"syscall"

	log "github.com/sirupsen/logrus"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type ClusterState struct {
	MbgIP           string      `json:"MbgIP"`
	IP              string      `json:"IP"`
	Id              string      `json:"Id"`
	Cport           ClusterPort `json:"Cport"`
	Services        map[string]ClusterService
	OpenConnections map[string]OpenConnection
}

type ClusterPort struct {
	Local    string
	External string
}

type ClusterService struct {
	Service service.Service
}

type OpenConnection struct {
	SvcId     string
	SvcIdDest string
	PId       int
}

const (
	ConstPort = 5000
)

var s = ClusterState{MbgIP: "", IP: "", Id: "", Services: make(map[string]ClusterService), OpenConnections: make(map[string]OpenConnection)}

func GetMbgIP() string {
	return s.MbgIP
}

func GetIP() string {
	return s.IP
}

func GetId() string {
	return s.Id
}

func GetCport() ClusterPort {
	return s.Cport
}
func SetState(ip, id, mbgIp, cportLocal, cportExternal string) {
	s.Id = id
	s.IP = ip
	s.Cport.Local = cportLocal
	s.Cport.External = cportExternal
	s.MbgIP = mbgIp

	SaveState()
}

func UpdateState() {
	s = readState()
}

//Return Function fields
func GetService(id string) ClusterService {
	val, ok := s.Services[id]
	if !ok {
		log.Fatalf("Service %v is not exist", id)
	}
	return val
}

func AddService(id, ip, domain string) {
	if s.Services == nil {
		s.Services = make(map[string]ClusterService)
	}

	s.Services[id] = ClusterService{Service: service.Service{id, ip, domain}}
	SaveState()
	log.Infof("[Cluster %v] Add service: %v", s.Id, s.Services[id])
	s.Print()
}

func (s *ClusterState) Print() {
	log.Infof("[Cluster %v]: Id: %v ip: %v mbgip: %v", s.Id, s.Id, s.IP, s.MbgIP)
	log.Infof("[Cluster %v]: services %v", s.Id, s.Services)
}

func AddOpenConnection(svcId, svcIdDest string, pId int) {
	s.OpenConnections[svcId+":"+svcIdDest] = OpenConnection{SvcId: svcId, SvcIdDest: svcIdDest, PId: pId}
	SaveState()
	log.Info(s.OpenConnections)
}

func CloseOpenConnection(svcId, svcIdDest string) {
	val, ok := s.OpenConnections[svcId+":"+svcIdDest]
	if ok {
		delete(s.OpenConnections, svcId+":"+svcIdDest)
		syscall.Kill(val.PId, syscall.SIGINT)
		log.Infof("[Cluster %v]: Delete connection: %v", s.Id, val)
		SaveState()
	} else {
		log.Fatal("[Cluster %v]: connection from service %v to service %v is not exist \n", s.Id, svcId, svcIdDest)
	}
}

/// Json code ////
func configPath() string {
	cfgFile := ".clusterApp"
	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, cfgFile)

	//set cfg file in the git
	//_, filename, _, _ := runtime.Caller(1)
	//return path.Join(path.Dir(filename), cfgFile)

}

func SaveState() {
	jsonC, _ := json.MarshalIndent(s, "", "\t")
	ioutil.WriteFile(configPath(), jsonC, 0644) // os.ModeAppend)
}

func readState() ClusterState {
	data, _ := ioutil.ReadFile(configPath())
	var s ClusterState
	json.Unmarshal(data, &s)
	return s
}
