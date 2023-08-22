package controlplane

// Runtime Types
const (
	k8s = "k8s"
)

// MyRunTimeEnv defines the runtime environment where the controlplane is deployed
var MyRunTimeEnv runtimeEnv

type runtimeEnv struct {
	rtenvType string
}

// IsRuntimeEnvK8s returns if the runtime environment of the controlplane is Kubernetes based
func (r *runtimeEnv) IsRuntimeEnvK8s() bool {
	return (r.rtenvType == k8s)
}

// SetRuntimeEnv sets the runtime environment of the controlplane
func (r *runtimeEnv) SetRuntimeEnv(rtenv string) {
	r.rtenvType = rtenv
}
