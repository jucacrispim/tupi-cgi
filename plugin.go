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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var INTERNAL_SERVER_ERROR_MSG = "Internal server error"

var MissingConfigError = errors.New("[tupi-cgi] No config")
var NoCgiDirError = errors.New("[tupi-cgi] CGI_DIR missing from config")
var BadCgiDirError = errors.New("[tupi-cgi] CGI_DIR wrong config value")
var UnknownSchemeError = errors.New("[tupi-cgi] Unknown scheme")
var ConfusionError = errors.New("[tupi-cgi] Im'm confused")
var InvalidCgiResponse = errors.New("[tupi-cgi] Invalid cgi response")

func Init(domain string, conf *map[string]any) error {
	c := (*conf)
	if c == nil {
		return MissingConfigError
	}

	d, exists := c["CGI_DIR"]
	if !exists {
		return NoCgiDirError
	}

	cgiDir, ok := d.(string)
	if !ok {
		return BadCgiDirError
	}

	_, err := os.Stat(cgiDir)
	return err

}

func Serve(w http.ResponseWriter, r *http.Request, conf *map[string]any) {
	c := (*conf)
	d, _ := c["CGI_DIR"]
	cgiDir, _ := d.(string)

	m, err := getMetaVars(r, cgiDir)
	if err != nil {
		log.Printf(err.Error())
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, 500)
		return
	}
	if m["SCRIPT_NAME"] == "" {
		http.Error(w, "NOT FOUND", http.StatusNotFound)
		return
	}
	var rawBody []byte = nil
	if r.ContentLength > 0 && r.Body != nil {
		defer r.Body.Close()
		rawBody, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad request", 400)
			return
		}
	}
	output, err := execCmd(&m, &rawBody)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}
	var headers *map[string]string
	var body *[]byte
	headers, body, err = parseCgiResponse(output)
	if headers == nil {
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}
	h := (*headers)
	sts, exits := h["Status"]
	if !exits {
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
		return
	}
	stsInt, err := strconv.Atoi(sts)
	if err != nil {
		http.Error(w, INTERNAL_SERVER_ERROR_MSG, http.StatusInternalServerError)
	}

	for k, v := range *headers {
		w.Header().Add(k, v)
	}
	w.WriteHeader(stsInt)
	w.Write([]byte(*body))
}

func isNewLine(s string) bool {
	if s == "\n" || s == "\n\r" || s == "\r" || s == "\r\n" || s == "" {
		return true
	}
	return false
}

func parseCgiResponse(response *[]byte) (*map[string]string, *[]byte, error) {
	headers := make(map[string]string, 0)
	body := make([]byte, 0)
	delim := byte('\n')
	previousDelim := 0
	for i, b := range *response {
		if b == delim {
			line := string((*response)[previousDelim:i])
			if isNewLine(line) {
				body = (*response)[i+1:]
				return &headers, &body, nil
			}
			previousDelim = i + 1
			line = strings.Trim(line, "\n")
			parts := strings.Split(line, ":")
			headers[strings.Trim(parts[0], " ")] = strings.Trim(parts[1], " ")

		}
	}
	return nil, nil, InvalidCgiResponse
}

func execCmd(m *map[string]string, rawBody *[]byte) (*[]byte, error) {
	meta := (*m)
	envVars := make([]string, 15)
	for k, v := range meta {
		envVar := fmt.Sprintf("%s=%s", k, v)
		envVars = append(envVars, envVar)
	}
	cmdPath := meta["SCRIPT_NAME"]
	cmd := exec.Command(cmdPath)
	cmdEnv := append(cmd.Env, envVars...)
	cmd.Env = cmdEnv
	if rawBody != nil {
		cmd.Stdin = bytes.NewReader(*rawBody)
	}
	o, err := cmd.CombinedOutput()
	return &o, err

}

func getMetaVars(r *http.Request, cgiDir string) (map[string]string, error) {
	headers := []string{
		"Auth-Type",
		"Remote-User",
		"Content-Type",
		"Server-Software",
	}
	meta := make(map[string]string)

	for _, h := range headers {
		rHeader := r.Header.Get(h)
		if rHeader != "" {
			meta[strings.ReplaceAll(strings.ToUpper(h), "-", "_")] = rHeader
		}
	}

	path := r.URL.Path
	scriptPath, pathInfo := findScript(cgiDir, path)
	pathTranslated := ""

	if pathInfo != "" {
		pathTranslated = cgiDir + pathInfo
	}
	query := r.URL.RawQuery

	meta["CONTENT_LENGTH"] = strconv.FormatInt(r.ContentLength, 10)
	meta["GATEWAY_INTERFACE"] = "CGI/1.1"
	meta["PATH_INFO"] = pathInfo
	meta["PATH_TRANSLATED"] = pathTranslated
	meta["SCRIPT_NAME"] = scriptPath
	meta["QUERY_STRING"] = query
	meta["REMOTE_ADDR"] = getIp(r)
	meta["REQUEST_METHOD"] = r.Method
	meta["SERVER_NAME"] = getDomainForRequest(r)
	port, err := getPortForRequest(r)
	if err != nil {
		return nil, err
	}
	meta["SERVER_PORT"] = strconv.Itoa(port)
	meta["SERVER_PROTOCOL"] = r.Proto

	return meta, nil
}

func getDomainForRequest(req *http.Request) string {
	domain := strings.Split(req.Host, ":")[0]
	domain = strings.ToLower(domain)
	return domain
}

func getPortForRequest(r *http.Request) (int, error) {
	hostParts := strings.Split(r.Host, ":")
	partsLen := len(hostParts)
	if partsLen > 2 {
		return 0, ConfusionError
	}
	if len(hostParts) == 2 {
		return strconv.Atoi(hostParts[1])
	}

	sc := r.URL.Scheme
	switch sc {
	case "http":
		return 80, nil

	case "https":
		return 443, nil
	}
	return 0, UnknownSchemeError
}

func getIp(req *http.Request) string {
	return req.RemoteAddr
}

func findScript(cgiDir string, path string) (string, string) {
	pathparts := strings.Split(path, string(os.PathSeparator))
	scriptPath := cgiDir
	pathInfo := ""
	for i, p := range pathparts {
		if p == "" {
			continue
		}
		testPath := scriptPath + string(os.PathSeparator) + p
		_, err := os.Stat(testPath)
		if err == nil {
			scriptPath = testPath
			continue
		}
		pathInfo = "/" + strings.Join(pathparts[i:], string(os.PathSeparator))
		break
	}
	if scriptPath == cgiDir {
		scriptPath = ""
	}
	return scriptPath, pathInfo
}
