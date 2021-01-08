package models

import (
	"github.com/snowlyg/LogSync/utils"
)

func Init() (err error) {
	utils.GetSQLite().AutoMigrate(LogMsg{}, ServerMsg{}, CfDevice{}, TelphoneGroup{}, Telphone{}, RestfulMsg{})
	return
}
