package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getConfigMapList(t string) (cml *v1.ConfigMapList) {
	client, err := newKubernetesClient()
	if err != nil {
		log.Print("[ERROR] ", err)
		return
	}

	cml, err = client.CoreV1().ConfigMaps("default").List(context.Background(), meta_v1.ListOptions{LabelSelector: fmt.Sprintf("gocat.zaim.net/configmap-type=%s", t)})
	if err != nil {
		log.Print("[ERROR] ", err)
		return
	}

	return cml
}

func newKubernetesClient() (kubernetes.Interface, error) {
	if os.Getenv("LOCAL") != "" {
		config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
