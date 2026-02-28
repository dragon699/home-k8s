package main

import (
	"os"

	"notifications-controller/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
