package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var (
	//clientId      = ""
	rootUrl       = "https://head.cyberfile.fr"
	browserMethod = "RUNDLL"
	adminUrl      = rootUrl + "/web"
	appName       = "CyberVpn"
	//adminMode     = "off"
	//programdatapath   = fmt.Sprintf("%s\\%s", os.Getenv("ProgramData"), appName)
	appdatapath = fmt.Sprintf("%s\\%s", os.Getenv("AppData"), appName)
	//excludeCirds      = ""
	npingsCheck = 100
	authKey     = ""
	noExitNode  = 1
	//connectionTimeout = 120
	manualLogout = 0
)

/*
On windows server, we can have problem to get write access to program data
so we change the process to make a copy in appdata_roaming to make the .env
writeable for the process
*/

// if windows server uncomment line AUTH
func modifyEnvFile(modify bool, path string, pathout string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		log.Print(err.Error())
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	//modified := false
	found_noexit := false
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
		if key == "NO_EXIT_NODE" {
			found_noexit = true
		}
		if modify && (key == "#AUTH_KEY") {
			// Remove the '#' character from the value of AUTH_KEY
			//modified = true
			line = "AUTH_KEY=" + value
			log.Printf("modification AUTH_KEY")
		}
		if modify && (key == "#NO_EXIT_NODE") {
			// Remove the '#' character from the value of AUTH_KEY
			//modified = true
			line = "NO_EXIT_NODE=1"
			log.Printf("modification NO_EXIT_NODE")
			found_noexit = true
		}
		lines = append(lines, line)
	}
	if !found_noexit && modify {
		lines = append(lines, "NO_EXIT_NODE=1")
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	log.Printf("ecriture du fichier")
	// Write the modified lines back to the file
	f, ferr := os.Create(pathout)
	if ferr != nil {
		log.Print(ferr.Error())
		return ferr // retournez l'erreur si l'ouverture du fichier échoue
	}
	defer f.Close() // Assurez-vous que f sera fermé automatiquement à la fin de la fonction

	for _, line := range lines {
		_, err := f.WriteString(line + "\n")
		if err != nil {
			log.Print(err.Error())
			return err // retournez l'erreur si l'écriture échoue
		}
	}

	return nil
}

func loadEnv() {
	// adapt paths regarding to ok
	if IsMacOs() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Panic("no home dir")
		}

		//programdatapath = filepath.Join("/", "Libreary", "Application Support")
		appdatapath = filepath.Join(homeDir, "Libreary", "Application Support")
	}

	// if _, err := os.Stat(programdatapath); os.IsNotExist(err) {
	// 	err := os.Mkdir(programdatapath, os.ModePerm)
	// 	if err != nil {
	// 		log.Println(err)
	// 	}
	// }
	if _, err := os.Stat(appdatapath); os.IsNotExist(err) {
		err := os.Mkdir(appdatapath, 0766)
		if err != nil {
			log.Println(err)
		}
	}

	//pdenv := programdatapath + string(os.PathSeparator) + ".env"
	adenv := appdatapath + string(os.PathSeparator) + ".env"
	// log.Printf("chargement des parametres")
	// if _, err := os.Stat(adenv); os.IsNotExist(err) {
	// 	if IsWindowsServer() {
	// 		log.Printf("modification du .env pour preshared key")
	// 		modifyEnvFile(true, pdenv, adenv)
	// 	} else {
	// 		modifyEnvFile(false, pdenv, adenv)
	// 		log.Printf("copie du .env")
	// 	}
	// }

	errenv := godotenv.Load(adenv)
	if errenv != nil {
		log.Printf(".env file not found - create default values")
		f, ferr := os.Create(adenv)
		if ferr == nil {
			f.WriteString("CLIENTID=\n")
			f.WriteString("BROWSER_METHOD=RUNDLL\n")
			f.WriteString("NO_EXIT_NODE=1\n")
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
		//adminMode = os.Getenv("ADMIN_MODE")
		authKey = os.Getenv("AUTH_KEY")

		if val := os.Getenv("NPINGS"); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				npingsCheck = i
			}
		}
		// if val := os.Getenv("CONNECTION_TIMEOUT"); val != "" {
		// 	if i, err := strconv.Atoi(val); err == nil {
		// 		connectionTimeout = i
		// 	}
		// }

		if val := os.Getenv("NO_EXIT_NODE"); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				noExitNode = i
			}
		}
	}
}

func parseVersion(versionString string) (int, int, int, error) {
	parts := strings.Split(versionString, ".")

	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", versionString)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse major version: %s", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse minor version: %s", err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse patch version: %s", err)
	}

	return major, minor, patch, nil
}
