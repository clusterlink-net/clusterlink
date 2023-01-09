/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A start command set all parameter state of the cluster",
	Long: `A start command set all parameter state of the cluster-
			1) The MBG that the cluster is connected
			2) The IP of the cluster
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")
		mbgIP, _ := cmd.Flags().GetString("mbgIP")
		cportLocal, _ := cmd.Flags().GetString("cportLocal")
		cport, _ := cmd.Flags().GetString("cport")

		state.SetState(ip, id, mbgIP, cportLocal, cport)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "Cluster Id")
	startCmd.Flags().String("ip", "", "Cluster IP")
	startCmd.Flags().String("mbgIP", "", "IP address of the MBG connected to the Cluster")
	startCmd.Flags().String("cportLocal", "50051", "Multi-cloud Border Gateway control local port inside the MBG")
	startCmd.Flags().String("cport", "", "Multi-cloud Border Gateway control external port for the MBG neighbors ")
}
