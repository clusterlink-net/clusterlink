package config

import (
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"path"

	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/client"
	"github.ibm.com/mbg-agent/pkg/util"
)

const (
	projectFolder = "/.gw/"
	configFile    = "gwctl"
)

// ClientConfig contain all Client configuration to send requests to the GW
type ClientConfig struct {
	GwIP             string        `json:"gwIP"`
	GwPort           uint16        `json:"gwPort"`
	ID               string        `json:"ID"`
	CaFile           string        `json:"CaFile"`
	CertFile         string        `json:"CertFile"`
	KeyFile          string        `json:"KeyFile"`
	Dataplane        string        `json:"Dataplane"`
	PolicyEngineIP   string        `json:"PolicyEngineIP"`
	MetricsManagerIP string        `json:"MetricsManagerIP"`
	logger           *logrus.Entry `json:"-"`
}

// NewClientConfig create config file with all the configuration of the Client
func NewClientConfig(cfg ClientConfig) (*ClientConfig, error) {
	c := ClientConfig{
		ID:               cfg.ID,
		GwIP:             cfg.GwIP,
		GwPort:           cfg.GwPort,
		Dataplane:        cfg.Dataplane,
		CertFile:         cfg.CertFile,
		KeyFile:          cfg.KeyFile,
		CaFile:           cfg.CaFile,
		PolicyEngineIP:   cfg.PolicyEngineIP,
		MetricsManagerIP: cfg.MetricsManagerIP,
		logger:           logrus.WithField("component", "gwctl/config"),
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

// GetGwPort return the gw port that the Client is connected
func (c *ClientConfig) GetGwPort() uint16 {
	return c.GwPort
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

// GetMetricsManagerIP return the metrics manager address
func (c *ClientConfig) GetMetricsManagerIP() string {
	return c.MetricsManagerIP
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
	err = os.WriteFile(f, jsonC, 0600) // RW by owner only
	c.logger.Println("Create Client config File:", f)
	if err != nil {
		c.logger.Errorln("Creating client config File", err)
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

	err = os.WriteFile(f, jsonC, 0600) // RW by owner only
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
	data, err := os.ReadFile(file)
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

// GetClientFromID loads Client from file according to the id.
func GetClientFromID(id string) (*client.Client, error) {
	c, err := GetConfigFromID(id)
	if err != nil {
		return nil, err
	}

	parsedCertData, err := util.ParseTLSFiles(c.CaFile, c.CertFile, c.KeyFile)
	if err != nil {
		return nil, err
	}

	return client.New(c.GwIP, c.GwPort, parsedCertData.ClientConfig(id)), nil
}

// ClientPath get CLI config file from id
func ClientPath(id string) string {
	cfgFile := configFile
	if id != "" {
		cfgFile += "_" + id
	}
	//set cfg file in home directory
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, projectFolder, cfgFile)
}
