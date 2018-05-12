package k8s

import (
	"io"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type (
	Client        = kubernetes.Clientset
	PodLogOptions = v1.PodLogOptions
	LabelSelector = metav1.LabelSelector
)

// NewClient creates a new kubernetes client
//
// It will use the current context in the kubeconfig.
func NewClient(kubeconfig string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// GetDeployment gets a deployment object
func GetDeployment(k8s *Client, ns, name string) (*appsv1.Deployment, error) {
	deploySvc := k8s.AppsV1().Deployments(ns)
	return deploySvc.Get(name, metav1.GetOptions{})
}

// ListPods lists pods matching labels
func ListPods(k8s *Client, ns string, selector *LabelSelector) ([]v1.Pod, error) {
	podsSvc := k8s.CoreV1().Pods(ns)
	pods, err := podsSvc.List(metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

// GetPodLogs gets logs of a pod
func GetPodLogs(k8s *Client, ns, name string, opts *PodLogOptions) (io.ReadCloser, error) {
	podsSvc := k8s.CoreV1().Pods(ns)
	req := podsSvc.GetLogs(name, opts)
	return req.Stream()
}
