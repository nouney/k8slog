// Copyright © 2018 Valentin Tjoncke <valtjo@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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

		klog := k8slog.New(k8s, k8slog.WithOptsJSONFields(flagJSONFields...), k8slog.WithOptsFollow(flagFollow))
		out, err := klog.Logs(args...)
		if err != nil {
			return err
		}

		cp := colorpicker.New()
		format := formatter(cp, len(args) > 1)
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

func formatter(cp *colorpicker.ColorPicker, multiple bool) func(logline *k8slog.Line) string {
	if !multiple {
		return func(logline *k8slog.Line) string {
			return logline.Line
		}
	}
	var podName func(*k8slog.Line) string
	if flagColors {
		podName = func(logline *k8slog.Line) string {
			color := cp.Pick(logline.Pod)
			return color.Sprint(logline.Pod)
		}
	} else {
		podName = func(logline *k8slog.Line) string {
			return logline.Pod
		}
	}
	return func(logline *k8slog.Line) string {
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
	cmd.PersistentFlags().BoolVar(&flagColors, "colors", true, "enable colors")
	cmd.Flags().BoolVarP(&flagFollow, "follow", "f", false, "follow the logs")
	cmd.Flags().StringSliceVarP(&flagJSONFields, "json", "j", nil, "json log only, print a specific field")
}
