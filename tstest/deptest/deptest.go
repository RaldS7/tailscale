// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// The deptest package contains a shared implementation of negative
// dependency tests for other packages, making sure we don't start
// depending on certain packages.
package deptest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

type DepChecker struct {
	GOOS    string            // optional
	GOARCH  string            // optional
	BadDeps map[string]string // package => why
}

func (c DepChecker) Check(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Slow and avoid caring about "go.exe" etc.
		t.Skip("skipping dep tests on windows hosts")
	}
	t.Helper()
	cmd := exec.Command("go", "list", "-json", ".")
	var extraEnv []string
	if c.GOOS != "" {
		extraEnv = append(extraEnv, "GOOS="+c.GOOS)
	}
	if c.GOARCH != "" {
		extraEnv = append(extraEnv, "GOARCH="+c.GOARCH)
	}
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Deps []string
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatal(err)
	}

	for _, dep := range res.Deps {
		if why, ok := c.BadDeps[dep]; ok {
			t.Errorf("package %q is not allowed as a dependency (env: %q); reason: %s", dep, extraEnv, why)
		}
	}
	t.Logf("got %d dependencies", len(res.Deps))
}

// ImportAliasCheck checks that all packages are imported according to Tailscale
// conventions.
func ImportAliasCheck(t testing.TB, relDir string) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir = filepath.Join(dir, relDir)

	cmd := exec.Command("git", "grep", "-n", "-F", `"golang.org/x/exp/`)
	cmd.Dir = dir
	matches, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("ignoring error: %v, %s", err, matches)
		return
	}
	badRx := regexp.MustCompile(`^([^:]+:\d+):\s+"golang.org/x/exp/(slices|maps)"`)
	if s := strings.TrimSpace(string(matches)); s != "" {
		for _, line := range strings.Split(s, "\n") {
			if m := badRx.FindStringSubmatch(line); m != nil {
				t.Errorf("%s: the x/exp/%s package should be imported as x%s", m[1], m[2], m[2])
			}
		}
	}
}
