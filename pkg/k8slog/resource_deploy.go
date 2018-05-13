package k8slog

import (
	"github.com/nouney/k8slog/pkg/k8s"
)

// Deployment is a deployment resource
type Deployment struct {
	resource
}

// GetLogs retrieve logs for the deployment resource
//
// This will get logs from all the pods matching the deployment selector
func (d Deployment) GetLogs(opts *k8s.PodLogOptions) (<-chan LogLine, error) {
	deploy, err := k8s.GetDeployment(d.k8s, d.Namespace, d.Name)
	if err != nil {
		return nil, err
	}

	var out <-chan LogLine
	if opts.Follow {
		// If we follow the log stream, we must watch the deployment's pods
		// so we can handle new ones as they're created
		c := make(chan LogLine)
		out = c
		d.watchPodsAndGetLogs(c, deploy.Spec.Selector, opts)
	} else {
		out, err = d.listPodsAndGetLogs(deploy.Spec.Selector, opts)
	}
	return out, err
}
