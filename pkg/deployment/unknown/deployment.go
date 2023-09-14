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

// NewDeployment returns a new unknown deployment.
func NewDeployment() *Deployment {
	return &Deployment{}
}
