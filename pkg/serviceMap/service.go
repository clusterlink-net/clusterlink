/**********************************************************/
/* Package Service contain all Service and data structure
/* related to Service that can run in mbg
/**********************************************************/
package service

var Arr = make(map[string]Service)

type Service struct {
	Id string
	Ip string
}

//Init all Functions inside the mbg
func init() {
}

//Return Function fields
func GetService(s string) Service {
	return Arr[s]
}

func UpdateService(id, ip string) {
	Arr[id] = Service{id, ip}
}
