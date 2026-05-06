package cli

import (
	"flag"
	"fmt"
	"time"
)

type Globals struct {
	Host    string
	Port    int
	Company string
	Timeout time.Duration
	Pretty  bool
}

func registerGlobals(fs *flag.FlagSet) *Globals {
	g := &Globals{}
	fs.StringVar(&g.Host, "host", "localhost", "Tally host")
	fs.IntVar(&g.Port, "port", 9000, "Tally port")
	fs.StringVar(&g.Company, "company", "", "Tally current company name")
	fs.DurationVar(&g.Timeout, "timeout", 30*time.Second, "HTTP timeout")
	fs.BoolVar(&g.Pretty, "pretty", false, "Pretty-print response")
	return g
}

func (g *Globals) URL() string {
	return fmt.Sprintf("http://%s:%d/", g.Host, g.Port)
}
