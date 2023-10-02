package create

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/templates"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	dpapi "github.com/clusterlink-net/clusterlink/pkg/dataplane/api"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/net/idna"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/util"
)

// PeerOptions contains everything necessary to create and run a 'create peer' subcommand.
type PeerOptions struct {
	// Name of the peer to create.
	Name string
	// Dataplanes is the number of dataplanes to create.
	Dataplanes uint16
}

// AddFlags adds flags to fs and binds them to options.
func (o *PeerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "name", "", "Peer name.")
	fs.Uint16Var(&o.Dataplanes, "dataplanes", 1, "Number of dataplanes.")
}

// RequiredFlags are the names of flags that must be explicitly specified.
func (o *PeerOptions) RequiredFlags() []string {
	return []string{"name"}
}

func (o *PeerOptions) createControlplane() error {
	if err := os.Mkdir(config.ControlplaneDirectory(o.Name), 0755); err != nil {
		return err
	}

	// create certificate
	peerDirectory := config.PeerDirectory(o.Name)
	controlplaneDirectory := config.ControlplaneDirectory(o.Name)
	return util.CreateCertificate(&util.CertificateConfig{
		Name:              "cl-controlplane",
		IsServer:          true,
		IsClient:          true,
		DNSNames:          []string{o.Name, api.GRPCServerName(o.Name)},
		CAPath:            filepath.Join(peerDirectory, config.CertificateFileName),
		CAKeyPath:         filepath.Join(peerDirectory, config.PrivateKeyFileName),
		CertOutPath:       filepath.Join(controlplaneDirectory, config.CertificateFileName),
		PrivateKeyOutPath: filepath.Join(controlplaneDirectory, config.PrivateKeyFileName),
	})
}

func (o *PeerOptions) createDataplane() error {
	if err := os.Mkdir(config.DataplaneDirectory(o.Name), 0755); err != nil {
		return err
	}

	// create certificate
	peerDirectory := config.PeerDirectory(o.Name)
	dataplaneDirectory := config.DataplaneDirectory(o.Name)
	return util.CreateCertificate(&util.CertificateConfig{
		Name:              "dataplane",
		IsServer:          true,
		IsClient:          true,
		DNSNames:          []string{dpapi.DataplaneServerName(o.Name)},
		CAPath:            filepath.Join(peerDirectory, config.CertificateFileName),
		CAKeyPath:         filepath.Join(peerDirectory, config.PrivateKeyFileName),
		CertOutPath:       filepath.Join(dataplaneDirectory, config.CertificateFileName),
		PrivateKeyOutPath: filepath.Join(dataplaneDirectory, config.PrivateKeyFileName),
	})
}

func (o *PeerOptions) createGWCTL() error {
	if err := os.Mkdir(config.GWCTLDirectory(o.Name), 0755); err != nil {
		return err
	}

	// create certificate
	peerDirectory := config.PeerDirectory(o.Name)
	gwctlDirectory := config.GWCTLDirectory(o.Name)
	return util.CreateCertificate(&util.CertificateConfig{
		Name:              "gwctl",
		IsClient:          true,
		CAPath:            filepath.Join(peerDirectory, config.CertificateFileName),
		CAKeyPath:         filepath.Join(peerDirectory, config.PrivateKeyFileName),
		CertOutPath:       filepath.Join(gwctlDirectory, config.CertificateFileName),
		PrivateKeyOutPath: filepath.Join(gwctlDirectory, config.PrivateKeyFileName),
	})
}

// Run the 'create peer' subcommand.
func (o *PeerOptions) Run() error {
	if _, err := idna.Lookup.ToASCII(o.Name); err != nil {
		return fmt.Errorf("peer name is not a valid DNS name: %v", err)
	}

	if err := verifyNotExists(o.Name); err != nil {
		return err
	}

	peerDirectory := config.PeerDirectory(o.Name)
	if err := os.Mkdir(peerDirectory, 0755); err != nil {
		return err
	}

	err := util.CreateCertificate(&util.CertificateConfig{
		Name:              o.Name,
		IsCA:              true,
		DNSNames:          []string{o.Name},
		CAPath:            config.CertificateFileName,
		CAKeyPath:         config.PrivateKeyFileName,
		CertOutPath:       filepath.Join(peerDirectory, config.CertificateFileName),
		PrivateKeyOutPath: filepath.Join(peerDirectory, config.PrivateKeyFileName),
	})
	if err != nil {
		return err
	}

	if err := o.createControlplane(); err != nil {
		return err
	}

	if err := o.createDataplane(); err != nil {
		return err
	}

	if err := o.createGWCTL(); err != nil {
		return err
	}

	// deployment configuration
	args, err := templates.Config{
		Peer:       o.Name,
		Dataplanes: o.Dataplanes,
	}.TemplateArgs()
	if err != nil {
		return err
	}

	// create docker run script
	if err := templates.CreateDockerRunScripts(args, peerDirectory); err != nil {
		return err
	}

	// create k8s deployment yaml
	return templates.CreateK8SConfig(args, peerDirectory)
}

// NewCmdCreatePeer returns a cobra.Command to run the 'create peer' subcommand.
func NewCmdCreatePeer() *cobra.Command {
	opts := &PeerOptions{}

	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Create a peer",
		Long:  `Create a peer`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run()
		},
	}

	opts.AddFlags(cmd.Flags())

	for _, flag := range opts.RequiredFlags() {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			fmt.Printf("Error marking required flag '%s': %v\n", flag, err)
			os.Exit(1)
		}
	}

	return cmd
}

// verifyNotExists verifies a given path does not exist.
func verifyNotExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("path %s exists", path)
	}

	if !os.IsNotExist(err) {
		return err
	}

	return nil
}
