package main

import (
	"context"
	"fmt"

	//netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	clientset "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetNetattachments(cfg *rest.Config) {
	exampleClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}

	list, err := exampleClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}
	for _, nad := range list.Items {
		fmt.Printf("network attachment definition %s with config %q\n", nad.Name, nad.Spec.Config)
		fmt.Printf("Using vlan %q\n", nad.Labels["vlan"])
	}
}

func main() {
	configLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	cfg, err := configLoader.ClientConfig()
	if err != nil {
		panic(err)
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err.Error())
	}

	// watch options
	opts := metav1.ListOptions{
		Watch: true,
	}

	//ask for the crds
	fmt.Printf("Starting my watch for Pods' \n")
	GetNetattachments(cfg)

	nsc := cs.CoreV1().Pods("test")

	// create the watcher
	watcher, err := nsc.Watch(context.TODO(), opts)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Starting my watch for Pods' \n")

	// iterate all the events
	for event := range watcher.ResultChan() {
		// retrieve the Pods
		item := event.Object.(*corev1.Pod)

		switch event.Type {

		// when a namespace is deleted...
		case watch.Deleted:
			// let's say hello!
			fmt.Printf("'%s' %v \n", item.GetName(), event.Type)

		// when a namespace is added...
		case watch.Added:
			fmt.Printf("'%s' %v \n ", item.GetName(), event.Type)

		case watch.Error:
			fmt.Printf("'%s' %v \n ", item.GetName(), event.Type)
		}

	}
}
