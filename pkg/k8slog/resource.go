package k8slog

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/nouney/k8slog/pkg/k8s"
	"github.com/pkg/errors"
)

// ResourceType represents a k8s resource type
type ResourceType int

const (
	// TypeUnknown is an unknown resource type
	TypeUnknown ResourceType = iota
	// TypePod is the resource type for pods
	TypePod
	// TypeDeploy is the resource type for deployments
	TypeDeploy

	defaultNamespace string = "default"
)

// Resource is a k8s resource (namespace/type/name)
type Resource interface {
	GetLogs(*k8s.PodLogOptions) (<-chan LogLine, error)
}

// NewResource creates new Resource object
//
// A resource can be a pod, a deployment, a statefulsets, etc.
// It can has the following forms:
//	- X/Y/Z: all the pods of the resource "Z" of type "Y" in namespace "X"
//	- Y/Z: all the pods of the resource "Z" of type "Y" in namespace "default"
//	- Z: the pod "Z" in namespace "default"
// Examples:
//	- mysvc-abcd: the pod "mysvc-abcd" in namespace "default"
//	- deploy/mysvc: all the pods of the deployment "mysvc" in namespace "default"
//	- prod/deploy/mysvc: all the pods of the deployment "mysvc" in namespace "prod"
// Lists of resource type:
//	- pod, po
//	- deployment, deploy
func NewResource(k8s *k8s.Client, res string) (Resource, error) {
	var err error
	r := resource{
		k8s:       k8s,
		Namespace: defaultNamespace,
		Type:      TypePod,
	}
	chunks := strings.Split(res, "/")
	nbc := len(chunks)
	if nbc == 1 {
		// Z: the pod "Z" in namespace "default"
		r.Name = chunks[0]
	} else if nbc == 2 {
		// Y/Z: all the pods of the resource "Z" of type "Y" in namespace "default"
		r.Type, err = validateResourceType(chunks[0])
		r.Name = chunks[1]
	} else if nbc == 3 {
		// X/Y/Z: all the pods of the resource "Z" of type "Y" in namespace "X"
		r.Namespace = chunks[0]
		r.Type, err = validateResourceType(chunks[1])
		r.Name = chunks[2]
	}
	if err != nil {
		return nil, err
	}
	var ret Resource
	switch r.Type {
	case TypePod:
		ret = &Pod{r}
	case TypeDeploy:
		ret = &Deployment{r}
	}
	return ret, nil
}

type resource struct {
	k8s       *k8s.Client
	Type      ResourceType
	Namespace string
	Name      string
}

// watchAndGetLogs watch pods matching the label selector in a specific namespace and retrieve their logs
//
// Async function
func watchPodsAndGetLogs(out chan<- LogLine, k8sclient *k8s.Client, ns string, selector *k8s.LabelSelector, opts *k8s.PodLogOptions) {
	k8s.WatchPods(k8sclient, ns, selector,
		func(pod *k8s.Pod) {
			// a pod matching the selector was created
			log.Printf("new pod \"%s\"", pod.ObjectMeta.Name)

			// we need a retry mechanism since the pod can take a moment to be running
			// (image pull, init containers, etc.)
			err := backoff.Retry(
				func() error {
					return getPodLogs(out, k8sclient, pod.Namespace, pod.Name, opts)
				},
				backoff.NewConstantBackOff(1*time.Second),
			)
			if err != nil {
				log.Printf("error: %s", err.Error())
				return
			}
		}, nil, nil)
}

// listPodsAndGetLogs lists pods maching the label selector in a specific namespace and retrieve their logs
//
// Async function
func listPodsAndGetLogs(k8sclient *k8s.Client, ns string, selector *k8s.LabelSelector, opts *k8s.PodLogOptions) (<-chan LogLine, error) {
	pods, err := k8s.ListPods(k8sclient, ns, selector)
	if err != nil {
		return nil, err
	}
	out := make(chan LogLine)
	go func() {
		var wg sync.WaitGroup
		wg.Add(len(pods))
		for _, pod := range pods {
			go func(pod *k8s.Pod) {
				defer wg.Done()
				err := getPodLogs(out, k8sclient, ns, pod.Name, opts)
				if err != nil {
					log.Printf("error: %s", err.Error())
				}
			}(&pod)
		}
		wg.Wait()
		close(out)
	}()
	return out, nil
}

// getPodLogs retrieve logs of a pod
//
// Sync function
func getPodLogs(out chan<- LogLine, k8sclient *k8s.Client, ns, name string, opts *k8s.PodLogOptions) error {
	rc, err := k8s.GetPodLogs(k8sclient, ns, name, opts)
	if err != nil {
		return errors.Wrap(err, "get logs")
	}
	r := bufio.NewReader(rc)
	if opts.Follow {
		log.Printf("pod \"%s\": start streaming", name)
	}
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			if opts.Follow {
				log.Printf("pod \"%s\": end streaming", name)
			}
			break
		}
		if err != nil {
			return errors.Wrap(err, "read")
		}
		out <- LogLine{ns, name, string(line)}
	}
	return nil
}

func validateResourceType(t string) (ResourceType, error) {
	switch t {
	case "pod", "po":
		return TypePod, nil
	case "deployment", "deploy":
		return TypeDeploy, nil
	default:
		return TypeUnknown, fmt.Errorf("unknown resource type: %s", t)
	}
}
