package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/pkg/api"
	"github.ibm.com/mbg-agent/pkg/client"
	"github.ibm.com/mbg-agent/pkg/util"
)

const (
	gw1crt  = "mbg1.crt"
	gw1key  = "mbg1.key"
	gw1Name = "mbg1"
	gw2crt  = "mbg2.crt"
	gw2key  = "mbg2.key"
	gw2Name = "mbg2"

	caCrt         = "ca.crt"
	cPortUint     = uint16(30443)
	cPort         = "30443"
	cPortLocal    = "443"
	kindDestPort  = "30001"
	curlClient    = "curl-client"
	pingerService = "pinger-server"
	pingerPort    = uint16(3000)
)

var (
	mtlsFolder = ProjDir + "/e2e/utils/mtls/"
	manifests  = ProjDir + "/e2e/utils/manifests/"
	gwctl1     *client.Client
	gwctl2     *client.Client
)

// ProjDir is the current directory of the project
var ProjDir string = getProjFolder()

func getProjFolder() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(path.Dir(path.Dir(filename)))
}

// StartClusterSetup starts a two cluster setup
func StartClusterSetup() error {
	StartClusterLink(gw1Name, cPortLocal, cPort, manifests)
	StartClusterLink(gw2Name, cPortLocal, cPort, manifests)
	return startTestPods()
}

// GetClient returns a gwctl client given a cluster name
func GetClient(name string) (*client.Client, error) {
	gwIP, err := GetKindIP(name)
	if err != nil {
		return nil, err
	}
	parsedCertData, err := util.ParseTLSFiles(mtlsFolder+caCrt, mtlsFolder+gw1crt, mtlsFolder+gw1key)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	gwctl := client.New(gwIP, cPortUint, parsedCertData.ClientConfig(name))
	return gwctl, nil
}

// CleanUp deletes the clusters that were created
func CleanUp() {
	DeleteCluster(gw1Name)
	DeleteCluster(gw2Name)
}

func startTestPods() error {
	err := LaunchApp(gw1Name, curlClient, "curlimages/curl", manifests+curlClient+".yaml")
	if err != nil {
		return err
	}
	err = LaunchApp(gw2Name, pingerService, "subfuzion/pinger", manifests+pingerService+".yaml")
	if err != nil {
		return err
	}
	err = CreateK8sService(pingerService, strconv.Itoa(int(pingerPort)), kindDestPort)
	if err != nil {
		return err
	}
	return nil
}

// GetPolicyFromFile returns a policy json object from the file
func GetPolicyFromFile(filename string) (api.Policy, error) {
	fileBuf, err := os.ReadFile(filename)
	if err != nil {
		return api.Policy{}, fmt.Errorf("error reading policy file: %w", err)
	}
	var policy api.Policy
	err = json.Unmarshal(fileBuf, &policy)
	if err != nil {
		return api.Policy{}, fmt.Errorf("error parsing Json in policy file: %w", err)
	}
	policy.Spec.Blob = fileBuf
	return policy, nil
}

func runCmdB(c string) error {
	log.Println(c)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	if err := cmd.Start(); err != nil {
		log.Error("Error starting command:", err)
		return err
	}
	time.Sleep(time.Second)
	return nil
}

//nolint:gosec // Ignore G204: Subprocess launched with a potential tainted input or cmd arguments
func runCmd(c string) error {
	log.Println(c)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	// Start command execution
	if err := cmd.Start(); err != nil {
		log.Error("Error starting command:", err)
		return err
	}

	// Set up goroutines to read output pipes
	go readOutput(stdout)
	go readOutput(stderr)

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		log.Error("Command returned error:", err)
		return err
	}
	return nil
}

func readOutput(pipe io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			fmt.Print(string(buf[:n]))
		}

		if err != nil {
			if err != io.EOF && err != io.ErrClosedPipe && !strings.Contains(err.Error(), "file already closed") {
				log.Error("Error reading output:", err, err.Error())
			}
			break
		}
	}
}

// GetOutput returns the output of a specified command
func GetOutput(c string) (string, error) {
	log.Println(c)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	return string(stdout), nil
}
