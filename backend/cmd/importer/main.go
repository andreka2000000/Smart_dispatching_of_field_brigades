package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"magistral/backend/internal/dataset"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	dir := flag.String("dir", "/app/data/synthetic", "directory containing the organizer-format files")
	date := flag.String("date", "2026-10-28", "model day")
	now := flag.String("now", "", "model time; defaults to date at 08:30 UTC+5")
	validate := flag.Bool("validate-only", false, "validate all files without contacting the API")
	replace := flag.Bool("replace", false, "explicitly replace the target database's reference and days")
	flag.Parse()
	if *now == "" {
		*now = *date + "T08:30:00+05:00"
	}
	data, e := dataset.Load(*dir, *date, *now)
	if e != nil {
		return e
	}
	fmt.Printf("Validated: %d staff, %d service centers, %d work types, %d day orders, %d matrix points\n", len(data.Reference.Staff), len(data.Reference.ServiceCenters), len(data.Reference.WorkTypes), len(data.Day.Orders), len(data.Matrix.Points))
	if *validate || !*replace {
		fmt.Println("No data changed. Use -replace only against an empty/disposable training database.")
		return nil
	}
	base := strings.TrimRight(os.Getenv("API_URL"), "/")
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	client := &http.Client{Timeout: 65 * time.Second}
	// The target must be empty or contain a compatible matrix. An arbitrary
	// dataset replacement may require a separate DB; it is never bypassed here.
	for _, item := range []struct {
		path  string
		value any
	}{
		{"/api/v1/reference", data.Reference},
		{"/api/v1/reference/travel-matrix", data.Matrix},
		{"/api/v1/days/" + *date, data.Day},
	} {
		raw, e := json.Marshal(item.value)
		if e != nil {
			return e
		}
		req, e := http.NewRequest(http.MethodPut, base+item.path, bytes.NewReader(raw))
		if e != nil {
			return e
		}
		req.Header.Set("Content-Type", "application/json")
		resp, e := client.Do(req)
		if e != nil {
			return e
		}
		body, e := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		resp.Body.Close()
		if e != nil {
			return e
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("%s: HTTP %d: %s", item.path, resp.StatusCode, body)
		}
		fmt.Printf("Loaded %s\n", item.path)
	}
	return nil
}
