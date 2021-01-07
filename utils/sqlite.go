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
	var single sync.Once
	dbFile := DBFile()
	single.Do(func() {
		var err error
		sqlite, err = gorm.Open("sqlite3", fmt.Sprintf("%s?loc=Asia/Shanghai", dbFile))
		if err != nil {
			panic(fmt.Sprintf("sqlite init err %+v", err))
		}
		// Sqlite cannot handle concurrent writes, so we limit sqlite to one connection.
		// see https://github.com/mattn/go-sqlite3/issues/274
		sqlite.DB().SetMaxOpenConns(100)
		sqlite.SetLogger(DefaultGormLogger)
		sqlite.LogMode(false)
	})

	return sqlite
}
