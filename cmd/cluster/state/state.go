package state

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os/user"
	"path"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type ClusterService struct {
	Service service.Service
}

type ClusterState struct {
	MbgIP    string `json:"MbgIP"`
	IP       string `json:"IP"`
	Id       string `json:"Id"`
	Cport    string `json:"Cport"`
	Services map[string]ClusterService
}

const (
	ConstPort = 5000
)

var s = ClusterState{MbgIP: "", IP: "", Id: "", Services: make(map[string]ClusterService)}

func GetMbgIP() string {
	log.Println(s.MbgIP)
	return s.MbgIP
}

func GetIP() string {
	return s.IP
}

func GetId() string {
	return s.Id
}

func GetCport() string {
	return s.Cport
}
func SetState(ip, id, mbgIp, cport string) {
	log.Println(s)
	s.Id = id
	s.IP = ip
	s.Cport = cport
	s.MbgIP = mbgIp

	SaveState()
}

func UpdateState() {
	s = readState()
}

//Return Function fields
func GetService(id string) ClusterService {
	return s.Services[id]
}

func AddService(id, ip, domain string) {
	if s.Services == nil {
		s.Services = make(map[string]ClusterService)
	}

	policy := "" //default:No policy
	s.Services[id] = ClusterService{Service: service.Service{id, ip, domain, policy}}
	log.Printf("[Cluster %v] Add service: %v", s.Id, s.Services[id])
	s.Print()
	SaveState()

}

func (s *ClusterState) Print() {
	log.Printf("[Cluster %v]: Id: %v ip: %v mbgip: %v", s.Id, s.Id, s.IP, s.MbgIP)
	log.Printf("[Cluster %v]: services %v", s.Id, s.Services)
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
	log.Println(s)
	jsonC, _ := json.MarshalIndent(s, "", "\t")
	ioutil.WriteFile(configPath(), jsonC, 0644) // os.ModeAppend)
}

func readState() ClusterState {
	data, _ := ioutil.ReadFile(configPath())
	var s ClusterState
	json.Unmarshal(data, &s)
	return s
}
