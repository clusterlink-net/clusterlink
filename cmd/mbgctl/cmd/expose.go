package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/api"
)

// exposeCmd represents the expose command
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Expose command send an expose message to Multi-cloud Border Gateway",
	Long:  `Expose command send an expose message to Multi-cloud Border Gateway`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		serviceId, _ := cmd.Flags().GetString("service")
		m := api.Mbgctl{mId}
		err := m.ExposeService(serviceId)
		if err != nil {
			fmt.Printf("Failed to expose service :%v", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)
	exposeCmd.Flags().String("myid", "", "MBGCtl Id")
	exposeCmd.Flags().String("service", "", "Service Id for exposing")
}
