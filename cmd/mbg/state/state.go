package state

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os/user"
	"path"
	"strconv"
	"strings"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type mbgState struct {
	MyInfo         MbgInfo
	ClusterArr     map[string]LocalCluster
	MbgArr         map[string]MbgInfo
	MyServices     map[string]LocalService
	RemoteServices map[string]RemoteService
}

type MbgInfo struct {
	Id                  string
	Ip                  string
	Cport               string
	LocalDataPortRange  int
	ExposeDataPortRange int
}

type LocalCluster struct {
	Id string
	Ip string
}

type RemoteService struct {
	Service        service.Service
	MbgId          string // For now to identify a service to a MBG
	LocalDataPort  string
	ExposeDataPort string
}

type LocalService struct {
	Service        service.Service
	LocalDataPort  string
	ExposeDataPort string
}

var s = mbgState{MyInfo: MbgInfo{},
	ClusterArr:     make(map[string]LocalCluster),
	MbgArr:         make(map[string]MbgInfo),
	MyServices:     make(map[string]LocalService),
	RemoteServices: make(map[string]RemoteService)}

func GetMyIp() string {
	return s.MyInfo.Ip
}

func GetMyId() string {
	return s.MyInfo.Id
}

func GetMyCport() string {
	return s.MyInfo.Cport
}

func GetMyInfo() MbgInfo {
	return s.MyInfo
}

func GetMbgArr() map[string]MbgInfo {
	return s.MbgArr
}

func GetLocalClusterArr() map[string]LocalCluster {
	return s.ClusterArr
}

func SetState(id, ip, cport string, localDataPortRange, exposeDataPortRange int) {
	s.MyInfo.Id = id
	s.MyInfo.Ip = ip
	s.MyInfo.Cport = cport
	s.MyInfo.LocalDataPortRange = localDataPortRange
	s.MyInfo.ExposeDataPortRange = exposeDataPortRange
	SaveState()
}

func SetLocalCluster(id, ip string) {
	log.Println(s)
	s.ClusterArr[id] = LocalCluster{Id: id, Ip: ip}
	SaveState()
}

func UpdateState() {
	s = readState()
}

//Return Function fields
func GetLocalService(id string) LocalService {
	return s.MyServices[id]
}

func GetRemoteService(id string) RemoteService {
	return s.RemoteServices[id]
}

func GetServiceMbgIp(Ip string) string {
	svcIp := strings.Split(Ip, ":")[0]
	for _, m := range s.MbgArr {
		if m.Ip == svcIp {
			mbgIp := m.Ip + ":" + m.Cport
			return mbgIp
		}
	}

	log.Panicln("[MBG %v]Error]Service %v is not defined", s.MyInfo.Id, Ip)
	return ""
}
func IsServiceLocal(id string) bool {
	_, exist := s.MyServices[id]
	return exist
}

func AddMbgNbr(id, ip, cport string) {
	s.MbgArr[id] = MbgInfo{Id: id, Ip: ip, Cport: cport}
	log.Printf("[MBG %v] add MBG neighbors array %v", s.MyInfo.Id, s.MbgArr[id])
	s.Print()
	SaveState()

}

func AddLocalService(id, ip, domain string) {
	var lp, ep string

	if val, ok := s.MyServices[id]; ok {
		lp = val.LocalDataPort
		ep = val.ExposeDataPort
	} else { //create new allocation for the ports
		lp = strconv.Itoa(s.MyInfo.LocalDataPortRange + len(s.MyServices))
		ep = strconv.Itoa(s.MyInfo.ExposeDataPortRange + len(s.MyServices))
	}

	if s.MyServices == nil {
		s.MyServices = make(map[string]LocalService)
	}

	s.MyServices[id] = LocalService{Service: service.Service{id, ip, domain, ""}, LocalDataPort: lp, ExposeDataPort: ep}
	log.Printf("[MBG %v] addd service %v", s.MyInfo.Id, service.GetService(id))
	s.Print()
	SaveState()
}

func AddRemoteService(id, ip, domain, MbgId string) {
	var lp, ep string

	if val, ok := s.RemoteServices[id]; ok {
		lp = val.LocalDataPort
		ep = val.ExposeDataPort
	} else { //create new allocation for the ports
		lp = strconv.Itoa(s.MyInfo.LocalDataPortRange + len(s.RemoteServices))
		ep = strconv.Itoa(s.MyInfo.ExposeDataPortRange + len(s.RemoteServices))
	}

	if s.RemoteServices == nil {
		s.RemoteServices = make(map[string]RemoteService)
	}

	s.RemoteServices[id] = RemoteService{Service: service.Service{id, ip, domain, "Forward"}, MbgId: MbgId, LocalDataPort: lp, ExposeDataPort: ep}
	log.Printf("[MBG %v] addd service %v", s.MyInfo.Id, service.GetService(id))
	s.Print()
	SaveState()
}

func (s *mbgState) Print() {
	log.Printf("[MBG %v]: Id: %v Ip: %v Cport %v", s.MyInfo.Id, s.MyInfo.Id, s.MyInfo.Ip, s.MyInfo.Cport)
	log.Printf("[MBG %v]: MBG neighbors : %v", s.MyInfo.Id, s.MbgArr)
	log.Printf("[MBG %v]: Myservices: %v", s.MyInfo.Id, s.MyServices)
	log.Printf("[MBG %v]: Remoteservices: %v", s.MyInfo.Id, s.RemoteServices)
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
	log.Println("Update MBG state")
	jsonC, _ := json.MarshalIndent(s, "", "\t")
	log.Println("[DEBUG]: state save in", configPath())
	ioutil.WriteFile(configPath(), jsonC, 0644) // os.ModeAppend)
}

func readState() mbgState {
	data, _ := ioutil.ReadFile(configPath())
	var s mbgState
	json.Unmarshal(data, &s)
	return s
}
