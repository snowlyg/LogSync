# LogSync

查询日志信息

#### 注册/卸载服务/启动/停止/重启

```shell script

LogSync.exe install 
LogSync.exe remove 
LogSync.exe start
LogSync.exe stop
LogSync.exe restart
LogSync.exe version 查看版本号

```

#### 配置

```ini
[config]
;exts 后缀
exts = .log,.txt

;根目录名称，默认 log
root = log

;外网服务器配置
host = test.ims.com:80
appid = wxcw846rde12w3fb9p
appsecret = 489r67esrqa341vcnb1m16azfm789nbmnhj8

[ftp]
;ftp 服务器ip,账号，密码
ip = 10.0.0.23
username = admin
password = Chindeo

[time]
;h,m,s 对应时分秒
;扫描日志
sync_log_time = 4
sync_log = m
;同步数据到外网
sync_data_time = 1
sync_data = h

[web]
; 大屏ip 多个大屏用 , 分隔
ip = 10.0.0.149,
account = administrator
password = chindeo888

[android]
account = root
password = Chindeo

[http]
port = 8001


[mysql]
;只有访问权限的账号密码
account=visible
pwd=Chindeo
```

##### 手动执行（已去除）

http://localhost:8001/sync_log?sync_log=1 // 通讯录，设备
http://localhost:8001/sync_device_log?sync_log=1 // 设备日志

#### 问题

1.

```text
运行一段时间后报错bind: An operation on a socket could not be performed because the system lacked sufficient buffer sp

上网搜查后确定问题源:代码连接端口的频次超出windows默认最大值

当然其中牵扯最大连接数量，起始中止端口号，释放连接资源时间（windows10默认120s）
```

2.

```text
 日志超时15分钟，主动查询设备信息逻辑耗时较多
```

3.接口监控返回数据格式


```json5
 {
  "code": 0, // 200代表成功，其他均是异常
  "message": "接口无法访问"
}
```

#### 编译

```shell script
go build -ldflags "-w -s -X main.Version=v1.9  -o ./cmd/LogSync.exe

CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=/usr/local/bin/x86_64-w64-mingw32-gcc CXX=/usr/local/bin/x86_64-w64-mingw32-g+ go build -ldflags " -w -s -X main.Version=v2.1" -race  -o ./cmd/LogSync.exe
```

#### 版本更新

- v1.1 增加 pscp 输入参数 y,重启程序后不检查超时15分钟
- v1.2 修改设备数据请求方式未本地方式
- v1.3 增加初次启动不报故障的判断
- v1.4 优化服务监控连接，修改为5秒超时
- v1.5 尝试调试服务自动退出，目前推测试服务监控和日志扫描同时连接 ftp,mysql 导致，解决方案去除服务监控 ftp,mysql 连接。
- v1.6 调整日志为一次性提交，修改 FTP 超时时间
- v1.7 增加文件异常报错后自动恢复程序
- v1.8 增加服务报错3次才上报，增加 ftp 连接超时宕机恢复
- v1.9 增加接口监控，修改请求为并发请求
- v2.0 跳过检测时间增加为凌晨1点
- v2.1 修复日志打印数据竞争,去除手动同步接口
- v2.2 增加设备离线设备提醒，增加设备离线状态字段，方便后台实时更新设备状态
- v2.21 修复设备推送数据跳过无设备目录
