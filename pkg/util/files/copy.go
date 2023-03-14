// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package files

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"github.com/onosproject/helmit/pkg/util/k8s"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"path"
	"strings"
)

// CopyOptions is options for copying files from a source to a destination
type CopyOptions struct {
	From      string
	To        string
	Namespace string
	Pod       string
	Container string
}

// Do executes the copy to the pod
func (c *CopyOptions) Do(ctx context.Context) error {
	if c.From == "" || c.Pod == "" {
		return errors.New("source and destination cannot be empty")
	}

	config, err := k8s.GetConfig()
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	pod, err := client.CoreV1().Pods(c.Namespace).Get(ctx, c.Pod, metav1.GetOptions{})
	if err != nil {
		return err
	}

	containerName := c.Container
	if len(containerName) == 0 {
		if len(pod.Spec.Containers) > 1 {
			return errors.New("destination container is ambiguous")
		}
		containerName = pod.Spec.Containers[0].Name
	}

	reader, writer := io.Pipe()

	if c.To == "" {
		c.To = c.From
	}

	// strip trailing slash (if any)
	if c.From != "/" && strings.HasSuffix(string(c.From[len(c.From)-1]), "/") {
		c.From = c.From[:len(c.From)-1]
	}
	if c.To != "/" && strings.HasSuffix(string(c.To[len(c.To)-1]), "/") {
		c.To = c.To[:len(c.To)-1]
	}

	go func() {
		defer writer.Close()
		err := makeTar(c.From, c.To, writer)
		if err != nil {
			fmt.Println(err)
		}
	}()

	cmd := []string{"tar", "-xf", "-"}
	req := client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(c.Pod).
		Namespace(c.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   cmd,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  reader,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		return err
	}
	return nil
}

func makeTar(srcPath, destPath string, writer io.Writer) error {
	// TODO: use compression here?
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)
	return recursiveTar(path.Dir(srcPath), path.Base(srcPath), path.Dir(destPath), path.Base(destPath), tarWriter)
}

func recursiveTar(srcBase, srcFile, destBase, destFile string, tw *tar.Writer) error {
	filepath := path.Join(srcBase, srcFile)
	stat, err := os.Lstat(filepath)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		files, err := ioutil.ReadDir(filepath)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			//case empty directory
			hdr, _ := tar.FileInfoHeader(stat, filepath)
			hdr.Name = destFile
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		}
		for _, f := range files {
			if err := recursiveTar(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), tw); err != nil {
				return err
			}
		}
		return nil
	} else if stat.Mode()&os.ModeSymlink != 0 {
		//case soft link
		hdr, _ := tar.FileInfoHeader(stat, filepath)
		target, err := os.Readlink(filepath)
		if err != nil {
			return err
		}

		hdr.Linkname = target
		hdr.Name = destFile
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
	} else {
		//case regular file or other file type like pipe
		hdr, err := tar.FileInfoHeader(stat, filepath)
		if err != nil {
			return err
		}
		hdr.Name = destFile

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(filepath)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		return f.Close()
	}
	return nil
}
