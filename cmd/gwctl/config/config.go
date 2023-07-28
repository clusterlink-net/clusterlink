package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/sirupsen/logrus"
)

const (
	projectFolder = "/.gw/"
	configFile    = "gwctl"
)

// ClientConfig contain all Client configuration to send requests to the GW
type ClientConfig struct {
	GwIP           string        `json:"gwIP"`
	ID             string        `json:"ID"`
	CaFile         string        `json:"CaFile"`
	CertFile       string        `json:"CertFile"`
	KeyFile        string        `json:"KeyFile"`
	Dataplane      string        `json:"Dataplane"`
	PolicyEngineIP string        `json:"PolicyEngineIP"`
	logger         *logrus.Entry `json:"-"`
	ClientConfigInterface
}

// ClientConfigInterface contain all the method of Client
type ClientConfigInterface interface {
	GetGwIP() string
	GetID() string
	GetDataplane() string
	GetCert() string
	GetCaFile() string
	GetKeyFile() string
	GetPolicyEngineIP() string
	NewConfig(id, GwIP, caFile, certificateFile, keyFile, dataplane string)
}

// NewClientConfig create config file with all the configuration of the Client
func NewClientConfig(cfg ClientConfig) (*ClientConfig, error) {
	c := ClientConfig{
		ID:             cfg.ID,
		GwIP:           cfg.GwIP,
		Dataplane:      cfg.Dataplane,
		CertFile:       cfg.CertFile,
		KeyFile:        cfg.KeyFile,
		CaFile:         cfg.CaFile,
		PolicyEngineIP: cfg.PolicyEngineIP,
		logger:         logrus.WithField("component", "gwctl/config"),
	}

	_, err := c.createProjectfolder()
	if err != nil {
		return nil, err
	}
	err = c.createConfigFile()
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetConfigFromID return configuration of Client according to the Client ID
func GetConfigFromID(id string) (*ClientConfig, error) {
	c, err := readConfigFromFile(id)
	return &c, err
}

// GetGwIP return the gw IP that the Client is connected
func (c *ClientConfig) GetGwIP() string {
	return c.GwIP
}

// GetID return the Client ID
func (c *ClientConfig) GetID() string {
	return c.ID
}

// GetDataplane return the Client dataplane type (MTLS or TCP)
func (c *ClientConfig) GetDataplane() string {
	return c.Dataplane
}

// GetCert return the Client certificate
func (c *ClientConfig) GetCert() string {
	return c.CertFile
}

// GetCaFile return the Client certificate Authority
func (c *ClientConfig) GetCaFile() string {
	return c.CaFile
}

// GetKeyFile return the Client key file
func (c *ClientConfig) GetKeyFile() string {
	return c.KeyFile
}

// GetPolicyEngineIP return the policy server address
func (c *ClientConfig) GetPolicyEngineIP() string {
	return c.PolicyEngineIP
}

/********************************/
/******** Config functions **********/
/********************************/
func (c *ClientConfig) createProjectfolder() (string, error) {
	usr, _ := user.Current()
	fol := path.Join(usr.HomeDir, projectFolder)
	//Create folder
	err := os.MkdirAll(fol, 0755)
	if err != nil {
		c.logger.Errorln(err)
		return "", err
	}
	return fol, nil
}

func (c *ClientConfig) createConfigFile() error {
	jsonC, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		c.logger.Errorln("Client create config File", err)
		return err
	}
	f := ClientPath(c.ID)
	err = ioutil.WriteFile(f, jsonC, 0644) // os.ModeAppend)
	c.logger.Println("Create Client config File:", f)
	if err != nil {
		c.logger.Errorln("Creating clinent config File", err)
		return err
	}
	c.SetDefaultClient(c.ID)
	return nil
}

func (c *ClientConfig) saveConfig() error {
	jsonC, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		c.logger.Errorln("Client save config File", err)
		return err
	}
	f := ClientPath(c.ID)
	if c.ID == "" { //get original file
		f, _ = os.Readlink(ClientPath(c.ID))
	}

	err = ioutil.WriteFile(f, jsonC, 0644) // os.ModeAppend)
	if err != nil {
		c.logger.Errorln("Saving config File", err)
		return err
	}
	return nil
}

// SetDefaultClient set the default Client the CLI will use.
func (c *ClientConfig) SetDefaultClient(id string) error {
	file := ClientPath(id)
	link := ClientPath("")
	//Check if the file exist
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		c.logger.Errorf("Client config File with id %v is not exist\n", id)
		return err
	}
	//Remove if the link exist
	if _, err := os.Lstat(link); err == nil {
		os.Remove(link)
	}
	//Create a link
	err := os.Symlink(file, link)
	if err != nil {
		c.logger.Errorln("Error creating symlink:", err)
		return err
	}
	return nil
}

func readConfigFromFile(id string) (ClientConfig, error) {
	file := ClientPath(id)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return ClientConfig{}, err
	}
	var s ClientConfig
	err = json.Unmarshal(data, &s)
	if err != nil {
		return ClientConfig{}, err
	}
	return s, nil
}
func ClientPath(id string) string {
	cfgFile := configFile
	if id != "" {
		cfgFile += "_" + id
	}
	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, projectFolder, cfgFile)
}
