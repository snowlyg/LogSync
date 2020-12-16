package sync

import (
	"encoding/json"
	"fmt"
	"github.com/jander/golog/logger"
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
	var restfulMsgs []*RestfulMsg
	var restfulUrl []string
	logging.Errorf("read article err : %+v\n", err)
	logging.Err.
	logging.Norm.("<========================>")
	logger.Println("接口监控开始")

	restfuls, err := utils.GetRestfuls()
	if err != nil {
		logger.Println(err)
		return
	}

	if len(restfuls) == 0 {
		logger.Println("未获取到接口数据")
		logger.Println("接口监控结束")
		return
	}

	for _, restful := range restfuls {
		restfulMsg := &models.RestfulMsg{Url: restful.Url, Model: gorm.Model{CreatedAt: time.Now()}}
		getRestful(restfulMsg)
		// 本机存储数据
		var oldRestfulMsg models.RestfulMsg
		utils.SQLite.Where("url = ?", restful.Url).First(&oldRestfulMsg)
		if oldRestfulMsg.ID > 0 {
			oldRestfulMsg.Status = restfulMsg.Status
			oldRestfulMsg.ErrMsg = restfulMsg.ErrMsg
			utils.SQLite.Save(&oldRestfulMsg)
		} else {
			utils.SQLite.Save(&restfulMsg)
		}

		restfulMsgResponse := &RestfulMsg{restful.Url, restfulMsg.Status, restfulMsg.ErrMsg}
		restfulMsgs = append(restfulMsgs, restfulMsgResponse)
		restfulUrl = append(restfulUrl, restful.Url)
	}

	var restfulMsgJson []byte
	restfulMsgJson, err = json.Marshal(restfulMsgs)
	if err != nil {
		logger.Printf("restfulMsgs: %+v\n  json化错误: %+v\n", restfulMsgs, err)
	}
	data := fmt.Sprintf("restful_data=%s", string(restfulMsgJson))
	var res interface{}
	res,err = utils.SyncServices("platform/report/restful", data)
	if err != nil{
		logger.Println(err)
	}
	logger.Printf("推送返回信息: %v\n", res)

	logger.Println(fmt.Sprintf("%d 个接口监控推送完成 : %v", len(restfulMsgs), restfulUrl))
	logger.Println("接口监控结束")
}

// RestfulResponse
type RestfulResponse struct {
	Status int64  `json:"status"`
	Msg    string `json:"msg"`
}

// getRestful 请求接口
func getRestful(restfulMsg *models.RestfulMsg) {
	var re RestfulResponse
	conCount := 0
	for conCount < 3 && !restfulMsg.Status {
		result := utils.Request("GET",restfulMsg.Url,"",true)
		//if err != nil {
		//	str := fmt.Sprintf("接口无法访问，报错如下：%v", err)
		//	restfulMsg.Status = false
		//	restfulMsg.ErrMsg = str
		//	conCount++
		//	continue
		//}

		err := json.Unmarshal(result, &re)
		if err != nil {
			str := fmt.Sprintf("接口可以访问，但返回数据无法解析，报错如下：%v", err)
			restfulMsg.Status = false
			restfulMsg.ErrMsg = str
			conCount++
			continue
		}

		if re.Status == 0 {
			restfulMsg.Status = false
			restfulMsg.ErrMsg = re.Msg
			conCount++
			continue
		} else {
			restfulMsg.Status = true
			restfulMsg.ErrMsg = re.Msg
			conCount = 0
			return
		}
	}

	// 故障显示连接次数
	if conCount > 0 {
		logger.Printf("%s 连接次数: %d", restfulMsg.Url, conCount)
	}

	conCount = 0
	return
}
