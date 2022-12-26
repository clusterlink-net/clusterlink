/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	log "github.com/sirupsen/logrus"

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
		log.Infof("Hello command called")
		MbgArr := state.GetMbgArr()
		MyInfo := state.GetMyInfo()
		for _, m := range MbgArr {
			log.Info(m)
			mbgControlplane.HelloReq(*m, MyInfo)
		}
		log.Infof("Finish sending Hello to all Mbgs")
	},
}

func init() {
	rootCmd.AddCommand(helloCmd)
}
