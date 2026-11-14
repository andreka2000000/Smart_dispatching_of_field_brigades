// Explicit local bootstrap. Reads credentials from stdin; never resets users.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"magistral/backend/internal/auth"
	"magistral/backend/internal/storage/postgres"
	"os"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, e := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
	if e != nil {
		return e
	}
	defer store.Pool.Close()
	var users []auth.NewUser
	if e = json.NewDecoder(os.Stdin).Decode(&users); e != nil {
		return e
	}
	manager := auth.Manager{Store: store}
	for _, in := range users {
		var exists bool
		if e = store.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM app_users WHERE username=$1)", in.Username).Scan(&exists); e != nil {
			return e
		}
		if exists {
			fmt.Println(in.Username + ": existing account retained")
			continue
		}
		if _, e = manager.Create(ctx, in); e != nil {
			return e
		}
		fmt.Println(in.Username + ": created")
	}
	return nil
}
