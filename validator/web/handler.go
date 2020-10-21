package web

import (
	"mime"
	"net/http"
	"net/url"
	"path"

	"github.com/sirupsen/logrus"
)

const prefix = "external/prysm_web_ui/prysm-web-ui"

var (
	log = logrus.WithField("prefix", "prysm-web")
)

// Handler serves web requests from the bundled site data.
var Handler = func(res http.ResponseWriter, req *http.Request) {
	log.Info("Hit")
	u, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		log.WithError(err).Error("Cannot parse request URI")
		return
	}
	p := u.Path
	if p == "/" {
		p = "/index.html"
	}
	log.Info(p)
	p = path.Join(prefix, p)
	log.Info(p)

	if d, ok := site[p]; ok {
		log.Info("In basic site")
		m := mime.TypeByExtension(path.Ext(p))
		res.Header().Add("Content-Type", m)
		res.WriteHeader(200)
		if _, err := res.Write(d); err != nil {
			log.WithError(err).Error("Failed to write http response")
		}
	} else if d, ok := site[path.Join(prefix, "index.html")]; ok {
		log.Info("In index")
		// Angular routing expects that routes are rewritten to serve index.html. For example, if
		// requesting /login, this should serve the single page app index.html.
		m := mime.TypeByExtension(".html")
		res.Header().Add("Content-Type", m)
		res.WriteHeader(200)
		if _, err := res.Write(d); err != nil {
			log.WithError(err).Error("Failed to write http response")
		}
	} else { // If index.html is not present, serve 404. This should never happen.
		log.Info("not present")
		log.WithField("URI", req.RequestURI).Error("Path not found")
		res.WriteHeader(404)
		if _, err := res.Write([]byte("Not found")); err != nil {
			log.WithError(err).Error("Failed to write http response")
		}
	}
}
