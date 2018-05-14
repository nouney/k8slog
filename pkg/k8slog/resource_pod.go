package k8slog

import (
	"log"

	"github.com/nouney/k8slog/pkg/k8s"
)

// Pod is a pod resource
type Pod struct {
	resource
}

// GetLogs retrieve logs for the pod resource
func (p Pod) GetLogs(opts *k8s.PodLogOptions) (<-chan LogLine, error) {
	out := make(chan LogLine)
	go func() {
		err := p.getPodLogs(out, p.Name, opts)
		if err != nil {
			log.Fatal(err)
		}
		close(out)
	}()
	return out, nil
}

func init() {
	registerType(
		TypePod,
		func(r resource) Resource {
			return &Pod{r}
		},
		"pod", "po",
	)
}
