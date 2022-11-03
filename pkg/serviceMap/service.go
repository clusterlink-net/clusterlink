/**********************************************************/
/* Package Service contain all Service and data structure
/* related to Service that can run in mbg
/**********************************************************/
package service

var Arr = make(map[string]Service)

type Service struct {
	Name   string
	Id     string
	Ip     string
	Domain string
	Policy string `default:"Forward"`
}

//Init all Functions inside the mbg
func init() {
}

//Return Function fields
func GetService(s string) Service {
	return Arr[s]
}

func UpdateService(name, id, ip, domain, policy string) {
	Arr[name+id] = Service{name, id, ip, domain, policy}
}
