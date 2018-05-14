package k8slog

import (
	"github.com/nouney/k8slog/pkg/k8s"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Service is a Service resource
type Service struct {
	resource
}

// GetLogs retrieve logs for the Service resource
//
// This will get logs from all the pods matching the Service selector
func (s Service) GetLogs(opts *k8s.PodLogOptions) (<-chan LogLine, error) {
	svc, err := k8s.GetService(s.k8s, s.Namespace, s.Name)
	if err != nil {
		return nil, err
	}
	selector := &k8s.LabelSelector{}
	err = v1.Convert_map_to_unversioned_LabelSelector(&svc.Spec.Selector, selector, nil)
	if err != nil {
		return nil, err
	}
	return s.getLogs(opts, selector)
}

func init() {
	registerType(
		TypeService,
		func(r resource) Resource {
			return &Service{r}
		},
		"service", "svc",
	)
}
