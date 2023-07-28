package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var (
	versionurl = "https://cybervpnqtgklfnh-get-runtime-versions.functions.fnc.fr-par.scw.cloud"
	dwnBaseUrl = "https://dwn.s3-website.fr-par.scw.cloud/"
)

// Struct to hold the JSON response
type VersionResponse struct {
	Version string `json:"version"`
}

func fetchContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Vérifier le code de statut de la réponse HTTP
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch content, status code: %d", resp.StatusCode)
	}

	var versionResp VersionResponse
	err = json.NewDecoder(resp.Body).Decode(&versionResp)
	if err != nil {
		return "", err
	}

	return versionResp.Version, nil
}

func checkVersion(maj int, min int, patch int) (string, bool, error) {
	remote, err := fetchContent(versionurl)
	if err != nil {
		return "", false, err
	}
	rmaj, rmin, rpatch, err := parseVersion(remote)
	if maj < rmaj {
		return remote, true, nil
	}
	if min < rmin {
		return remote, true, nil
	}
	if patch < rpatch {
		return remote, true, nil
	}
	return remote, false, nil
}

func checkAndDownload() (string, string, error) {
	m, mi, p, err := parseVersion(myVersion)
	if err != nil {
		return "bad version", "", err
	}
	remote, update, err := checkVersion(m, mi, p)
	if err != nil {
		return "remote check failed", "", err
	}
	if !update {
		return "up to date", "", nil
	}
	name := fmt.Sprintf("cybervpn.%s.msi", remote)
	targetFilePath := filepath.Join(appdatapath, name)
	if _, err := os.Stat(targetFilePath); err == nil {
		return "already downloaded", targetFilePath, nil
	} else if os.IsNotExist(err) {
		status, err := downloadVersion(remote)
		return status, targetFilePath, err
	} else {
		return "existence verification error", "", err
	}
}

func downloadVersion(version string) (status string, err error) {

	name := fmt.Sprintf("cybervpn.%s.msi", version)
	url := fmt.Sprintf("%s/%s", dwnBaseUrl, name)
	err = downloadBinaryFromURL(url, name)
	if err != nil {
		return "download failed", err
	} else {
		return "download succceed", nil
	}
}

func downloadBinaryFromURL(url string, name string) error {
	// Get the user's home directory
	targetFilePath := filepath.Join(appdatapath, name)
	// Create the target file
	file, err := os.Create(targetFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Perform the HTTP GET request to download the binary
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the status code of the HTTP response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary, status code: %d", resp.StatusCode)
	}

	// Copy the contents of the response body to the target file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Binary downloaded and saved to:", targetFilePath)
	return nil
}
