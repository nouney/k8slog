package k8slog

import (
	"github.com/nouney/k8slog/pkg/k8s"
)

// ReplicaSet is a ReplicaSet resource
type ReplicaSet struct {
	resource
}

// GetLogs retrieve logs for the ReplicaSet resource
//
// This will get logs from all the pods matching the ReplicaSet selector
func (rs ReplicaSet) GetLogs(opts *k8s.PodLogOptions) (<-chan LogLine, error) {
	repset, err := k8s.GetReplicaSet(rs.k8s, rs.Namespace, rs.Name)
	if err != nil {
		return nil, err
	}
	return rs.getLogs(opts, repset.Spec.Selector)
}

func init() {
	registerType(
		TypeReplicaSet,
		func(r resource) Resource {
			return &ReplicaSet{r}
		},
		"replicaset", "rs",
	)
}
