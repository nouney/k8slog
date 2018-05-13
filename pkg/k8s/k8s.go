package k8s

import (
	"io"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	// auth against GKE clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type (
	// Client is a kubernetes client
	Client = kubernetes.Clientset
	// PodLogOptions is an alias to kubernetes' PodLogOptions
	PodLogOptions = v1.PodLogOptions
	// Pod is an alias to kubernetes' Pod
	Pod = v1.Pod
	// LabelSelector is an alias to kubernetes' LabelSelector
	LabelSelector = metav1.LabelSelector
)

// NewClient creates a new kubernetes client
//
// It uses the current context in the kubeconfig file
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

// ListPods lists pods matching the label selector
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

// WatchPods watches pods matching the label selector
func WatchPods(k8s *Client, ns string, selector *LabelSelector, onAdd func(*v1.Pod), onUpdate func(*v1.Pod, *v1.Pod), onDelete func(*v1.Pod)) func() {
	_, eController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = metav1.FormatLabelSelector(selector)
				return k8s.CoreV1().Pods(ns).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = metav1.FormatLabelSelector(selector)
				return k8s.CoreV1().Pods(ns).Watch(options)
			},
		},
		&v1.Pod{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if onAdd != nil {
					onAdd(obj.(*v1.Pod))
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if onUpdate != nil {
					onUpdate(old.(*v1.Pod), new.(*v1.Pod))
				}
			},
			DeleteFunc: func(obj interface{}) {
				if onDelete != nil {
					onDelete(obj.(*v1.Pod))
				}
			},
		},
	)
	stop := make(chan struct{})
	go eController.Run(stop)
	return func() {
		stop <- struct{}{}
	}
}
