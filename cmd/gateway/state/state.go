package state

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"runtime"
	"strconv"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type GwService struct {
	Service   service.Service
	LocalPort string
}

type GatewayState struct {
	MbgIP    string `json:"MbgIP"`
	GwIP     string `json:"GwIP"`
	GwName   string `json:"GwName"`
	Services map[string]GwService
}

const (
	ConstPort = 5000
)

var s = GatewayState{MbgIP: "", GwIP: "", GwName: "", Services: make(map[string]GwService)}

func GetMbgIP() string {
	log.Println(s.MbgIP)
	return s.MbgIP
}

func GetGwIP() string {
	return s.GwIP
}

func GetGwName() string {
	return s.GwName
}

func SetState(mbgIp, ip, name string) {
	log.Println(s)
	s.GwName = name
	s.GwIP = ip
	s.MbgIP = mbgIp
	SaveState()
}

func UpdateState() {
	s = readState()
}

//Return Function fields
func GetService(name, id string) GwService {
	return s.Services[name+"_"+id]
}

func UpdateService(name, id, ip, domain, policy string) {
	p := strconv.Itoa(ConstPort + len(s.Services) + 1)
	if s.Services == nil {
		s.Services = make(map[string]GwService)
	}

	s.Services[name+"_"+id] = GwService{Service: service.Service{name, id, ip, domain, policy}, LocalPort: p}
	log.Printf("[Gateway %v] Update service %v", s.GwName, service.GetService(name+id))
	s.Print()
}

func (s *GatewayState) Print() {
	log.Printf("[Gateway %v]: Name: %v ip: %v mbgip: %v", s.GwName, s.GwName, s.GwIP, s.MbgIP)
	log.Printf("[Gateway %v]: services %v", s.GwName, s.Services)
}

/// Json code ////
func configPath() string {
	cfgFile := ".gatewayApp"
	//usr, _ := user.Current()
	//return path.Join(usr.HomeDir, cfgFile)
	_, filename, _, _ := runtime.Caller(1)

	return path.Join(path.Dir(filename), cfgFile)

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
