package subcommand

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/clusterlink-org/clusterlink/cmd/gwctl/config"
)

// MetricsGetOptions is the command line options for 'get metrics'
type metricsGetOptions struct {
	myID string
}

// MetricsGetCmd - get a policy command
func MetricsGetCmd() *cobra.Command {
	o := metricsGetOptions{}
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Get metrics from the GW",
		Long:  `Get Connection-level metrics from the GW. This is a test command, Ideally metrics have to be scraped from prometheus endpoint provided`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *metricsGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
}

// run performs the execution of the 'delete policy' subcommand
func (o *metricsGetOptions) run() error {
	m, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	metrics, err := m.GetMetrics()
	if err != nil {
		fmt.Printf("Unable to get metrics %v\n", err)
	} else {
		fmt.Printf("Metrics\n")
		for _, conn := range metrics {
			fmt.Printf("%v\n", conn)
		}
	}
	return nil
}
