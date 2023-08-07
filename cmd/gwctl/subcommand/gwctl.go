package subcommand

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.ibm.com/mbg-agent/cmd/gwctl/config"
	cmdutil "github.ibm.com/mbg-agent/cmd/util"
	"github.ibm.com/mbg-agent/pkg/admin"
)

// initOptions is the command line options for 'init'
type initOptions struct {
	id             string
	gwIP           string
	gwPort         uint16
	certCa         string
	cert           string
	key            string
	dataplane      string
	policyEngineIP string
}

// InitCmd represents the init command.
func InitCmd() *cobra.Command {
	o := initOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "A start command set all parameter state of gwctl (gw control)",
		Long:  `A start command set all parameter state of gwctl (gw control)`,
		RunE: func(cmd *cobra.Command, args []string) error {

			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"id", "gwIP", "gwPort", "certca", "cert", "key"})
	return cmd
}

// addFlags registers flags for the CLI.
func (o *initOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.id, "id", "", "gwctl ID")
	fs.StringVar(&o.gwIP, "gwIP", "", "IP address of the gateway (that the gwctl is connected)")
	fs.Uint16Var(&o.gwPort, "gwPort", 0, "Port of the gateway (that the gwctl is connected)")
	fs.StringVar(&o.certCa, "certca", "", "Path to the Root Certificate Auth File (.pem)")
	fs.StringVar(&o.cert, "cert", "", "Path to the Certificate File (.pem)")
	fs.StringVar(&o.key, "key", "", "Path to the Key File (.pem)")
	fs.StringVar(&o.dataplane, "dataplane", "mtls", "tcp/mtls based dataplane proxies")
	fs.StringVar(&o.policyEngineIP, "policyEngineIP", "", "IP address of the policy engine, if empty will use the same value as gwIP")
}

// run performs the execution of the 'init' subcommand
func (o *initOptions) run() error {
	_, err := admin.NewClient(config.ClientConfig{
		ID:             o.id,
		GwIP:           o.gwIP,
		GwPort:         o.gwPort,
		CaFile:         o.certCa,
		CertFile:       o.cert,
		KeyFile:        o.key,
		Dataplane:      o.dataplane,
		PolicyEngineIP: o.policyEngineIP})
	return err
}

// stateGetOptions is the command line options for 'get state'
type stateGetOptions struct {
	myID string
}

// StateGetCmd  - get gwctl parameters.
func StateGetCmd() *cobra.Command {
	o := stateGetOptions{}
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Get gwctl information",
		Long:  `Get gwctl information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *stateGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
}

// run performs the execution of the 'get state' subcommand
func (o *stateGetOptions) run() error {
	d, err := config.GetConfigFromID(o.myID)
	if err != nil {
		return err
	}

	sJSON, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("Error: %v", err.Error())
	}

	fmt.Println(string(sJSON))
	return nil
}
