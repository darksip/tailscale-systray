package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/lxn/walk"
)

//go:embed icon
var iconFS embed.FS

var iconsWalk map[string]*walk.Icon
var iconPath string

// icons are embeded in exe file but needs to be extracted to
// be used in walk functions
// we extract them in appdata when starting the exe
func extractIcons(iconPath string) {
	embs, err := iconFS.ReadDir("icon")
	if err == nil {
		for _, emb := range embs {
			log.Printf(emb.Name())
			dest := fmt.Sprintf("%s\\%s", iconPath, emb.Name())
			src := fmt.Sprintf("%s/%s", "icon", emb.Name())
			if fcont, err := iconFS.ReadFile(src); err == nil {
				if err := os.WriteFile(dest, fcont, 0666); err != nil {
					log.Printf("error os.WriteFile error: %v", err)
				}
			}
		}
	}
}

func addIconFromFile(key string, name string) {
	icon, err := walk.NewIconFromFile(fmt.Sprintf("%s\\%s.ico", iconPath, name))
	if err == nil {
		iconsWalk[key] = icon
	}
}

func addIconFromDll(key string, index int) {
	icon, err := walk.NewIconFromSysDLLWithSize("imageres", index, 256)
	if err == nil {
		iconsWalk[key] = icon
	}
}

// we can refer to Program data directly as we know walk is on ly on Windows
func initIconsWalk() {

	iconsWalk = map[string]*walk.Icon{}
	// is directory existing ?
	iconPath = fmt.Sprintf("%s\\%s\\%s", os.Getenv("ProgramData"), appName, "icons")
	if _, err := os.Stat(iconPath); err != nil {
		os.Mkdir(iconPath, os.FileMode(0644))
	}
	// extract all icons
	extractIcons(iconPath)
	addIconFromFile("empty", "empty16")
	addIconFromFile("on", "on")
	addIconFromFile("off", "off")
	addIconFromFile("off16", "off")
	addIconFromFile("caution", "caution")
	addIconFromFile("blueballoon", "bluebaloon")
	addIconFromFile("redballoon", "redbaloon")
	addIconFromFile("greyballoon", "greybaloon")
	addIconFromFile("bluearrow", "fleche16")
	addIconFromFile("greyarrow", "fleche16Off")
	addIconFromDll("connected", 28)
	addIconFromDll("disconnected", 26)
	addIconFromDll("error", 100)
	addIconFromDll("attention", 102)
	addIconFromDll("needslogin", 300)
	addIconFromDll("browser", 170)
	addIconFromDll("exitnode", 114)
	addIconFromDll("unknown", 99)
	addIconFromDll("info", 99)
}
