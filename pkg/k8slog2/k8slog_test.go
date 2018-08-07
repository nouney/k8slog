package k8slog

import (
	"testing"
)

func TestParseResource(t *testing.T) {
	tests := []struct {
		test     string
		expected *resource
	}{
		{"name", newResource(DefaultNamespace, DefaultType, "name")},
		{"type/name", newResource(DefaultNamespace, "type", "name")},
		{"ns/type/name", newResource("ns", "type", "name")},
	}

	for _, test := range tests {
		res, _ := parseResource(test.test)
		cmpResource(t, res, test.expected)
	}
}

func TestLog(t *testing.T) {
	c := K8SLog{}
	c.log("totof")
}

func newResource(ns, typ, name string) *resource {
	return &resource{ns, typ, name, nil}
}

func cmpResource(t *testing.T, r1, r2 *resource) {
	if r1.ns != r2.ns {
		t.Errorf("ns is different: \"%s\" instead of \"%s\"", r1.ns, r2.name)
	}
	if r1.typ != r2.typ {
		t.Errorf("typ is different: \"%s\" instead of \"%s\"", r1.typ, r2.typ)
	}
	if r1.name != r2.name {
		t.Errorf("name is different: \"%s\" instead of \"%s\"", r1.name, r2.name)
	}
}
