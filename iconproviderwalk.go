package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/lxn/walk"
)

type widthAndDllIdx struct {
	width int
	idx   int32
	dll   string
}

var cachedSystemIconsForWidthAndDllIdx = make(map[widthAndDllIdx]*walk.Icon)

//go:embed icon
var iconFS embed.FS

var iconsWalk map[string]*walk.Icon
var iconPath string

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

func addIcon(key string, name string) {
	icon, err := walk.NewIconFromFile(fmt.Sprintf("%s\\%s.ico", iconPath, name))
	if err == nil {
		iconsWalk[key] = icon
	}
}

func initIconsWalk() {

	iconsWalk = map[string]*walk.Icon{}
	// is directory existing ?
	iconPath = fmt.Sprintf("%s\\%s\\%s", os.Getenv("ProgramData"), appName, "icons")
	if _, err := os.Stat(iconPath); err != nil {
		os.Mkdir(iconPath, os.FileMode(0644))
	}
	// extract all icons
	extractIcons(iconPath)
	addIcon("empty", "empty16")
	addIcon("on", "on")
	addIcon("off", "off")
	addIcon("off16", "off")
	addIcon("blueballoon", "bluebaloon")
	addIcon("greyballoon", "greybaloon")
	addIcon("bluearrow", "fleche16")
	addIcon("greyarrow", "fleche16Off")

}

func loadSystemIcon(dll string, index int32, size int) (icon *walk.Icon, err error) {
	icon = cachedSystemIconsForWidthAndDllIdx[widthAndDllIdx{size, index, dll}]
	if icon != nil {
		return
	}
	icon, err = walk.NewIconFromSysDLLWithSize(dll, int(index), size)
	if err == nil {
		cachedSystemIconsForWidthAndDllIdx[widthAndDllIdx{size, index, dll}] = icon
	}
	return
}

func loadShieldIcon(size int) (icon *walk.Icon, err error) {
	icon, err = loadSystemIcon("imageres", 73, size)
	if err != nil {
		icon, err = loadSystemIcon("imageres", 1, size)
	}
	return
}

var cachedLogoIconsForWidth = make(map[int]*walk.Icon)

func loadLogoIcon(size int) (icon *walk.Icon, err error) {
	icon = cachedLogoIconsForWidth[size]
	if icon != nil {
		return
	}
	icon, err = walk.NewIconFromResourceIdWithSize(7, walk.Size{size, size})
	if err == nil {
		cachedLogoIconsForWidth[size] = icon
	}
	return
}
