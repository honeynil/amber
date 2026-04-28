package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var distFS embed.FS

func Handler() http.Handler {
	dist, _ := fs.Sub(distFS, "dist")
	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" {
			f, err := dist.Open(path[1:])
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r2 := *r
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, &r2)
	})
}
