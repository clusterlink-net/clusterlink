package state

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

var log = logrus.WithField("component", "mbgctl")

const (
	ProjectFolder = "/.mbg/"
	DBFile        = "mbgctl"
)

type MbgctlState struct {
	MbgIP                  string `json:"MbgIP"`
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

var s = MbgctlState{MbgIP: "", Id: "", Services: make(map[string]MbgctlService), OpenConnections: make(map[string]OpenConnection)}

func GetMbgIP() string {
	return s.MbgIP
}

func GetId() string {
	return s.Id
}

func SetState(id, mbgIp, caFile, certificateFile, keyFile, dataplane string) error {
	s.Id = id
	s.MbgIP = mbgIp
	s.Dataplane = dataplane
	s.CertificateFile = certificateFile
	s.KeyFile = keyFile
	s.CaFile = caFile
	s.PolicyDispatcherTarget = GetAddrStart() + mbgIp + "/policy"
	CreateProjectfolder()
	return SaveState(s.Id)
}

func UpdateState(id string) error {
	var err error
	s, err = readState(id)
	return err
}

// Return Function fields
func GetService(id string) MbgctlService {
	val, ok := s.Services[id]
	if !ok {
		fmt.Printf("Service %v does not exist", id)
	}
	return val
}

func AddService(mId, id, ip, description string) {
	if s.Services == nil {
		s.Services = make(map[string]MbgctlService)
	}

	s.Services[id] = MbgctlService{Service: service.Service{id, ip, description}}
	SaveState(mId)
}
func DelService(mId, id string) {
	if _, ok := s.Services[id]; ok {
		delete(s.Services, id)
		SaveState(mId)
		fmt.Printf("Service %v deleted\n", id)
		return
	} else {
		fmt.Printf("Service %v does not exist\n", id)
	}
}

func (s *MbgctlState) Print() {
	fmt.Printf("Id: %v,  mbgTarget: %v", s.Id, s.MbgIP)
	fmt.Printf("Services %v", s.Services)
}

func AssignPolicyDispatcher(mId, targetUrl string) error {
	s.PolicyDispatcherTarget = targetUrl
	return SaveState(mId)
}

func GetPolicyDispatcher() string {
	return s.PolicyDispatcherTarget
}

func AddOpenConnection(mId, svcId, svcIdDest string, pId int) {
	s.OpenConnections[svcId+":"+svcIdDest] = OpenConnection{SvcId: svcId, SvcIdDest: svcIdDest, PId: pId}
	SaveState(mId)
	log.Info(s.OpenConnections)
}

func CloseOpenConnection(mId, svcId, svcIdDest string) {
	val, ok := s.OpenConnections[svcId+":"+svcIdDest]
	if ok {
		delete(s.OpenConnections, svcId+":"+svcIdDest)
		syscall.Kill(val.PId, syscall.SIGINT)
		log.Infof("[%v]: Delete connection: %v", s.Id, val)
		SaveState(mId)
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

/** logfile **/
func CreateProjectfolder() string {
	usr, _ := user.Current()
	fol := path.Join(usr.HomeDir, ProjectFolder)
	//Create folder
	err := os.MkdirAll(fol, 0755)
	if err != nil {
		log.Println(err)
	}
	return fol
}

// / Json code ////
func configPath(id string) string {
	cfgFile := DBFile
	if id != "" {
		cfgFile += "_" + id
	}

	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ProjectFolder, cfgFile)
}

func SaveState(id string) error {
	jsonC, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath(id), jsonC, 0644) // os.ModeAppend)
}

func readState(id string) (MbgctlState, error) {
	data, err := ioutil.ReadFile(configPath(id))
	if err != nil {
		return MbgctlState{}, err
	}
	var s MbgctlState
	err = json.Unmarshal(data, &s)
	if err != nil {
		return MbgctlState{}, err
	}
	return s, nil
}
