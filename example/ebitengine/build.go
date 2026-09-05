// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
)

// ebitengineModule is the import path of the Ebitengine module the host is built against. The guest
// must be built against the same version, since the host and guest speak a version-locked protocol.
const ebitengineModule = "github.com/hajimehoshi/ebiten/v2"

// ebitengineRequireVersion is the version in the generated module's require directive. It is a
// placeholder matching the module path's major version: the replace directive overrides every version
// of the module, so the required version is never fetched.
const ebitengineRequireVersion = "v2.0.0"

// resolveEbitengineReplacement returns the replace-directive target that pins a guest to the host's
// Ebitengine version — "<module>@<version>" or a directory — determined once from the host's own build
// information.
var resolveEbitengineReplacement = sync.OnceValues(func() (string, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("the host has no build information; cannot determine its Ebitengine version")
	}
	for _, dep := range bi.Deps {
		if dep.Path != ebitengineModule {
			continue
		}
		m := dep
		if dep.Replace != nil {
			m = dep.Replace
		}
		// A directory replacement is reproducible for the guest only when it is an absolute path.
		if filepath.IsAbs(m.Path) {
			return m.Path, nil
		}
		if m.Version != "" {
			return m.Path + "@" + m.Version, nil
		}
		return "", fmt.Errorf("the host pins %s to a non-absolute path %q, which cannot be reproduced for the guest", ebitengineModule, m.Path)
	}
	return "", fmt.Errorf("%s is not a dependency of the host", ebitengineModule)
})

// buildGuest builds spec into a binary at bin with the ebitenginevmguest build tag, forcing the guest onto
// the host's Ebitengine version. spec is either a local path, built in its own module, or an import
// path with an optional @version query, built in a module generated under workDir.
func buildGuest(workDir, bin, spec string) error {
	if isFileSystemPath(spec) {
		// A local package is built in its own module, which already pins its Ebitengine version.
		build := exec.Command("go", "build", "-tags", "ebitenginevmguest", "-o", bin, spec)
		build.Stdout = os.Stderr
		build.Stderr = os.Stderr
		return build.Run()
	}

	replacement, err := resolveEbitengineReplacement()
	if err != nil {
		return err
	}

	pkg, version, _ := strings.Cut(spec, "@")

	// The package is built inside a generated module that requires it and replaces Ebitengine with the
	// host's version; -mod=mod lets the package's other dependencies resolve into the generated module.
	md, err := os.MkdirTemp(workDir, "mod")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(md); err != nil {
			slog.Error("removing the temporary module failed", "error", err)
		}
	}()

	if err := goModuleCmd(md, "mod", "init", "ebitenginevmguest"); err != nil {
		return err
	}
	if err := goModuleCmd(md, "mod", "edit",
		"-require="+ebitengineModule+"@"+ebitengineRequireVersion,
		"-replace="+ebitengineModule+"="+replacement); err != nil {
		return err
	}

	if isWithinModule(pkg, ebitengineModule) {
		// A package inside the Ebitengine module is already provided by the pinned require above, so it
		// must not be fetched separately (and cannot be independently versioned).
		if version != "" {
			return fmt.Errorf("a version query is not allowed on %s, which is part of %s", pkg, ebitengineModule)
		}
	} else {
		query := pkg + "@latest"
		if version != "" {
			query = pkg + "@" + version
		}
		if err := goModuleCmd(md, "get", query); err != nil {
			return err
		}
	}

	return goModuleCmd(md, "build", "-mod=mod", "-tags", "ebitenginevmguest", "-o", bin, pkg)
}

// goModuleCmd runs a go command in dir with the workspace disabled, so an enclosing go.work cannot
// override the generated module's pins.
func goModuleCmd(dir string, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isFileSystemPath reports whether spec refers to a package by file system path rather than import path.
func isFileSystemPath(spec string) bool {
	if filepath.IsAbs(spec) {
		return true
	}
	if spec == "." || spec == ".." {
		return true
	}
	for _, prefix := range []string{"./", "../", `.\`, `..\`} {
		if strings.HasPrefix(spec, prefix) {
			return true
		}
	}
	return false
}

// isWithinModule reports whether the import path pkg is provided by the module.
func isWithinModule(pkg, module string) bool {
	return pkg == module || strings.HasPrefix(pkg, module+"/")
}
