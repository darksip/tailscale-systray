package main

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestEnvServer(t *testing.T) {
	pathOut := os.TempDir() + "\\.env"
	modifyEnvFile(true, ".env", pathOut)
	errenv := godotenv.Load(pathOut)
	if errenv != nil {
		t.Fatal(errenv.Error())
	} else {
		if os.Getenv("AUTH_KEY") == "" {
			t.Fatal("AUTH_KEY not defined")
		}
		if os.Getenv("NO_EXIT_NODE") == "" {
			t.Fatal("NO_EXIT_NODE not defined")
		}
		if os.Getenv("NO_EXIT_NODE") == "0" {
			t.Fatal("NO_EXIT_NODE bad value for server")
		}
	}
}

func TestAutoUpdateFetch(t *testing.T) {
	version, err := fetchContent(versionurl)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(version) < 4 {
		t.Fatalf("bad version")
	}
}

func TestCheckVersion(t *testing.T) {
	remote, update, err := checkVersion(1, 20, 0)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !update {
		t.Fatal("update should be true")
	}
	if len(remote) < 4 {
		t.Fatal("remote should be longer")
	}
}

func TestDownloadVersion(t *testing.T) {
	status, err := downloadVersion("1.20.2")
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(status) < 4 {
		t.Fatal("status should be longer")
	}
}

func TestCheckAndDownload(t *testing.T) {
	status, fname, err := checkAndDownload()
	if err != nil {
		t.Fatal(err.Error())
	}
	log.Printf("status : %s", status)
	if len(status) < 4 {
		t.Fatal("status should be longer")
	}
	if len(fname) < 4 {
		t.Fatal("filename should be longer")
	}
}
