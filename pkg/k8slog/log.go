package k8slog

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/nouney/k8slog/pkg/k8s"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// ResourceType represents a k8s resource type
type ResourceType = string

const (
	// TypePod is the resource type for pods
	TypePod ResourceType = "pod"
	// TypeDeploy is the resource type for deployments
	TypeDeploy ResourceType = "deploy"

	defaultNamespace string = "default"
)

var (
	// ErrInvalidResourceType is returned when the given resource type is invalid
	ErrInvalidResourceType = errors.New("invalid resource type")
)

// Line is a log line of a pod
type Line struct {
	// Namespace is the namespace of the pod
	Namespace string
	// Pod is the name of the pod
	Pod string
	// Line is the log line itself
	Line string
}

// Client allows to retrieve logs of differents resources on k8s
type Client struct {
	k8s        *kubernetes.Clientset
	jsonFields []string
	follow     bool
}

// Opts is an option used to configure Client
type Opts func(c *Client)

// WithOptsFollow configure the follow option
//
// If the follow option is enabled, the client will follow the log stream of the resources.
// If the given resource is not a pod, the client will also watch for new pods of the resource.
func WithOptsFollow(value bool) Opts {
	return func(c *Client) {
		c.follow = value
	}
}

// WithOptsJSONFIelds configure the json option
//
// If enabled, log lines will be handled as JSON objects and only the given fields will be printed.
func WithOptsJSONFields(fields ...string) Opts {
	return func(c *Client) {
		c.jsonFields = fields
	}
}

// New creates a new Client
func New(k8s *kubernetes.Clientset, opts ...Opts) *Client {
	c := &Client{k8s: k8s}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Logs retrieve logs of on or multiple resource.
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
func (c Client) Logs(ress ...string) (<-chan Line, error) {
	out := make(chan Line)
	go func() {
		var wg sync.WaitGroup
		for _, res := range ress {
			err := c.logs(res, out, &wg)
			if err != nil {
				log.Println("Error:", err)
			}
		}
		if !c.follow {
			wg.Wait()
			close(out)
		}
	}()
	return out, nil
}

func (c Client) logs(res string, out chan<- Line, wg *sync.WaitGroup) error {
	ns, typ, name, err := parseResource(res)
	if err != nil {
		return err
	}
	switch typ {
	case TypePod:
		err = c.podLogs(ns, name, out, wg)
	case TypeDeploy:
		err = c.deployLogs(ns, name, out, wg)
	}
	return err
}

func (c Client) podLogs(ns, name string, out chan<- Line, wg *sync.WaitGroup) error {
	wg.Add(1)
	go func() {
		c.getPodLogs(ns, name, out)
		wg.Done()
	}()
	return nil
}

func (c Client) deployLogs(ns, name string, out chan<- Line, wg *sync.WaitGroup) error {
	deploy, err := k8s.GetDeployment(c.k8s, ns, name)
	if err != nil {
		return err
	}

	if c.follow {
		c.watchAndGetLogs(ns, deploy.Spec.Selector, out)
		return nil
	}
	pods, err := k8s.ListPods(c.k8s, ns, deploy.Spec.Selector)
	if err != nil {
		return err
	}

	for _, pod := range pods {
		c.podLogs(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, out, wg)
	}
	return nil
}

func (c Client) watchAndGetLogs(ns string, selector *k8s.LabelSelector, out chan<- Line) {
	k8s.WatchPods(c.k8s, ns, selector,
		func(pod *v1.Pod) {
			// a pod matching the selector was created
			go func() {
				log.Printf("new pod \"%s\"", pod.ObjectMeta.Name)
				// retry mechanism since the pod can take a moment to be up
				operation := func() error {
					return c.getPodLogs(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, out)
				}
				err := backoff.Retry(operation, backoff.NewConstantBackOff(1*time.Second))
				if err != nil {
					log.Printf("error: %s", err.Error())
				}
				log.Printf("pod \"%s\": start streaming", pod.ObjectMeta.Name)
			}()

		}, nil, nil)
}

func (c Client) getPodLogs(ns, name string, out chan<- Line) error {
	rc, err := k8s.GetPodLogs(c.k8s, ns, name, &k8s.PodLogOptions{Timestamps: len(c.jsonFields) == 0, Follow: c.follow})
	if err != nil {
		return errors.Wrap(err, "get logs")
	}
	r := bufio.NewReader(rc)
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			log.Printf("pod \"%s\": end streaming\n", name)
			break
		}
		if err != nil {
			return errors.Wrap(err, "read")
		}
		out <- Line{ns, name, c.refineLine(line)}
	}
	return nil
}

func (c Client) refineLine(line []byte) string {
	nbField := len(c.jsonFields)
	if nbField == 0 {
		return string(line)
	}
	var buffer bytes.Buffer
	field := c.jsonFields[0]
	value := gjson.Get(string(line), field)
	buffer.WriteString(value.String())
	for i := 1; i < nbField; i++ {
		buffer.WriteRune(' ')
		field := c.jsonFields[i]
		value := gjson.Get(string(line), field)
		buffer.WriteString(value.String())
	}
	buffer.WriteRune('\n')
	return buffer.String()
}

func parseResource(res string) (ns, typ, name string, err error) {
	ns = defaultNamespace
	typ = TypePod
	chunks := strings.Split(res, "/")
	nbc := len(chunks)
	if nbc == 1 {
		name = chunks[0]
	} else if nbc == 2 {
		typ = chunks[0]
		name = chunks[1]
		if !validResourceType(typ) {
			err = ErrInvalidResourceType
		}
	} else if nbc == 3 {
		ns = chunks[0]
		typ = chunks[1]
		name = chunks[2]
		if !validResourceType(typ) {
			err = ErrInvalidResourceType
		}
	}
	return
}

func validResourceType(t ResourceType) bool {
	return t == TypeDeploy || t == TypePod
}
