package executor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// lookPathIn is exec.LookPath against a given PATH rather than this
// process's own. exec.Command resolves a bare name with os.Getenv("PATH")
// when it is constructed, before c.Env applies. Without this, the dirs
// localEnv adds (Docker Desktop's CLI dir for an app started from Finder or
// Explorer) would never be searched.
func lookPathIn(name, pathList, goos, pathext string) (string, error) {
	if strings.ContainsAny(name, `/\`) {
		return name, nil
	}
	sep, exts := ":", []string{""}
	if goos == "windows" {
		sep, exts = ";", nil
		if filepath.Ext(name) != "" {
			exts = append(exts, "")
		}
		if pathext == "" {
			pathext = ".COM;.EXE;.BAT;.CMD"
		}
		for _, e := range strings.Split(pathext, ";") {
			if e != "" {
				exts = append(exts, strings.ToLower(e))
			}
		}
	}
	for _, dir := range strings.Split(pathList, sep) {
		if dir == "" {
			continue
		}
		for _, ext := range exts {
			p := filepath.Join(dir, name+ext)
			fi, err := os.Stat(p)
			if err != nil || !fi.Mode().IsRegular() {
				continue
			}
			if goos == "windows" || fi.Mode()&0o111 != 0 {
				return p, nil
			}
		}
	}
	return "", exec.ErrNotFound
}
