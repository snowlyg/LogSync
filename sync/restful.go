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

type RestfulMsg struct {
	Url    string `json:"url" gorm:"column:url"`
	Status bool   `json:"status" gorm:"column:status"`
	ErrMsg string `json:"err_msg" gorm:"column:err_msg"`
}

func CheckRestful() {
	logger := logging.GetMyLogger("restful")
	var restfulMsgs []*RestfulMsg
	var restfulUrl []string

	logger.Info("<========================>")
	logger.Info("接口监控开始")

	restfuls, err := utils.GetRestfuls()
	if err != nil {
		logger.Error(err)
		return
	}

	if len(restfuls) == 0 {
		logger.Info("未获取到接口数据")
		logger.Info("接口监控结束")
		return
	}

	for _, restful := range restfuls {
		func() {
			restfulMsg := &models.RestfulMsg{Url: restful.Url, Model: gorm.Model{CreatedAt: time.Now()}}
			getRestful(restfulMsg, logger)
			//// 本机存储数据
			//var oldRestfulMsg models.RestfulMsg
			//utils.GetSQLite().Where("url = ?", restful.Url).First(&oldRestfulMsg)
			//if oldRestfulMsg.ID > 0 {
			//	oldRestfulMsg.Status = restfulMsg.Status
			//	oldRestfulMsg.ErrMsg = restfulMsg.ErrMsg
			//	utils.GetSQLite().Save(&oldRestfulMsg)
			//} else {
			//	utils.GetSQLite().Save(&restfulMsg)
			//}

			restfulMsgResponse := &RestfulMsg{restful.Url, restfulMsg.Status, restfulMsg.ErrMsg}
			restfulMsgs = append(restfulMsgs, restfulMsgResponse)
			restfulUrl = append(restfulUrl, restful.Url)
		}()

	}

	var restfulMsgJson []byte
	restfulMsgJson, err = json.Marshal(restfulMsgs)
	if err != nil {
		logger.Errorf("restfulMsgs: %+v\n  json化错误: %+v\n", restfulMsgs, err)
	}
	data := fmt.Sprintf("restful_data=%s", string(restfulMsgJson))
	var res interface{}
	res, err = utils.SyncServices("platform/report/restful", data)
	if err != nil {
		logger.Error(err)
	}

	logger.Infof("推送返回信息: %v\n", res)
	logger.Info(fmt.Sprintf("%d 个接口监控推送完成 : %v", len(restfulMsgs), restfulUrl))
	logger.Info("接口监控结束")
}

// RestfulResponse
type RestfulResponse struct {
	Status int64       `json:"code"`
	Msg    string      `json:"message"`
	Data   interface{} `json:"data"`
}

// getRestful 请求接口
func getRestful(restfulMsg *models.RestfulMsg, logger *logging.Logger) {
	var re RestfulResponse
	conCount := 0
	for conCount < 3 && !restfulMsg.Status {
		result := utils.Request("GET", restfulMsg.Url, "", true)
		if len(result) == 0 {
			str := fmt.Sprintf("接口无法访问")
			restfulMsg.Status = false
			restfulMsg.ErrMsg = str
			conCount++
			continue
		}
		logger.Info(string(result))
		err := json.Unmarshal(result, &re)
		if err != nil {
			str := fmt.Sprintf("接口可以访问，但返回数据 %s 无法解析，报错如下：%v", string(result), err)
			restfulMsg.Status = false
			restfulMsg.ErrMsg = str
			conCount++
			continue
		}

		// 200代表成功，其他均是异常
		if re.Status == 200 {
			restfulMsg.Status = true
			restfulMsg.ErrMsg = re.Msg
			conCount = 0
			return
		} else {
			restfulMsg.Status = false
			restfulMsg.ErrMsg = re.Msg
			conCount++
			continue
		}
	}

	// 故障显示连接次数
	if conCount > 0 {
		logger.Infof("%s 连接次数: %d", restfulMsg.Url, conCount)
	}

	conCount = 0
	return
}
