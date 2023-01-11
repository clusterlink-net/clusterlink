/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
	handler "github.ibm.com/mbg-agent/pkg/protocol/http/cluster"
)

// helloCmd represents the hello command
var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Hello command send hello message to all MBGs in thr MBG neighbor list",
	Long:  `Hello command send hello message to all MBGs in thr MBG neighbor list.`,
	Run: func(cmd *cobra.Command, args []string) {
		mbgPeerId, _ := cmd.Flags().GetString("mbgId")
		state.UpdateState()

		myMbgIP := state.GetMbgIP()
		if mbgPeerId == "" {
			handler.Hello2AllReq(myMbgIP)
		} else {
			handler.HelloReq(myMbgIP, mbgPeerId)
		}
	},
}

func init() {
	rootCmd.AddCommand(helloCmd)
	helloCmd.Flags().String("mbgId", "", "Send hello to specific MBG peer")

}
