package main

import (
	"fmt"
	"os"
)

func main() {
	method, _ := os.LookupEnv("REQUEST_METHOD")
	qs, _ := os.LookupEnv("QUERY_STRING")
	b := "method was: " + method
	if qs != "" {
		b += "\n" + "query string: " + qs
	}

	fmt.Fprintf(os.Stdout, "Status: 200\nContent-Type: text/plain\n\n"+b)
	os.Exit(0)
}
