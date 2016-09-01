// Package prerender integrates prerender.io with net/http.
package prerender

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type handler struct {
	sub                 http.Handler
	botUserAgents       []string
	ignoredExtension    []string
	prerenderServiceURL string
	prerenderToken      string
	prerenderUsername   string
	prerenderPassword   string
	log                 *log.Logger
}

type Option func(*handler)

// Handler returns a new prerender handler. app must be your HTTP app.
func Handler(app http.Handler, options ...Option) http.Handler {
	if app == nil {
		app = http.DefaultServeMux
	}

	h := &handler{sub: app}

	// Defaults
	Bots(crawlerUserAgents)(h)
	IgnoredExtensions(extensionsToIgnore)(h)
	ServiceURL(prerenderServiceURL)(h)

	if v := os.Getenv("PRERENDER_SERVICE_URL"); v != "" {
		ServiceURL(v)(h)
	}

	if v := os.Getenv("PRERENDER_TOKEN"); v != "" {
		ServiceToken(v)(h)
	}

	if u, p := os.Getenv("PRERENDER_USERNAME"), os.Getenv("PRERENDER_PASSWORD"); u != "" || p != "" {
		ServiceAuth(u, p)(h)
	}

	// User provided
	for _, option := range options {
		option(h)
	}

	return h
}

// Bots replaces the default list of bot User-Agents with a custom list.
func Bots(userAgents []string) Option {
	return func(h *handler) {
		h.botUserAgents = userAgents
	}
}

// IgnoredExtensions replaces the default list of ignored extentions with a custom list.
func IgnoredExtensions(exts []string) Option {
	return func(h *handler) {
		h.ignoredExtension = exts
	}
}

// ServiceURL sets the prerender service url.
func ServiceURL(url string) Option {
	return func(h *handler) {
		h.prerenderServiceURL = url
	}
}

// ServiceToken sets the prerender service token.
func ServiceToken(token string) Option {
	return func(h *handler) {
		h.prerenderToken = token
	}
}

// ServiceAuth sets the prerender username and password.
func ServiceAuth(username, password string) Option {
	return func(h *handler) {
		h.prerenderUsername, h.prerenderPassword = username, password
	}
}

// Logger sets a logger.
func Logger(logger *log.Logger) Option {
	return func(h *handler) {
		h.log = logger
	}
}

// ServeHTTP serves the HTTP.
func (h *handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !h.shouldShowPrerenderedPage(req) {
		h.sub.ServeHTTP(rw, req)
		return
	}

	h.getPrerenderedPage(rw, req)
}

func (h *handler) shouldShowPrerenderedPage(req *http.Request) bool {
	const (
		X_BUFFERBOT      = "X-Bufferbot"
		ESCAPED_FRAGMENT = "_escaped_fragment_"
	)

	var (
		userAgent                   = req.UserAgent()
		bufferAgent                 = req.Header.Get(X_BUFFERBOT)
		isRequestingPrerenderedPage = false
	)

	if userAgent == "" {
		return false
	}
	if req.Method != "GET" {
		return false
	}

	if q, f := req.URL.Query()[ESCAPED_FRAGMENT]; f && len(q) > 0 {
		isRequestingPrerenderedPage = true
	}

	if h.isBot(userAgent) {
		isRequestingPrerenderedPage = true
	}

	if bufferAgent != "" {
		isRequestingPrerenderedPage = true
	}

	if h.containsIgnoredExtension(req.URL.Path) {
		return false
	}

	return isRequestingPrerenderedPage
}

func (h *handler) isBot(ua string) bool {
	ua = strings.ToLower(ua)
	for _, name := range h.botUserAgents {
		if strings.Contains(ua, name) {
			return true
		}
	}
	return false
}

func (h *handler) containsIgnoredExtension(path string) bool {
	path = strings.ToLower(path)
	for _, name := range h.ignoredExtension {
		if strings.Contains(path, name) {
			return true
		}
	}
	return false
}

func (h *handler) getPrerenderedPage(rw http.ResponseWriter, req1 *http.Request) {
	h.logf("prerender: %q", req1.URL)

	rawurl, err := h.buildApiUrl(req1)
	if err != nil {
		h.logf("prerender error: %s", err)
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
		return
	}

	req2, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		h.logf("prerender error: %s", err)
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
		return
	}

	req2.Header.Set("User-Agent", req1.UserAgent())

	if h.prerenderToken != "" {
		req2.Header.Set(x_PRERENDER_TOKEN, h.prerenderToken)
	}

	if h.prerenderUsername != "" || h.prerenderPassword != "" {
		req2.SetBasicAuth(h.prerenderUsername, h.prerenderPassword)
	}

	httpClient := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("Redirect")
		},
	}

	resp, err := httpClient.Do(req2)

	if err != nil && strings.HasSuffix(err.Error(), "Redirect") == false {
		h.logf("prerender error: %s", err)
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
		return
	} else if err != nil && strings.HasSuffix(err.Error(), "Redirect") == true {

		if resp.Header != nil {
			for key, values := range resp.Header {
				for _, value := range values {
					rw.Header().Set(key, value)
				}
			}
		}

		rw.WriteHeader(301)
		rw.Write([]byte(""))
		return
	}

	rw.WriteHeader(resp.StatusCode)

	if resp.Header != nil {
		for key, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(key, value)
			}
		}
	}

	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logf("prerender error: %s", err)
		fmt.Println(err)
		http.Error(rw, "Internal server error", http.StatusInternalServerError)
		return
	}

	rw.Write(content)

}

func (h *handler) buildApiUrl(req *http.Request) (string, error) {
	const (
		CF_VISITOR        = "Cf-Visitor"
		CF_HTTPS          = `"scheme":"https"`
		X_FORWARDED_PROTO = "X-Forwarded-Proto"
		X_FORWARDED_HTTPS = "https,"
		HTTP_HOST         = "Host"
	)

	var (
		rawurl string
		u      *url.URL
		err    error
	)

	u, err = url.ParseRequestURI(req.RequestURI)
	if err != nil {
		return "", err
	}

	u.Host = req.Header.Get(HTTP_HOST)
	if u.Host == "" {
		u.Host = req.URL.Host
	}
	if u.Host == "" {
		u.Host = req.Host
	}
	if u.Host == "" {
		return "", errors.New("undetectable host")
	}

	u.Scheme = "http"

	if strings.Contains(req.Header.Get(CF_VISITOR), CF_HTTPS) {
		u.Scheme = "https"
	} else if strings.HasPrefix(req.Header.Get(X_FORWARDED_PROTO), X_FORWARDED_HTTPS) {
		u.Scheme = "https"
	}

	rawurl = h.prerenderServiceURL
	if !strings.HasSuffix(rawurl, "/") {
		rawurl += "/"
	}
	rawurl += url.QueryEscape(u.String())

	return rawurl, nil
}

func (h *handler) logf(format string, args ...interface{}) {
	if h.log != nil {
		h.log.Printf(format, args...)
	}
}
