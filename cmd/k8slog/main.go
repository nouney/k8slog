package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nouney/k8slog/pkg/colorpicker"
	"github.com/nouney/k8slog/pkg/k8s"
	"github.com/nouney/k8slog/pkg/k8slog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

var (
	flagFollow     = false
	flagColors     = true
	flagTimestamp  = true
	flagPrefix     = true
	flagKubeconfig = ""
	flagJSONFields = []string{}
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var cmd = &cobra.Command{
	Use:   "k8slog",
	Short: "A brief description of your application",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		k8s, err := k8s.NewClient(flagKubeconfig)
		if err != nil {
			return err
		}

		klog := k8slog.New(
			k8s,
			k8slog.WithOptsTimestamps(flagTimestamp),
			k8slog.WithOptsJSONFields(flagJSONFields...),
			k8slog.WithOptsFollow(flagFollow),
		)
		out, err := klog.Logs(args...)
		if err != nil {
			return err
		}

		cp := colorpicker.New()
		format := formatter(cp)
		for {
			logline, ok := <-out
			if !ok {
				break
			}
			fmt.Print(format(&logline))
		}
		return nil
	},
}

func formatter(cp *colorpicker.ColorPicker) func(logline *k8slog.LogLine) string {
	if !flagPrefix {
		return func(logline *k8slog.LogLine) string {
			return logline.Line
		}
	}
	var podName func(*k8slog.LogLine) string
	if flagColors {
		podName = func(logline *k8slog.LogLine) string {
			color := cp.Pick(logline.Namespace + "/" + string(logline.Type) + "/" + logline.Name)
			return color.Sprint(logline.Pod)
		}
	} else {
		podName = func(logline *k8slog.LogLine) string {
			return logline.Pod
		}
	}
	return func(logline *k8slog.LogLine) string {
		return concat("[", logline.Namespace, "][", podName(logline), "]: ", logline.Line)
	}
}

func concat(strs ...string) string {
	var buffer bytes.Buffer
	for _, str := range strs {
		buffer.WriteString(str)
	}
	return buffer.String()
}

func init() {
	defaultKubeconfig := ""
	if home := homedir.HomeDir(); home != "" {
		defaultKubeconfig = filepath.Join(home, ".kube", "config")
	}
	cmd.PersistentFlags().StringVar(&flagKubeconfig, "kubeconfig", defaultKubeconfig, "absolute path to the kubeconfig file")
	cmd.Flags().BoolVarP(&flagColors, "colors", "c", true, "enable colors")
	cmd.Flags().BoolVarP(&flagFollow, "follow", "f", false, "follow the logs")
	cmd.Flags().BoolVarP(&flagTimestamp, "timestamp", "t", true, "print timestamp")
	cmd.Flags().BoolVarP(&flagPrefix, "prefix", "p", true, "print prefix")
	cmd.Flags().StringSliceVarP(&flagJSONFields, "json", "j", nil, "json log only, print a specific field")
}
