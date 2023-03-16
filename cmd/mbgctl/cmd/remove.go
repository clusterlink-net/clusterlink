package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/api"
)

// updateCmd represents the update command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove",
	Long:  `Remove`,
	Run:   emptyRun,
}

var peerRemCmd = &cobra.Command{
	Use:   "peer",
	Short: "Add MBG peer to MBG",
	Long:  `Add MBG peer to MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		id, _ := cmd.Flags().GetString("id")
		m := api.Mbgctl{mId}
		err := m.RemovePeer(id)
		if err != nil {
			fmt.Printf("Failed to add peer :%v", err)
			return
		}
		fmt.Printf("Peer added successfully")
	},
}

var ServiceRemCmd = &cobra.Command{
	Use:   "service",
	Short: "delete local service to the MBG",
	Long:  `delete local service to the MBG and save it also in the state of the mbgctl`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		serviceId, _ := cmd.Flags().GetString("id")
		serviceType, _ := cmd.Flags().GetString("type")
		serviceMbg, _ := cmd.Flags().GetString("mbg")

		m := api.Mbgctl{mId}
		if serviceType == "local" {
			m.RemoveLocalService(serviceId)
		} else {
			m.RemoveRemoteService(serviceId, serviceMbg)
		}

	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	// remove peer
	removeCmd.AddCommand(peerRemCmd)
	peerRemCmd.Flags().String("myid", "", "MBGCtl Id")
	peerRemCmd.Flags().String("id", "", "MBG peer id")
	// remove service
	removeCmd.AddCommand(ServiceRemCmd)
	ServiceRemCmd.Flags().String("myid", "", "MBGCtl Id")
	ServiceRemCmd.Flags().String("id", "", "Service id")
	ServiceRemCmd.Flags().String("type", "local", "Service type : remote/local")
	ServiceRemCmd.Flags().String("peer", "", "Optional Service from a remote peer")
}
