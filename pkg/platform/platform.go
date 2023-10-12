package platform

// Platform abstracts all operations which are handled by the specific platform (e.g. Kubernetes).
type Platform interface {
	CreateService(name, targetApp string, port, targetPort uint16)
	UpdateService(name, targetApp string, port, targetPort uint16)
	DeleteService(name string)
	CreateEndpoint(name, targetIP string, targetPort uint16)
	UpdateEndpoint(name, targetIP string, targetPort uint16)
	DeleteEndpoint(name string)
}
