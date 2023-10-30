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

package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
)

const (
	dockerRunTemplate = `#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
FABRIC_DIR=$SCRIPT_DIR/..

docker run -itd \
--name {{.peer}}-controlplane \
-v $FABRIC_DIR/{{.fabricCAPath}}:{{.controlplaneCAMountPath}} \
-v $FABRIC_DIR/{{.controlplaneCertPath}}:{{.controlplaneCertMountPath}} \
-v $FABRIC_DIR/{{.controlplaneKeyPath}}:{{.controlplaneKeyMountPath}} \
-v $FABRIC_DIR/{{.controlplanePersistencyDirectory}}:{{.persistencyDirectoryMountPath}} \
cl-controlplane \
cl-controlplane \
--log-level info \
--log-file {{.persistencyDirectoryMountPath}}/log.log \

docker run -itd \
--name {{.peer}}-dataplane \
-v $FABRIC_DIR/{{.fabricCAPath}}:{{.dataplaneCAMountPath}} \
-v $FABRIC_DIR/{{.dataplaneCertPath}}:{{.dataplaneCertMountPath}} \
-v $FABRIC_DIR/{{.dataplaneKeyPath}}:{{.dataplaneKeyMountPath}} \
-v $FABRIC_DIR/{{.dataplanePersistencyDirectory}}:{{.persistencyDirectoryMountPath}} \
{{ if (eq .dataplaneType .dataplaneTypeEnvoy)  }}cl-dataplane \
cl-dataplane \{{ else }}cl-go-dataplane \
cl-go-dataplane \{{ end }}
--controlplane-host {{.peer}}-controlplane \
--log-level info \
--log-file {{.persistencyDirectoryMountPath}}/log.log

docker run -itd \
--name {{.peer}}-gwctl \
-v $FABRIC_DIR/{{.peerCAPath}}:/root/ca.pem \
-v $FABRIC_DIR/{{.gwctlCertPath}}:/root/cert.pem \
-v $FABRIC_DIR/{{.gwctlKeyPath}}:/root/key.pem \
gwctl \
/bin/sh -c "gwctl init --id {{.peer}} \
                       --gwIP {{.peer}}-dataplane \
                       --gwPort 443 \
                       --certca /root/ca.pem \
                       --cert /root/cert.pem \
                       --key /root/key.pem &&
            gwctl config use-context --myid {{.peer}} &&
            sleep infinity"`
)

// CreateDockerRunScripts creates docker run shell scripts for running the various clusterlink components.
func CreateDockerRunScripts(args map[string]interface{}, outDir string) error {
	var dockerRunScript bytes.Buffer
	t := template.Must(template.New("").Parse(dockerRunTemplate))
	if err := t.Execute(&dockerRunScript, args); err != nil {
		return fmt.Errorf("cannot create docker run script off template: %w", err)
	}

	outPath := filepath.Join(outDir, config.DockerRunFile)
	//#nosec G306 -- script needs to be runnable
	return os.WriteFile(outPath, dockerRunScript.Bytes(), 0700)
}
