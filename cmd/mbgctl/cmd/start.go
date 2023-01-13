/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A start command set all parameter state of mbgctl (mbg control)",
	Long: `A start command set all parameter state of mbgctl (mbg control)-
			1) The MBG that the mbgctl is connected
			2) The IP of the mbgctl
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")
		mbgIP, _ := cmd.Flags().GetString("mbgIP")
		state.SetState(ip, id, mbgIP)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "mbgctl Id")
	startCmd.Flags().String("ip", "", "mbgctl IP")
	startCmd.Flags().String("mbgIP", "", "IP address of the MBG (that the mbgctl is connected)")
}
