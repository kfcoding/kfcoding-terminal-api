package handler

import (
	"github.com/kfcoding-terminal-controller/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"github.com/satori/go.uuid"
	v12 "k8s.io/api/core/v1"
	"encoding/json"
	"github.com/kfcoding-terminal-controller/config"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateTerminal(body *types.TerminalBody) (string, error) {
	log.Printf("CreateTerminal: %+v\n", body)
	return createTerminalPod(body)
}

func DeleteTerminal(name string) (string, error) {
	log.Print("DeleteTerminal: ", name)
	return deleteTerminalPod(name)
}

func createTerminalPod(body *types.TerminalBody) (string, error) {
	var podBody v12.Pod
	err := json.Unmarshal([]byte(types.TerminalPod), &podBody)
	if nil != err {
		log.Print("createTerminalPod error: ", err)
		return "", err
	}
	var name = "terminal-" + uuid.Must(uuid.NewV4()).String()
	podBody.Name = name
	podBody.Namespace = config.Namespace
	podBody.Labels["app"] = name
	podBody.Spec.Containers[0].Image = body.Image

	podInterface.Create(&podBody)

	if nil != err {
		log.Print("createTerminalPod error: ", err)
		return "", err
	} else {
		log.Printf("createTerminalPod ok")
		return name, nil
	}

	return name, nil
}

func deleteTerminalPod(name string) (string, error) {
	racePeriodSeconds := int64(0)
	var propagationPolicy v13.DeletionPropagation
	propagationPolicy = "Background"

	options := &v13.DeleteOptions{
		TypeMeta: v13.TypeMeta{
			Kind:       "DeleteOptions",
			APIVersion: "v1",
		},
		GracePeriodSeconds: &racePeriodSeconds,
		PropagationPolicy:  &propagationPolicy,
	}
	err := podInterface.Delete(name, options)

	if nil != err {
		log.Print("deleteTerminalPod error: ", err)
		return "", err
	} else {
		log.Printf("deleteTerminalPod ok")
		return "", nil
	}
	return "", nil
}

var podInterface v1.PodInterface
var serviceInterface v1.ServiceInterface

func Init() {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Could not init in cluster config: ", err.Error())
	}
	K8sClient, err := kubernetes.NewForConfig(cfg)
	podInterface = K8sClient.CoreV1().Pods(config.Namespace)
	serviceInterface = K8sClient.CoreV1().Services(config.Namespace)
}
