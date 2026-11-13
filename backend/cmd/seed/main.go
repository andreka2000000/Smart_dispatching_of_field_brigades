package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Explicit fixture import through the same validated API used by organizers.
// Re-running this command deliberately replaces the active reference and day.
func main() {
	base := os.Getenv("API_URL")
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	client := &http.Client{Timeout: 65 * time.Second}
	for _, f := range []struct{ file, path string }{{"reference.json", "/api/v1/reference"}, {"matrix.json", "/api/v1/reference/travel-matrix"}, {"day.json", "/api/v1/days/2026-10-26"}} {
		b, e := os.ReadFile("testdata/" + f.file)
		if e != nil {
			panic(e)
		}
		req, e := http.NewRequest(http.MethodPut, base+f.path, bytes.NewReader(b))
		if e != nil {
			panic(e)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, e := client.Do(req)
		if e != nil {
			panic(e)
		}
		body, e := io.ReadAll(resp.Body)
		resp.Body.Close()
		if e != nil {
			panic(e)
		}
		fmt.Printf("%s %d %s\n", f.file, resp.StatusCode, body)
		if resp.StatusCode != 200 {
			os.Exit(1)
		}
	}
}
