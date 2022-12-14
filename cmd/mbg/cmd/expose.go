/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
)

// exposeCmd represents the expose command
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Expose command send an expose message to Multi-cloud Border Gateway",
	Long:  `Expose command send an expose message to Multi-cloud Border Gateway`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("serviceId")
		state.UpdateState()
		mbgControlplane.ExposeToMbg(serviceId)

	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)
	exposeCmd.Flags().String("serviceId", "", "Service Id for exposing")

}
