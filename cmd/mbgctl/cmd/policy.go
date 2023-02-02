/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"
	event "github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

const (
	acl     = "acl"
	acl_add = "acl_add"
	acl_del = "acl_del"
	lb      = "lb"
	lb_set  = "lb_set"
	show    = "show"
)

const (
	add int = iota
	del
	mod
)

// PolicyCmd represents the applyPolicy command
var PolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "An applyPolicy command send the MBG the policy for dedicated service",
	Long:  `An applyPolicy command send the MBG the policy for dedicated service.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Policy command called")
		pType, _ := cmd.Flags().GetString("command")
		state.UpdateState()
		switch pType {
		case acl_add:
			serviceSrc, _ := cmd.Flags().GetString("serviceSrc")
			serviceDst, _ := cmd.Flags().GetString("serviceDst")
			mbgDest, _ := cmd.Flags().GetString("mbgDest")
			priority, _ := cmd.Flags().GetInt("priority")
			action, _ := cmd.Flags().GetInt("action")
			sendAclPolicy(serviceSrc, serviceDst, mbgDest, priority, event.Action(action), add)
		case acl_del:
			serviceSrc, _ := cmd.Flags().GetString("serviceSrc")
			serviceDst, _ := cmd.Flags().GetString("serviceDst")
			mbgDest, _ := cmd.Flags().GetString("mbgDest")
			priority, _ := cmd.Flags().GetInt("priority")
			action, _ := cmd.Flags().GetInt("action")
			sendAclPolicy(serviceSrc, serviceDst, mbgDest, priority, event.Action(action), del)
		case lb_set:
			service, _ := cmd.Flags().GetString("serviceDst")
			policy, _ := cmd.Flags().GetString("policy")
			sendLBPolicy(service, policyEngine.PolicyLoadBalancer(policy))
		case show:
			showAclPolicies()
			showLBPolicies()
		default:
			fmt.Println("Unknown policy type")
			os.Exit(1)
		}

	},
}

func init() {
	rootCmd.AddCommand(PolicyCmd)
	PolicyCmd.Flags().String("command", "", "Policy agent command (For now, acl_add/del lb_add/del/show)")
	PolicyCmd.Flags().String("serviceSrc", "*", "Name of Source Service (* for wildcard)")
	PolicyCmd.Flags().String("serviceDst", "*", "Name of Dest Service (* for wildcard)")
	PolicyCmd.Flags().String("mbgDest", "*", "Name of MBG the dest service belongs to (* for wildcard)")
	PolicyCmd.Flags().Int("priority", 0, "Priority of the acl rule (0 -> highest)")
	PolicyCmd.Flags().Int("action", 0, "acl 0 -> allow, 1 -> deny")
	PolicyCmd.Flags().String("policy", "random", "lb policy: random,ecmp")
}

func sendAclPolicy(serviceSrc string, serviceDst string, mbgDest string, priority int, action event.Action, command int) {
	url := state.GetPolicyDispatcher() + "/" + acl
	httpClient := http.Client{}
	switch command {
	case add:
		url += "/add"
	case del:
		url += "/delete"
	default:
		fmt.Println("Unknown Command")
		os.Exit(1)

	}
	log.Infof("Sending ACL Policy %s", url)
	jsonReq, err := json.Marshal(policyEngine.AclRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, MbgDest: mbgDest, Priority: priority, Action: action})
	if err != nil {
		log.Errorf("Unable to marshal json %v", err)
		return
	}
	httpAux.HttpPost(url, jsonReq, httpClient)
}

func sendLBPolicy(service string, policy policyEngine.PolicyLoadBalancer) {
	url := state.GetPolicyDispatcher() + "/" + lb + "/setPolicy"
	httpClient := http.Client{}

	jsonReq, err := json.Marshal(policyEngine.LoadBalancerRule{Service: service, Policy: policy})
	if err != nil {
		log.Errorf("Unable to marshal json %v", err)
		return
	}
	httpAux.HttpPost(url, jsonReq, httpClient)
}

func showAclPolicies() {
	var rules policyEngine.ACL
	httpClient := http.Client{}
	url := state.GetPolicyDispatcher() + "/" + acl
	resp := httpAux.HttpGet(url, httpClient)
	err := json.NewDecoder(bytes.NewBuffer(resp)).Decode(&rules)
	if err != nil {
		log.Infof("Unable to decode response %v", err)
	}
	log.Info("MBG ACL rules")
	for r, v := range rules {
		log.Infof("Rule %v , %v", r, v)
	}
}

func showLBPolicies() {
	var policies map[string]policyEngine.PolicyLoadBalancer
	httpClient := http.Client{}
	url := state.GetPolicyDispatcher() + "/" + lb
	resp := httpAux.HttpGet(url, httpClient)

	if err := json.Unmarshal(resp, &policies); err != nil {
		log.Fatal("Unable to decode response:", err)
	}
	log.Info("MBG Load-balancing policies")
	for p, r := range policies {
		log.Infof("Service: %v ,Policy: %v", p, r)
	}

}
