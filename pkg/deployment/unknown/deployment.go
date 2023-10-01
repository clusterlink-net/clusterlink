package unknown

// Deployment represents an unknown deployment.
type Deployment struct {
}

// CreateService creates a service.
func (d *Deployment) CreateService(_ string, _, _ uint16) error {
	return nil
}

// UpdateService updates a service.
func (d *Deployment) UpdateService(_ string, _, _ uint16) error {
	return nil
}

// DeleteService deletes a service.
func (d *Deployment) DeleteService(_ string) error {
	return nil
}

// CreateEndpoint creates a endpoint.
func (d *Deployment) CreateEndpoint(_, _ string, _ uint16) error {
	return nil
}

// UpdateEndpoint updates a endpoint.
func (d *Deployment) UpdateEndpoint(_, _ string, _ uint16) error {
	return nil
}

// DeleteEndpoint deletes a endpoint.
func (d *Deployment) DeleteEndpoint(_ string) error {
	return nil
}

// GetPodLabelsByIP returns all the labels that match the pod IP.
func (d *Deployment) GetPodLabelsByIP(_ string) (map[string]string, error) {
	return nil, nil
}

// NewDeployment returns a new unknown deployment.
func NewDeployment() *Deployment {
	return &Deployment{}
}
