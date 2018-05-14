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
	return d.getLogs(opts, deploy.Spec.Selector)
}

func init() {
	registerType(
		TypeDeploy,
		func(r resource) Resource {
			return &Deployment{r}
		},
		"deployment", "deploy",
	)
}
