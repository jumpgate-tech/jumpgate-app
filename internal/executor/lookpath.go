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
//
// On Windows only .exe and .com are ever started, whatever PATHEXT says: a
// .bat or .cmd runs through cmd.exe, which re-parses the arguments
// CreateProcess was given (BatBadBut), and the argv path exists precisely so
// no shell parses them.
func lookPathIn(name, pathList, goos, pathext string) (string, error) {
	if strings.ContainsAny(name, `/\`) {
		if goos == "windows" && !windowsDirectExt(filepath.Ext(name)) {
			return "", exec.ErrNotFound
		}
		return name, nil
	}
	sep, exts := ":", []string{""}
	if goos == "windows" {
		sep, exts = ";", nil
		if ext := filepath.Ext(name); ext != "" {
			if !windowsDirectExt(ext) {
				return "", exec.ErrNotFound
			}
			exts = append(exts, "")
		}
		if pathext == "" {
			pathext = ".COM;.EXE"
		}
		for _, e := range strings.Split(pathext, ";") {
			if e != "" && windowsDirectExt(e) {
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

// windowsDirectExt reports whether a file with extension ext is started by
// CreateProcess itself, with no interpreter in between.
func windowsDirectExt(ext string) bool {
	return strings.EqualFold(ext, ".exe") || strings.EqualFold(ext, ".com")
}
