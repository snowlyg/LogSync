package utils

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Model struct {
	ID        string   `structs:"id" gorm:"primary_key" form:"id" json:"id"`
	CreatedAt DateTime `structs:"-" json:"createdAt" gorm:"type:datetime"`
	UpdatedAt DateTime `structs:"-" json:"updatedAt" gorm:"type:datetime"`
	// DeletedAt *time.Time `sql:"index" structs:"-"`
}

var sqlite *gorm.DB

func GetSQLite() *gorm.DB {
	if sqlite != nil {
		return sqlite
	}
	dbFile := DBFile()
	var err error
	sqlite, err = gorm.Open("sqlite3", fmt.Sprintf("%s?loc=Asia/Shanghai", dbFile))
	if err != nil {
		panic(fmt.Sprintf("sqlite init err %+v", err))
	}
	sqlite.DB().SetMaxIdleConns(10)
	sqlite.DB().SetMaxOpenConns(10)
	sqlite.SetLogger(DefaultGormLogger)
	sqlite.LogMode(false)

	return sqlite
}
