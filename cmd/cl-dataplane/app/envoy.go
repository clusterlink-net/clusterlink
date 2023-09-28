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

		"importHeader":        cpapi.ImportHeader,
		"clientIPHeader":      cpapi.ClientIPHeader,
		"authorizationHeader": cpapi.AuthorizationHeader,
		"targetClusterHeader": cpapi.TargetClusterHeader,
	}

	var envoyConf bytes.Buffer
	t := template.Must(template.New("").Parse(envoyConfigurationTemplate))
	if err := t.Execute(&envoyConf, envoyConfArgs); err != nil {
		return fmt.Errorf("cannot create Envoy configuration off template: %v", err)
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
