package models

import (
	"github.com/snowlyg/LogSync/utils"
)

func Init() (err error) {
	err = utils.Init()
	if err != nil {
		return
	}
	utils.SQLite.AutoMigrate(LogMsg{}, ServerMsg{}, CfDevice{}, TelphoneGroup{}, Telphone{}, RestfulMsg{})

	return
}

func Close() {
	utils.Close()
}
