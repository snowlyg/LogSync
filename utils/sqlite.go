package utils

import (
	"fmt"
	"sync"

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
	dbFile := DBFile()
	var single sync.Mutex
	single.Lock()
	var err error
	sqlite, err = gorm.Open("sqlite3", fmt.Sprintf("%s?loc=Asia/Shanghai", dbFile))
	if err != nil {
		panic(fmt.Sprintf("sqlite init err %+v", err))
	}
	sqlite.DB().SetMaxIdleConns(100)
	sqlite.DB().SetMaxOpenConns(100)
	sqlite.SetLogger(DefaultGormLogger)
	sqlite.LogMode(false)
	single.Unlock()

	return sqlite
}
