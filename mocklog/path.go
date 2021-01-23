package main

import (
	"github.com/snowlyg/LogSync/utils"
	"path/filepath"
	"time"
)

const root = "/Users/snowlyg/ftp/admin/log"

type device struct {
	Code     string
	Type     string
	FileName string
}

func GetDevices() []device {
	return []device{
		{"A4580F48337E", "bis", "fault.log"},
		{"A4580F48337F", "bis", "fault.log"},
		{"4CEDFB5F7187", "nis", "fault.txt"},
		{"4CEDFB698175", "nis", "fault.txt"},
	}
}

func GetPath(device device) string {
	location, _ := utils.GetLocation()
	return filepath.Join(root, device.Type, device.Code, time.Now().In(location).Format(utils.DateLayout))
}
