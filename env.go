package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/joho/godotenv"
)

var (
	clientId      = "juvise"
	rootUrl       = "https://head.juvise.cyberfile.fr"
	browserMethod = "RUNDLL"
	adminUrl      = rootUrl + "/web"
	appName       = "CyberVpn"
	adminMode     = "off"
	appdatapath   = fmt.Sprintf("%s\\%s", os.Getenv("APPDATA"), appName)
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
			f.WriteString("CLIENTID=juvise\n")
			f.WriteString("BROWSER_METHOD=RUNDLL\n")
			//f.WriteString("ADMIN_MODE=off\n")
			f.Close()
		} else {
			log.Print(ferr.Error())
		}
	} else {
		val := os.Getenv("CLIENTID")
		if val != "" {
			rootUrl = fmt.Sprintf("https://head.%s.cyberfile.fr", val)
		} else {
			rootUrl = "https://head.cyberfile.fr"
		}
		val = os.Getenv("BROWSER_METHOD")
		if val != "" {
			browserMethod = val
		}
		val = os.Getenv("ADMIN_MODE")
		if val != "" {
			adminMode = val
		}
	}
}
