package kubernetes

// This function initiates the informers to keep a watch on pod/services/replicasets.
func InitializeKubeDeployment(KubeConfigPath string) error {
	err := Data.InitFromConfig(KubeConfigPath)
	if err != nil {
		return err
	}
	return nil
}
