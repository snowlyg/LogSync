package models

import (
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/snowlyg/LogSync/utils"
)

//dev_id as local_device_id,
//dev_code as device_code,
//dev_desc as device_desc,
//dev_position as device_position ,
//dev_type as device_type_id,
//dev_ip as device_ip,
//dev_active as device_active,
//dev_create_time as create_at

type CfDevice struct {
	DevId         int64  `json:"local_device_id" gorm:"column:dev_id"`                        // 服务id
	DevType       int64  `json:"device_type_id" gorm:"column:dev_type"`                       // 类型
	DevStatus     int64  `json:"device_status" gorm:"column:dev_status"`                      // 状态
	DevActive     int64  `json:"device_active" gorm:"column:dev_active"`                      // 状态
	DevCode       string `json:"device_code" gorm:"column:dev_code;type:varchar(50)"`         // 设备代码
	DevDesc       string `json:"device_desc" gorm:"column:dev_desc;type:varchar(50)"`         // 设备描述
	DevPosition   string `json:"device_position" gorm:"column:dev_position;type:varchar(50)"` // 位置
	DevIp         string `json:"device_ip" gorm:"column:dev_ip;type:varchar(40)"`             // ip
	LocDesc       string `json:"loc_desc" gorm:"column:loc_desc;type:varchar(60)"`            // loc_desc
	RoomDesc      string `json:"room_desc" gorm:"column:room_desc;type:varchar(60)"`          // room_desc
	BedCode       string `json:"bed_code" gorm:"column:bed_code;type:varchar(255)"`           // bed_code
	DevCreateTime string `json:"create_at" gorm:"column:dev_create_time"`                     // 创建时间
}

func GetCfDevice() ([]*CfDevice, error) {
	sqlDb, err := gorm.Open("mysql", utils.Config.DB)
	if err != nil {
		return nil, err
	}
	defer sqlDb.Close()

	sqlDb.DB().SetMaxOpenConns(100)
	sqlDb.DB().SetMaxIdleConns(100)
	sqlDb.SetLogger(utils.DefaultGormLogger)
	sqlDb.LogMode(false)

	var cfDevices []*CfDevice
	query := "select ct_loc.loc_desc as loc_desc,pac_room.room_desc as room_desc, pac_bed.bed_code as bed_code, dev_id ,dev_code ,dev_desc ,dev_position ,dev_type,dev_active,dev_status,dev_create_time,mm.ipaddr as dev_ip from cf_device"
	query += " left join mqtt.mqtt_device as mm on mm.username = cf_device.dev_code"
	query += " left join ct_loc on ct_loc.loc_id = cf_device.ct_loc_id"
	query += " left join pac_room on pac_room.room_id = cf_device.pac_room_id"
	query += " left join pac_bed on pac_bed.bed_id = cf_device.pac_bed_id"
	query += " where cf_device.dev_active = '1'"

	rows, _ := sqlDb.Raw(query).Rows()
	defer rows.Close()
	if rows == nil {
		return nil, errors.New("get 0 data")
	}
	for rows.Next() {
		var cfDevice CfDevice
		sqlDb.ScanRows(rows, &cfDevice)
		cfDevices = append(cfDevices, &cfDevice)
	}
	return cfDevices, nil
}
