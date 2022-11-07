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

type MbgService struct {
	Service   service.Service
	LocalPort string
}

type MbgInfo struct {
	Name string
	Id   string
	Ip   string
}
type LocalGw struct {
	Ip string
}

type mbgState struct {
	MyInfo   MbgInfo
	Gw       LocalGw
	MbgArr   map[string]MbgInfo
	Services map[string]MbgService
}

const (
	ConstPort = 5000
)

var s = mbgState{MyInfo: MbgInfo{}, MbgArr: make(map[string]MbgInfo), Services: make(map[string]MbgService)}

func GetMyIp() string {
	return s.MyInfo.Ip
}

func GetMyName() string {
	return s.MyInfo.Name
}
func GetMyInfo() MbgInfo {
	return s.MyInfo
}

func GetMbgArr() map[string]MbgInfo {
	return s.MbgArr
}

func SetState(name, id, ip string) {
	s.MyInfo.Name = name
	s.MyInfo.Id = id
	s.MyInfo.Ip = ip
	SaveState()
}

func SetLocalGw(ip string) {
	log.Println(s)
	s.Gw.Ip = ip
	SaveState()
}

func UpdateState() {
	s = readState()
}

//Return Function fields
func GetService(name, id string) MbgService {
	return s.Services[name+"_"+id]
}

func UpdateMbgArr(name, id, ip string) {
	s.MbgArr[name+"_"+id] = MbgInfo{Name: name, Ip: ip, Id: id}
	log.Printf("[MBG %v] Update MBG neighbors array %v", s.MyInfo.Name, s.MbgArr[name+"_"+id])
	s.Print()
	SaveState()

}

func UpdateService(name, id, ip, domain, policy string) {
	p := strconv.Itoa(ConstPort + len(s.Services) + 1)
	if s.Services == nil {
		s.Services = make(map[string]MbgService)
	}

	s.Services[name+"_"+id] = MbgService{Service: service.Service{name, id, ip, domain, policy}, LocalPort: p}
	log.Printf("[MBG %v] Update service %v", s.MyInfo.Name, service.GetService(name+id))
	s.Print()
}

func (s *mbgState) Print() {
	log.Printf("[MBG %v]: Name: %v Id: %v Ip: %v", s.MyInfo.Name, s.MyInfo.Id, s.MyInfo.Ip)
	log.Printf("[MBG %v]: MBG neighbors : %v", s.MyInfo.Name, s.MbgArr)
	log.Printf("[MBG %v]: services: %v", s.MyInfo.Name, s.Services)
}

/// Json code ////
func configPath() string {
	cfgFile := ".mbgApp"
	//usr, _ := user.Current()
	//return path.Join(usr.HomeDir, cfgFile)
	_, filename, _, _ := runtime.Caller(1)

	return path.Join(path.Dir(filename), cfgFile)

}

func SaveState() {
	log.Println("Update MBG state")
	jsonC, _ := json.MarshalIndent(s, "", "\t")
	ioutil.WriteFile(configPath(), jsonC, 0644) // os.ModeAppend)
}

func readState() mbgState {
	data, _ := ioutil.ReadFile(configPath())
	var s mbgState
	json.Unmarshal(data, &s)
	return s
}
