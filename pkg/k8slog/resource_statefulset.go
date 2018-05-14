package k8slog

import (
	"github.com/nouney/k8slog/pkg/k8s"
)

// StatefulSet is a statefulset resource
type StatefulSet struct {
	resource
}

// GetLogs retrieve logs for the statefulset resource
//
// This will get logs from all the pods matching the statefulset selector
func (ss StatefulSet) GetLogs(opts *k8s.PodLogOptions) (<-chan LogLine, error) {
	sttst, err := k8s.GetStatefulSet(ss.k8s, ss.Namespace, ss.Name)
	if err != nil {
		return nil, err
	}
	return ss.getLogs(opts, sttst.Spec.Selector)
}

func init() {
	registerType(
		TypeStatefulSet,
		func(r resource) Resource {
			return &StatefulSet{r}
		},
		"statefulset", "sts",
	)
}
