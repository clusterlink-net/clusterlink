/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyEngine

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
)

type ACL map[string]rule

type AclRule struct {
	ServiceSrc string
	ServiceDst string
	MbgDest    string
	Priority   int
	Action     event.Action
}

type rule struct {
	Priority int
	Action   event.Action
	Bitrate  int
}

type AccessControl struct {
	ACLRules    ACL
	DefaultRule event.Action
}

func (A *AccessControl) Init() {
	A.ACLRules = make(ACL)
}

func (A *AccessControl) AddRuleReq(w http.ResponseWriter, r *http.Request) {
	var requestAttr AclRule
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Add Rule request : %+v", requestAttr)

	A.AddRule(requestAttr.ServiceSrc, requestAttr.ServiceDst, requestAttr.MbgDest, requestAttr.Priority, requestAttr.Action)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (A *AccessControl) DelRuleReq(w http.ResponseWriter, r *http.Request) {
	var requestAttr AclRule
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Delete Rule request : %+v", requestAttr)

	A.DeleteRule(requestAttr.ServiceSrc, requestAttr.ServiceDst, requestAttr.MbgDest)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (A *AccessControl) GetRuleReq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(A.ACLRules); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}

	A.displayRules()
}

func (A *AccessControl) AddRule(serviceSrc string, serviceDst string, mbgDest string, priority int, action event.Action) {
	if A.ACLRules == nil {
		A.ACLRules = make(ACL)
	}
	A.ACLRules[getKey(serviceSrc, serviceDst, mbgDest)] = rule{Priority: priority, Action: action}
	plog.Infof("Rule added %+v-> %+v ", getKey(serviceSrc, serviceDst, mbgDest), A.ACLRules[getKey(serviceSrc, serviceDst, mbgDest)])
}

func (A *AccessControl) DeleteRule(serviceSrc string, serviceDst string, mbgDest string) {
	delete(A.ACLRules, getKey(serviceSrc, serviceDst, mbgDest))
}

func (A *AccessControl) RemoveDestService(serviceDst, mbg string) {
	str := "-" + serviceDst + "-"
	if mbg != "" {
		str = str + mbg
	}
	for key := range A.ACLRules {
		if strings.Contains(key, str) {
			delete(A.ACLRules, key)
		}
	}
}
func (A *AccessControl) RulesLookup(serviceSrc string, serviceDst string, mbgDst string) (int, event.Action, int) {
	resultAction := event.Allow
	priority := math.MaxInt
	bitrate := 0
	if myRule, ok := A.ACLRules[getKey(serviceSrc, serviceDst, mbgDst)]; ok {
		if myRule.Priority < priority {
			priority = myRule.Priority
			resultAction = myRule.Action
			bitrate = myRule.Bitrate
		}
		//plog.Infof("Rules Matched.. action=%d", myRule.Action)
	}
	return priority, resultAction, bitrate
}

// TODO : Parallelize lookups
func (A *AccessControl) Lookup(serviceSrc string, serviceDst string, mbgDst string, defaultAction event.Action) (event.Action, int) {
	resultAction := defaultAction
	priority := math.MaxInt
	bitrate := 0
	plog.Infof("ACL Lookup (%s, %s, %s)", serviceSrc, serviceDst, mbgDst)
	// For now, we perform something like an LPM (Longest Prefix Match) with priority
	// Return the first matching rule if priority is 0, Otherwise, check next matches and
	// return the match with the highest priority (0 is high priority, MaxInt is low priority)

	// 111
	prio, action, rate := A.RulesLookup(serviceSrc, serviceDst, mbgDst)
	if prio == 0 {
		return action, rate
	}
	if prio < priority {
		resultAction = action
		bitrate = rate
	}
	// 110
	prio, action, rate = A.RulesLookup(serviceSrc, serviceDst, event.Wildcard)
	if prio == 0 {
		return action, rate
	}
	if prio < priority {
		priority = prio
		resultAction = action
		bitrate = rate
	}

	// 101
	prio, action, rate = A.RulesLookup(serviceSrc, event.Wildcard, mbgDst)
	if prio == 0 {
		return action, rate
	}
	if prio < priority {
		priority = prio
		resultAction = action
		bitrate = rate
	}

	// 011
	prio, action, rate = A.RulesLookup(event.Wildcard, serviceDst, mbgDst)
	if prio == 0 {
		return action, rate
	}
	if prio < priority {
		priority = prio
		resultAction = action
		bitrate = rate
	}

	// 100
	prio, action, rate = A.RulesLookup(serviceSrc, event.Wildcard, event.Wildcard)
	if prio == 0 {
		return action, rate
	}
	if prio < priority {
		priority = prio
		resultAction = action
		bitrate = rate
	}
	// 010
	prio, action, rate = A.RulesLookup(event.Wildcard, serviceDst, event.Wildcard)
	if prio == 0 {
		return action, rate
	}
	if prio < priority {
		priority = prio
		resultAction = action
		bitrate = rate
	}

	// 001
	prio, action, rate = A.RulesLookup(event.Wildcard, event.Wildcard, mbgDst)
	if prio < priority {
		resultAction = action
		bitrate = rate
	}

	return resultAction, bitrate
}

func (A *AccessControl) LookupTarget(service string, peerMbgs *[]string) (event.Action, []string) {
	myAction := event.AllowAll
	mbgList := []string{}
	for _, mbg := range *peerMbgs {
		plog.Infof("Checking %s to expose", mbg)
		action, _ := A.Lookup(event.Wildcard, service, mbg, event.Allow)
		if action == event.Allow {
			mbgList = append(mbgList, mbg)
		} else {
			myAction = event.AllowPartial
		}
	}
	if len(mbgList) == 0 {
		myAction = event.Deny
	}
	return myAction, mbgList
}

func (A *AccessControl) displayRules() {
	for key, rule := range A.ACLRules {
		plog.Infof("%s -> %+v", key, rule)
	}

}

func getKey(serviceSrc string, serviceDst string, mbgDst string) string {
	return serviceSrc + "-" + serviceDst + "-" + mbgDst
}

func (r rule) String() string {
	return fmt.Sprintf("Action: %s Priority: %d Bitrate: %d", r.Action, r.Priority, r.Bitrate)
}
