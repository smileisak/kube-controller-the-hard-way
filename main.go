package main

import (
    "kube-controlller-the-hard-way/pkg/controller"
    "kube-controlller-the-hard-way/pkg/utils"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
)

func main() {
    var kubeClient kubernetes.Interface
    if _, err := rest.InClusterConfig(); err != nil {
        kubeClient = utils.GetClientOutOfCluster()
    } else {
        kubeClient = utils.GetClient()
    }

    dummyController := controller.NewPodsController(kubeClient)
    stopChan := make(chan struct{})
    defer close(stopChan)

    dummyController.Run(stopChan)
}
