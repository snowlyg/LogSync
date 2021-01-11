package models

type ServerMsg struct {
	PlatformServiceId int64  `json:"platform_service_id" gorm:"column:platform_service_id"`       // 服务id
	ServiceTypeId     int64  `json:"service_type_id" gorm:"column:service_type_id"`               // 服务类型id
	ServiceName       string `json:"service_name" gorm:"column:service_name;type:varchar(256)"`   // 服务类型名称
	ServiceTitle      string `json:"service_title" gorm:"column:service_title;type:varchar(256)"` // 服务类型名称
	FaultMsg          string `json:"fault_msg" gorm:"column:fault_msg;type:varchar(256)"`         // 错误信息
	Status            bool   `json:"status" gorm:"column:status"`                                 // 服务装填
}
