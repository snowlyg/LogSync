package routers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/snowlyg/LogSync/sync"
	"net/http"
)

func SyncLog(c *gin.Context) {
	syncLog := c.Query("sync_log")
	println(fmt.Sprintf("sync_log:%d", syncLog))
	if syncLog == "1" {
		go func() {
			sync.SyncDevice()
		}()

		go func() {
			sync.SyncDeviceLog()
		}()

		c.String(http.StatusOK, "成功执行同步")
	} else {
		c.String(http.StatusOK, "参数错误，未执行同步")
	}

}
