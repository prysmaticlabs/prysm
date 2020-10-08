package web

import (
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
		res.WriteHeader(200)
		if _, err := res.Write(d); err != nil {
			log.WithError(err).Error("Failed to write http response")
		}
	} else {
		res.WriteHeader(404)
		if _, err := res.Write([]byte("Not found")); err != nil {
			log.WithError(err).Error("Failed to write http response")
		}
	}
}