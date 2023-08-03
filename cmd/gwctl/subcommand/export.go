package subcommand

import (
	"fmt"
	"net"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	cmdutil "github.ibm.com/mbg-agent/cmd/util"
	"github.ibm.com/mbg-agent/pkg/admin"
	"github.ibm.com/mbg-agent/pkg/api"
)

// exportCreateOptions is the command line options for 'create export'
type exportCreateOptions struct {
	myID     string
	name     string
	host     string
	port     uint16
	external string
}

// ExportCreateCmd - Create an exported service.
func ExportCreateCmd() *cobra.Command {
	o := exportCreateOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Create an exported service",
		Long:  `Create an exported service that can be accessed by other peers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "port"})
	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportCreateOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name")
	fs.StringVar(&o.host, "host", "", "Exported service endpoint hostname (IP/DNS), if unspecified, uses the service name")
	fs.Uint16Var(&o.port, "port", 0, "Exported service port")
	fs.StringVar(&o.external, "external", "", "External endpoint <host>:<port, which the exported service will be connected")
}

// run performs the execution of the 'create export' subcommand
func (o *exportCreateOptions) run() error {
	var exEndpoint api.Endpoint
	g, err := admin.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	if o.external != "" {
		exHost, exPort, err := net.SplitHostPort(o.external)
		if err != nil {
			return err
		}

		if exHost == "" {
			return fmt.Errorf("missing host in address")
		}

		exPortInt, err := strconv.Atoi(exPort)
		if err != nil {
			return err
		}

		exEndpoint = api.Endpoint{
			Host: exHost,
			Port: uint16(exPortInt),
		}
	}

	err = g.CreateExportService(api.Export{
		Name: o.name,
		Spec: api.ExportSpec{
			Service: api.Endpoint{
				Host: o.host,
				Port: o.port},
			ExternalService: exEndpoint,
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("Exported service created successfully\n")
	return nil
}

// exportDeleteOptions is the command line options for 'delete export'
type exportDeleteOptions struct {
	myID string
	name string
}

// ExportDeleteCmd - delete an exported service command.
func ExportDeleteCmd() *cobra.Command {
	o := exportDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Delete an exported service",
		Long:  `Delete an exported service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportDeleteOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name")
}

// run performs the execution of the 'delete export' subcommand
func (o *exportDeleteOptions) run() error {
	g, err := admin.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.DeleteExportService(api.Export{Name: o.name})
	if err != nil {
		return err
	}

	fmt.Println("Exported service was deleted successfully")
	return nil
}

// exportGetOptions is the command line options for 'get export'
type exportGetOptions struct {
	myID string
	name string
}

// ExportGetCmd - get an exported service command.
func ExportGetCmd() *cobra.Command {
	o := exportGetOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Get an exported service list",
		Long:  `Get an exported service list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name. If empty gets all exported services.")
}

// run performs the execution of the 'get export' subcommand
func (o *exportGetOptions) run() error {
	g, err := admin.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	if o.name == "" {
		sArr, err := g.GetExportServices()
		if err != nil {
			return err
		}
		fmt.Printf("Exported services:\n")
		for i, s := range sArr {
			fmt.Printf("%d. Service Name: %s. Endpoint: %v\n", i+1, s.Name, s.Spec.Service)
			i++
		}
	} else {
		s, err := g.GetExportService(api.Export{Name: o.name})
		if err != nil {
			return err
		}
		fmt.Printf("Exported service :%+v\n", s)
	}

	return nil
}
