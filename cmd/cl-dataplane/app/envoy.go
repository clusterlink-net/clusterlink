// Copyright (c) The ClusterLink Authors.
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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
)

const (
	envoyPath = "/usr/local/bin/envoy"
)

func (o *Options) runEnvoy(peerName, dataplaneID string) error {
	envoyConfArgs := map[string]interface{}{
		"peerName":    peerName,
		"dataplaneID": dataplaneID,

		"controlplaneHost": o.ControlplaneHost,
		"controlplanePort": cpapi.ListenPort,

		"dataplaneListenPort": api.ListenPort,

		"certificateFile": CertificateFile,
		"keyFile":         KeyFile,
		"caFile":          CAFile,

		"controlplaneInternalHTTPCluster": cpapi.ControlplaneInternalHTTPCluster,
		"controlplaneExternalHTTPCluster": cpapi.ControlplaneExternalHTTPCluster,
		"controlplaneGRPCCluster":         cpapi.ControlplaneGRPCCluster,
		"egressRouterCluster":             cpapi.EgressRouterCluster,

		"egressRouterListener":  cpapi.EgressRouterListener,
		"ingressRouterListener": cpapi.IngressRouterListener,

		"certificateSecret": cpapi.CertificateSecret,
		"validationSecret":  cpapi.ValidationSecret,

		"controlplaneGRPCSNI": cpapi.GRPCServerName(peerName),
		"dataplaneSNI":        api.DataplaneSNI(peerName),

		"dataplaneEgressAuthorizationPrefix":  strings.TrimSuffix(cpapi.DataplaneEgressAuthorizationPath, "/"),
		"dataplaneIngressAuthorizationPrefix": strings.TrimSuffix(cpapi.DataplaneIngressAuthorizationPath, "/"),

		"importNameHeader":      cpapi.ImportNameHeader,
		"importNamespaceHeader": cpapi.ImportNamespaceHeader,
		"clientIPHeader":        cpapi.ClientIPHeader,
		"authorizationHeader":   cpapi.AuthorizationHeader,
		"targetClusterHeader":   cpapi.TargetClusterHeader,
	}

	var envoyConf bytes.Buffer
	t := template.Must(template.New("").Parse(envoyConfigurationTemplate))
	if err := t.Execute(&envoyConf, envoyConfArgs); err != nil {
		return fmt.Errorf("cannot create Envoy configuration off template: %w", err)
	}

	args := []string{
		"--log-level", o.LogLevel,
		"--config-yaml", envoyConf.String(),
	}
	if o.LogFile != "" {
		args = append(args, "--log-path", o.LogFile)
	}

	cmd := exec.Command(envoyPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
