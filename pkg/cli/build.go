// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/iancoleman/strcase"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type walker func(ast.Node) bool

func (w walker) Visit(node ast.Node) ast.Visitor {
	if w(node) {
		return w
	}
	return nil
}

func getSuites(pkgPath string, suiteType string) ([]string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	pkg, err := build.Import(pkgPath, workDir, build.ImportComment)
	if err != nil {
		return nil, err
	}

	fileset := token.NewFileSet()
	files, err := ioutil.ReadDir(pkg.Dir)
	if err != nil {
		return nil, err
	}

	suites := make([]string, 0)
	for _, info := range files {
		if file, err := parser.ParseFile(fileset, filepath.Join(pkg.Dir, info.Name()), nil, 0); err == nil {
			suiteImp := ""
			for _, imp := range file.Imports {
				if imp.Path.Value == fmt.Sprintf("\"github.com/onosproject/helmit/pkg/%s\"", suiteType) {
					if imp.Name != nil {
						suiteImp = imp.Name.Name
					} else {
						suiteImp = suiteType
					}
				}
			}

			ast.Walk(walker(func(node ast.Node) bool {
				switch v := node.(type) {
				case *ast.GenDecl:
					switch v.Tok {
					case token.TYPE:
						for _, spec := range v.Specs {
							switch s := spec.(type) {
							case *ast.TypeSpec:
								name := s.Name.Name
								if ast.IsExported(name) {
									switch v := s.Type.(type) {
									case *ast.StructType:
										for _, field := range v.Fields.List {
											if e, ok := field.Type.(*ast.SelectorExpr); ok && e.X.(*ast.Ident).Name == suiteImp && e.Sel.Name == "Suite" {
												suites = append(suites, s.Name.Name)
											}
										}
									}
								}
							}
						}
					}
				}
				return true
			}), file)
		} else {
			return nil, err
		}
	}
	return suites, nil
}

func buildMain(pkgPath string, suiteType string) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	pkg, err := build.Import(pkgPath, workDir, build.ImportComment)
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(pkg.Dir)))
	tempPath := filepath.Join(os.TempDir(), "helmit", hash)
	binPath := filepath.Join(tempPath, "helmit", "bin")
	mainPath := filepath.Join(tempPath, "helmit", "main.go")

	if pkg.IsCommand() {
		err = buildBinary(workDir, pkgPath, binPath)
		if err != nil {
			return "", err
		}
		return binPath, nil
	}

	mod, err := getModuleInfo()
	if err != nil {
		return "", err
	}

	err = copyDir(pkg.SrcRoot, tempPath)
	if err != nil {
		return "", err
	}

	suites, err := getSuites(pkgPath, "test")
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(filepath.Dir(mainPath), os.ModePerm)
	if err != nil {
		return "", err
	}

	mainFile, err := os.Create(mainPath)
	if err != nil {
		return "", err
	}

	parent, child := filepath.Split(pkg.Dir)
	impPath := strings.Trim(child, "/")
	for strings.TrimSuffix(parent, "/") != mod.Dir {
		parent, child = filepath.Split(parent)
		impPath = strings.Trim(filepath.Join(child, impPath), "/")
	}

	fmt.Fprintf(mainFile, "package main\n")
	fmt.Fprintf(mainFile, "import (\n")
	fmt.Fprintf(mainFile, "	suites \"%s/%s\"\n", mod.Path, impPath)
	fmt.Fprintf(mainFile, "	\"github.com/onosproject/helmit/pkg/registry\"\n")
	fmt.Fprintf(mainFile, "	\"github.com/onosproject/helmit/pkg/%s\"\n", suiteType)
	fmt.Fprintf(mainFile, ")\n")
	fmt.Fprintf(mainFile, "func main() {\n")
	for _, suite := range suites {
		fmt.Fprintf(mainFile, "	registry.Register%sSuite(\"%s\", &suites.%s{})\n", strcase.ToCamel(suiteType), suite, suite)
	}
	fmt.Fprintf(mainFile, "	%s.Main()\n", suiteType)
	fmt.Fprintf(mainFile, "}\n")

	mainFile.Close()

	err = buildBinary(tempPath, "./helmit", binPath)
	if err != nil {
		return "", err
	}
	return binPath, nil
}

func buildBinary(workDir, pkgPath, binPath string) error {
	pkg, err := build.Import(pkgPath, workDir, build.ImportComment)
	if err != nil {
		return err
	}

	if !pkg.IsCommand() {
		return errors.New("package must be a command")
	}

	// Build the command
	build := exec.Command("go", "build", "-o", binPath, pkgPath)
	build.Dir = workDir
	build.Stderr = os.Stderr
	build.Stdout = os.Stdout
	env := os.Environ()
	env = append(env, "GOOS=linux", "CGO_ENABLED=0")
	build.Env = env
	return build.Run()
}

type modInfo struct {
	Path string
	Dir  string
}

func getModuleInfo() (*modInfo, error) {
	output, err := exec.Command("go", "list", "-mod=readonly", "-m", "-json").Output()
	if err != nil {
		return nil, err
	}
	var info modInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func copyDir(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err == nil {
		err = os.RemoveAll(dst)
		if err != nil {
			return err
		}
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = copyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}
