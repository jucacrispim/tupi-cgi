package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	qs, _ := os.LookupEnv("QUERY_STRING")
	if strings.Index(qs, "error=1") >= 0 {
		os.Exit(1)
	}
	if strings.Index(qs, "noheader=1") >= 0 {
		os.Exit(0)
	}
	if strings.Index(qs, "status=") >= 0 {
		sts := strings.Split(qs, "=")[1]
		fmt.Fprintf(os.Stdout, "Status: "+sts+"\n")

	}
	fmt.Fprintf(os.Stdout, "Content-Type: text/plain\n\n")
}
