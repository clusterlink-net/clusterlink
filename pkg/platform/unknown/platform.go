package unknown

// Platform represents an unknown platform.
type Platform struct {
}

// CreateService creates a service.
func (d *Platform) CreateService(_, _ string, _, _ uint16) {
}

// UpdateService updates a service.
func (d *Platform) UpdateService(_, _ string, _, _ uint16) {
}

// DeleteService deletes a service.
func (d *Platform) DeleteService(_ string) {
}

// CreateEndpoint creates a endpoint.
func (d *Platform) CreateEndpoint(_, _ string, _ uint16) {
}

// UpdateEndpoint updates a endpoint.
func (d *Platform) UpdateEndpoint(_, _ string, _ uint16) {
}

// DeleteEndpoint deletes a endpoint.
func (d *Platform) DeleteEndpoint(_ string) {
}

// NewPlatform returns a new unknown platform.
func NewPlatform() *Platform {
	return &Platform{}
}
