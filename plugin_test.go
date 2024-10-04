// Copyright 2024 Juca Crispim <juca@poraodojuca.net>

// This file is part of tupi-cgi.

// tupi-cgi is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// tupi-cgi is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with tupi-cgi. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

type ErrBody int

func (ErrBody) Read(p []byte) (int, error) {
	return 0, errors.New("some error")
}

func TestInit_BadConfs(t *testing.T) {
	var tests = []struct {
		name string
		conf map[string]any
		err  error
	}{
		{
			"missing config",
			nil,
			MissingConfigError},
		{
			"missing cgi dir",
			map[string]any{},
			NoCgiDirError},
		{
			"bad cgi dir",
			map[string]any{"CGI_DIR": 1},
			BadCgiDirError},
		{
			"cgi dir does not exist",
			map[string]any{"CGI_DIR": "./dont-exist"},
			os.ErrNotExist},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Init("some.domain", &test.conf)
			if !errors.Is(err, test.err) {
				t.Fatal(err, test.err)
			}
		})
	}

}

func TestInit(t *testing.T) {
	var tests = []struct {
		conf map[string]any
	}{
		{map[string]any{"CGI_DIR": "./build"}},
	}

	for _, test := range tests {
		err := Init("some.domain", &test.conf)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetMetaVars(t *testing.T) {

	var testCases = []struct {
		name     string
		r        *http.Request
		expected map[string]string
		err      error
	}{
		{
			"simple",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/something", nil)
				r.URL.Scheme = "http"
				r.Header.Add("Server-Software", "tupi")
				return r
			}(),
			map[string]string{
				"QUERY_STRING":      "",
				"REMOTE_ADDR":       "",
				"REQUEST_METHOD":    "GET",
				"SERVER_NAME":       "",
				"SERVER_PORT":       "80",
				"SCRIPT_NAME":       "./build/something",
				"PATH_INFO":         "",
				"PATH_TRANSLATED":   "",
				"CONTENT_LENGTH":    "0",
				"GATEWAY_INTERFACE": "CGI/1.1",
				"SERVER_PROTOCOL":   "HTTP/1.1",
				"SERVER_SOFTWARE":   "tupi",
			},
			nil,
		},
		{
			"script does not exist",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/bad.cgi", nil)
				r.URL.Scheme = "http"
				r.Header.Add("Server-Software", "tupi")
				return r
			}(),
			map[string]string{
				"QUERY_STRING":      "",
				"REMOTE_ADDR":       "",
				"REQUEST_METHOD":    "GET",
				"SERVER_NAME":       "",
				"SERVER_PORT":       "80",
				"SCRIPT_NAME":       "",
				"PATH_INFO":         "/bad.cgi",
				"PATH_TRANSLATED":   "./build/bad.cgi",
				"CONTENT_LENGTH":    "0",
				"GATEWAY_INTERFACE": "CGI/1.1",
				"SERVER_PROTOCOL":   "HTTP/1.1",
				"SERVER_SOFTWARE":   "tupi",
			},
			nil,
		},
		{
			"with path info",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/something/the/path", nil)
				r.TLS = &tls.ConnectionState{}
				r.Header.Add("Server-Software", "tupi")
				return r
			}(),
			map[string]string{
				"QUERY_STRING":      "",
				"REMOTE_ADDR":       "",
				"REQUEST_METHOD":    "GET",
				"SERVER_NAME":       "",
				"SERVER_PORT":       "443",
				"SCRIPT_NAME":       "./build/something",
				"PATH_INFO":         "/the/path",
				"PATH_TRANSLATED":   "./build/the/path",
				"CONTENT_LENGTH":    "0",
				"GATEWAY_INTERFACE": "CGI/1.1",
				"SERVER_PROTOCOL":   "HTTP/1.1",
				"SERVER_SOFTWARE":   "tupi",
			},
			nil,
		},
		{
			"with query string",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/something?the=query&other=param", nil)
				r.TLS = &tls.ConnectionState{}
				r.Header.Add("Server-Software", "tupi")
				return r
			}(),
			map[string]string{
				"QUERY_STRING":      "the=query&other=param",
				"REMOTE_ADDR":       "",
				"REQUEST_METHOD":    "GET",
				"SERVER_NAME":       "",
				"SERVER_PORT":       "443",
				"SCRIPT_NAME":       "./build/something",
				"PATH_INFO":         "",
				"PATH_TRANSLATED":   "",
				"CONTENT_LENGTH":    "0",
				"GATEWAY_INTERFACE": "CGI/1.1",
				"SERVER_PROTOCOL":   "HTTP/1.1",
				"SERVER_SOFTWARE":   "tupi",
			},
			nil,
		},
		{
			"custom port",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/something?the=query&other=param", nil)
				r.Host = "localhost:1234"
				r.Header.Add("Server-Software", "tupi")
				return r
			}(),
			map[string]string{
				"QUERY_STRING":      "the=query&other=param",
				"REMOTE_ADDR":       "",
				"REQUEST_METHOD":    "GET",
				"SERVER_NAME":       "localhost",
				"SERVER_PORT":       "1234",
				"SCRIPT_NAME":       "./build/something",
				"PATH_INFO":         "",
				"PATH_TRANSLATED":   "",
				"CONTENT_LENGTH":    "0",
				"GATEWAY_INTERFACE": "CGI/1.1",
				"SERVER_PROTOCOL":   "HTTP/1.1",
				"SERVER_SOFTWARE":   "tupi",
			},
			nil,
		},
	}

	cgiDir := "./build"
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			meta, err := getMetaVars(test.r, cgiDir)
			if err != nil && errors.Is(err, test.err) {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(meta, test.expected) {
				t.Fatalf("Bad meta vars\n %+v\n %+v", meta, test.expected)
			}
		})
	}
}

