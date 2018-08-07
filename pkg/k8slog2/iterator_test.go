package k8slog

import (
	"io"
	"testing"
)

func TestNewIterator(t *testing.T) {
	nb := 10
	iter := newIterator(newChannel(nb))

	count := 0
	for {
		_, err := iter()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Error(err)
		}
		count++
	}

	if count != nb {
		t.Errorf("%d result instead of %d", count, nb)
	}
}

func TestMergeIterator(t *testing.T) {
	nb := 10
	iter1 := newIterator(newChannel(nb))
	iter2 := newIterator(newChannel(nb))
	iter := mergeIterators(iter1, iter2)

	count := 0
	for {
		_, err := iter()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Error(err)
		}
		count++
	}
	if count != nb*2 {
		t.Errorf("%d result instead of %d", count, nb*2)
	}
}

func newChannel(nb int) chan *retrieveLogResult {
	out := make(chan *retrieveLogResult)
	go func() {
		for i := 0; i < nb; i++ {
			out <- &retrieveLogResult{nil, nil}
		}
		close(out)
	}()

	return out
}
