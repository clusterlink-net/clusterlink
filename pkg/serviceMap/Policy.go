/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package service

var PolicyArr = make(map[string]Policy)

type Policy struct {
	Id   uint32
	Name string
	Ip   string
}

//Init all Policys inside the mbg
func init() {
	PolicyArr["Forward"] = Policy{1, "Forward", ""} //no need port for forwarding
	PolicyArr["TCP-split"] = Policy{2, "TCP-split", "split-service:5300"}
	PolicyArr["Encryption"] = Policy{3, "Encryption", "5400"}
}

//Return Policy fields
func GetPolicy(p string) Policy {
	return PolicyArr[p]
}
func GetPolicyIp(p string) string {
	return PolicyArr[p].Ip
}

//Check if Policy exist
func CheckPolicyExist(k string) bool {
	_, flag := PolicyArr[k]
	return flag
}

//Return the Policy name according Policy Id
func ConvertId2Name(id uint32) string {
	for _, element := range PolicyArr {
		if element.Id == id {
			return element.Name
		}
	}
	println("Error MBG not support this Policy")
	return ""
}

//Return the Policy IP according Policy Id
func ConvertId2Ip(id uint32) string {
	for _, element := range PolicyArr {
		if element.Id == id {
			return element.Ip
		}
	}
	println("Error MBG not support this Policy")
	return ""
}
