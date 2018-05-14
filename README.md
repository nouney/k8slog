# k8slog [![Build Status](https://travis-ci.org/nouney/k8slog.svg?branch=master)](https://travis-ci.org/nouney/k8slog)

k8slog aims to provide an lightweight, quick and easy way to retrieve logs from pods running in a Kubernetes cluster (kind of `kubectl logs` on steroids).

This project is in its early stages and will evolve quickly.

## Features

- Retrieve logs from various Kubernetes resources (pod, deployments, statefulsets, ...)
- Log streaming: follow the logs as they come through
- Live reload: handle creation and deletion of pods
- JSON: parse each log line as a JSON object and prints only certain fields
- Colors: colorized output to easely distinguish logs between resources

## Quick start

```
# Help
$ k8slog --help

# Print logs of pods controlled by deployment "mysvc" in namespace "prod"
$ k8slog prod/deploy/mysvc

# Print logs of pods controlled by deployment "mysvc" in namespace "default"
$ k8slog deploy/mysvc # same as default/deploy/mysvc

# Print logs of pod "mysvc-abcd" in namespace "default"
$ k8slog mysvc-abcd # same as default/pod/mysvc-abcd

# Get multiple logs at once
$ k8slog mypod preprod/svc/mysvc prod/statefulset/mysts

# Follow the logs
$ k8slog deploy/mysvc

# Handle log lines as json object, only prints the fields "timestamp", "user" and "message"
$ k8slog deploy/mysvc --json timestamp,user,message

# You can use -f and --json  at the same time
$ k8slog deploy/mysvc -f --json timestamp,user,message

# Disable timestamp at the beginning of the line
$ k8slog deploy/mysvc --timestamp=false

# Disable colors used by the prefix
$ k8slog deploy/mysvc --colors=false

# Disable prefix ([namespace][pod-name])
$ k8slog deploy/mysvc --prefix=false
```

## Documentation

### Installation

#### Source

To install from code source:

```shell
$ go install github.com/nouney/k8slog/cmd/k8slog
```

>   $GOPATH/bin must be in your $PATH

### Resource string

k8slog uses a string to represent a Kubernetes resource. This string has the following form:  `namespace/resource-type/resource-name`. `namespace` defaults to `default` and `resource-type` defaults to `pod`.

Some examples:
- `prod/deploy/mysvc`: deployment "mysvc" in namespace "prod"
- `svc/mysvc`: service "mysvc" in namespace "default"
- `preprod/pod/mysvc-abcd`: pod "mysvc-abcd" in namespace "preprod"
- `mysvc-abcd`: pod "mysvc-abcd" in namespace "default"

### Retrieve logs

##### Types

- pod, po
- deployment, deploy
- statefulset, sts
- replicaset, rs
- service, svc

#### Snapshot

``` shell
$ k8slog [resources...]
```

Retrieve logs. Same as `kubectl logs`.

#### Stream

```shell
$ k8slog -f [resources...]
```

You can retrieve logs as they come through by using the `-f` or `--follow` flags. In this case, k8slog never returns and waits for new logs to print. Use Ctrl-C to quit it.

k8slog will watch the resources and get logs from pods controlled by them (except for pod resources). So by example if you retrieve logs of a deployment that you scale it up just after, k8slog will also handle the new pods.

### Output

#### JSON

```shell
$ k8slog --json [field1,field2,...] [resources...]
```

This feature is useful if you format your logs as JSON objects. k8slog will parse the log line
and only print the fields you want. The fields are printed in the given order.

#### Prefix

```shell
$ k8slog --prefix=[false|true] [resources...]
```
k8slog begins each line with a prefix of the form `[namespace][pod-name]` to differenciate the resources.
You can disable the prefix by setting the flag `--prefix` to false.

#### Colors

```shell
$ k8slog --colors=[false|true] [resources...]
```

k8slog colorizes the pod name in the prefix to easely differenciate the resources.
You can disable the colors by setting the flag `--colors` to false.

#### Timestamps

```
$ k8slog --timestamp=[false|true] [resources...]
```

By default, k8slog retrieve the timestamp of the log lines and print it just after the prefix.
You can disable the colors by setting the flag `--timestamp` to false.

`--timestamp` is forced to false if `--json` is set.