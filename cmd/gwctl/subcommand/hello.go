package cmd

import (
	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/controlplane/api"
)

// helloCmd represents the hello command
var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Hello command send hello message to all MBGs in the MBG neighbor list",
	Long:  `Hello command send hello message to all MBGs in the MBG neighbor list.`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		peerId, _ := cmd.Flags().GetString("peer")
		m := api.Gwctl{Id: mId}

		if peerId == "" {
			m.SendHello()
		} else {
			m.SendHello(peerId)
		}
	},
}

func init() {
	rootCmd.AddCommand(helloCmd)
	helloCmd.Flags().String("myid", "", "Gwctl Id")
	helloCmd.Flags().String("peerId", "", "Send hello to specific peer")
}
