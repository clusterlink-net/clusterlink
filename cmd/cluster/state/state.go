package state

import (
	"encoding/json"
	"io/ioutil"
	"os/user"
	"path"

	log "github.com/sirupsen/logrus"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type ClusterService struct {
	Service service.Service
}

type ClusterState struct {
	MbgIP    string      `json:"MbgIP"`
	IP       string      `json:"IP"`
	Id       string      `json:"Id"`
	Cport    ClusterPort `json:"Cport"`
	Services map[string]ClusterService
}

type ClusterPort struct {
	Local    string
	External string
}

const (
	ConstPort = 5000
)

var s = ClusterState{MbgIP: "", IP: "", Id: "", Services: make(map[string]ClusterService)}

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

	policy := "" //default:No policy
	s.Services[id] = ClusterService{Service: service.Service{id, ip, domain, policy}}
	SaveState()
	log.Infof("[Cluster %v] Add service: %v", s.Id, s.Services[id])
	s.Print()
}

func (s *ClusterState) Print() {
	log.Infof("[Cluster %v]: Id: %v ip: %v mbgip: %v", s.Id, s.Id, s.IP, s.MbgIP)
	log.Infof("[Cluster %v]: services %v", s.Id, s.Services)
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
