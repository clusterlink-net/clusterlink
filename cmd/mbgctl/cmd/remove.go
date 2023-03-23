package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/api"
	event "github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
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

var PolicyRemCmd = &cobra.Command{
	Use:   "policy",
	Short: "Remove service policy from MBG.",
	Long:  `Remove service policy from MBG.`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		pType, _ := cmd.Flags().GetString("type")
		serviceSrc, _ := cmd.Flags().GetString("serviceSrc")
		serviceDst, _ := cmd.Flags().GetString("serviceDst")
		mbgDest, _ := cmd.Flags().GetString("mbgDest")
		policy, _ := cmd.Flags().GetString("policy")
		priority := 0 //Doesn't matter when deleting a rule
		action := 0   //Doesn't matter when deleting a rule
		m := api.Mbgctl{mId}
		switch pType {
		case acl:
			m.SendACLPolicy(serviceSrc, serviceDst, mbgDest, priority, event.Action(action), api.Del)
		case lb:
			m.SendLBPolicy(serviceSrc, serviceDst, policyEngine.PolicyLoadBalancer(policy), mbgDest, api.Del)
		default:
			fmt.Println("Unknown policy type")
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
	// add policy
	removeCmd.AddCommand(PolicyRemCmd)
	PolicyRemCmd.Flags().String("myid", "", "MBGCtl Id")
	PolicyRemCmd.Flags().String("type", "", "Policy agent command (For now, acl,lb)")
	PolicyRemCmd.Flags().String("serviceSrc", "*", "Name of Source Service (* for wildcard)")
	PolicyRemCmd.Flags().String("serviceDst", "*", "Name of Dest Service (* for wildcard)")
	PolicyRemCmd.Flags().String("mbgDest", "*", "Name of MBG the dest service belongs to (* for wildcard)")
	PolicyRemCmd.Flags().String("policy", "random", "lb policy: random, ecmp, static")
}
