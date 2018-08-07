package k8slog

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/tidwall/gjson"

	"github.com/nouney/k8slog/pkg/k8s"
	"github.com/pkg/errors"
)

type LogLine struct {
	Namespace string
	Name      string
	Type      EnumResourceType
	Pod       string
	Line      string
}

func (k K8SLog) Logs(ress ...string) (iterator, error) {
	out := make(chan *retrieveLogResult)
	var wg sync.WaitGroup

	for _, res := range ress {
		wg.Add(1)
		go func(res string) {
			defer wg.Done()
			iter, err := k.log(res)
			if err != nil {
				out <- &retrieveLogResult{nil, err}
				return
			}
			forwardIterator(out, iter)
		}(res)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return newIterator(out), nil
}

func (k K8SLog) log(res string) (iterator, error) {
	resource, err := parseResource(res)
	if err != nil {
		return nil, errors.Wrap(err, "parse resource")
	}

	k.debugf("retrieve logs of resource %+v", resource)
	var iter iterator
	switch resource.typ {
	case ResourceTypePod:
		// list all pods
		// check if the name match
		// get logs for all of them
	case ResourceTypeService:
		iter, err = k.retrieveServiceLogs(resource)
		if err != nil {
			return nil, err
		}
	case ResourceTypeDeploy:
		iter, err = k.retrieveDeployLogs(resource)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrUnknownResourceType
	}
	return iter, nil
}

func (k K8SLog) retrieveLogsFromSelector(res resource, selector *k8s.LabelSelector) (iterator, error) {
	k.debugf("retrieve logs from selector \"%s\"", selector.String())
	if k.follow {
		return k.watchAndLogs(res, selector)
	}
	return k.listAndLogs(res, selector)
}

func (k K8SLog) listAndLogs(res resource, selector *k8s.LabelSelector) (iterator, error) {
	pods, err := k.k8s.ListPods(res.ns, selector)
	if err != nil {
		return nil, err
	}

	var iters []iterator
	for _, pod := range pods {
		iter, err := k.retrieveLogsFromPod(res, pod)
		if err != nil {
			return nil, err
		}
		iters = append(iters, iter)
	}
	return mergeIterators(iters...), nil
}

func (k K8SLog) watchAndLogs(res resource, selector *k8s.LabelSelector) (iterator, error) {
	out := make(chan *retrieveLogResult)
	k.k8s.WatchPods(res.ns, selector,
		func(pod *k8s.Pod) {
			go func(pod *k8s.Pod) {
				fmt.Printf("start streaming logs from pod \"%s\" in namespace \"%s\"\n", pod.Name, pod.Namespace)
				var iter iterator
				err := backoff.Retry(
					func() error {
						var err error
						iter, err = k.retrieveLogsFromPod(res, *pod)
						if err != nil {
							k.debugf("backoff: %s", err)
							return err
						}
						return nil
					},
					backoff.NewConstantBackOff(1*time.Second),
				)
				if err != nil {
					out <- &retrieveLogResult{nil, err}
				}
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
			}(pod)
		},
		nil,
		func(pod *k8s.Pod) {
			fmt.Printf("end streaming logs from pod \"%s\" in namespace \"%s\"\n", pod.Name, pod.Namespace)
		})
	return newIterator(out), nil
}

func (k K8SLog) retrieveLogsFromPod(res resource, pod k8s.Pod) (iterator, error) {
	rc, err := k.k8s.GetPodLogs(pod.Namespace, pod.Name, &k8s.PodLogOptions{Follow: k.follow})
	if err != nil {
		return nil, errors.Wrap(err, "k8s")
	}
	k.debugf("begin logs of pod \"%s\" in namespace \"%s\"", pod.Name, pod.Namespace)

	out := make(chan *retrieveLogResult)
	go func() {
		defer rc.Close()
		defer close(out)
		defer k.debugf("end logs of pod \"%s\" in namespace \"%s\"", pod.Name, pod.Namespace)
		rdr := bufio.NewReader(rc)
		for {
			line, err := rdr.ReadBytes('\n')
			if err == io.EOF {
				break
			} else if err != nil {
				out <- &retrieveLogResult{nil, errors.Wrap(err, "read bytes")}
				return
			}
			k.debugf("new line for pod \"%s\" in namespace \"%s\"", pod.Name, pod.Namespace)
			out <- &retrieveLogResult{&LogLine{
				Namespace: pod.Namespace,
				Name:      res.name,
				Type:      res.typ,
				Pod:       pod.Name,
				Line:      k.refineLine(string(line)),
			}, nil}
		}
	}()
	iter := newIterator(out)
	return iter, nil
}

func (k K8SLog) refineLine(line string) string {
	if k.jsonFieldsLen == 0 {
		return line
	}
	var buffer bytes.Buffer
	field := k.jsonFields[0]
	value := gjson.Get(line, field)
	buffer.WriteString(value.String())
	for i := 1; i < k.jsonFieldsLen; i++ {
		buffer.WriteRune(' ')
		field := k.jsonFields[i]
		value := gjson.Get(line, field)
		buffer.WriteString(value.String())
	}
	buffer.WriteRune('\n')
	return buffer.String()
}
