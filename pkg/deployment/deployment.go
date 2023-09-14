package deployment

// Deployment abstracts all operations which are handled by the specific deployment (e.g. Kubernetes).
type Deployment interface {
	CreateService(name string, port, targetPort uint16) error
	UpdateService(name string, port, targetPort uint16) error
	DeleteService(name string) error
}
