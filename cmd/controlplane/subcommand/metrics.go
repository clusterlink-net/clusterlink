package subcommand

import (
	"github.com/spf13/cobra"
)

// observe represents the addPolicy command
var observeCmd = &cobra.Command{
	Use:   "observe",
	Short: "add the metrics exporter target",
	Long:  `add the metrics exporter target`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		target, _ := cmd.Flags().GetString("target")
		start, _ := cmd.Flags().GetBool("start")
		m := Mbg{Id: mId}
		m.AddMetricsManager(target, start)
	},
}

func init() {
	rootCmd.AddCommand(observeCmd)
	observeCmd.Flags().String("target", "", "Target endpoint(e.g.ip:port) to reach the metrics manager")
	observeCmd.Flags().Bool("start", true, "Start the metrics manager (true/false)")
}
