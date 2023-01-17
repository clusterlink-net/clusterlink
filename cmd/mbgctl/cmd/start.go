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
		caFile, _ := cmd.Flags().GetString("rootCa")
		certificateFile, _ := cmd.Flags().GetString("certificate")
		dataplane, _ := cmd.Flags().GetString("dataplane")
		keyFile, _ := cmd.Flags().GetString("key")

		state.SetState(ip, id, mbgIP, caFile, certificateFile, keyFile, dataplane)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "mbgctl Id")
	startCmd.Flags().String("ip", "", "mbgctl IP")
	startCmd.Flags().String("mbgIP", "", "IP address of the MBG (that the mbgctl is connected)")
	startCmd.Flags().String("rootCa", "", "Path to the Root Certificate Auth File (.pem)")
	startCmd.Flags().String("certificate", "", "Path to the Certificate File (.pem)")
	startCmd.Flags().String("key", "", "Path to the Key File (.pem)")
	startCmd.Flags().String("dataplane", "tcp", "tcp/mtls based data-plane proxies")

}
