package database

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"time"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "gwctl")

const (
	ProjectFolder = "/.gw/"
	DBFile        = "gwctl"
)

type GwctlState struct {
	MbgIP                  string `json:"MbgIP"`
	Id                     string `json:"Id"`
	CaFile                 string
	CertificateFile        string
	KeyFile                string
	Dataplane              string
	PolicyDispatcherTarget string
}

var s = GwctlState{MbgIP: "", Id: ""}

func GetMbgIP() string {
	return s.MbgIP
}

func GetId() string {
	return s.Id
}
func GetState() (GwctlState, error) {
	m, err := readState(s.Id)
	return m, err
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
	return CreateState(s.Id)
}

func UpdateState(id string) error {
	var err error
	s, err = readState(id)
	return err
}

func (s *GwctlState) Print() {
	fmt.Printf("Id: %v,  mbgTarget: %v", s.Id, s.MbgIP)
}

func AssignPolicyDispatcher(mId, targetUrl string) error {
	s.PolicyDispatcherTarget = targetUrl
	return SaveState(mId)
}

func GetPolicyDispatcher() string {
	return s.PolicyDispatcherTarget
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

/** DB file **/
func CreateProjectfolder() string {
	usr, _ := user.Current()
	fol := path.Join(usr.HomeDir, ProjectFolder)
	//Create folder
	err := os.MkdirAll(fol, 0755)
	if err != nil {
		log.Errorln(err)
	}
	return fol
}

func GwctlPath(id string) string {
	cfgFile := DBFile
	if id != "" {
		cfgFile += "_" + id
	}
	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ProjectFolder, cfgFile)
}

func CreateState(id string) error {
	jsonC, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		log.Errorln("Gwctl create config File", err)
		return err
	}
	f := GwctlPath(id)
	err = ioutil.WriteFile(f, jsonC, 0644) // os.ModeAppend)
	log.Println("create Gwctl config File:", f)
	if err != nil {
		log.Errorln("Gwctl- create config File", err)
		return err
	}
	SetDefaultLink(id)
	return nil
}

func SaveState(id string) error {
	jsonC, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		log.Errorln("Gwctl save config File", err)
		return err
	}
	f := GwctlPath(id)
	if id == "" { //get original file
		f, _ = os.Readlink(GwctlPath(id))
	}

	err = ioutil.WriteFile(f, jsonC, 0644) // os.ModeAppend)
	if err != nil {
		log.Errorln("Gwctl save config File", err)
		return err
	}
	return nil
}

func readState(id string) (GwctlState, error) {
	file := GwctlPath(id)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Errorln("Gwctl config File", err)
		return GwctlState{}, err
	}
	var s GwctlState
	err = json.Unmarshal(data, &s)
	if err != nil {
		log.Errorln("Gwctl config File", err)
		return GwctlState{}, err
	}
	return s, nil
}

func SetDefaultLink(id string) error {
	file := GwctlPath(id)
	link := GwctlPath("")
	//Check if the file exist
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		log.Errorf("Gwctl config File with id %v is not exist\n", id)
		return err
	}
	//Remove if the link exist
	if _, err := os.Lstat(link); err == nil {
		os.Remove(link)
	}
	//Create a link
	err := os.Symlink(file, link)
	if err != nil {
		log.Errorln("Error creating symlink:", err)
		return err
	}
	return nil
}
