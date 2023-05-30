package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "gwctl")

const (
	ProjectFolder = "/.gw/"
	DBFile        = "gwctl"
)

type GwctlDbInterface interface {
	GetMbgIP() string
	GetId() string
	GetDataplane() string
	GetCertificate() string
	GetCaFile() string
	GetKeyFile() string
	GetPolicyDispatcher() string
	Set(id, mbgIp, caFile, certificateFile, keyFile, dataplane string)
	SetPolicyDispatcher(mId, targetUrl string)
}
type GwctlDb struct {
	MbgIP                  string `json:"MbgIP"`
	Id                     string `json:"Id"`
	CaFile                 string
	CertificateFile        string
	KeyFile                string
	Dataplane              string
	PolicyDispatcherTarget string
	GwctlDbInterface
}

func GetDb(id string) (GwctlDb, error) {
	d, err := readState(id)
	return d, err
}

func (d *GwctlDb) GetMbgIP() string {
	return d.MbgIP
}

func (d *GwctlDb) GetId() string {
	return d.Id
}
func (d *GwctlDb) GetDataplane() string {
	return d.Dataplane
}
func (d *GwctlDb) GetCertificate() string {
	return d.CertificateFile
}
func (d *GwctlDb) GetCaFile() string {
	return d.CaFile
}

func (d *GwctlDb) GetKeyFile() string {
	return d.KeyFile
}

func (d *GwctlDb) GetPolicyDispatcher() string {
	return d.PolicyDispatcherTarget
}

func (d *GwctlDb) Set(id, mbgIp, caFile, certificateFile, keyFile, dataplane string) error {
	d.Id = id
	d.MbgIP = mbgIp
	d.Dataplane = dataplane
	d.CertificateFile = certificateFile
	d.KeyFile = keyFile
	d.CaFile = caFile
	CreateProjectfolder()
	return d.CreateState()
}

func (d *GwctlDb) SetPolicyDispatcher(mId, targetUrl string) error {
	d.PolicyDispatcherTarget = targetUrl
	return d.SaveState()
}

func (d *GwctlDb) Print() {
	fmt.Printf("Id: %v,  mbgTarget: %v", d.Id, d.MbgIP)
}

/********************************/
/******** DB functions **********/
/********************************/
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

func (d *GwctlDb) CreateState() error {
	jsonC, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		log.Errorln("Gwctl create config File", err)
		return err
	}
	f := GwctlPath(d.Id)
	err = ioutil.WriteFile(f, jsonC, 0644) // os.ModeAppend)
	log.Println("create Gwctl config File:", f)
	if err != nil {
		log.Errorln("Gwctl- create config File", err)
		return err
	}
	SetDefaultLink(d.Id)
	return nil
}

func (d *GwctlDb) SaveState() error {
	jsonC, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		log.Errorln("Gwctl save config File", err)
		return err
	}
	f := GwctlPath(d.Id)
	if d.Id == "" { //get original file
		f, _ = os.Readlink(GwctlPath(d.Id))
	}

	err = ioutil.WriteFile(f, jsonC, 0644) // os.ModeAppend)
	if err != nil {
		log.Errorln("Gwctl save config File", err)
		return err
	}
	return nil
}

func readState(id string) (GwctlDb, error) {
	file := GwctlPath(id)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Errorln("Gwctl config File", err)
		return GwctlDb{}, err
	}
	var s GwctlDb
	err = json.Unmarshal(data, &s)
	if err != nil {
		log.Errorln("Gwctl config File", err)
		return GwctlDb{}, err
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
