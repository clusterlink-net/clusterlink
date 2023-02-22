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
	lb_add  = "lb_add"
	lb_del  = "lb_del"
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
		pType, _ := cmd.Flags().GetString("command")
		state.UpdateState()
		serviceSrc, _ := cmd.Flags().GetString("serviceSrc")
		serviceDst, _ := cmd.Flags().GetString("serviceDst")
		mbgDest, _ := cmd.Flags().GetString("mbgDest")
		priority, _ := cmd.Flags().GetInt("priority")
		action, _ := cmd.Flags().GetInt("action")
		policy, _ := cmd.Flags().GetString("policy")
		switch pType {
		case acl_add:
			sendAclPolicy(serviceSrc, serviceDst, mbgDest, priority, event.Action(action), add)
		case acl_del:
			sendAclPolicy(serviceSrc, serviceDst, mbgDest, priority, event.Action(action), del)
		case lb_add:
			sendLBPolicy(serviceSrc, serviceDst, policyEngine.PolicyLoadBalancer(policy), mbgDest, add)
		case lb_del:
			sendLBPolicy(serviceSrc, serviceDst, policyEngine.PolicyLoadBalancer(policy), mbgDest, del)
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
	PolicyCmd.Flags().String("policy", "random", "lb policy: random, ecmp, static")
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
	jsonReq, err := json.Marshal(policyEngine.AclRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, MbgDest: mbgDest, Priority: priority, Action: action})
	if err != nil {
		fmt.Errorf("Unable to marshal json %v", err)
		return
	}
	httpAux.HttpPost(url, jsonReq, httpClient)
}

func sendLBPolicy(serviceSrc, serviceDst string, policy policyEngine.PolicyLoadBalancer, mbgDest string, command int) {
	url := state.GetPolicyDispatcher() + "/" + lb
	switch command {
	case add:
		url += "/add"
	case del:
		url += "/delete"
	default:
		fmt.Println("Unknown Command")
		os.Exit(1)
	}
	httpClient := http.Client{}

	jsonReq, err := json.Marshal(policyEngine.LoadBalancerRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, Policy: policy, DefaultMbg: mbgDest})
	if err != nil {
		fmt.Errorf("Unable to marshal json %v", err)
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
		fmt.Printf("Unable to decode response %v\n", err)
	}
	fmt.Printf("MBG ACL rules\n")
	for r, v := range rules {
		fmt.Printf("[Match]: %v [Action]: %v\n", r, v)
	}
}

func showLBPolicies() {
	var policies map[string]map[string]policyEngine.PolicyLoadBalancer
	httpClient := http.Client{}
	url := state.GetPolicyDispatcher() + "/" + lb
	resp := httpAux.HttpGet(url, httpClient)

	if err := json.Unmarshal(resp, &policies); err != nil {
		fmt.Printf("Unable to decode response:", err)
	}
	fmt.Printf("MBG Load-balancing policies\n")
	for d, val := range policies {
		for s, p := range val {
			fmt.Printf("ServiceSrc: %v ServiceDst: %v Policy: %v\n", s, d, p)
		}
	}
}
