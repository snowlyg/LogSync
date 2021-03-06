package sync

import (
	"encoding/json"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
	"github.com/snowlyg/LogSync/utils/logging"
)

func SyncDevice(logger *logging.Logger) {
	// logger := logging.GetMyLogger("sync")
	// 同步设备和通讯录
	// http://fyxt.t.chindeo.com/platform/report/syncdevice 同步设备 post
	// http://fyxt.t.chindeo.com/platform/report/synctelgroup   同步通讯录组 post
	// http://fyxt.t.chindeo.com/platform/report/synctel  同步通讯录 post
	sqlDb, err := gorm.Open("mysql", utils.Config.DB)
	if err != nil {
		logger.Infof("mysql conn error: ", err)
		return
	} else {
		logger.Info("mysql conn success")
	}
	defer sqlDb.Close()

	sqlDb.DB().SetMaxOpenConns(1)
	sqlDb.SetLogger(utils.DefaultGormLogger)
	sqlDb.LogMode(false)

	createDevices(logger)
	createTelphones(sqlDb, logger)
	createTelphoneGroups(sqlDb, logger)

}

// 同步设备
func createDevices(logger *logging.Logger) {
	cfDevices, err := models.GetCfDevice()
	if err != nil {
		logger.Error(err)
	}

	if len(cfDevices) > 0 {
		cfDeviceJson, _ := json.Marshal(&cfDevices)
		data := fmt.Sprintf("data=%s", cfDeviceJson)
		var res interface{}
		res, err = utils.SyncServices("platform/report/syncdevice", data)
		if err != nil {
			logger.Error(err)
		}
		logger.Infof("设备数据提交返回信息:%v", res)
	}
}

// 同步通讯录
func createTelphones(sqlDb *gorm.DB, logger *logging.Logger) {
	var telphones []*models.Telphone
	rows, err := sqlDb.Raw("select *  from ss_telephone").Rows()
	if err != nil {
		logger.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		var telphone models.Telphone
		sqlDb.ScanRows(rows, &telphone)
		telphones = append(telphones, &telphone)
	}

	if len(telphones) > 0 {
		telphoneJson, _ := json.Marshal(&telphones)
		data := fmt.Sprintf("data=%s", telphoneJson)
		var res interface{}
		res, err = utils.SyncServices("platform/report/synctel", data)
		if err != nil {
			logger.Error(err)
		}
		logger.Infof("同步通讯录返回数据:%s", res)
	}
}

// 同步电话组
func createTelphoneGroups(sqlDb *gorm.DB, logger *logging.Logger) {
	var telphoneGroups []*models.TelphoneGroup
	rows, err := sqlDb.Raw("select *  from ss_telephone_group").Rows()
	if err != nil {
		logger.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		var telphoneGroup models.TelphoneGroup
		sqlDb.ScanRows(rows, &telphoneGroup)
		telphoneGroups = append(telphoneGroups, &telphoneGroup)
	}

	if len(telphoneGroups) > 0 {
		telphoneGroupJson, _ := json.Marshal(&telphoneGroups)
		data := fmt.Sprintf("data=%s", telphoneGroupJson)
		var res interface{}
		res, err = utils.SyncServices("platform/report/synctelgroup", data)
		if err != nil {
			logger.Error(err)
		}
		logger.Infof("同步通讯录组返回数据:%s", res)
	}
}
