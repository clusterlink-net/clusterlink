package service

var m = make(map[string]Service)

type Service struct {
	Id   uint32
	Name string
}

func init() {
	m["Forward"] = Service{1, "Forward"}
	m["TCP-split"] = Service{2, "TCP-split"}
	m["Encryption"] = Service{3, "Encryption"}
}

func GetService(k string) Service {
	return m[k]
}
func CheckServiceExist(k string) bool {
	_, flag := m[k]
	return flag
}

func ConvertId2Name(id uint32) string {
	for _, element := range m {
		if element.Id == id {
			return element.Name
		}
	}
	println("Error Service Node not support this service")
	return ""
}
