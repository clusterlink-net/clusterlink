package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/api"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get",
	Long:  `Get`,
	Run:   emptyRun,
}

// updateCmd represents the update command
var peerGetCmd = &cobra.Command{
	Use:   "peer",
	Short: "get peer list from the MBG",
	Long:  `get peer list from the MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		peerId, _ := cmd.Flags().GetString("id")
		m := api.Mbgctl{mId}
		if peerId == "" {
			pArr, err := m.GetPeers()
			if err != nil {
				fmt.Printf("Unable to get peers: %v", err)
				return
			}
			fmt.Printf("Peers :%+v", pArr)
		} else {
			m.GetPeer(peerId)
		}

	},
}

var serviceGetCmd = &cobra.Command{
	Use:   "service",
	Short: "get service list from the MBG",
	Long:  `get service list from the MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		serviceId, _ := cmd.Flags().GetString("id")
		serviceType, _ := cmd.Flags().GetString("type")
		m := api.Mbgctl{mId}
		if serviceId == "" {
			if serviceType == "local" {
				sArr, err := m.GetLocalServices()
				if err != nil {
					fmt.Printf("Unable to get local services: %v", err)
					return
				}
				fmt.Printf("Local Services :%+v", sArr)
			} else {
				sArr, err := m.GetRemoteServices()
				if err != nil {
					fmt.Printf("Unable to get remote services: %v", err)
					return
				}
				fmt.Printf("Remote Services :%+v", sArr)
			}
		} else {
			if serviceType == "local" {
				s, err := m.GetLocalService(serviceId)
				if err != nil {
					fmt.Printf("Unable to get local service: %v", err)
					return
				}
				fmt.Printf("Local Service :%+v", s)
			} else {
				sArr, err := m.GetRemoteService(serviceId)
				if err != nil {
					fmt.Printf("Unable to get remote service: %v", err)
					return
				}
				fmt.Printf("Remote Service :%+v", sArr)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	// Get Peer
	getCmd.AddCommand(peerGetCmd)
	peerGetCmd.Flags().String("myid", "", "MBGCtl Id")
	peerGetCmd.Flags().String("id", "", "Peer id field")
	// Get Service
	getCmd.AddCommand(serviceGetCmd)
	serviceGetCmd.Flags().String("myid", "", "MBGCtl Id")
	serviceGetCmd.Flags().String("id", "", "service id field")
	serviceGetCmd.Flags().String("type", "remote", "service type : remote/local")
}
