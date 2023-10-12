package unknown

// Deployment represents an unknown deployment.
type Deployment struct {
}

// CreateService creates a service.
func (d *Deployment) CreateService(_, _ string, _, _ uint16) {
}

// UpdateService updates a service.
func (d *Deployment) UpdateService(_, _ string, _, _ uint16) {
}

// DeleteService deletes a service.
func (d *Deployment) DeleteService(_ string) {
}

// CreateEndpoint creates a endpoint.
func (d *Deployment) CreateEndpoint(_, _ string, _ uint16) {
}

// UpdateEndpoint updates a endpoint.
func (d *Deployment) UpdateEndpoint(_, _ string, _ uint16) {
}

// DeleteEndpoint deletes a endpoint.
func (d *Deployment) DeleteEndpoint(_ string) {
}

// NewDeployment returns a new unknown deployment.
func NewDeployment() *Deployment {
	return &Deployment{}
}
