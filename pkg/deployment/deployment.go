package deployment

// Deployment abstracts all operations which are handled by the specific deployment (e.g. Kubernetes).
type Deployment interface {
	CreateService(name, targetApp string, port, targetPort uint16)
	UpdateService(name, targetApp string, port, targetPort uint16)
	DeleteService(name string)
	CreateEndpoint(name, targetIP string, targetPort uint16)
	UpdateEndpoint(name, targetIP string, targetPort uint16)
	DeleteEndpoint(name string)
}
