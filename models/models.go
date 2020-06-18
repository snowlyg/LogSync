package models

import (
	"github.com/snowlyg/LogSync/db"
)

func Init() (err error) {
	err = db.Init()
	if err != nil {
		return
	}
	db.SQLite.AutoMigrate(LogMsg{}, ServerMsg{}, CfDevice{}, TelphoneGroup{}, Telphone{})

	return
}

func Close() {
	db.Close()
}
