package common

import (
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/kfcoding-terminal-controller/config"
	"log"
)

type K8sClient struct {
	Config       *rest.Config
	K8sClient    *kubernetes.Clientset
	PodInterface v1.PodInterface
}

func InitK8sClient() *K8sClient {

	//cfg, err := config2.LoadKubeConfig()
	//if err != nil {
	//	panic(err.Error())
	//}
	//clientset := client.NewAPIClient(c)
	//
	//rest.
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Could not init in cluster config: ", err.Error())
	}
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal("Could not get clientset: ", err.Error())
	}
	return &K8sClient{
		K8sClient:    k8sClient,
		Config:       cfg,
		PodInterface: k8sClient.CoreV1().Pods(config.Namespace),
	}
}
