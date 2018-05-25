package logger

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// Options is a struct for specifying configuration parameters for the Logger middleware.
type Options struct {
	// Prefix is the outputted keyword in front of the log message. Logger automatically wraps the prefix in square brackets (ie. [myApp] ) unless the `DisableAutoBrackets` is set to true. A blank value will not have brackets added. Default is blank (with no brackets).
	Prefix string
	// DisableAutoBrackets if set to true, will remove the prefix and square brackets. Default is false.
	DisableAutoBrackets bool
	// RemoteAddressHeaders is a list of header keys that Logger will look at to determine the proper remote address. Useful when using a proxy like Nginx: `[]string{"X-Forwarded-Proto"}`. Default is an empty slice, and thus will use `reqeust.RemoteAddr`.
	RemoteAddressHeaders []string
	// Out is the destination to which the logged data will be written too. Default is `os.Stdout`.
	Out io.Writer
	// OutputFlags defines the logging properties. See http://golang.org/pkg/log/#pkg-constants. To disable all flags, set this to `-1`. Defaults to log.LstdFlags (2009/01/23 01:23:23).
	OutputFlags int
	// IgnoredRequestURIs is a list of path values we do not want logged out. Exact match only!
	IgnoredRequestURIs []string
}

// Logger is a HTTP middleware handler that logs a request. Outputted information includes status, method, URL, remote address, size, and the time it took to process the request.
type Logger struct {
	*log.Logger
	opt Options
}

// New returns a new Logger instance.
func New(opts ...Options) *Logger {
	var o Options
	if len(opts) == 0 {
		o = Options{}
	} else {
		o = opts[0]
	}

	// Determine prefix.
	prefix := o.Prefix
	if len(prefix) > 0 && o.DisableAutoBrackets == false {
		prefix = "[" + prefix + "] "
	}

	// Determine output writer.
	var output io.Writer
	if o.Out != nil {
		output = o.Out
	} else {
		// Default is stdout.
		output = os.Stdout
	}

	// Determine output flags.
	flags := log.LstdFlags
	if o.OutputFlags == -1 {
		flags = 0
	} else if o.OutputFlags != 0 {
		flags = o.OutputFlags
	}

	return &Logger{
		Logger: log.New(output, prefix, flags),
		opt:    o,
	}
}

// Handler wraps an HTTP handler and logs the request as necessary.
func (l *Logger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		crw := newCustomResponseWriter(w)
		next.ServeHTTP(crw, r)

		for _, ignoredURI := range l.opt.IgnoredRequestURIs {
			if ignoredURI == r.RequestURI {
				return
			}
		}

		addr := r.RemoteAddr
		for _, headerKey := range l.opt.RemoteAddressHeaders {
			if val := r.Header.Get(headerKey); len(val) > 0 {
				addr = val
				break
			}
		}

		l.Printf("(%s) \"%s %s %s\" %d %d %s", addr, r.Method, r.RequestURI, r.Proto, crw.status, crw.size, time.Since(start))
	})
}

type customResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (c *customResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

func (c *customResponseWriter) Write(b []byte) (int, error) {
	size, err := c.ResponseWriter.Write(b)
	c.size += size
	return size, err
}

func (c *customResponseWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *customResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := c.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not implement the Hijacker interface")
}

func newCustomResponseWriter(w http.ResponseWriter) *customResponseWriter {
	// When WriteHeader is not called, it's safe to assume the status will be 200.
	return &customResponseWriter{
		ResponseWriter: w,
		status:         200,
	}
}
