package k8slog

import (
	"errors"
	"fmt"

	"github.com/nouney/k8slog/pkg/k8s"
)

const (
	DefaultNamespace = "default"
	DefaultType      = ResourceTypePod
)

var (
	ErrUnknownResourceType = errors.New("unknown resource type")
)

type K8SLog struct {
	k8s           *k8s.Client
	jsonFields    []string
	jsonFieldsLen int
	follow        bool
	timestamps    bool

	debugEnabled bool
}

// New creates a new K8SLog object
func New(k8s *k8s.Client, opts ...Option) *K8SLog {
	c := &K8SLog{k8s: k8s, timestamps: true}
	for _, opt := range opts {
		opt(c)
	}
	c.debugf("internal value: %+v", c)
	return c
}

func (k K8SLog) debug(strs ...interface{}) {
	if k.debugEnabled {
		fmt.Print("debug: ")
		fmt.Println(strs...)
	}
}

func (k K8SLog) debugf(f string, args ...interface{}) {
	if k.debugEnabled {
		fmt.Printf("debug: "+f+"\n", args...)
	}
}
