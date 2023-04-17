package kubernetes

func InitializeKubeDeployment(KubeConfigPath string) error {
	err := Data.InitFromConfig(KubeConfigPath)
	if err != nil {
		return err
	}
	return nil
}

// Add support to create a service/Nodeport
