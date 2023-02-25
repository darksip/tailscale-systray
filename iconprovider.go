package main

import (
	_ "embed"
)

// TODO : have to make ppng icons for mac
var (
	//go:embed icon/on.png
	iconOnPng []byte
	//go:embed icon/off.png
	iconOffPng []byte
	//go:embed icon/on64.ico
	iconOnIco []byte
	//go:embed icon/off64.ico
	iconOffIco []byte
	//go:embed icon/empty16.ico
	iconEmpty []byte
	//go:embed icon/icoOn16.ico
	iconOn16 []byte
	//go:embed icon/bluebaloon.ico
	iconBlueBaloon []byte
	//go:embed icon/greybaloon.ico
	iconGreyBaloon []byte
	//go:embed icon/redbaloon.ico
	iconRedBaloon []byte
	//go:embed icon/fleche16.ico
	iconBlueArrow []byte
	//go:embed icon/fleche16Off.ico
	iconGreyArrow []byte

	iconOn  []byte
	iconOff []byte

	iconsGL map[string][]byte
)

func initGLIcons() {
	iconsGL = map[string][]byte{}
	iconsGL["empty"] = iconEmpty
	iconsGL["on16"] = iconOn16
	iconsGL["on"] = iconOnIco
	iconsGL["off"] = iconOffIco
	iconsGL["blueballoon"] = iconBlueBaloon
	iconsGL["greyballon"] = iconGreyBaloon
	iconsGL["bluearrow"] = iconBlueArrow
	iconsGL["greyarrow"] = iconGreyArrow

}
