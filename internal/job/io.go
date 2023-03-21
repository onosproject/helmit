package job

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"path"
)

func (j *Job) GetLogs(ctx context.Context) (io.ReadCloser, error) {
	if err := j.init(); err != nil {
		return nil, err
	}

	req := j.client.CoreV1().Pods(j.Namespace).GetLogs(j.pod.Name, &corev1.PodLogOptions{
		Container: "job",
		Follow:    true,
	})
	return req.Stream(ctx)
}

func (j *Job) Copy(ctx context.Context, dst, src string) error {
	if err := j.init(); err != nil {
		return err
	}

	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()
		err := makeTar(src, dst, writer)
		if err != nil {
			fmt.Println(err)
		}
	}()

	cmd := []string{"tar", "-xf", "-"}
	req := j.client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(j.pod.Name).
		Namespace(j.pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "job",
			Command:   cmd,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(j.config, "POST", req.URL())
	if err != nil {
		return err
	}
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
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

func (j *Job) Echo(ctx context.Context, dst string, data []byte) error {
	if err := j.init(); err != nil {
		return err
	}

	cmd := []string{"/bin/sh", "-c", fmt.Sprintf("echo \"%s\" > %s", string(data), dst)}
	req := j.client.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(j.pod.Name).
		Namespace(j.pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "job",
			Command:   cmd,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(j.config, "POST", req.URL())
	if err != nil {
		return err
	}
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
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
	return recursiveTar(path.Dir(srcPath), path.Base(srcPath), path.Base(destPath), tarWriter)
}

func recursiveTar(srcBase, srcFile, destFile string, tw *tar.Writer) error {
	filepath := path.Join(srcBase, srcFile)
	stat, err := os.Lstat(filepath)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		files, err := os.ReadDir(filepath)
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
			if err := recursiveTar(srcBase, path.Join(srcFile, f.Name()), path.Join(destFile, f.Name()), tw); err != nil {
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
