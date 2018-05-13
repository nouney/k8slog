package k8slog

import (
	"bytes"
	"log"
	"sync"

	"github.com/nouney/k8slog/pkg/k8s"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"k8s.io/client-go/kubernetes"
)

var (
	// ErrInvalidResourceType is returned when the given resource type is invalid
	ErrInvalidResourceType = errors.New("invalid resource type")
)

// LogLine is a log line of a pod
type LogLine struct {
	// Namespace is the namespace of the pod
	Namespace string
	// Pod is the name of the pod
	Pod string
	// Line is the log line itself
	Line string
}

// Client allows to retrieve logs of differents resources on k8s
type Client struct {
	k8s           *kubernetes.Clientset
	jsonFields    []string
	jsonFieldsLen int
	follow        bool
	timestamps    bool
}

// Opts is an option used to configure Client
type Opts func(c *Client)

// WithOptsFollow enable to follow log stream (default: false).
//
// If the follow option is enabled, the client will follow the log stream of the resources.
// If the given resource type is not a pod, the client will also watch for new pods of the resource.
func WithOptsFollow(value bool) Opts {
	return func(c *Client) {
		c.follow = value
	}
}

// WithOptsTimestamps enable timestamps at the beginning of the log line (default: true)
func WithOptsTimestamps(value bool) Opts {
	return func(c *Client) {
		c.timestamps = value
	}
}

// WithOptsJSONFields configure the json option (default: none).
//
// If enabled, log lines will be handled as JSON objects and only the given fields will be printed.
func WithOptsJSONFields(fields ...string) Opts {
	return func(c *Client) {
		c.jsonFields = fields
		c.jsonFieldsLen = len(fields)
		if c.jsonFieldsLen > 0 {
			// disable timestamps so we can parse the json object
			WithOptsTimestamps(true)(c)
		}
	}
}

// New creates a new Client
func New(k8s *kubernetes.Clientset, opts ...Opts) *Client {
	c := &Client{k8s: k8s, timestamps: true}
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
func (c Client) Logs(ress ...string) (<-chan LogLine, error) {
	out := make(chan LogLine)
	go func() {
		// no need to wait if we follow
		var wg sync.WaitGroup
		if !c.follow {
			wg.Add(len(ress))
		}

		for _, res := range ress {
			go func(res string) {
				defer wg.Done()
				err := c.logs(out, res)
				if err != nil {
					log.Println("Error:", err)
				}
			}(res)
		}

		// no need to wait if we follow
		if !c.follow {
			wg.Wait()
			close(out)
		}
	}()
	return out, nil
}

// logs retrieve logs of a resource
//
// Sync function
func (c Client) logs(out chan<- LogLine, res string) error {
	r, err := NewResource(c.k8s, res)
	if err != nil {
		return err
	}
	stream, err := r.GetLogs(&k8s.PodLogOptions{Timestamps: c.timestamps, Follow: c.follow})
	for {
		line, ok := <-stream
		if !ok {
			break
		}
		line.Line = c.refineLine(line.Line)
		out <- line
	}
	return nil
	// ns, typ, name, err := parseResource(res)
	// if err != nil {
	// 	return err
	// }
	// switch typ {
	// case TypePod:
	// 	err = c.podLogs(ns, name, out, wg)
	// case TypeDeploy:
	// 	err = c.deployLogs(ns, name, out, wg)
	// }
	// // ici: on cree une Resource et on appelle GetLogs dessus
	// return err
}

// // podLogs retrieve logs of a pod resource
// func (c Client) podLogs(ns, name string, out chan<- Line, wg *sync.WaitGroup) error {
// 	wg.Add(1)
// 	go func() {
// 		c.getPodLogs(ns, name, out)
// 		wg.Done()
// 	}()
// 	return nil
// }

// // deployLogs retrieve logs of a deployment resource
// func (c Client) deployLogs(ns, name string, out chan<- Line, wg *sync.WaitGroup) error {
// 	deploy, err := k8s.GetDeployment(c.k8s, ns, name)
// 	if err != nil {
// 		return err
// 	}

// 	if c.follow {
// 		c.watchAndGetLogs(ns, deploy.Spec.Selector, out)
// 		return nil
// 	}
// 	pods, err := k8s.ListPods(c.k8s, ns, deploy.Spec.Selector)
// 	if err != nil {
// 		return err
// 	}

// 	for _, pod := range pods {
// 		c.podLogs(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, out, wg)
// 	}
// 	return nil
// }

// // watchAndGetLogs watch pods matching the label selector in a specific namespace and retrieve their logs
// func (c Client) watchAndGetLogs(ns string, selector *k8s.LabelSelector, out chan<- Line) {
// 	k8s.WatchPods(c.k8s, ns, selector,
// 		func(pod *v1.Pod) {
// 			// a pod matching the selector was created
// 			// go func() {
// 			log.Printf("new pod \"%s\"", pod.ObjectMeta.Name)
// 			// we need a retry mechanism since the pod can take a moment to be running
// 			// (image pull, init containers, etc.)
// 			operation := func() error {
// 				return c.getPodLogs(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, out)
// 			}
// 			err := backoff.Retry(operation, backoff.NewConstantBackOff(1*time.Second))
// 			if err != nil {
// 				log.Printf("error: %s", err.Error())
// 			}
// 			log.Printf("pod \"%s\": start streaming", pod.ObjectMeta.Name)
// 			// }()

// 		}, nil, nil)
// }

// // getPodLogs retrieve logs of a pod
// func (c Client) getPodLogs(ns, name string, out chan<- Line) error {
// 	rc, err := k8s.GetPodLogs(c.k8s, ns, name, &k8s.PodLogOptions{Timestamps: c.timestamps, Follow: c.follow})
// 	if err != nil {
// 		return errors.Wrap(err, "get logs")
// 	}

// 	r := bufio.NewReader(rc)
// 	for {
// 		line, err := r.ReadBytes('\n')
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			return errors.Wrap(err, "read")
// 		}
// 		out <- Line{ns, name, c.refineLine(line)}
// 	}
// 	return nil
// }

func (c Client) refineLine(line string) string {
	if c.jsonFieldsLen == 0 {
		return line
	}
	var buffer bytes.Buffer
	field := c.jsonFields[0]
	value := gjson.Get(line, field)
	buffer.WriteString(value.String())
	for i := 1; i < c.jsonFieldsLen; i++ {
		buffer.WriteRune(' ')
		field := c.jsonFields[i]
		value := gjson.Get(line, field)
		buffer.WriteString(value.String())
	}
	buffer.WriteRune('\n')
	return buffer.String()
}
