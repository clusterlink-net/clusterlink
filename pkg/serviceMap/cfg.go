/**********************************************************/
/* Package service contain all functions and data structure
/* related to service that can run in mbg
/**********************************************************/
package service

var m = make(map[string]Service)

type Service struct {
	Id   uint32
	Name string
	Ip   string
}

//Init all services inside the mbg
func init() {
	m["Forward"] = Service{1, "Forward", ""} //no need port for forwarding
	m["TCP-split"] = Service{2, "TCP-split", "split-service:5300"}
	m["Encryption"] = Service{3, "Encryption", "5400"}
}

//Return service fields
func GetService(k string) Service {
	return m[k]
}

//Check if service exist
func CheckServiceExist(k string) bool {
	_, flag := m[k]
	return flag
}

//Return the service name according service Id
func ConvertId2Name(id uint32) string {
	for _, element := range m {
		if element.Id == id {
			return element.Name
		}
	}
	println("Error Service Node not support this service")
	return ""
}

//Return the service IP according service Id
func ConvertId2Ip(id uint32) string {
	for _, element := range m {
		if element.Id == id {
			return element.Ip
		}
	}
	println("Error Service Node not support this service")
	return ""
}
