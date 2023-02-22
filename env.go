package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	clientId      = ""
	rootUrl       = "https://head.cyberfile.fr"
	browserMethod = "RUNDLL"
	adminUrl      = rootUrl + "/web"
	appName       = "CyberVpn"
	adminMode     = "off"
	appdatapath   = fmt.Sprintf("%s\\%s", os.Getenv("ProgramData"), appName)
	excludeCirds  = ""
	npingsCheck   = 100
)

func loadEnv() {
	if _, err := os.Stat(appdatapath); os.IsNotExist(err) {
		err := os.Mkdir(appdatapath, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
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
		if val := os.Getenv("NPINGS"); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				npingsCheck = i
			}
		}
	}
}
