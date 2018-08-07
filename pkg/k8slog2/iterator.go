package k8slog

import (
	"io"
	"sync"
)

type (
	iterator func() (*LogLine, error)

	retrieveLogResult struct {
		line *LogLine
		err  error
	}
)

func newIterator(out <-chan *retrieveLogResult) iterator {
	return func() (*LogLine, error) {
		res, ok := <-out
		if !ok {
			return nil, io.EOF
		}
		return res.line, res.err
	}
}

func forwardIterator(out chan<- *retrieveLogResult, iter iterator) {
	for {
		line, err := iter()
		if err == io.EOF {
			break
		} else if err != nil {
			out <- &retrieveLogResult{nil, err}
		}
		out <- &retrieveLogResult{line, nil}
	}
}

func mergeIterators(iters ...iterator) iterator {
	var wg sync.WaitGroup
	out := make(chan *retrieveLogResult)

	for _, iter := range iters {
		wg.Add(1)
		go func(iter iterator) {
			defer wg.Done()
			for {
				line, err := iter()
				if err == io.EOF {
					break
				}
				out <- &retrieveLogResult{line, err}
			}
		}(iter)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return newIterator(out)
}
