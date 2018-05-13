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

type ResourceType = string

const (
	TypePod    ResourceType = "pod"
	TypeDeploy ResourceType = "deploy"

	defaultNamespace string = "default"
)

var (
	ErrInvalidResourceType = errors.New("invalid resource type")
)

type Line struct {
	Namespace string
	Pod       string
	Line      string
}

type Client struct {
	k8s        *kubernetes.Clientset
	jsonFields []string
	follow     bool
}

type Opts func(c *Client)

func WithOptsFollow(value bool) Opts {
	return func(c *Client) {
		c.follow = value
	}
}

func WithOptsJSONFields(fields ...string) Opts {
	return func(c *Client) {
		c.jsonFields = fields
	}
}

func New(k8s *kubernetes.Clientset, opts ...Opts) *Client {
	c := &Client{k8s: k8s}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

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
