package cli

import (
	"flag"
	"fmt"
	"time"
)

type Globals struct {
	Scheme  string
	Host    string
	Port    int
	Company string
	Timeout time.Duration
	Pretty  bool
	Format  string
}

func registerGlobals(fs *flag.FlagSet) *Globals {
	g := &Globals{}
	fs.StringVar(&g.Scheme, "scheme", "http", "URL scheme (http or https)")
	fs.StringVar(&g.Host, "host", "localhost", "Tally host")
	fs.IntVar(&g.Port, "port", 9000, "Tally port")
	fs.StringVar(&g.Company, "company", "", "Tally current company name")
	fs.DurationVar(&g.Timeout, "timeout", 30*time.Second, "HTTP timeout")
	fs.BoolVar(&g.Pretty, "pretty", false, "Pretty-print response (XML)")
	fs.StringVar(&g.Format, "format", "xml", "Output format: xml or json")
	return g
}

func (g *Globals) URL() string {
	scheme := g.Scheme
	if scheme == "" {
		scheme = "http"
	}
	if (scheme == "https" && g.Port == 443) || (scheme == "http" && g.Port == 80) {
		return fmt.Sprintf("%s://%s/", scheme, g.Host)
	}
	return fmt.Sprintf("%s://%s:%d/", scheme, g.Host, g.Port)
}
