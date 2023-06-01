package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/controlplane/api"
)

// exposeCmd represents the expose command
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Expose command send an expose message to Multi-cloud Border Gateway",
	Long:  `Expose command send an expose message to Multi-cloud Border Gateway`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		serviceId, _ := cmd.Flags().GetString("service")
		peer, _ := cmd.Flags().GetString("peer")

		m := api.Gwctl{Id: mId}
		err := m.ExposeService(serviceId, peer)
		if err != nil {
			fmt.Printf("Failed to expose service :%v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)
	exposeCmd.Flags().String("myid", "", "Gwctl Id")
	exposeCmd.Flags().String("service", "", "Service Id for exposing")
	exposeCmd.Flags().String("peer", "", "Peer to expose ,if empty expose to all peers")

}
