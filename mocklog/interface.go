package main

import (
	"bytes"
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
		//[2020-12-30 15:17:08]
		//Message: http://10.0.0.23/app/telephone/list, 服务器出错500
		for i := 0; i < 5; i++ {
			b := bytes.NewBufferString("[2020-12-30 15:17:08]" + "\r\n" + "Message: http://10.0.0.23/app/telephone/list, 服务器出错500")
			err := utils.AppendFile(filepath.Join(GetPath(device), device.InterfaceFileName), b.Bytes())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
