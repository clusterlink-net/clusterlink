package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/controlplane/api"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
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
	Short: "Remove MBG peer",
	Long:  `Remove MBG peer`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		id, _ := cmd.Flags().GetString("id")
		m := api.Gwctl{Id: mId}
		err := m.RemovePeer(id)
		if err != nil {
			fmt.Printf("Failed to remove peer :%v", err)
			return
		}
		fmt.Printf("Peer removed successfully")
	},
}

var ServiceRemCmd = &cobra.Command{
	Use:   "service",
	Short: "delete local service from the MBG or from a MBG peer",
	Long:  `delete local service from the MBG  and save it also in the state of the gwctl`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		serviceId, _ := cmd.Flags().GetString("id")
		serviceType, _ := cmd.Flags().GetString("type")
		serviceMbg, _ := cmd.Flags().GetString("mbg")
		peer, _ := cmd.Flags().GetString("peer")
		allFlag, _ := cmd.Flags().GetBool("all")

		m := api.Gwctl{Id: mId}
		if allFlag {
			fmt.Println("Start to remove all services")
			serviceId = "*"
		} else {
			fmt.Println("Start to remove service ", serviceId)
		}

		if serviceType == "local" {
			if peer == "" {
				m.RemoveLocalService(serviceId)
			} else {
				m.RemoveLocalServiceFromPeer(serviceId, peer)
			}
		} else {
			m.RemoveRemoteService(serviceId, serviceMbg)
		}

	},
}

var bindingRemCmd = &cobra.Command{
	Use:   "binding",
	Short: "Removes binding of a remote service to a k8s service port",
	Long:  `Removes a K8s service with the port binding for a remote service`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		serviceId, _ := cmd.Flags().GetString("service")

		m := api.Gwctl{Id: mId}
		err := m.DeleteServiceEndpoint(serviceId)
		if err != nil {
			fmt.Printf("Failed to delete binding :%v\n", err)
			return
		}
		fmt.Printf("Binding service delete successfully\n")
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
		m := api.Gwctl{Id: mId}
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
	peerRemCmd.Flags().String("myid", "", "Gwctl Id")
	peerRemCmd.Flags().String("id", "", "MBG peer id")
	// remove service
	removeCmd.AddCommand(ServiceRemCmd)
	ServiceRemCmd.Flags().String("myid", "", "Gwctl Id")
	ServiceRemCmd.Flags().String("id", "", "Service id to remove. Use '*' to remove all services.")
	ServiceRemCmd.Flags().String("type", "local", "Service type : remote/local")
	ServiceRemCmd.Flags().String("peer", "", "Optional, allow to remove local service from a remote peer."+
		"If this option is specified it will not remove the local service from the local MBg")
	ServiceRemCmd.PersistentFlags().Bool("all", false, "Remove all services")

	// remove service binding
	removeCmd.AddCommand(bindingRemCmd)
	bindingRemCmd.Flags().String("myid", "", "Gwctl Id")
	bindingRemCmd.Flags().String("service", "", "Service id")
	// remove policy
	removeCmd.AddCommand(PolicyRemCmd)
	PolicyRemCmd.Flags().String("myid", "", "Gwctl Id")
	PolicyRemCmd.Flags().String("type", "", "Policy agent command (For now, acl,lb)")
	PolicyRemCmd.Flags().String("serviceSrc", "*", "Name of Source Service (* for wildcard)")
	PolicyRemCmd.Flags().String("serviceDst", "*", "Name of Dest Service (* for wildcard)")
	PolicyRemCmd.Flags().String("mbgDest", "*", "Name of MBG the dest service belongs to (* for wildcard)")
	PolicyRemCmd.Flags().String("policy", "random", "lb policy: random, ecmp, static")
}
