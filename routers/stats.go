package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
	"github.com/snowlyg/LogSync/utils/logging"
	"net/http"
	"time"
)

func SyncDeviceLog(c *gin.Context) {
	syncLog := c.Query("sync_log")
	if syncLog == "1" {
		err := utils.GetToken()
		if err != nil {
			c.String(http.StatusOK, "get token err %v", err)
			return
		}
		go func() {
			sync.SyncDevice()
		}()

		c.String(http.StatusOK, "通讯录/设备数据成功执行同步")
	} else {
		c.String(http.StatusOK, "参数错误，未执行同步")
	}

}

func SyncLog(c *gin.Context) {
	syncLog := c.Query("sync_log")
	if syncLog == "1" {
		err := utils.GetToken()
		if err != nil {
			c.String(http.StatusOK, "get token err %v", err)
			return
		}

		go func() {
			start := time.Now() // 获取当前时间
			sync.CheckRestful()
			elapsed1 := time.Since(start)
			logging.CommonLogger.Info("完成接口监控耗时：", elapsed1)
			sync.CheckService()
			elapsed2 := time.Since(start)
			logging.CommonLogger.Info("完成服务监控耗时：", elapsed2)
			// 进入当天目录,跳过 23点45 当天凌晨 0点15 分钟，给设备创建目录的时间
			if !((time.Now().Hour() == 0 && time.Now().Minute() < 59) || (time.Now().Hour() == 23 && time.Now().Minute() > 45)) {
				sync.SyncDeviceLog()
			}
			elapsed := time.Since(start)
			logging.CommonLogger.Info("完成日志扫描耗时：", elapsed)
		}()

		c.String(http.StatusOK, "执行成功")
	} else {
		c.String(http.StatusOK, "参数错误，未执行同步")
	}

}
