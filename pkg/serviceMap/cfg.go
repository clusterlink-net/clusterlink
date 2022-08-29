package service

var m = make(map[string]Service)

type Service struct {
	Id   uint32
	Name string
}

func init() {
	m["5001"] = Service{1, "TCP-split"}
	m["5002"] = Service{2, "Forward"}
	m["5003"] = Service{3, "Encryption"}
}

func GetService(port string) Service {
	return m[port]
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
