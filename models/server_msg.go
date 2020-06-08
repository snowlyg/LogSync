package models

import (
	"github.com/jinzhu/gorm"
)

type ServerMsg struct {
	gorm.Model
	PlatformServiceId int64  // 服务id
	ServiceTypeId     int64  // 服务类型id
	ServiceName       string `gorm:"type:varchar(256)"` // 服务类型名称
	ServiceTitle      string `gorm:"type:varchar(256)"` // 服务类型名称
	FaultMsg          string `gorm:"type:varchar(256)"` // 错误信息
	Status            bool   // 服务装填
}
