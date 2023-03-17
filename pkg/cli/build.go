package cli

import (
	"errors"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultRunnerImage = "onosproject/helmit-runner"
)

func buildBinary(pkgPath, binPath string) error {
	build := exec.Command("go", "build", "-o", binPath, pkgPath)
	build.Stderr = os.Stderr
	build.Stdout = os.Stdout
	env := os.Environ()
	env = append(env, "GOOS=linux", "CGO_ENABLED=0")
	build.Env = env
	return build.Run()
}

func validatePackage(pkgPath string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	pkg, err := build.Import(pkgPath, workDir, build.ImportComment)
	if err != nil {
		return err
	}

	if !pkg.IsCommand() {
		return errors.New("main not found in package")
	}
	return nil
}

func parseFiles(files []string) (map[string][]string, error) {
	if len(files) == 0 {
		return map[string][]string{}, nil
	}

	values := make(map[string][]string)
	for _, path := range files {
		index := strings.Index(path, "=")
		if index == -1 {
			return nil, errors.New("values file must be in the format {release}={file}")
		}
		release, path := path[:index], path[index+1:]
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		_, err = os.Stat(path)
		if err != nil {
			return nil, err
		}
		releaseValues, ok := values[release]
		if !ok {
			releaseValues = make([]string, 0)
		}
		values[release] = append(releaseValues, path)
	}
	return values, nil
}

func parseOverrides(values []string) (map[string][]string, error) {
	overrides := make(map[string][]string)
	for _, set := range values {
		index := strings.Index(set, ".")
		if index == -1 {
			return nil, errors.New("values must be in the format {release}.{path}={value}")
		}
		release, value := set[:index], set[index+1:]
		override, ok := overrides[release]
		if !ok {
			override = make([]string, 0)
		}
		overrides[release] = append(override, value)
	}
	return overrides, nil
}

func parseSecrets(secrets []string) (map[string]string, error) {
	if len(secrets) == 0 {
		return map[string]string{}, nil
	}

	values := make(map[string]string)
	for _, secret := range secrets {
		index := strings.Index(secret, "=")
		if index == -1 {
			return nil, errors.New("secrets must be in the format {key}={value}")
		}
		key, value := secret[:index], secret[index+1:]
		values[key] = value
	}
	return values, nil
}
