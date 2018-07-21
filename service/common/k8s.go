package common

import (
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/kfcoding-terminal-controller/config"
	"log"
)

type K8sClient struct {
	Interface    kubernetes.Interface
	Config       *rest.Config
	PodInterface v1.PodInterface
}

func InitK8sClient() *K8sClient {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Could not init in cluster config: ", err.Error())
	}
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal("Could not get clientset: ", err.Error())
	}
	return &K8sClient{
		Interface:    k8sClient,
		Config:       cfg,
		PodInterface: k8sClient.CoreV1().Pods(config.Namespace),
	}
}

func InitOutClusterK8sClient() *K8sClient {
	client, cfg, err := GetClientAndConfig()

	if nil != err {
		log.Fatal(err)
	}
	return &K8sClient{
		Interface:    client,
		Config:       cfg,
		PodInterface: client.CoreV1().Pods(config.Namespace),
	}
}
