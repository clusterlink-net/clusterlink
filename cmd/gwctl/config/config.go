package config

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

var clog = logrus.WithField("component", "gwctl")

const (
	ProjectFolder = "/.gw/"
	ConfigFile    = "gwctl"
)

type GwctlConfigInterface interface {
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
type GwctlConfig struct {
	MbgIP                  string `json:"MbgIP"`
	Id                     string `json:"Id"`
	CaFile                 string
	CertificateFile        string
	KeyFile                string
	Dataplane              string
	PolicyDispatcherTarget string
	GwctlConfigInterface
}

func GetConfig(id string) (GwctlConfig, error) {
	c, err := readState(id)
	return c, err
}

func (c *GwctlConfig) GetMbgIP() string {
	return c.MbgIP
}

func (c *GwctlConfig) GetId() string {
	return c.Id
}
func (c *GwctlConfig) GetDataplane() string {
	return c.Dataplane
}
func (c *GwctlConfig) GetCertificate() string {
	return c.CertificateFile
}
func (c *GwctlConfig) GetCaFile() string {
	return c.CaFile
}

func (c *GwctlConfig) GetKeyFile() string {
	return c.KeyFile
}

func (c *GwctlConfig) GetPolicyDispatcher() string {
	return c.PolicyDispatcherTarget
}

func (c *GwctlConfig) Set(id, mbgIp, caFile, certificateFile, keyFile, dataplane string) error {
	c.Id = id
	c.MbgIP = mbgIp
	c.Dataplane = dataplane
	c.CertificateFile = certificateFile
	c.KeyFile = keyFile
	c.CaFile = caFile
	CreateProjectfolder()
	return c.CreateState()
}

func (c *GwctlConfig) SetPolicyDispatcher(mId, targetUrl string) error {
	c.PolicyDispatcherTarget = targetUrl
	return c.SaveState()
}

func (c *GwctlConfig) Print() {
	fmt.Printf("Id: %v,  mbgTarget: %v", c.Id, c.MbgIP)
}

/********************************/
/******** Config functions **********/
/********************************/
func CreateProjectfolder() string {
	usr, _ := user.Current()
	fol := path.Join(usr.HomeDir, ProjectFolder)
	//Create folder
	err := os.MkdirAll(fol, 0755)
	if err != nil {
		clog.Errorln(err)
	}
	return fol
}

func GwctlPath(id string) string {
	cfgFile := ConfigFile
	if id != "" {
		cfgFile += "_" + id
	}
	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ProjectFolder, cfgFile)
}

func (c *GwctlConfig) CreateState() error {
	jsonC, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		clog.Errorln("Gwctl create config File", err)
		return err
	}
	f := GwctlPath(c.Id)
	err = ioutil.WriteFile(f, jsonC, 0644) // os.ModeAppend)
	clog.Println("create Gwctl config File:", f)
	if err != nil {
		clog.Errorln("Gwctl- create config File", err)
		return err
	}
	SetDefaultLink(c.Id)
	return nil
}

func (c *GwctlConfig) SaveState() error {
	jsonC, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		clog.Errorln("Gwctl save config File", err)
		return err
	}
	f := GwctlPath(c.Id)
	if c.Id == "" { //get original file
		f, _ = os.Readlink(GwctlPath(c.Id))
	}

	err = ioutil.WriteFile(f, jsonC, 0644) // os.ModeAppend)
	if err != nil {
		clog.Errorln("Gwctl save config File", err)
		return err
	}
	return nil
}

func readState(id string) (GwctlConfig, error) {
	file := GwctlPath(id)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		clog.Errorln("Gwctl config File", err)
		return GwctlConfig{}, err
	}
	var s GwctlConfig
	err = json.Unmarshal(data, &s)
	if err != nil {
		clog.Errorln("Gwctl config File", err)
		return GwctlConfig{}, err
	}
	return s, nil
}

func SetDefaultLink(id string) error {
	file := GwctlPath(id)
	link := GwctlPath("")
	//Check if the file exist
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		clog.Errorf("Gwctl config File with id %v is not exist\n", id)
		return err
	}
	//Remove if the link exist
	if _, err := os.Lstat(link); err == nil {
		os.Remove(link)
	}
	//Create a link
	err := os.Symlink(file, link)
	if err != nil {
		clog.Errorln("Error creating symlink:", err)
		return err
	}
	return nil
}
