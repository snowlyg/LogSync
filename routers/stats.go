package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/snowlyg/LogSync/sync"
	"net/http"
	"time"
)

func SyncDeviceLog(c *gin.Context) {
	syncLog := c.Query("sync_log")
	//println(fmt.Sprintf("sync_log:%d", syncLog))
	if syncLog == "1" {
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
		go func() {
			// 进入当天目录,跳过 23点45 当天凌晨 0点15 分钟，给设备创建目录的时间
			if !((time.Now().Hour() == 0 && time.Now().Minute() < 15) || (time.Now().Hour() == 23 && time.Now().Minute() > 45)) {
				sync.SyncDeviceLog()
			}
			sync.CheckDevice()
		}()

		c.String(http.StatusOK, "设备日志成功执行同步")
	} else {
		c.String(http.StatusOK, "参数错误，未执行同步")
	}

}
