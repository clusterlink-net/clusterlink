package subcommand

import (
	"fmt"

	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/controlplane/api"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
)

func emptyRun(*cobra.Command, []string) {}

const (
	acl    = "acl"
	lb     = "lb"
	mbgApp = "dataplane"
)

// updateCmd represents the update command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add",
	Long:  `Add`,
	Run:   emptyRun,
}

var peerCmd = &cobra.Command{
	Use:   "peer",
	Short: "Add MBG peer to MBG",
	Long:  `Add MBG peer to MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		target, _ := cmd.Flags().GetString("target")
		id, _ := cmd.Flags().GetString("id")
		cport, _ := cmd.Flags().GetString("port")
		m := api.Gwctl{Id: mId}
		err := m.AddPeer(id, target, cport)
		if err != nil {
			fmt.Printf("Failed to add peer :%v\n", err)
			return
		}
		fmt.Printf("Peer added successfully\n")
	},
}

var policyengineCmd = &cobra.Command{
	Use:   "policyengine",
	Short: "add the location of policy engine",
	Long:  `add the location of policy engine`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		target, _ := cmd.Flags().GetString("target")
		m := api.Gwctl{Id: mId}
		err := m.AddPolicyEngine(target)
		if err != nil {
			fmt.Printf("Failed to add policy engine :%v\n", err)
			return
		}
		fmt.Printf("Policy engine added successfully\n")
	},
}

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Add local service to the MBG",
	Long:  `Add local service to the MBG and save it also in the state of the gwctl`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		serviceId, _ := cmd.Flags().GetString("id")
		serviceIp, _ := cmd.Flags().GetString("target")
		servicePort, _ := cmd.Flags().GetString("port")

		description, _ := cmd.Flags().GetString("description")

		m := api.Gwctl{Id: mId}
		err := m.AddService(serviceId, serviceIp, servicePort, description)
		if err != nil {
			fmt.Printf("Failed to add service :%v\n", err)
			return
		}
		fmt.Printf("Service added successfully\n")
	},
}

var bindingCmd = &cobra.Command{
	Use:   "binding",
	Short: "Bind a remote service to a k8s service port",
	Long:  `Creates a K8s service with the port binding for a remote service`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		serviceId, _ := cmd.Flags().GetString("service")
		servicePort, _ := cmd.Flags().GetInt("port")
		namespace, _ := cmd.Flags().GetString("namespace")
		name, _ := cmd.Flags().GetString("name")

		m := api.Gwctl{Id: mId}
		if name == "" {
			name = serviceId
		}
		err := m.CreateServiceEndpoint(serviceId, servicePort, name, namespace, mbgApp)
		if err != nil {
			fmt.Printf("Failed to create binding :%v\n", err)
			return
		}
		fmt.Printf("Binding service added successfully\n")
	},
}

// PolicyCmd represents the applyPolicy command
var PolicyAddCmd = &cobra.Command{
	Use:   "policy",
	Short: "An applyPolicy command send the MBG the policy for dedicated service",
	Long:  `An applyPolicy command send the MBG the policy for dedicated service.`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		pType, _ := cmd.Flags().GetString("type")
		serviceSrc, _ := cmd.Flags().GetString("serviceSrc")
		serviceDst, _ := cmd.Flags().GetString("serviceDst")
		mbgDest, _ := cmd.Flags().GetString("mbgDest")
		priority, _ := cmd.Flags().GetInt("priority")
		action, _ := cmd.Flags().GetInt("action")
		policy, _ := cmd.Flags().GetString("policy")
		m := api.Gwctl{Id: mId}
		switch pType {
		case acl:
			m.SendACLPolicy(serviceSrc, serviceDst, mbgDest, priority, event.Action(action), api.Add)
		case lb:
			m.SendLBPolicy(serviceSrc, serviceDst, policyEngine.PolicyLoadBalancer(policy), mbgDest, api.Add)

		default:
			fmt.Println("Unknown policy type")
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	// add peer
	addCmd.AddCommand(peerCmd)
	peerCmd.Flags().String("myid", "", "Gwctl Id")
	peerCmd.Flags().String("id", "", "MBG peer id")
	peerCmd.Flags().String("target", "", "MBG peer target(IP/Hostname)")
	peerCmd.Flags().String("port", "443", "MBG peer control port")
	// add policyengine
	addCmd.AddCommand(policyengineCmd)
	policyengineCmd.Flags().String("myid", "", "Gwctl Id")
	policyengineCmd.Flags().String("target", "", "Target endpoint(e.g.ip:port) to reach the policy agent")
	// add service
	addCmd.AddCommand(serviceCmd)
	serviceCmd.Flags().String("myid", "", "Gwctl Id")
	serviceCmd.Flags().String("id", "", "Service id field ")
	serviceCmd.Flags().String("target", "", "Service IP/Hostname if not specified use service id as the target id")
	serviceCmd.Flags().String("port", "", "Service port")
	serviceCmd.Flags().String("description", "", "Service description (Optional)")
	// add service binding
	addCmd.AddCommand(bindingCmd)
	bindingCmd.Flags().String("myid", "", "Gwctl Id")
	bindingCmd.Flags().String("service", "", "Service id")
	bindingCmd.Flags().Int("port", 0, "local port to be bound for remote service")
	bindingCmd.Flags().String("namespace", "default", "Namespace where the service binding to be created")
	bindingCmd.Flags().String("name", "", "Name of k8s service by default is the service id")

	// add policy
	addCmd.AddCommand(PolicyAddCmd)
	PolicyAddCmd.Flags().String("myid", "", "Gwctl Id")
	PolicyAddCmd.Flags().String("type", "", "Policy agent command (For now, acl,lb)")
	PolicyAddCmd.Flags().String("serviceSrc", "*", "Name of Source Service (* for wildcard)")
	PolicyAddCmd.Flags().String("serviceDst", "*", "Name of Dest Service (* for wildcard)")
	PolicyAddCmd.Flags().String("mbgDest", "*", "Name of MBG the dest service belongs to (* for wildcard)")
	PolicyAddCmd.Flags().Int("priority", 0, "Priority of the acl rule (0 -> highest)")
	PolicyAddCmd.Flags().Int("action", 0, "acl 0 -> allow, 1 -> deny")
	PolicyAddCmd.Flags().String("policy", "random", "lb policy: random, ecmp, static")
}
