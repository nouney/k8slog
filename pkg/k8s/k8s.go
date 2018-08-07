package k8s

import (
	"io"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // gke clusters
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var _ = v1.PodLogOptions{}

type (
	// Client is a kubernetes client
	// PodLogOptions is an alias to kubernetes' PodLogOptions
	PodLogOptions = v1.PodLogOptions
	// Pod is an alias to kubernetes' Pod
	Pod     = v1.Pod
	Service = v1.Service
	Deploy  = appsv1.Deployment
	// LabelSelector is an alias to kubernetes' LabelSelector
	LabelSelector = metav1.LabelSelector
)

type Client struct {
	*kubernetes.Clientset
}

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
	return &Client{client}, nil
}

func (c Client) ListDeployments(ns string) ([]appsv1.Deployment, error) {
	deploySvc := c.AppsV1().Deployments(ns)
	deploys, err := deploySvc.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deploys.Items, nil
}

func (c Client) ListServices(ns string) ([]v1.Service, error) {
	serviceSvc := c.CoreV1().Services(ns)
	svcs, err := serviceSvc.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return svcs.Items, nil
}

// ListPods lists pods matching the label selector
func (c Client) ListPods(ns string, selector *LabelSelector) ([]v1.Pod, error) {
	podsSvc := c.CoreV1().Pods(ns)
	pods, err := podsSvc.List(metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

// GetPodLogs gets logs of a pod
func (c Client) GetPodLogs(ns, name string, opts *PodLogOptions) (io.ReadCloser, error) {
	podsSvc := c.CoreV1().Pods(ns)
	req := podsSvc.GetLogs(name, opts)
	return req.Stream()
}

// GetDeployment gets a Deployment object
func (c Client) GetDeployment(ns, name string) (*appsv1.Deployment, error) {
	deploySvc := c.AppsV1().Deployments(ns)
	return deploySvc.Get(name, metav1.GetOptions{})
}

// GetStatefulSet gets a StatefulSet object
func GetStatefulSet(k8s *Client, ns, name string) (*appsv1.StatefulSet, error) {
	ssSvc := k8s.AppsV1().StatefulSets(ns)
	return ssSvc.Get(name, metav1.GetOptions{})
}

// GetReplicaSet gets a ReplicaSet object
func GetReplicaSet(k8s *Client, ns, name string) (*appsv1.ReplicaSet, error) {
	ssSvc := k8s.AppsV1().ReplicaSets(ns)
	return ssSvc.Get(name, metav1.GetOptions{})
}

// GetService gets a Service object
func GetService(k8s *Client, ns, name string) (*v1.Service, error) {
	ssSvc := k8s.CoreV1().Services(ns)
	return ssSvc.Get(name, metav1.GetOptions{})
}

func ListAllPods(k8s *Client, ns string) ([]v1.Pod, error) {
	podsSvc := k8s.CoreV1().Pods(ns)
	pods, err := podsSvc.List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

// WatchPods watches pods matching the label selector
func (c Client) WatchPods(ns string, selector *LabelSelector, onAdd func(*v1.Pod), onUpdate func(*v1.Pod, *v1.Pod), onDelete func(*v1.Pod)) func() {
	_, eController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = metav1.FormatLabelSelector(selector)
				return c.CoreV1().Pods(ns).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = metav1.FormatLabelSelector(selector)
				return c.CoreV1().Pods(ns).Watch(options)
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

// WatchServices watches services
func (c Client) WatchServices(ns string, onAdd func(*v1.Service), onUpdate func(*v1.Service, *v1.Service), onDelete func(*v1.Service)) func() {
	_, eController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return c.CoreV1().Services(ns).List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return c.CoreV1().Services(ns).Watch(metav1.ListOptions{})
			},
		},
		&v1.Service{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if onAdd != nil {
					onAdd(obj.(*v1.Service))
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if onUpdate != nil {
					onUpdate(old.(*v1.Service), new.(*v1.Service))
				}
			},
			DeleteFunc: func(obj interface{}) {
				if onDelete != nil {
					onDelete(obj.(*v1.Service))
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

// WatchDeploys watches deploys
func (c Client) WatchDeploys(ns string, onAdd func(*Deploy), onUpdate func(*Deploy, *Deploy), onDelete func(*Deploy)) func() {
	_, eController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return c.AppsV1().Deployments(ns).List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return c.AppsV1().Deployments(ns).Watch(metav1.ListOptions{})
			},
		},
		&Deploy{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if onAdd != nil {
					onAdd(obj.(*Deploy))
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if onUpdate != nil {
					onUpdate(old.(*Deploy), new.(*Deploy))
				}
			},
			DeleteFunc: func(obj interface{}) {
				if onDelete != nil {
					onDelete(obj.(*Deploy))
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
