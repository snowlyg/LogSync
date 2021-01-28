package main

import (
	"path/filepath"
	"time"

	"github.com/snowlyg/LogSync/utils"
)

const root = "D:/env/FileZillaServer/share/log"

type device struct {
	Code              string
	Type              string
	FileName          string
	InterfaceFileName string
}

func GetDevices() []device {
	return []device{
		{"A4580F48337E", "bis", "fault.log", "interface.log"},
		{"A4580F48337F", "bis", "fault.log", "interface.log"},
		{"4CEDFB5F7187", "nis", "fault.txt", "error.txt"},
		{"4CEDFB698175", "nis", "fault.txt", "error.txt"},
	}
}

func GetPath(device device) string {
	location, _ := utils.GetLocation()
	return filepath.Join(root, device.Type, device.Code, time.Now().In(location).Format(utils.DateLayout))
}