func TestParseCgiResponse(t *testing.T) {

	var testCases = []struct {
		name            string
		response        []byte
		expectedHeaders map[string]string
		expectedBody    []byte
		err             error
	}{
		{
			"ok response",
			[]byte("Status: 200\nContent-Type: text/plain\n\nthe body"),
			map[string]string{
				"Status":       "200",
				"Content-Type": "text/plain",
			},
			[]byte("the body"),
			nil,
		},
		{
			"bad response",
			[]byte("Status: 200"),
			nil,
			nil,
			InvalidCgiResponse,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			header, body, err := parseCgiResponse(&test.response)
			if err != test.err {
				t.Fatal(err)
			}

			if err != nil {
				return
			}
			h := (*header)

			if !reflect.DeepEqual(h, test.expectedHeaders) {
				t.Fatalf("Ivalid headers\n %+v\n%+v", h, test.expectedHeaders)
			}
			b := (*body)

			if !reflect.DeepEqual(b, test.expectedBody) {
				t.Fatalf("Invalid body %s\n%s", b, test.expectedBody)
			}

		})
	}
}

func TestServe(t *testing.T) {

	type validateFn func(w *httptest.ResponseRecorder)
	var testCases = []struct {
		name     string
		r        *http.Request
		validate validateFn
	}{
		{
			"get request",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/something", nil)
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusOK {
					t.Fatalf("Invalid status code %d", w.Code)
				}
				b := string(w.Body.Bytes())
				if b != "method was: GET" {
					t.Fatalf("Invalid body %s", b)
				}
			},
		},
		{
			"get request with query string",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/something?query=string&a=1", nil)
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusOK {
					t.Fatalf("Invalid status code %d", w.Code)
				}
				b := string(w.Body.Bytes())
				if b != "method was: GET\nquery string: query=string&a=1" {
					t.Fatalf("Invalid body %s", b)
				}
			},
		},
		{
			"post request",
			func() *http.Request {
				r, _ := http.NewRequest("POST", "/something",
					bytes.NewBuffer([]byte("the post body")))
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusOK {
					t.Fatalf("Invalid status code %d", w.Code)
				}
				b := string(w.Body.Bytes())
				if b != "the post body" {
					t.Fatalf("Invalid body %s", b)
				}
			},
		},
		{
			"put request",
			func() *http.Request {
				r, _ := http.NewRequest("PUT", "/something",
					bytes.NewBuffer([]byte("the post body")))
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusMethodNotAllowed {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
		{
			"bad body",
			func() *http.Request {
				r, _ := http.NewRequest("POST", "/something", ErrBody(0))
				r.URL.Scheme = "http"
				r.ContentLength = 100
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusBadRequest {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
		{
			"cgi error",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/otherthing?error=1", nil)
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusInternalServerError {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
		{
			"cgi response without headers",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/otherthing?noheader=1", nil)
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusInternalServerError {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
		{
			"cgi response without status",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/otherthing", nil)
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusInternalServerError {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
		{
			"cgi response with bad status",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/otherthing?status=bla", nil)
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusInternalServerError {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
		{
			"cgi script not found",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/missing", nil)
				r.URL.Scheme = "http"
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusNotFound {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
		{
			"bad port",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "/something?the=query&other=param", nil)
				r.Host = "localhost:ss"
				r.Header.Add("Server-Software", "tupi")
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusInternalServerError {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
		{
			"containing dotdot",
			func() *http.Request {
				r, _ := http.NewRequest("GET", "../../../../../bin/ls", nil)
				r.Header.Add("Server-Software", "tupi")
				return r
			}(),
			func(w *httptest.ResponseRecorder) {
				if w.Code != http.StatusNotFound {
					t.Fatalf("Invalid status code %d", w.Code)
				}
			},
		},
	}

	conf := map[string]any{"CGI_DIR": "./build"}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Serve(w, test.r, &conf)
			test.validate(w)
		})
	}
}
