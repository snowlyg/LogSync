package sync

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/antlinker/go-mqtt/client"
	"github.com/jander/golog/logger"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
)

// 监控服务
func CheckService() {
	// 监控服务
	// platform_service_id ，service_type_id，create_at，fault_msg
	// http://fyxt.t.chindeo.com/platform/report/service  服务故障上报url

	var serverMsgs []*models.ServerMsg // 扫描设备名称
	var serverNames []string           // 扫描设备名称

	logger.Println("<========================>")
	logger.Println("服务监控开始")

	serverList := utils.GetServices()
	if len(serverList) > 0 {
		for _, server := range serverList {

			logger.Println(fmt.Sprintf("服务名称： %v", server.ServiceName))
			var serverMsg models.ServerMsg
			serverMsg.Status = false
			serverMsg.ServiceTypeId = server.ServiceTypeId
			serverMsg.ServiceName = server.ServiceName
			serverMsg.ServiceTitle = server.ServiceTitle
			serverMsg.PlatformServiceId = server.Id
			serverMsg.CreatedAt = time.Now()

			conCount := 0
			for conCount < 3 && !serverMsg.Status {
				switch server.ServiceName {
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
							serverMsg.FaultMsg = err.Error()
							logger.Printf("MQTT 客户端创建失败: %v ", err)
							conCount++
							return
						} else {
							if mqttClient == nil {
								serverMsg.FaultMsg = "连接失败"
								logger.Printf("MQTT 连接失败")
								conCount++
								return
							} else {
								defer mqttClient.Disconnect()
								//建立连接
								err = mqttClient.Connect()
								if err != nil {
									serverMsg.FaultMsg = err.Error()
									logger.Printf("MQTT 连接出错: %v ", err)
									conCount++
									return
								}
							}
						}
						serverMsg.Status = true
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
								serverMsg.FaultMsg = err.Error()
								logger.Printf("RabbitMq 连接错误: %v ", err)
								conCount++
								return
							}
						} else {
							if rabbitmq == nil {
								serverMsg.FaultMsg = "连接失败"
								logger.Printf("RabbitMq 连接失败: 连接失败 ")
								conCount++
								return
							} else {
								defer rabbitmq.Destory()
							}
						}
						serverMsg.Status = true
					}()
				default:
					func() {
						if err := utils.IsPortInUse(server.Ip, server.Port); err != nil {
							serverMsg.FaultMsg = err.Error()
							logger.Printf("%s连接错误: %v ", server.ServiceName, err)
							conCount++
							return
						}
						serverMsg.Status = true
					}()
				}
			}

			// 故障显示连接次数
			if conCount > 0 {
				logger.Printf("%s 连接次数: %d", server.ServiceName, conCount)
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
			serverNames = append(serverNames, serverMsg.ServiceName)

			conCount = 0
		}

		serverMsgJson, _ := json.Marshal(serverMsgs)
		data := fmt.Sprintf("fault_data=%s", string(serverMsgJson))
		res := utils.SyncServices("platform/report/service", data)
		logger.Printf("推送返回信息: %v", res)

	}
	logger.Println(fmt.Sprintf("%d 个服务监控推送完成 : %v", len(serverMsgs), serverNames))
	logger.Println("服务监控结束")

}
