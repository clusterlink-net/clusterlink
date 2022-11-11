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

type LocalService struct {
	Service   service.Service
	LocalPort string
}

type RemoteService struct {
	Service service.Service
	MbgId   string  // For now to identify a service to a MBG
}

type MbgInfo struct {
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
	MyServices map[string]LocalService
	RemoteServices map[string]RemoteService
}

const (
	ConstPort = 5000
)

var s = mbgState{MyInfo: MbgInfo{}, MbgArr: make(map[string]MbgInfo),
				MyServices: make(map[string]LocalService),
				RemoteServices: make(map[string]RemoteService)}

func GetMyIp() string {
	return s.MyInfo.Ip
}

func GetId() string {
	return s.MyInfo.Id
}
func GetMyInfo() MbgInfo {
	return s.MyInfo
}

func GetMbgArr() map[string]MbgInfo {
	return s.MbgArr
}

func SetState(id, ip string) {
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
func GetLocalService(id string) LocalService {
	return s.MyServices[id]
}

func GetRemoteService(id string) RemoteService {
	return s.RemoteServices[id]
}

func UpdateMbgArr(id, ip string) {
	s.MbgArr[id] = MbgInfo{Id: id, Ip: ip}
	log.Printf("[MBG %v] Update MBG neighbors array %v", s.MyInfo.Id, s.MbgArr[id])
	s.Print()
	SaveState()

}

func UpdateLocalService(id, ip, domain, policy string) {
	p := strconv.Itoa(ConstPort + len(s.MyServices) + 1)
	if s.MyServices == nil {
		s.MyServices = make(map[string]LocalService)
	}

	s.MyServices[id] = LocalService{Service: service.Service{id, ip, domain, policy}, LocalPort: p}
	log.Printf("[MBG %v] Update Local service %v", s.MyInfo.Id, service.GetService(id))
	s.Print()
}

func UpdateRemoteService(id, ip, domain, policy string, mbgId string) {
	if s.RemoteServices == nil {
		s.RemoteServices = make(map[string]RemoteService)
	}

	s.RemoteServices[id] = RemoteService{Service: service.Service{id, ip, domain, policy}, MbgId: mbgId}
	log.Printf("[MBG %v] Update Remote service %v -> Source MBG %v", s.MyInfo.Id, service.GetService(id), mbgId)
	s.Print()
}

func (s *mbgState) Print() {
	log.Printf("[MBG %v]: Id: %v Ip: %v", s.MyInfo.Id, s.MyInfo.Ip)
	log.Printf("[MBG %v]: MBG neighbors : %v", s.MyInfo.Id, s.MbgArr)
	log.Printf("[MBG %v]: Local Services: %v", s.MyInfo.Id, s.MyServices)
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
