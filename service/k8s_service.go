package service

import (
	"github.com/kfcoding-terminal-controller/types"
	"log"
	v12 "k8s.io/api/core/v1"
	"encoding/json"
	"github.com/kfcoding-terminal-controller/config"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/kfcoding-terminal-controller/service/common"
)

type K8sService struct {
	k8sClient *common.K8sClient
}

func GetK8sService(k8sClient *common.K8sClient) *K8sService {
	return &K8sService{
		k8sClient: k8sClient,
	}
}

func (service *K8sService) CreateTerminal(body *types.TerminalBody, hostname, podName string) (error) {
	var podBody v12.Pod
	err := json.Unmarshal([]byte(types.TerminalPod), &podBody)
	if nil != err {
		log.Print("createTerminalPod error: ", err)
		return err
	}
	podBody.Name = podName
	podBody.Namespace = config.Namespace
	podBody.Labels["app"] = podName
	podBody.Spec.Containers[0].Image = body.Image
	podBody.Spec.Hostname = hostname

	service.k8sClient.PodInterface.Create(&podBody)

	if nil != err {
		log.Print("createTerminalPod error: ", err)
		return err
	} else {
		log.Printf("createTerminalPod ok")
		return nil
	}
}

func (service *K8sService) DeleteTerminal(name string) (error) {
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

	err := service.k8sClient.PodInterface.Delete(name, options)

	if nil != err {
		log.Print("deleteTerminalPod error: ", err)
		return err
	} else {
		log.Printf("deleteTerminalPod ok")
		return nil
	}
}
