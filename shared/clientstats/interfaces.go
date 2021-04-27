package clientstats

import "io"

type Scraper interface {
	Scrape() (io.Reader, error)
}

type Updater interface {
	Update(io.Reader) error
}
