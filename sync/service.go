package sync

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/antlinker/go-mqtt/client"
	"github.com/jander/golog/logger"
	"github.com/jinzhu/gorm"
	"github.com/jlaffaye/ftp"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
)

var ServiceCount int      // 扫描设备数量
var ServiceNames []string // 扫描设备名称

// 监控服务
func CheckService() {
	// 监控服务
	// platform_service_id ，service_type_id，create_at，fault_msg
	// http://fyxt.t.chindeo.com/platform/report/service  服务故障上报url

	logger.Println("<========================>")
	logger.Println("服务监控开始")
	defer logger.Println("服务监控结束")
	defer logger.Println(fmt.Sprintf("%d 个服务监控推送完成 : %v", ServiceCount, ServiceNames))

	ServiceCount = 0
	ServiceNames = nil

	serverList := utils.GetServices()
	if len(serverList) > 0 {
		var serverMsgs []*models.ServerMsg
		for _, server := range serverList {
			logger.Println(fmt.Sprintf("服务名称： %v", server.ServiceName))
			var serverMsg models.ServerMsg
			serverMsg.Status = true
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
						serverMsg.Status = false
						serverMsg.FaultMsg = err.Error()
						logger.Printf("MYSQL 连接错误: %v ", err)
					}
					if sqlDb == nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = "MYSQL 连接失败"
					} else {
						defer sqlDb.Close()
					}
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
						logger.Printf("MQTT 客户端创建失败: %v ", err)
					} else {
						if mqttClient == nil {
							serverMsg.Status = false
							serverMsg.FaultMsg = "连接失败"
							logger.Printf("MQTT 连接失败")
						} else {
							defer mqttClient.Disconnect()
							//建立连接
							err = mqttClient.Connect()
							if err != nil {
								serverMsg.Status = false
								serverMsg.FaultMsg = err.Error()
								logger.Printf("MQTT 连接出错: %v ", err)
							}
						}
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
						} else {
							serverMsg.Status = false
							serverMsg.FaultMsg = err.Error()
							logger.Printf("RabbitMq 连接错误: %v ", err)
						}
					} else {
						if rabbitmq == nil {
							serverMsg.Status = false
							serverMsg.FaultMsg = "连接失败"
							logger.Printf("RabbitMq 连接失败: 连接失败 ")
						} else {
							defer rabbitmq.Destory()
						}
					}

				}()
			case "FileZilla Server":
				func() {
					c, err := ftp.Dial(fmt.Sprintf("%s:%d", server.Ip, server.Port), ftp.DialWithTimeout(5*time.Second))
					if err != nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = err.Error()
						logger.Printf("FTP 连接错误: %v ", err)
					} else {
						if c == nil {
							serverMsg.Status = false
							serverMsg.FaultMsg = "连接失败"
							logger.Printf("FTP 连接失败")
						} else {
							err = c.Login(server.Account, server.Pwd)
							if err != nil {
								serverMsg.Status = false
								serverMsg.FaultMsg = err.Error()
								logger.Printf("FTP 连接错误: %v ", err)
							} else {
								defer c.Quit()
								serverMsg.Status = true
								logger.Println("FTP 连接成功")
							}
						}
					}
				}()
			default:
				func() {
					if err := utils.IsPortInUse(server.Ip, server.Port); err != nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = err.Error()
						logger.Printf("%s连接错误: %v ", server.ServiceName, err)
					}
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
			setServiceCountAndNames(server)
		}

		serverMsgJson, _ := json.Marshal(&serverMsgs)
		data := fmt.Sprintf("fault_data=%s", string(serverMsgJson))
		res := utils.SyncServices("platform/report/service", data)
		logger.Printf("推送返回信息: %v", res)
	}

}

func setServiceCountAndNames(server *utils.Server) {
	ServiceCount++
	ServiceNames = append(ServiceNames, server.ServiceName)
}
