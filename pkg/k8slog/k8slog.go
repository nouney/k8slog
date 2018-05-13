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
	resource

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
	if err != nil {
		return err
	}
	for {
		line, ok := <-stream
		if !ok {
			break
		}
		line.Line = c.refineLine(line.Line)
		out <- line
	}
	return nil
}

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
