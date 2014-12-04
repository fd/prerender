package main

import (
	"net/http"

	"github.com/fd/prerender"
)

// Run:
//   go run main.go
//   curl -v -H 'Host: [HOST]' http://localhost:3000/[PATH]\?_escaped_fragment_

func main() {
	http.ListenAndServe(":3000", prerender.Handler(nil))
}
