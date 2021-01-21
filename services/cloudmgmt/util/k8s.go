package util

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewK8sClientSet create a kubernetes.Clientset
// using in cluster config
func NewK8sClientSet() (*kubernetes.Clientset, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(restConfig)
}

// GetPodNamespace - get namespace of cloudmgmt pod
// note: this function only works when called in a pod
func GetPodNamespace() string {
	// use the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return "default"
}

// GetPodIP get IP for the current pod.
// May wait up to 10 seconds.
// Require apiGroup "" pods list and get RBAC permission
func GetPodIP() (IP string, err error) {
	clientset, err := NewK8sClientSet()
	if err != nil {
		return
	}
	var pods *apiv1.PodList
	ns := GetPodNamespace()
	podName := os.Getenv("HOSTNAME")
	pi := clientset.CoreV1().Pods(ns)
	for iter := 1; iter < 10; iter++ {
		pods, err = pi.List(metav1.ListOptions{})
		if err != nil {
			return
		}
		for _, pod := range pods.Items {
			pod, _ := pi.Get(pod.Name, metav1.GetOptions{})
			if pod.Name == podName {
				IP = pod.Status.PodIP
				if IP != "" {
					return
				}
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	return
}
