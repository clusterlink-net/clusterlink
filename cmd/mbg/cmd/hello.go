/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
)

// helloCmd represents the hello command
var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Hello command send hello message to all MBGs in thr MBG neighbor list",
	Long:  `Hello command send hello message to all MBGs in thr MBG neighbor list.`,
	Run: func(cmd *cobra.Command, args []string) {
		state.UpdateState()
		MbgArr := state.GetMbgArr()
		MyInfo := state.GetMyInfo()
		for _, m := range MbgArr {
			mbgControlplane.HelloReq(m, MyInfo)
		}
	},
}

func init() {
	rootCmd.AddCommand(helloCmd)
}
