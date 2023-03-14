package main

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestEnvServer(t *testing.T) {
	pathOut := os.TempDir() + "\\.env"
	modifyEnvFile(true, ".env", pathOut)
	errenv := godotenv.Load(pathOut)
	if errenv != nil {
		t.Fatalf(errenv.Error())
	} else {
		if os.Getenv("AUTH_KEY") == "" {
			t.Fatalf("AUTH_KEY not defined")
		}
		if os.Getenv("NO_EXIT_NODE") == "" {
			t.Fatalf("NO_EXIT_NODE not defined")
		}
		if os.Getenv("NO_EXIT_NODE") == "0" {
			t.Fatalf("NO_EXIT_NODE bad value for server")
		}
	}
}
