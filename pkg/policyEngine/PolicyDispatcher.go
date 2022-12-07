/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyEngine

var PolicyMap = make(map[string]Policy)

//TODO Placeholder for policy type
const (
	connection int = iota
	service
	global
)

type Policy struct {
	Name   string
	Desc   string
	Target string
	PType  int
}

func AddPolicy(name string, desc string, target string, ptype int) {
	PolicyMap[name] = Policy{name, desc, target, ptype}
}

// This is applicable, if we attach a separate Policy Agent (e.g. )
func GetPolicyTarget(policy string) string {

	policyObj, found := PolicyMap[policy]
	if found {
		// Ideally we need to allocate a free port for the policy per service pair
		return policyObj.Target
	} else {
		return ""
	}
}

// Implement Global policies
func ApplyGlobalPolicies(serviceID string) {
	// Global access control policies can be applied here.
}

// Implement Service policies
func ApplyServicePolicies(serviceID string) {
	// Service-level access policies can be applied here.
	// e.g No. of outgoing connections per service
}

// Implement Connection-level policies
func ApplyConnectionPolicies(connID string) {
	// Connection-level access policies can be applied here.
	// e.g Rate-limit for  connections
}
