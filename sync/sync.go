package sync

import (
	"encoding/json"
	"fmt"
	"github.com/antlinker/go-mqtt/client"
	"github.com/jander/golog/logger"
	"github.com/jinzhu/gorm"
	"github.com/jlaffaye/ftp"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
	"net"
	"time"
)

func SyncDevice() {
	// 监控服务
	// 同步设备和通讯录
	// http://fyxt.t.chindeo.com/platform/report/syncdevice 同步设备 post
	// http://fyxt.t.chindeo.com/platform/report/synctelgroup   同步通讯录组 post
	// http://fyxt.t.chindeo.com/platform/report/synctel  同步通讯录 post
	// platform_service_id ，service_type_id，create_at，fault_msg
	// http://fyxt.t.chindeo.com/platform/report/service  服务故障上报url
	serverList := utils.GetServices()
	var serverMsgs []*models.ServerMsg
	for _, server := range serverList {

		var serverMsg models.ServerMsg
		serverMsg.ServiceTypeId = server.ServiceTypeId
		serverMsg.ServiceName = server.ServiceName
		serverMsg.ServiceTitle = server.ServiceTitle
		serverMsg.PlatformServiceId = server.Id
		serverMsg.CreatedAt = time.Now()

		switch server.ServiceName {
		case "MySQL":
			func() {
				conn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", server.Account, server.Pwd, server.Ip, server.Port, "dois")
				sqlDb, err := gorm.Open("mysql", conn)
				if err != nil {
					logger.Printf("mysql conn error: %v ", err)
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					return
				} else {
					logger.Println("mysql conn success")
					serverMsg.Status = true
				}
				defer sqlDb.Close()

				sqlDb.DB().SetMaxOpenConns(1)
				sqlDb.SetLogger(utils.DefaultGormLogger)
				sqlDb.LogMode(false)

				createDevices(sqlDb)
				createTelphones(sqlDb)
				createTelphoneGroups(sqlDb)
			}()

		case "EMQX":
			func() {
				addr := fmt.Sprintf("tcp://%s:%d", server.Ip, server.Port)
				mqttClient, err := client.CreateClient(client.MqttOption{
					Addr:               addr,
					ReconnTimeInterval: 1,
					UserName:           server.Account,
					Password:           server.Pwd,
				})

				if err != nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("mqtt client create error: %v ", err)
					return
				}
				//断开连接
				defer mqttClient.Disconnect()

				if mqttClient == nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = "连接失败"
					logger.Printf("mqtt conn error: 连接失败 ")
					return
				} else {
					//建立连接
					err = mqttClient.Connect()
					if err != nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = err.Error()
						logger.Printf("mqtt conn error: %v ", err)
						return
					}

					serverMsg.Status = true
					logger.Println("mqtt conn success")

					return
				}
			}()

		case "RabbitMQ":
			func() {
				mqurl := fmt.Sprintf("amqp://%s:%s@%s:%d/shop", server.Account, server.Pwd, server.Ip, server.Port)
				rabbitmq, err := NewRabbitMQSimple("imoocSimple", mqurl)
				if err != nil {
					if err.Error() == "Exception (403) Reason: \"no access to this vhost\"" {
						serverMsg.Status = true
						logger.Println("RabbitMq conn success")
						return
					}

					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("RabbitMq conn error: %v ", err)
					return
				}

				defer rabbitmq.Destory()

				if rabbitmq == nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = "连接失败"
					logger.Printf("RabbitMq conn error: 连接失败 ")
					return
				} else {
					serverMsg.Status = true
					logger.Println("RabbitMq conn success")
					//断开连接
					return
				}

			}()

		case "FileZilla Server":
			func() {
				c, err := ftp.Dial(fmt.Sprintf("%s:%d", server.Ip, server.Port), ftp.DialWithTimeout(5*time.Second))
				if err != nil {
					if err.Error() == "Exception (403) Reason: \"no access to this vhost\"" {
						serverMsg.Status = true
						logger.Println("FTP conn success")
						return
					}
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("FTP conn error: %v ", err)
					return
				}

				defer c.Quit()

				if c == nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = "连接失败"
					logger.Printf("FTP conn error: 连接失败 ")
					return
				} else {
					err = c.Login(server.Account, server.Pwd)
					if err != nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = err.Error()
						logger.Printf("FTP conn error: %v ", err)
						return
					} else {
						serverMsg.Status = true
						logger.Println("FTP conn success")
						return
					}
				}
			}()
		default:
			func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", server.Ip, server.Port))
				if err != nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("FTP conn error: %v ", err)
					return
				}
				defer conn.Close()
				if conn == nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = "连接失败"
					logger.Printf("FTP conn error: 连接失败 ")
					return
				}

				serverMsg.Status = true
				logger.Printf("%s conn success", server.ServiceName)
				return

			}()
		}

		// 本机存储数据
		var oldServerMsg models.ServerMsg
		utils.SQLite.Where("service_type_id = ?", server.Id).First(&oldServerMsg)
		if oldServerMsg.ID > 0 {
			oldServerMsg.Status = serverMsg.Status
			oldServerMsg.FaultMsg = serverMsg.FaultMsg
			utils.SQLite.Save(&oldServerMsg)
		} else {
			utils.SQLite.Save(&serverMsg)
		}

		serverMsgs = append(serverMsgs, &serverMsg)
	}

	serverMsgJson, _ := json.Marshal(&serverMsgs)
	data := fmt.Sprintf("fault_data=%s", string(serverMsgJson))
	res := utils.PostServices("platform/report/service", data)

	logger.Printf("推送返回信息: %v", res)
	logger.Printf("服务监控推送完成")
}

// 同步设备
func createDevices(sqlDb *gorm.DB) {
	var cfDevices []*models.CfDevice
	query := "select ct_loc.loc_desc as loc_desc,pac_room.room_desc as room_desc, pac_bed.bed_code as bed_code, dev_id ,dev_code ,dev_desc ,dev_position ,dev_type,dev_active ,dev_create_time,mm.ipaddr as dev_ip from cf_device"
	query += " left join mqtt.mqtt_device as mm on mm.username = cf_device.dev_code"
	query += " left join ct_loc on ct_loc.loc_id = cf_device.ct_loc_id"
	query += " left join pac_room on pac_room.room_id = cf_device.pac_room_id"
	query += " left join pac_bed on pac_bed.bed_id = cf_device.pac_bed_id"

	rows, err := sqlDb.Raw(query).Rows()
	if err != nil {
		logger.Println(err)
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
			res := utils.PostServices("platform/report/syncdevice", data)
			logger.Error("PostDevice:%s", res)

		} else {
			logger.Println("db.SQLite is null")
		}
	}

}

// 同步通讯录
func createTelphones(sqlDb *gorm.DB) {
	var telphones []*models.Telphone

	rows, err := sqlDb.Raw("select *  from ss_telephone").Rows()
	if err != nil {
		logger.Println(err)
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
			res := utils.PostServices("platform/report/synctel", data)
			logger.Error("PostTel:%s", res)
		} else {
			logger.Println("db.SQLite is null")
		}
	}
}

// 同步电话组
func createTelphoneGroups(sqlDb *gorm.DB) {
	var telphoneGroups []*models.TelphoneGroup
	//var telphoneGroups []*models.TelphoneGroup

	rows, err := sqlDb.Raw("select *  from ss_telephone_group").Rows()
	if err != nil {
		logger.Println(err)
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
			res := utils.PostServices("platform/report/synctelgroup", data)

			logger.Error("PostTelGroup:%s", res)
		} else {
			logger.Println("db.SQLite is null")
		}
	}
}
