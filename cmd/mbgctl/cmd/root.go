package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mbgctl",
	Short: "A mbgctl that send control message to the MBG",
	Long: `mbgctl is part from Multi-cloud Border Gateway(MBG) project,
	that allow sending control messages (HTTPS) to publish, connect and update policy for services to MBG`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
