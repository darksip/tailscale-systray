package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var (
	clientId          = ""
	rootUrl           = "https://head.cyberfile.fr"
	browserMethod     = "RUNDLL"
	adminUrl          = rootUrl + "/web"
	appName           = "CyberVpn"
	adminMode         = "off"
	appdatapath       = fmt.Sprintf("%s\\%s", os.Getenv("ProgramData"), appName)
	excludeCirds      = ""
	npingsCheck       = 100
	authKey           = ""
	noExitNode        = 0
	connectionTimeout = 120
	manualLogout      = 0
)

// if windows server uncomment line AUTH

func modifyEnvFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	modified := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			lines = append(lines, line)
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			lines = append(lines, line)
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "#AUTH_KEY" {
			// Remove the '#' character from the value of AUTH_KEY
			modified = true
			line = "AUTH_KEY=" + value
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if !modified {
		return nil
	}

	// Write the modified lines back to the file
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			writer.Flush()
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	return nil
}

func loadEnv() {
	if _, err := os.Stat(appdatapath); os.IsNotExist(err) {
		err := os.Mkdir(appdatapath, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
	if IsWindowsServer() {
		modifyEnvFile(path.Join(appdatapath, ".env"))
	}
	errenv := godotenv.Load(path.Join(appdatapath, ".env"))
	if errenv != nil {
		log.Printf(".env file not found - create default values")
		f, ferr := os.Create(path.Join(appdatapath, ".env"))
		if ferr == nil {
			f.WriteString("CLIENTID=\n")
			f.WriteString("BROWSER_METHOD=RUNDLL\n")
			//f.WriteString("ADMIN_MODE=off\n")
			f.Close()
		} else {
			log.Print(ferr.Error())
		}
	} else {
		if val := os.Getenv("CLIENTID"); val != "" {
			rootUrl = fmt.Sprintf("https://head.%s.cyberfile.fr", val)
		}
		browserMethod = os.Getenv("BROWSER_METHOD")
		adminMode = os.Getenv("ADMIN_MODE")
		authKey = os.Getenv("AUTH_KEY")

		if val := os.Getenv("NPINGS"); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				npingsCheck = i
			}
		}
		if val := os.Getenv("CONNECTION_TIMEOUT"); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				connectionTimeout = i
			}
		}

		if val := os.Getenv("NO_EXIT_NODE"); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				noExitNode = i
			}
		}
	}
}
