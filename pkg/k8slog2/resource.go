package k8slog

import (
	"strings"

	"github.com/gobwas/glob"
	"github.com/pkg/errors"
)

type EnumResourceType = string

const (
	ResourceTypePod     EnumResourceType = "pod"
	ResourceTypeService                  = "svc"
	ResourceTypeDeploy                   = "deploy"
)

type resource struct {
	ns    string
	typ   EnumResourceType
	name  string
	gname glob.Glob
}

func parseResource(res string) (*resource, error) {
	var err error
	var name string
	ret := &resource{
		ns:  DefaultNamespace,
		typ: DefaultType,
	}

	chunks := strings.Split(res, "/")
	switch l := len(chunks); {
	case l == 1:
		name = chunks[0]
	case l == 2:
		ret.typ = chunks[0]
		name = chunks[1]
	case l == 3:
		ret.ns = chunks[0]
		ret.typ = chunks[1]
		name = chunks[2]
	}
	ret.gname, err = glob.Compile(name)
	if err != nil {
		return nil, errors.Wrap(err, "glob")
	}
	return ret, nil
}
