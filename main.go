package main

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/matt/helpmetalktome/cmd"
)

func main() {
	exe, _ := os.Executable()
	envPath := filepath.Join(filepath.Dir(exe), ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		_ = godotenv.Load()
	} else {
		_ = godotenv.Load(envPath)
	}

	cmd.Execute()
}
