package state

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os/user"
	"path"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type GwService struct {
	Service service.Service
}

type GatewayState struct {
	MbgIP    string `json:"MbgIP"`
	GwIP     string `json:"GwIP"`
	GwId     string `json:"GwId"`
	GwCport  string `json:"GwCport"`
	Services map[string]GwService
}

const (
	ConstPort = 5000
)

var s = GatewayState{MbgIP: "", GwIP: "", GwId: "", Services: make(map[string]GwService)}

func GetMbgIP() string {
	log.Println(s.MbgIP)
	return s.MbgIP
}

func GetGwIP() string {
	return s.GwIP
}

func GetGwId() string {
	return s.GwId
}

func SetState(mbgIp, ip, id, cport string) {
	log.Println(s)
	s.GwId = id
	s.GwIP = ip
	s.MbgIP = mbgIp
	s.GwCport = cport

	SaveState()
}

func UpdateState() {
	s = readState()
}

//Return Function fields
func GetService(id string) GwService {
	return s.Services[id]
}

func AddService(id, ip, domain string) {
	if s.Services == nil {
		s.Services = make(map[string]GwService)
	}

	policy := "" //default:No policy
	s.Services[id] = GwService{Service: service.Service{id, ip, domain, policy}}
	log.Printf("[Gateway %v] Add service: %v", s.GwId, s.Services[id])
	s.Print()
	SaveState()

}

func (s *GatewayState) Print() {
	log.Printf("[Gateway %v]: Id: %v ip: %v mbgip: %v", s.GwId, s.GwId, s.GwIP, s.MbgIP)
	log.Printf("[Gateway %v]: services %v", s.GwId, s.Services)
}

/// Json code ////
func configPath() string {
	cfgFile := ".gatewayApp"
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

func readState() GatewayState {
	data, _ := ioutil.ReadFile(configPath())
	var s GatewayState
	json.Unmarshal(data, &s)
	return s
}
