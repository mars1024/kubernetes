package filter

import (
	"net/http"
	"os"
	"strings"
)

const (
	ENV_VAR_USERAGENT_BLACKLIST = "BLACKLIST_USER_AGENT"

	blacklist_separator = "##"
)

func WithBlacklist(delegate http.Handler) http.Handler {
	var blacklistUserAgents []string
	if list := os.Getenv(ENV_VAR_USERAGENT_BLACKLIST); len(list) > 0 {
		blacklistUserAgents = strings.Split(list, blacklist_separator)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if len(req.Header.Get("User-Agent")) == 0 {
			w.Write([]byte("Request with empty User-Agent is forbidden"))
			w.WriteHeader(403)
			return
		}
		for _, agent := range blacklistUserAgents {
			if strings.HasPrefix(req.Header.Get("User-Agent"), agent) {
				w.Write([]byte("AKS blacklisted"))
				w.WriteHeader(403)
				return
			}
		}
		delegate.ServeHTTP(w, req)
	})
}
