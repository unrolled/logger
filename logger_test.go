package logger

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	myHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bar"))
	})
	myHandlerWithError = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	})
)

func TestNoConfig(t *testing.T) {
	l := New()

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/should/be/stdout/", nil)
	req.RemoteAddr = "111.222.333.444"
	l.Handler(myHandler).ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), "bar")
}

func TestDefaultConfig(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out: buf,
	})

	res := httptest.NewRecorder()
	url := "/foo/wow?q=search-term&print=1#comments"
	req, _ := http.NewRequest("GET", url, nil)
	req.RequestURI = url
	l.Handler(myHandler).ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), "bar")

	expectContainsTrue(t, buf.String(), fmt.Sprintf("%d", http.StatusOK))
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsTrue(t, buf.String(), url)

	// LstdFlags output.
	curDate := time.Now().Format("2006/01/02 15:04")
	expectContainsTrue(t, buf.String(), curDate)
}

func TestDefaultConfigPostError(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out: buf,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/foo", nil)
	l.Handler(myHandlerWithError).ServeHTTP(res, req)

	expect(t, res.Code, http.StatusBadGateway)
	expect(t, strings.TrimSpace(res.Body.String()), strings.TrimSpace(http.StatusText(http.StatusBadGateway)))

	expectContainsTrue(t, buf.String(), fmt.Sprintf("%d", http.StatusBadGateway))
	expectContainsTrue(t, buf.String(), "POST")

	// LstdFlags output.
	curDate := time.Now().Format("2006/01/02 15:04")
	expectContainsTrue(t, buf.String(), curDate)
}

func TestResponseSize(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out: buf,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	l.Handler(myHandler).ServeHTTP(res, req)

	// Result of myHandler should be three bytes.
	expectContainsTrue(t, buf.String(), " 3 ")
}

func TestCustomPrefix(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Prefix: "testapp_-_yo",
		Out:    buf,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	l.Handler(myHandler).ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), "bar")

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsTrue(t, buf.String(), "[testapp_-_yo] ")
}

func TestCustomPrefixWithNoBrackets(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Prefix:              "testapp_-_yo2()",
		DisableAutoBrackets: true,
		Out:                 buf,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	l.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsTrue(t, buf.String(), "testapp_-_yo2()")
	expectContainsFalse(t, buf.String(), "[testapp_-_yo2()] ")
}

func TestCustomFlags(t *testing.T) {
	buf := bytes.NewBufferString("")

	r := New(Options{
		OutputFlags: log.Lshortfile,
		Out:         buf,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	r.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")

	// Log should start with...
	expect(t, buf.String()[0:10], "logger.go:")

	// Should not include a date now.
	curDate := time.Now().Format("2006/01/02")
	expectContainsFalse(t, buf.String(), curDate)
}

func TestCustomFlagsZero(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		OutputFlags: -1,
		Out:         buf,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	l.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")

	// Should not include a date now.
	curDate := time.Now().Format("2006/01/02")
	expectContainsFalse(t, buf.String(), curDate)
}

func TestDefaultRemoteAddress(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out: buf,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.RemoteAddr = "8.8.4.4"
	l.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsTrue(t, buf.String(), req.RemoteAddr)
}

func TestDefaultRemoteAddressWithXForwardFor(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out:                  buf,
		RemoteAddressHeaders: []string{"X-Forwarded-Proto"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.RemoteAddr = "8.8.4.4"
	req.Header.Add("X-Forwarded-Proto", "12.34.56.78")
	l.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsTrue(t, buf.String(), "12.34.56.78")
	expectContainsFalse(t, buf.String(), req.RemoteAddr)
}

func TestDefaultRemoteAddressWithXForwardForFallback(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out:                  buf,
		RemoteAddressHeaders: []string{"X-Forwarded-Proto"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.RemoteAddr = "8.8.4.4"
	l.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsTrue(t, buf.String(), req.RemoteAddr)
}

func TestDefaultRemoteAddressMultiples(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out:                  buf,
		RemoteAddressHeaders: []string{"X-Real-IP", "X-Forwarded-Proto"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.RemoteAddr = "8.8.4.4"
	req.Header.Add("X-Forwarded-Proto", "12.34.56.78")
	req.Header.Add("X-Real-IP", "98.76.54.32")
	l.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsTrue(t, buf.String(), "98.76.54.32")
	expectContainsFalse(t, buf.String(), "12.34.56.78")
	expectContainsFalse(t, buf.String(), req.RemoteAddr)
}

func TestDefaultRemoteAddressMultiplesFallback(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out:                  buf,
		RemoteAddressHeaders: []string{"X-Real-IP", "X-Forwarded-Proto"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.RemoteAddr = "8.8.4.4"
	req.Header.Add("X-Forwarded-Proto", "12.34.56.78")
	l.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsFalse(t, buf.String(), "98.76.54.32")
	expectContainsTrue(t, buf.String(), "12.34.56.78")
	expectContainsFalse(t, buf.String(), req.RemoteAddr)
}

func TestIgnoreMultipleConfigs(t *testing.T) {
	buf := bytes.NewBufferString("")

	opt1 := Options{Out: buf}
	opt2 := Options{Out: os.Stderr, OutputFlags: -1}

	l := New(opt1, opt2)

	res := httptest.NewRecorder()
	url := "/should/output/to/buf/only/"
	req, _ := http.NewRequest("GET", url, nil)
	req.RequestURI = url
	l.Handler(myHandler).ServeHTTP(res, req)

	expect(t, res.Code, http.StatusOK)
	expect(t, res.Body.String(), "bar")

	expectContainsTrue(t, buf.String(), fmt.Sprintf("%d", http.StatusOK))
	expectContainsTrue(t, buf.String(), "GET")
	expectContainsTrue(t, buf.String(), url)

	// LstdFlags output.
	curDate := time.Now().Format("2006/01/02 15:04")
	expectContainsTrue(t, buf.String(), curDate)
}

func TestIgnoredURIsNoMatch(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out:                buf,
		IgnoredRequestURIs: []string{"/favicon.ico"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	l.Handler(myHandler).ServeHTTP(res, req)

	expectContainsTrue(t, buf.String(), "200")
	expectContainsTrue(t, buf.String(), "GET")
}

func TestIgnoredURIsMatchig(t *testing.T) {
	buf := bytes.NewBufferString("")

	l := New(Options{
		Out:                buf,
		IgnoredRequestURIs: []string{"/favicon.ico", "/foo"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/foo", nil)
	req.RequestURI = "/foo"
	l.Handler(myHandler).ServeHTTP(res, req)

	expect(t, buf.String(), "")
}

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected [%v] (type %v) - Got [%v] (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func expectContainsTrue(t *testing.T, a, b string) {
	if !strings.Contains(a, b) {
		t.Errorf("Expected [%s] to contain [%s]", a, b)
	}
}

func expectContainsFalse(t *testing.T, a, b string) {
	if strings.Contains(a, b) {
		t.Errorf("Expected [%s] to contain [%s]", a, b)
	}
}
