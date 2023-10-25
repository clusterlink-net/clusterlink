// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/client"
	"github.com/clusterlink-net/clusterlink/pkg/util"
)

const (
	gw1Name = "gw1"
	gw2Name = "gw2"

	ControlPort = uint16(30443)
	cPortLocal  = "443"
	pingerPort  = uint16(3000)
)

var (
	testOutputFolder = ProjDir + "/bin/tests/e2e/"
)

// ProjDir is the current directory of the project
var ProjDir = getProjFolder()

func getProjFolder() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(path.Dir(path.Dir(path.Dir(filename))))
}

// StartClusterSetup starts a two cluster setup
func StartClusterSetup() error {

	clAdm := ProjDir + "/bin/cl-adm "
	// Create test folder
	err := createFolder(testOutputFolder)
	if err != nil {
		return err
	}

	err = runCmdInDir(clAdm+" create fabric", testOutputFolder)
	if err != nil {
		return err
	}

	err = StartClusterLink(gw1Name, cPortLocal, ControlPort)
	if err != nil {
		return err
	}

	return StartClusterLink(gw2Name, cPortLocal, ControlPort)

}

// GetClient returns a gwctl client given a cluster name
func GetClient(name string) (*client.Client, error) {
	var parsedCertData *util.ParsedCertData
	gwIP, err := GetKindIP(name)
	if err != nil {
		return nil, err
	}

	parsedCertData, err = util.ParseTLSFiles(testOutputFolder+"/cert.pem", testOutputFolder+name+"/gwctl/cert.pem", testOutputFolder+name+"/gwctl/key.pem")
	if err != nil {
		log.Error(err)
		return nil, err
	}

	gwctl := client.New(gwIP, ControlPort, parsedCertData.ClientConfig(name))
	return gwctl, nil
}

// CleanUp deletes the clusters that were created
func CleanUp() error {
	err := DeleteCluster(gw1Name)
	if err != nil {
		return err
	}

	return DeleteCluster(gw2Name)
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
	cmd := exec.Command(argSplit[0], argSplit[1:]...) //nolint:gosec
	if err := cmd.Start(); err != nil {
		log.Error("Error starting command:", err)
		return err
	}

	time.Sleep(time.Second)
	return nil
}

func runCmdInDir(c, dir string) error {
	log.Println(c)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...) //nolint:gosec
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		log.Error("Error running command:", err)
		return err
	}
	return nil
}

//nolint:gosec // Ignore G204: Subprocess launched with a potential tainted input or cmd arguments
func runCmd(c string) error {
	log.Println(c)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...) //nolint:gosec
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
	cmd := exec.Command(argSplit[0], argSplit[1:]...) //nolint:gosec
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	return string(stdout), nil
}

// CreateFolder - Remove and create a folder for test
func createFolder(dir string) error {
	err := runCmd(fmt.Sprintf("rm -rf %s ", dir))
	if err != nil {
		return err
	}

	return runCmd(fmt.Sprintf("mkdir -p %s", testOutputFolder))
}
