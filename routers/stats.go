package routers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
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
		err := utils.GetToken()
		if err != nil {
			c.String(http.StatusOK, "get token err %v", err)
			return
		}
		sync.NotFirst = true
		go func() {
			start := time.Now() // 获取当前时间
			sync.CheckRestful()
			sync.CheckService()
			// 进入当天目录,跳过 23点45 当天凌晨 0点15 分钟，给设备创建目录的时间
			if !((time.Now().Hour() == 0 && time.Now().Minute() < 15) || (time.Now().Hour() == 23 && time.Now().Minute() > 45)) {
				sync.SyncDeviceLog()
			}

			elapsed := time.Since(start)
			fmt.Println("该函数执行完成耗时：", elapsed)
		}()

		c.String(http.StatusOK, "执行成功")
	} else {
		c.String(http.StatusOK, "参数错误，未执行同步")
	}

}
