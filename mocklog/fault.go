package main

import (
	"encoding/json"
	"path/filepath"
	"strconv"

	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
)

func CreateFaultFile(device device, plugin, interf sync.Plugin, timestamp, mqtt, reason string) error {
	if device.Type == "bis" {
		faultLog := sync.FaultLog{
			AppType:      device.Type,
			Call:         plugin,
			Face:         plugin,
			Interf:       interf,
			Iptv:         plugin,
			Mqtt:         plugin,
			IsBackground: true,
			Timestamp:    timestamp,
		}
		b, _ := json.Marshal(faultLog)
		return utils.CreateFile(filepath.Join(GetPath(device), device.FileName), b)
	} else if device.Type == "nis" {
		bl, _ := strconv.ParseBool(mqtt)
		faultTxt := sync.FaultTxt{
			Mqtt:      bl,
			Reason:    reason,
			Timestamp: timestamp,
		}
		b, _ := json.Marshal(faultTxt)
		return utils.CreateFile(filepath.Join(GetPath(device), device.FileName), b)
	}
	return nil
}
