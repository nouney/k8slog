package k8slog

import "fmt"

// ResourceType represents a k8s resource type
type ResourceType int

const (
	// TypeUnknown is an unknown resource type
	TypeUnknown ResourceType = iota
	// TypePod is the resource type for pods
	TypePod
	// TypeDeploy is the resource type for deployments
	TypeDeploy
	// TypeStatefulSet is the resource type for statefulsets
	TypeStatefulSet
	// TypeReplicaSet is the resource type for replicasets
	TypeReplicaSet
	// TypeService is the resource type for services
	TypeService

	lastType = TypeService + 1
)

var types [lastType]func(resource) Resource
var strTypes map[string]ResourceType

func strTypeToConst(str string) (ResourceType, error) {
	c, ok := strTypes[str]
	if !ok || c == TypeUnknown {
		return TypeUnknown, fmt.Errorf("unknown resource type: %s", str)
	}
	return c, nil
}

func registerType(typ ResourceType, f func(resource) Resource, strs ...string) {
	if strTypes == nil {
		strTypes = make(map[string]ResourceType)
	}
	types[typ] = f
	for _, str := range strs {
		strTypes[str] = typ
	}
}
