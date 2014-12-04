prerender
=========

[![GoDoc](https://godoc.org/github.com/fd/prerender?status.svg)](https://godoc.org/github.com/fd/prerender)

```go
func main() {
  http.ListenAndServe(":3000", prerender.Handler(app))
}
```
