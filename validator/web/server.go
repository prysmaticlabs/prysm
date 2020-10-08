package web

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
)

const prefix = "external/prysm_web_ui"

func Run() {
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		u, err := url.ParseRequestURI(req.RequestURI)
		if err != nil {
			panic(err)
		}
		p := u.Path
		if p == "/" {
			p = "/index.html"
		}
		p = path.Join(prefix, p)

		// DEBUG: Print the path.
		fmt.Println(p)

		if d, ok := site[p]; ok {
			res.WriteHeader(200)
			if _, err := res.Write(d); err != nil {
				panic(err)
			}
		} else {
			res.WriteHeader(404)
		}

	})

	log.Fatal(http.ListenAndServe(":3000", nil))
}
