package main

import (
	"os"

	"connector-notifications/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
