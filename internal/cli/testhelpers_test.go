package cli

import (
	"net/url"
	"strconv"
)

func hostOf(rawURL string) string {
	u, _ := url.Parse(rawURL)
	return u.Hostname()
}

func portOf(rawURL string) string {
	u, _ := url.Parse(rawURL)
	p, _ := strconv.Atoi(u.Port())
	return strconv.Itoa(p)
}
