package k8slog

import (
	"io"

	"github.com/nouney/k8slog/pkg/k8s"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k K8SLog) retrieveServiceLogs(res *resource) (iterator, error) {
	if k.follow {
		return k.watchAndLogsService(res)
	}
	services, err := k.k8s.ListServices(res.ns)
	if err != nil {
		return nil, errors.Wrap(err, "list services")
	}

	var iters []iterator
	for _, s := range services {
		if !res.gname.Match(s.Name) {
			continue
		}
		k.debugf("service \"%s\" in namespace \"%s\" matches", s.Name, s.Namespace)
		iter, err := k.retrieveServicePodsLogs(res, &s)
		if err != nil {
			return nil, errors.Wrap(err, "retrieve logs from selector")
		}
		iters = append(iters, iter)
	}
	return mergeIterators(iters...), nil
}

func (k K8SLog) retrieveServicePodsLogs(res *resource, s *k8s.Service) (iterator, error) {
	res.name = s.Name
	selector := &k8s.LabelSelector{}
	err := v1.Convert_map_to_unversioned_LabelSelector(&s.Spec.Selector, selector, nil)
	if err != nil {
		return nil, errors.Wrap(err, "convert label selector")
	}
	iter, err := k.retrieveLogsFromSelector(*res, selector)
	return iter, err
}

func (k K8SLog) watchAndLogsService(res *resource) (iterator, error) {
	out := make(chan *retrieveLogResult)
	k.debug("watching services...")
	k.k8s.WatchServices(res.ns, func(s *k8s.Service) {
		if !res.gname.Match(s.Name) {
			return
		}
		res.name = s.Name
		k.debugf("new service \"%s\" in namespace \"%s\" matches", s.Name, s.Namespace)
		selector := &k8s.LabelSelector{}
		err := v1.Convert_map_to_unversioned_LabelSelector(&s.Spec.Selector, selector, nil)
		if err != nil {
			out <- &retrieveLogResult{nil, err}
			return
		}
		iter, err := k.retrieveLogsFromSelector(*res, selector)
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
