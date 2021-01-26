package main

import (
	"encoding/json"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
	"path/filepath"
)

func CreateInterfaceFile(device device, infos []map[string]string) error {
	if device.Type == "bis" {
		for _, info := range infos {
			faultLog := sync.InterfaceLog{
				PostParamJson: "",
				PostParamType: 1,
				Remark:        "",
				RequestType:   1,
				Msg:           info["msg"],
				Url:           info["url"],
				Timestamp:     info["timestamp"],
			}
			b, _ := json.Marshal(faultLog)
			err := utils.AppendFile(filepath.Join(GetPath(device), device.InterfaceFileName), b)
			if err != nil {
				return err
			}
		}

	} else if device.Type == "nis" {
		//bl, _ := strconv.ParseBool(mqtt)
		//faultTxt := sync.FaultTxt{
		//	Mqtt:      bl,
		//	Reason:    reason,
		//	Timestamp: timestamp,
		//}
		//b, _ := json.Marshal(faultTxt)
		//return utils.CreateFile(filepath.Join(GetPath(device), device.FileName), b)
	}
	return nil
}
