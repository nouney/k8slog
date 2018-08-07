package k8slog

// Option is an option used to configure K8SLog
type Option func(c *K8SLog)

// WithFollow enable to follow log stream (default: false).
//
// If the follow option is enabled, the client will follow the log stream of the resources.
// If the given resource type is not a pod, the K8SLog will also watch for new pods of the resource.
func WithFollow(value bool) Option {
	return func(c *K8SLog) {
		c.follow = value
	}
}

// WithTimestamps enable timestamps at the beginning of the log line (default: true)
func WithTimestamps(value bool) Option {
	return func(c *K8SLog) {
		c.timestamps = value
	}
}

// WithJSONFields configure the json option (default: none).
// If enabled, log lines will be handled as JSON objects and only the given fields will be printed.
func WithJSONFields(fields ...string) Option {
	return func(c *K8SLog) {
		c.jsonFields = fields
		c.jsonFieldsLen = len(fields)
		if c.jsonFieldsLen > 0 {
			// disable timestamps so we can parse the json object
			WithTimestamps(false)(c)
		}
	}
}

func WithDebug(value bool) Option {
	return func(c *K8SLog) {
		c.debugEnabled = value
	}
}
