package sync

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
	"github.com/snowlyg/LogSync/utils/logging"
	"time"
)

func SyncDevice() {
	// 同步设备和通讯录
	// http://fyxt.t.chindeo.com/platform/report/syncdevice 同步设备 post
	// http://fyxt.t.chindeo.com/platform/report/synctelgroup   同步通讯录组 post
	// http://fyxt.t.chindeo.com/platform/report/synctel  同步通讯录 post
	serverList, err := utils.GetServices()
	if err != nil {
		logging.Err.Error(err)
		return
	}
	account := utils.Conf().Section("mysql").Key("account").MustString("visible")
	pwd := utils.Conf().Section("mysql").Key("pwd").MustString("Chindeo")
	for _, server := range serverList {
		switch server.ServiceName {
		case "MySQL":
			func() {
				conn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", account, pwd, server.Ip, server.Port, "dois")
				sqlDb, err := gorm.Open("mysql", conn)
				if err != nil {
					logging.Norm.Infof("mysql conn error: %v ,id:%s", err, server.Ip)
					return
				} else {
					logging.Norm.Info("mysql conn success")
				}
				defer sqlDb.Close()

				sqlDb.DB().SetMaxOpenConns(1)
				sqlDb.SetLogger(utils.DefaultGormLogger)
				sqlDb.LogMode(false)

				createDevices(sqlDb)
				createTelphones(sqlDb)
				createTelphoneGroups(sqlDb)
				deleteMsg()
			}()
		default:
			continue
		}
	}
}

// 删除3天前的日志记录
func deleteMsg() {
	lastWeek := time.Now().AddDate(0, 0, -3).Format("2006-01-02 15:04:05")
	utils.SQLite.Unscoped().Where("created_at < ?", lastWeek).Delete(models.LogMsg{})
	utils.SQLite.Unscoped().Where("created_at < ?", lastWeek).Delete(models.ServerMsg{})

	logging.Norm.Infof("删除3天前数据库日志记录 :%s", lastWeek)
}

// 同步设备
func createDevices(sqlDb *gorm.DB) {
	var cfDevices []*models.CfDevice
	query := "select ct_loc.loc_desc as loc_desc,pac_room.room_desc as room_desc, pac_bed.bed_code as bed_code, dev_id ,dev_code ,dev_desc ,dev_position ,dev_type,dev_active,dev_status,dev_create_time,mm.ipaddr as dev_ip from cf_device"
	query += " left join mqtt.mqtt_device as mm on mm.username = cf_device.dev_code"
	query += " left join ct_loc on ct_loc.loc_id = cf_device.ct_loc_id"
	query += " left join pac_room on pac_room.room_id = cf_device.pac_room_id"
	query += " left join pac_bed on pac_bed.bed_id = cf_device.pac_bed_id"

	rows, err := sqlDb.Raw(query).Rows()
	if err != nil {
		logging.Err.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		var cfDevice models.CfDevice
		// ScanRows 扫描一行记录到 user
		sqlDb.ScanRows(rows, &cfDevice)

		cfDevices = append(cfDevices, &cfDevice)
	}

	if len(cfDevices) > 0 {
		if utils.SQLite != nil {
			utils.SQLite.Exec("DELETE FROM t_cf_devices;")
			for _, cfD := range cfDevices {
				utils.SQLite.Create(&cfD)
			}

			cfDeviceJson, _ := json.Marshal(&cfDevices)
			data := fmt.Sprintf("data=%s", cfDeviceJson)
			var res interface{}
			res, err = utils.SyncServices("platform/report/syncdevice", data)
			if err != nil {
				logging.Err.Error(err)
			}
			logging.Norm.Infof("数据提交返回信息:%v", res)

		} else {
			logging.Norm.Infof("db.SQLite is null")
		}
	}

}

// 同步通讯录
func createTelphones(sqlDb *gorm.DB) {
	var telphones []*models.Telphone

	rows, err := sqlDb.Raw("select *  from ss_telephone").Rows()
	if err != nil {
		logging.Err.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		var telphone models.Telphone
		// ScanRows 扫描一行记录到 user
		sqlDb.ScanRows(rows, &telphone)

		telphones = append(telphones, &telphone)
	}

	if len(telphones) > 0 {
		if utils.SQLite != nil {
			utils.SQLite.Exec("DELETE FROM t_telphones;")
			for _, cfD := range telphones {
				utils.SQLite.Create(&cfD)
			}

			telphoneJson, _ := json.Marshal(&telphones)
			data := fmt.Sprintf("data=%s", telphoneJson)
			var res interface{}
			res, err = utils.SyncServices("platform/report/synctel", data)
			if err != nil {
				logging.Err.Error(err)
			}
			logging.Norm.Infof("同步通讯录返回数据:%s", res)
		} else {
			logging.Norm.Infof("db.SQLite is null")
		}
	}
}

// 同步电话组
func createTelphoneGroups(sqlDb *gorm.DB) {
	var telphoneGroups []*models.TelphoneGroup
	rows, err := sqlDb.Raw("select *  from ss_telephone_group").Rows()
	if err != nil {
		logging.Err.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		var telphoneGroup models.TelphoneGroup
		// ScanRows 扫描一行记录到 user
		sqlDb.ScanRows(rows, &telphoneGroup)

		telphoneGroups = append(telphoneGroups, &telphoneGroup)
	}

	if len(telphoneGroups) > 0 {
		if utils.SQLite != nil {
			utils.SQLite.Exec("DELETE FROM t_telphone_groups;")
			for _, cfD := range telphoneGroups {
				utils.SQLite.Create(&cfD)
			}

			telphoneGroupJson, _ := json.Marshal(&telphoneGroups)
			data := fmt.Sprintf("data=%s", telphoneGroupJson)
			var res interface{}
			res, err = utils.SyncServices("platform/report/synctelgroup", data)
			if err != nil {
				logging.Err.Error(err)
			}
			logging.Norm.Infof("同步通讯录组返回数据:%s", res)
		} else {
			logging.Norm.Infof("db.SQLite is null")
		}
	}
}
