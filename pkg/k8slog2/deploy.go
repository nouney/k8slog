package k8slog

import (
	"io"

	"github.com/nouney/k8slog/pkg/k8s"
	"github.com/pkg/errors"
)

func (k K8SLog) retrieveDeployLogs(res *resource) (iterator, error) {
	if k.follow {
		return k.watchAndLogsDeploy(res)
	}

	deploys, err := k.k8s.ListDeployments(res.ns)
	if err != nil {
		return nil, errors.Wrap(err, "list deployments")
	}

	var iters []iterator
	for _, dep := range deploys {
		if !res.gname.Match(dep.Name) {
			continue
		}
		k.debugf("deploy \"%s\" in namespace \"%s\" matches", dep.Name, dep.Namespace)
		iter, err := k.retrieveDeployPodsLogs(res, &dep)
		if err != nil {
			return nil, errors.Wrap(err, "retrieve logs from selector")
		}
		iters = append(iters, iter)
	}
	return mergeIterators(iters...), nil
}

func (k K8SLog) retrieveDeployPodsLogs(res *resource, dep *k8s.Deploy) (iterator, error) {
	res.name = dep.Name
	iter, err := k.retrieveLogsFromSelector(*res, dep.Spec.Selector)
	return iter, err
}

func (k K8SLog) watchAndLogsDeploy(res *resource) (iterator, error) {
	out := make(chan *retrieveLogResult)
	k.debug("watching deployments...")
	k.k8s.WatchDeploys(res.ns, func(dep *k8s.Deploy) {
		if !res.gname.Match(dep.Name) {
			return
		}
		k.debugf("new deploy \"%s\" in namespace \"%s\" matches", dep.Name, dep.Namespace)
		iter, err := k.retrieveDeployPodsLogs(res, dep)
		if err != nil {
			out <- &retrieveLogResult{nil, err}
			return
		}
		go func() {
			for {
				line, err := iter()
				if err == io.EOF {
					break
				} else if err != nil {
					out <- &retrieveLogResult{nil, err}
					return
				}
				out <- &retrieveLogResult{line, nil}
			}
		}()
	}, nil, nil)
	return newIterator(out), nil
}
