// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"errors"
	"fmt"
	"github.com/onosproject/helmit/internal/logging"
	"go/types"
	"golang.org/x/tools/go/packages"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

func newBuilder(suiteType reflect.Type, suiteMatchers []string, template string, log logging.Logger) *Builder {
	return &Builder{
		log:           log,
		template:      template,
		suiteType:     suiteType,
		suiteMatchers: suiteMatchers,
	}
}

// Builder is a builder for dynamically generating and compiling test/benchmark suite binaries.
type Builder struct {
	log           logging.Logger
	template      string
	suiteType     reflect.Type
	suiteMatchers []string
}

// Build parses the given pkgPaths to locate test/benchmark suites, generates a main to run the
// matching suites, and builds a binary from the main, outputting the resulting executable to binPath.
func (b *Builder) Build(binPath string, pkgPaths ...string) error {
	info, err := b.getBuildInfo(pkgPaths...)
	if err != nil {
		return err
	}

	if len(info.Suites) == 0 {
		return fmt.Errorf("no matching suites found in packages %s", strings.Join(pkgPaths, ","))
	}

	mainDir := filepath.Join(info.Module.Dir, ".helmit")
	if err := os.MkdirAll(mainDir, os.ModePerm); err != nil {
		return err
	}
	mainFile := filepath.Join(mainDir, "main.go")
	defer func() {
		_ = os.RemoveAll(mainDir)
	}()

	if err := b.applyTemplate(mainFile, info); err != nil {
		return err
	}

	if err := b.buildBinary(mainDir, binPath); err != nil {
		return err
	}
	return nil
}

// getBuildInfo parses the given Go package paths to locate matching suites within those packages, returning
// buildInfo suitable for templating to generate Go mains.
func (b *Builder) getBuildInfo(pkgPaths ...string) (buildInfo, error) {
	b.log.Logf("Parsing packages %s", strings.Join(pkgPaths, ","))

	var build buildInfo
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule,
	}
	pkgs, err := packages.Load(cfg, pkgPaths...)
	if err != nil {
		return build, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return build, errors.New("failed to parse packages")
	}

	imports := make(map[string]importInfo)
	aliases := make(map[string]bool)
	for _, pkg := range pkgs {
		if build.Module.Path != "" && build.Module.Path != pkg.Module.Path {
			return build, errors.New("all suites must be under the same Go module")
		}

		build.Module.Path = pkg.Module.Path
		build.Module.Dir = pkg.Module.Dir

		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if ok, err := b.isMatch(obj); err != nil {
				return build, err
			} else if !ok {
				continue
			}

			imp, ok := imports[obj.Pkg().Path()]
			if !ok {
				imp = importInfo{
					Path: obj.Pkg().Path(),
				}
				var i int
				for {
					alias := fmt.Sprintf("%s%d", strings.ReplaceAll(filepath.Base(obj.Pkg().Path()), "-", "_"), i)
					if _, ok := aliases[alias]; !ok {
						imp.Alias = alias
						aliases[alias] = true
						break
					}
					i++
				}
				imports[obj.Pkg().Path()] = imp
				build.Imports = append(build.Imports, imp)
			}

			build.Suites = append(build.Suites, suiteInfo{
				Name:   obj.Name(),
				Import: imp,
			})
		}
	}
	return build, nil
}

func (b *Builder) isMatch(obj types.Object) (bool, error) {
	if !obj.Exported() {
		return false, nil
	}
	for _, suiteMatcher := range b.suiteMatchers {
		if ok, err := regexp.MatchString(suiteMatcher, obj.Name()); err != nil {
			return false, err
		} else if ok {
			return b.isSuite(obj), nil
		}
	}
	return false, nil
}

func (b *Builder) isSuite(obj types.Object) bool {
	if obj == nil || obj.Type() == nil || obj.Type().Underlying() == nil {
		return false
	}
	st, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		named, ok := f.Type().(*types.Named)
		if !ok {
			continue
		}
		if named.Obj().Pkg().Path() == b.suiteType.PkgPath() && named.Obj().Name() == b.suiteType.Name() {
			return true
		}
		if b.isSuite(f) {
			return true
		}
	}
	return false
}

func (b *Builder) applyTemplate(path string, info buildInfo) error {
	b.log.Logf("Generating %s", path)
	tpl, err := template.New("main").Parse(b.template)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return tpl.Execute(file, info)
}

func (b *Builder) buildBinary(mainDir, binPath string) error {
	b.log.Logf("Building binary %s", binPath)
	build := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-o", binPath, mainDir)
	build.Stderr = os.Stderr
	build.Stdout = os.Stdout
	env := os.Environ()
	env = append(env, "GOOS=linux", "CGO_ENABLED=0")
	build.Env = env
	return build.Run()
}

type buildInfo struct {
	Module  moduleInfo
	Imports []importInfo
	Suites  []suiteInfo
}

type moduleInfo struct {
	Path string
	Dir  string
}

type importInfo struct {
	Path  string
	Alias string
}

type suiteInfo struct {
	Name   string
	Import importInfo
}
