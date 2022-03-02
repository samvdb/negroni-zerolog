package negronizerolog

import (
	"fmt"
	"net/http"
	"github.com/urfave/negroni"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog"
	"net/url"
	"time"
)

type timer interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

type realClock struct{}

func (rc *realClock) Now() time.Time {
	return time.Now()
}

func (rc *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Middleware is a middleware handler that logs the request as it goes in and the response as it goes out.
type Middleware struct {
	// Logger is the log.Logger instance used to log messages with the Logger middleware
	Logger zerolog.Logger
	// Name is the name of the application as recorded in latency metrics
	Name   string
	Before func(zerolog.Logger, *http.Request, string) zerolog.Logger
	After  func(zerolog.Logger, negroni.ResponseWriter, time.Duration, string) zerolog.Logger

	logStarting  bool
	logCompleted bool

	clock timer

	// Exclude URLs from logging
	excludeURLs []string
}

// NewMiddleware returns a new *Middleware, yay!
func NewMiddleware() *Middleware {
	return NewCustomMiddleware(zerolog.InfoLevel, "web")
}

// NewCustomMiddleware builds a *Middleware with the given level
func NewCustomMiddleware(level zerolog.Level,  name string) *Middleware {
	subLogger := log.Logger
	subLogger = subLogger.Level(level)

	return NewMiddlewareFromLogger(subLogger, name)
}

// NewMiddlewareFromLogger returns a new *Middleware which writes to a given zerolog logger.
func NewMiddlewareFromLogger(logger zerolog.Logger, name string) *Middleware {

	subLogger := logger.With().Str("component", "negroni").Logger()
	return &Middleware{
		Logger: subLogger,
		Name:   name,
		Before: DefaultBefore,
		After:  DefaultAfter,

		logStarting:  true,
		logCompleted: true,
		clock:        &realClock{},
	}
}

// SetLogStarting accepts a bool to control the logging of "started handling
// request" prior to passing to the next middleware
func (m *Middleware) SetLogStarting(v bool) {
	m.logStarting = v
}

// SetLogCompleted accepts a bool to control the logging of "completed handling
// request" after returning from the next middleware
func (m *Middleware) SetLogCompleted(v bool) {
	m.logCompleted = v
}

// ExcludeURL adds a new URL u to be ignored during logging. The URL u is parsed, hence the returned error
func (m *Middleware) ExcludeURL(u string) error {
	if _, err := url.Parse(u); err != nil {
		return err
	}
	m.excludeURLs = append(m.excludeURLs, u)
	return nil
}

// ExcludedURLs returns the list of excluded URLs for this middleware
func (m *Middleware) ExcludedURLs() []string {
	return m.excludeURLs
}

func (m *Middleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if m.Before == nil {
		m.Before = DefaultBefore
	}

	if m.After == nil {
		m.After = DefaultAfter
	}

	for _, u := range m.excludeURLs {
		if r.URL.Path == u {
			next(rw, r)
			return
		}
	}

	start := m.clock.Now()

	// Try to get the real IP
	remoteAddr := r.RemoteAddr
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		remoteAddr = realIP
	}

	sLog := m.Logger.With().Logger()

	if reqID := r.Header.Get("X-Request-Id"); reqID != "" {
		sLog = sLog.With().Str("request_id", reqID).Logger()
	}

	sLog = m.Before(sLog, r, remoteAddr)

	if m.logStarting {
		sLog.Info().Msg("started handling request")
	}

	next(rw, r)

	latency := m.clock.Since(start)
	res := rw.(negroni.ResponseWriter)

	if m.logCompleted {
		m.After(sLog, res, latency, m.Name).Info().Msg("completed handling request")
	}
}

// BeforeFunc is the func type used to modify or replace the zerolog.Logger prior
// to calling the next func in the middleware chain
type BeforeFunc func(zerolog.Logger, *http.Request, string) zerolog.Logger

// AfterFunc is the func type used to modify or replace the zerolog.Logger after
// calling the next func in the middleware chain
type AfterFunc func(zerolog.Logger, negroni.ResponseWriter, time.Duration, string) zerolog.Logger

// DefaultBefore is the default func assigned to *Middleware.Before
func DefaultBefore(entry zerolog.Logger, req *http.Request, remoteAddr string) zerolog.Logger {
	return entry.With().
		Str("request", req.RequestURI).
		Str("method", req.Method).
		Str("remote", remoteAddr).Logger()
}

// DefaultAfter is the default func assigned to *Middleware.After
func DefaultAfter(entry zerolog.Logger, res negroni.ResponseWriter, latency time.Duration, name string) zerolog.Logger {

	return entry.With().
		Int("status", res.Status()).
		Str("text_status", http.StatusText(res.Status())).
		Dur("took", latency).
		Int64(fmt.Sprintf("measure#%s.latency", name), latency.Nanoseconds()).Logger()
}