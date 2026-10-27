// Command relsrv is a stand-in for GitHub's release download layout, for
// scripts/e2e-install.sh: it serves DIR/download/<tag>/<file> over TLS,
// answers /tag/<tag> for any tag dir, and redirects /latest (302) to the
// newest tag dir (semver order; a prerelease sorts before its release).
//
//	go run ./scripts/e2e/relsrv -dir D -cert C -key K -addr :8443
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	dir := flag.String("dir", "", "release dir holding download/<tag>/<file>")
	cert := flag.String("cert", "", "TLS certificate (PEM)")
	key := flag.String("key", "", "TLS key (PEM)")
	addr := flag.String("addr", ":8443", "listen address")
	flag.Parse()
	if *dir == "" || *cert == "" || *key == "" {
		log.Fatal("usage: relsrv -dir D -cert C -key K [-addr :8443]")
	}
	dl := filepath.Join(*dir, "download")
	mux := http.NewServeMux()
	mux.Handle("/download/", http.StripPrefix("/download/", http.FileServer(http.Dir(dl))))
	mux.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		tag := newest(dl)
		if tag == "" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/tag/"+tag, http.StatusFound)
	})
	mux.HandleFunc("/tag/", func(w http.ResponseWriter, r *http.Request) {
		tag := strings.TrimPrefix(r.URL.Path, "/tag/")
		fi, err := os.Stat(filepath.Join(dl, filepath.Base(tag)))
		if err != nil || !fi.IsDir() || tag != filepath.Base(tag) {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("release " + tag + "\n"))
	})
	log.Fatal(http.ListenAndServeTLS(*addr, *cert, *key, mux))
}

// newest is the highest vX.Y.Z[-pre] dir name under dl.
func newest(dl string) string {
	ents, err := os.ReadDir(dl)
	if err != nil {
		return ""
	}
	best := ""
	for _, e := range ents {
		if e.IsDir() && parse(e.Name()) != nil && (best == "" || less(best, e.Name())) {
			best = e.Name()
		}
	}
	return best
}

type ver struct {
	n   [3]int
	pre string
}

func parse(s string) *ver {
	if !strings.HasPrefix(s, "v") {
		return nil
	}
	core, pre, _ := strings.Cut(s[1:], "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return nil
	}
	v := &ver{pre: pre}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		v.n[i] = n
	}
	return v
}

// less orders semver tags; a prerelease sorts before its release.
func less(a, b string) bool {
	x, y := parse(a), parse(b)
	for i := range x.n {
		if x.n[i] != y.n[i] {
			return x.n[i] < y.n[i]
		}
	}
	switch {
	case x.pre == y.pre, x.pre == "":
		return false
	case y.pre == "":
		return true
	}
	return x.pre < y.pre
}
