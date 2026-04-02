package main

import (
	"os"

	"github.com/Hermit-Avon/runpost/internal/app"
)

func main() {
	os.Exit(app.Main(os.Args[1:], os.Stderr))
}
