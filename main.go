package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/couchbase/goutils/logging"
	clientset "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type NetworkSelectionElement struct {
	// Name contains the name of the Network object this element selects
	Name string `json:"name"`
	// Namespace contains the optional namespace that the network referenced
	// by Name exists in
	Namespace string `json:"namespace,omitempty"`
	// IPRequest contains an optional requested IP address for this network
	// attachment
	IPRequest []string `json:"ips,omitempty"`
	// MacRequest contains an optional requested MAC address for this
	// network attachment
	MacRequest string `json:"mac,omitempty"`
	// InfinibandGUID request contains an optional requested Infiniband GUID address
	// for this network attachment
	InfinibandGUIDRequest string `json:"infiniband-guid,omitempty"`
	// InterfaceRequest contains an optional requested name for the
	// network interface this attachment will create in the container
	InterfaceRequest string `json:"interface,omitempty"`
	// DeprecatedInterfaceRequest is obsolated parameter at pre 3.2.
	// This will be removed in 4.0 release.
	DeprecatedInterfaceRequest string `json:"interfaceRequest,omitempty"`
	// PortMappingsRequest contains an optional requested port mapping
	// for the network
	// BandwidthRequest contains an optional requested bandwidth for
	// the network
	// DeviceID contains an optional requested deviceID the network
	DeviceID string `json:"deviceID,omitempty"`
	// CNIArgs contains additional CNI arguments for the network interface
	CNIArgs *map[string]interface{} `json:"cni-args"`
	// GatewayRequest contains default route IP address for the pod
	GatewayRequest *[]net.IP `json:"default-route,omitempty"`
}

func ParsePodNetworkAnnotation(podNetworks, defaultNamespace string) ([]NetworkSelectionElement, error) {
	var networks []NetworkSelectionElement

	logging.Debugf("parsePodNetworkAnnotation: %s, %s", podNetworks, defaultNamespace)
	if podNetworks == "" {
		fmt.Println("parsePodNetworkAnnotation: pod annotation does not have \"network\" as key")
	}

	if strings.ContainsAny(podNetworks, "[{\"") {
		if err := json.Unmarshal([]byte(podNetworks), &networks); err != nil {
			fmt.Printf("parsePodNetworkAnnotation: failed to parse pod Network Attachment Selection Annotation JSON format: %v", err)
		}
	} else {
		// Comma-delimited list of network attachment object names
		for _, item := range strings.Split(podNetworks, ",") {
			// Remove leading and trailing whitespace.
			item = strings.TrimSpace(item)

			// Parse network name (i.e. <namespace>/<network name>@<ifname>)
			netNsName, networkName, netIfName, err := ParsePodNetworkObjectName(item)
			if err != nil {
				fmt.Printf("parsePodNetworkAnnotation: %v", err)
			}

			networks = append(networks, NetworkSelectionElement{
				Name:             networkName,
				Namespace:        netNsName,
				InterfaceRequest: netIfName,
			})
		}
	}

	for _, n := range networks {
		if n.Namespace == "" {
			n.Namespace = defaultNamespace
		}
		if n.MacRequest != "" {
			// validate MAC address
			if _, err := net.ParseMAC(n.MacRequest); err != nil {
				fmt.Printf("parsePodNetworkAnnotation: failed to mac: %v", err)
			}
		}
		if n.InfinibandGUIDRequest != "" {
			// validate GUID address
			if _, err := net.ParseMAC(n.InfinibandGUIDRequest); err != nil {
				fmt.Printf("parsePodNetworkAnnotation: failed to validate infiniband GUID: %v", err)
			}
		}
		if n.IPRequest != nil {
			for _, ip := range n.IPRequest {
				// validate IP address
				if strings.Contains(ip, "/") {
					if _, _, err := net.ParseCIDR(ip); err != nil {
						fmt.Printf("failed to parse CIDR %q: %v", ip, err)
					}
				} else if net.ParseIP(ip) == nil {
					fmt.Printf("failed to parse IP address %q", ip)
				}
			}
		}
		// compatibility pre v3.2, will be removed in v4.0
		if n.DeprecatedInterfaceRequest != "" && n.InterfaceRequest == "" {
			n.InterfaceRequest = n.DeprecatedInterfaceRequest
		}
	}

	return networks, nil
}

func ParsePodNetworkObjectName(podnetwork string) (string, string, string, error) {
	var netNsName string
	var netIfName string
	var networkName string

	logging.Debugf("parsePodNetworkObjectName: %s", podnetwork)
	slashItems := strings.Split(podnetwork, "/")
	if len(slashItems) == 2 {
		netNsName = strings.TrimSpace(slashItems[0])
		networkName = slashItems[1]
	} else if len(slashItems) == 1 {
		networkName = slashItems[0]
	} else {
		fmt.Println("parsePodNetworkObjectName: Invalid network object (failed at '/')")
	}

	atItems := strings.Split(networkName, "@")
	networkName = strings.TrimSpace(atItems[0])
	if len(atItems) == 2 {
		netIfName = strings.TrimSpace(atItems[1])
	} else if len(atItems) != 1 {
		fmt.Println("parsePodNetworkObjectName: Invalid network object (failed at '@')")
	}

	logging.Debugf("parsePodNetworkObjectName: parsed: %s, %s, %s", netNsName, networkName, netIfName)
	return netNsName, networkName, netIfName, nil
}

func GetNetattachments(cfg *rest.Config, nadname string) []string {
	exampleClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}

	list, err := exampleClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	var nads []string
	for _, nad := range list.Items {
		//fmt.Printf("network attachment definition %s with config %q\n", nad.Name, nad.Spec.Config)
		//fmt.Printf("Using vlan %q\n", nad.Labels["vlan"])
		nads = append(nads, nad.Name)
	}
	return nads
}

func GetNetattachmentVlan(cfg *rest.Config, nadname string) string {
	exampleClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		fmt.Println(err)
	}

	list, err := exampleClient.K8sCniCncfIoV1().NetworkAttachmentDefinitions("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	var Vlan string
	for _, nad := range list.Items {
		if nad.Name == nadname {
			Vlan = nad.Labels["vlan"]
		}
	}
	return Vlan
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

	//logging to stdout to start.
	fmt.Printf("Starting my watch for Pods \n")

	nsc := cs.CoreV1().Pods("test")

	// create the watcher.
	watcher, err := nsc.Watch(context.TODO(), opts)
	if err != nil {
		panic(err)
	}

	// iterate all the events
	for event := range watcher.ResultChan() {
		// retrieve the Pods
		item := event.Object.(*corev1.Pod)

		switch event.Type {

		// when a pod is deleted...
		case watch.Deleted:
			// let's say hello!
			fmt.Printf("'%s' %v \n", item.GetName(), event.Type)

		// when a pod is added...
		case watch.Added:
			fmt.Printf("'%s' %v \n ", item.GetName(), event.Type)
			netmaps := item.Annotations
			val, ok := netmaps["k8s.v1.cni.cncf.io/networks"]
			if ok {
				for _, n := range GetNetattachments(cfg, val) {
					a, err := ParsePodNetworkAnnotation(val, "test")
					if err != nil {
						fmt.Println(err)
					}
					for _, aa := range a {
						if aa.Name == n {
							fmt.Printf("Found network attachment of '%s' \n", n)
							fmt.Printf("Use vlan id of '%s' \n", GetNetattachmentVlan(cfg, n))
							//Would have to Add start all the cvx logic here.
						}
					}

				}
			}
		}
	}
}
