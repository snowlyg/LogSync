package models

import "github.com/jinzhu/gorm"

type RestfulMsg struct {
	gorm.Model
	Url    string `json:"url" gorm:"column:url"`
	Status bool   `json:"status" gorm:"column:status"`
	ErrMsg string `json:"err_msg" gorm:"column:err_msg"`
}
