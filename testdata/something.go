package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	method, _ := os.LookupEnv("REQUEST_METHOD")
	me := strings.ToLower(method)
	if me != "get" && me != "post" {
		fmt.Printf("Status: 405\n\n")
		os.Exit(0)
	}
	qs, _ := os.LookupEnv("QUERY_STRING")
	b := "method was: " + method
	if qs != "" {
		b += "\n" + "query string: " + qs
	}

	if me == "post" {
		body, _ := io.ReadAll(os.Stdin)
		b = string(body)

	}

	fmt.Fprintf(os.Stdout, "Status: 200\nContent-Type: text/plain\n\n"+b)
	os.Exit(0)
}
