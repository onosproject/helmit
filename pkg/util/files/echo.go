// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package files

import (
	"context"
	"errors"
	"fmt"
	"github.com/onosproject/helmit/pkg/util/k8s"
	"k8s.io/client-go/kubernetes"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// EchoOptions is options for echoing output to a file
type EchoOptions struct {
	Namespace string
	Pod       string
	Container string
	File      string
	Bytes     []byte
}

// Do executes the copy to the pod
func (o *EchoOptions) Do(ctx context.Context) error {
	if o.Pod == "" || o.File == "" {
		return errors.New("target file cannot be empty")
	}

	config, err := k8s.GetConfig()
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	pod, err := client.CoreV1().Pods(o.Namespace).Get(context.Background(), o.Pod, metav1.GetOptions{})
	if err != nil {
		return err
	}

	containerName := o.Container
	if len(containerName) == 0 {
		if len(pod.Spec.Containers) > 1 {
			return errors.New("destination container is ambiguous")
		}
		containerName = pod.Spec.Containers[0].Name
	}

	cmd := []string{"/bin/sh", "-c", fmt.Sprintf("echo \"%s\" > %s", string(o.Bytes), o.File)}
	req := client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(o.Pod).
		Namespace(o.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   cmd,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		return err
	}
	return nil
}
