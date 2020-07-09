package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/snowlyg/LogSync/sync"
	"net/http"
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
	//println(fmt.Sprintf("sync_log:%d", syncLog))
	if syncLog == "1" {
		go func() {
			sync.SyncDeviceLog()
			sync.CheckDevice()
		}()

		c.String(http.StatusOK, "设备日志成功执行同步")
	} else {
		c.String(http.StatusOK, "参数错误，未执行同步")
	}

}
