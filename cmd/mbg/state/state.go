package state

import (
	"encoding/json"
	"io/ioutil"
	"os/user"
	"path"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

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
	Id            string
	Ip            string
	Cport         ClusterPort
	DataPortRange ClusterPort
}

type LocalCluster struct {
	Id string
	Ip string
}

type RemoteService struct {
	Service  service.Service
	MbgId    string // For now to identify a service to a MBG
	DataPort ClusterPort
}

type LocalService struct {
	Service  service.Service
	DataPort ClusterPort
}

type ClusterPort struct {
	Local    string
	External string
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

func GetMyCport() ClusterPort {
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

func SetState(id, ip, cportLocal, cportExternal, localDataPortRange, externalDataPortRange string) {
	s.MyInfo.Id = id
	s.MyInfo.Ip = ip
	s.MyInfo.Cport.Local = cportLocal
	s.MyInfo.Cport.External = cportExternal
	s.MyInfo.DataPortRange.Local = localDataPortRange
	s.MyInfo.DataPortRange.External = externalDataPortRange
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

func AddMbgNbr(id, ip, cport string) {
	s.MbgArr[id] = MbgInfo{Id: id, Ip: ip, Cport: ClusterPort{External: cport, Local: ""}}
	log.Infof("[MBG %v] add MBG neighbors array %v", s.MyInfo.Id, s.MbgArr[id])
	s.Print()
	SaveState()

}

func AddLocalService(id, ip, domain string) {
	var lp, ep string

	if val, ok := s.MyServices[id]; ok {
		lp = val.DataPort.Local
		ep = val.DataPort.External
	} else { //create new allocation for the ports
		lval, _ := strconv.Atoi(s.MyInfo.DataPortRange.Local)
		eval, _ := strconv.Atoi(s.MyInfo.DataPortRange.External)
		lp = strconv.Itoa(lval + len(s.MyServices))
		ep = strconv.Itoa(eval + len(s.MyServices))
	}

	if s.MyServices == nil {
		s.MyServices = make(map[string]LocalService)
	}

	s.MyServices[id] = LocalService{Service: service.Service{id, ip, domain, ""}, DataPort: ClusterPort{Local: lp, External: ep}}
	log.Infof("[MBG %v] addd service %v", s.MyInfo.Id, service.GetService(id))
	s.Print()
	SaveState()
}

func AddRemoteService(id, ip, domain, MbgId string) {
	var lp, ep string

	if val, ok := s.RemoteServices[id]; ok {
		lp = val.DataPort.Local
		ep = val.DataPort.External
	} else { //create new allocation for the ports
		lval, _ := strconv.Atoi(s.MyInfo.DataPortRange.Local)
		eval, _ := strconv.Atoi(s.MyInfo.DataPortRange.External)
		lp = strconv.Itoa(lval + len(s.RemoteServices))
		ep = strconv.Itoa(eval + len(s.RemoteServices))
	}

	if s.RemoteServices == nil {
		s.RemoteServices = make(map[string]RemoteService)
	}

	s.RemoteServices[id] = RemoteService{Service: service.Service{id, ip, domain, "Forward"}, MbgId: MbgId, DataPort: ClusterPort{Local: lp, External: ep}}
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
