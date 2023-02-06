package state

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/user"
	"path"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

var log = logrus.WithField("component", "mbgctl")

type MbgctlState struct {
	MbgIP                  string `json:"MbgIP"`
	IP                     string `json:"IP"`
	Id                     string `json:"Id"`
	CaFile                 string
	CertificateFile        string
	KeyFile                string
	Dataplane              string
	Services               map[string]MbgctlService
	OpenConnections        map[string]OpenConnection
	PolicyDispatcherTarget string
}

type MbgctlService struct {
	Service service.Service
}

type OpenConnection struct {
	SvcId     string
	SvcIdDest string
	PId       int
}

var s = MbgctlState{MbgIP: "", IP: "", Id: "", Services: make(map[string]MbgctlService), OpenConnections: make(map[string]OpenConnection)}

func GetMbgIP() string {
	return s.MbgIP
}

func GetIP() string {
	return s.IP
}

func GetId() string {
	return s.Id
}

func SetState(ip, id, mbgIp, caFile, certificateFile, keyFile, dataplane string) {
	s.Id = id
	s.IP = ip
	s.MbgIP = mbgIp
	s.Dataplane = dataplane
	s.CertificateFile = certificateFile
	s.KeyFile = keyFile
	s.CaFile = caFile

	SaveState()
}

func UpdateState() {
	s = readState()
}

//Return Function fields
func GetService(id string) MbgctlService {
	val, ok := s.Services[id]
	if !ok {
		log.Fatalf("Service %v is not exist", id)
	}
	return val
}

func AddService(id, ip, description string) {
	if s.Services == nil {
		s.Services = make(map[string]MbgctlService)
	}

	s.Services[id] = MbgctlService{Service: service.Service{id, ip, description}}
	SaveState()
	log.Debugf("[%v] Add service: %v", s.Id, s.Services[id])
	s.Print()
}

func (s *MbgctlState) Print() {
	log.Debugf("[%v]: Id: %v ip: %v mbgip: %v", s.Id, s.Id, s.IP, s.MbgIP)
	log.Debugf("[%v]: services %v", s.Id, s.Services)
}

func AssignPolicyDispatcher(targetUrl string) {
	s.PolicyDispatcherTarget = targetUrl
	SaveState()
}

func GetPolicyDispatcher() string {
	return s.PolicyDispatcherTarget
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
		log.Infof("[%v]: Delete connection: %v", s.Id, val)
		SaveState()
	}
}

func GetAddrStart() string {
	if s.Dataplane == "mtls" {
		return "https://"
	} else {
		return "http://"
	}
}

func GetHttpClient() http.Client {
	if s.Dataplane == "mtls" {
		cert, err := ioutil.ReadFile(s.CaFile)
		if err != nil {
			log.Fatalf("could not open certificate file: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)

		certificate, err := tls.LoadX509KeyPair(s.CertificateFile, s.KeyFile)
		if err != nil {
			log.Fatalf("could not load certificate: %v", err)
		}

		client := http.Client{
			Timeout: time.Minute * 3,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{certificate},
					ServerName:   s.Id,
				},
			},
		}
		return client
	} else {
		return http.Client{}
	}
}

/// Json code ////
func configPath() string {
	cfgFile := ".mbgctl"
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

func readState() MbgctlState {
	data, _ := ioutil.ReadFile(configPath())
	var s MbgctlState
	json.Unmarshal(data, &s)
	return s
}
