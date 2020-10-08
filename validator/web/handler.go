package web

import (
	"mime"
	"net/http"
	"net/url"
	"path"

	log "github.com/sirupsen/logrus"
)

const prefix = "external/prysm_web_ui"

// webHandler serves web requests from the bundled site data.
var webHandler = func(res http.ResponseWriter, req *http.Request) {
	u, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		panic(err)
	}
	p := u.Path
	if p == "/" {
		p = "/index.html"
	}
	p = path.Join(prefix, p)

	if d, ok := site[p]; ok {
		m := mime.TypeByExtension(path.Ext(p))
		res.Header().Add("Content-Type", m)
		res.WriteHeader(200)
		if _, err := res.Write(d); err != nil {
			log.WithError(err).Error("Failed to write http response")
		}
	// Serve index.html as default.
	} else if d, ok := site[path.Join(prefix, "index.html")]; ok {
		m := mime.TypeByExtension("html")
		res.Header().Add("Content-Type", m)
		res.WriteHeader(200)
		if _, err := res.Write(d); err != nil {
			log.WithError(err).Error("Failed to write http response")
		}
	} else { // If index.html is not present, serve 404.
		res.WriteHeader(404)
		if _, err := res.Write([]byte("Not found")); err != nil {
			log.WithError(err).Error("Failed to write http response")
		}
	}
}