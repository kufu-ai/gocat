package deploy

import (
	"os"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// kubeconfigPath returns the path to the KUBECONFIG file,
// which is either specified by the KUBECONFIG environment variable,
// or the default path ~/.kube/config.
func (c *Coordinator) kubeconfigPath() string {
	kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
	if path := os.Getenv("KUBECONFIG"); path != "" {
		kubeconfig = path
	}

	return kubeconfig
}

// kubernetesClientSet creates a Kubernetes API client
// that uses either the KUBECONFIG file if exists, or the in-cluster configuration.
func (c *Coordinator) kubernetesClientSet() (clientset.Interface, error) {
	if c.clientset != nil {
		return c.clientset, nil
	}

	var restConfig *rest.Config

	kubeconfig := c.kubeconfigPath()
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	c.clientset = clientset

	return clientset, nil
}
