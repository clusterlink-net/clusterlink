package kubernetes

// InitializeKubeDeployment initiates the informers to keep a watch on pod/services/replicasets.
func InitializeKubeDeployment(k8sConfigPath string) error {
	err := Data.InitFromConfig(k8sConfigPath)
	if err != nil {
		return err
	}
	return nil
}
