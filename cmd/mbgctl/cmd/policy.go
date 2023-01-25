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
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

const (
	acl     = "acl"
	acl_add = "acl_add"
	acl_del = "acl_del"
	lb      = "lb"
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
		log.Println("applyPolicy called")
		pType, _ := cmd.Flags().GetString("command")
		state.UpdateState()
		switch pType {
		case acl_add:
			serviceSrc, _ := cmd.Flags().GetString("serviceSrc")
			serviceDst, _ := cmd.Flags().GetString("serviceDst")
			mbgDest, _ := cmd.Flags().GetString("mbgDest")
			priority, _ := cmd.Flags().GetInt("priority")
			action, _ := cmd.Flags().GetInt("action")
			sendAclPolicy(serviceSrc, serviceDst, mbgDest, priority, action, add)
		case acl_del:
			serviceSrc, _ := cmd.Flags().GetString("serviceSrc")
			serviceDst, _ := cmd.Flags().GetString("serviceDst")
			mbgDest, _ := cmd.Flags().GetString("mbgDest")
			priority, _ := cmd.Flags().GetInt("priority")
			action, _ := cmd.Flags().GetInt("action")
			sendAclPolicy(serviceSrc, serviceDst, mbgDest, priority, action, add)
		case lb:
			service, _ := cmd.Flags().GetString("serviceDst")
			policy, _ := cmd.Flags().GetInt("policy")
			sendLbPolicy(service, policy)
		case show:
			showAclPolicies()
			showLbPolicies()
		default:
			fmt.Println("Unknown policy type")
			os.Exit(1)
		}

	},
}

func init() {
	rootCmd.AddCommand(PolicyCmd)
	PolicyCmd.Flags().String("command", "", "Policy agent command (For now, acl_add/del lb_add/del)")
	PolicyCmd.Flags().String("serviceSrc", "*", "Name of Source Service (* for wildcard)")
	PolicyCmd.Flags().String("serviceDst", "*", "Name of Dest Service (* for wildcard)")
	PolicyCmd.Flags().String("mbgDest", "*", "Name of MBG the dest service belongs to (* for wildcard)")
	PolicyCmd.Flags().Int("priority", 0, "Priority of the acl rule (0 -> highest)")
	PolicyCmd.Flags().Int("action", 0, "acl 0 -> allow, 1 -> deny")
	PolicyCmd.Flags().Int("policy", 0, "lb policy , 0-> random, 1-> ecmp")
}

func sendAclPolicy(serviceSrc string, serviceDst string, mbgDest string, priority int, action int, command int) {
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
	log.Infof("Sending Policy %+v", jsonReq)
	httpAux.HttpPost(url, jsonReq, httpClient)
}

func sendLbPolicy(service string, policy int) {
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
	fmt.Printf("%+v\n", rules)
}

func showLbPolicies() {
	var rules map[string]int
	httpClient := http.Client{}
	url := state.GetPolicyDispatcher() + "/" + lb
	resp := httpAux.HttpGet(url, httpClient)
	err := json.NewDecoder(bytes.NewBuffer(resp)).Decode(&rules)
	if err != nil {
		log.Infof("Unable to decode response %v", err)
	}
	fmt.Printf("%+v", rules)
}
