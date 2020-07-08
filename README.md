# LogSync
查询日志信息

#### 注册/卸载服务/启动/停止/重启

```shell script

LogSync.exe install 
LogSync.exe remove 
LogSync.exe start
LogSync.exe stop
LogSync.exe restart

```

#### 配置

```ini
[config]
;exts 后缀
exts=.log,.txt

;根目录名称，默认 log
root=log

;外网服务器配置
host=test.ims.com:80
appid=wxcw846rde12w3fb9p
appsecret=489r67esrqa341vcnb1m16azfm789nbmnhj8

[ftp]
;ftp 服务器ip,账号，密码
ip=10.0.0.23
username=admin
password=Chindeo

[time]
;h,m,s 对应时分秒
;扫描日志
sync_log_time=4
sync_log=m
;同步数据到外网
sync_data_time=1
sync_data=h

[web]
; 大屏ip 多个大屏用 , 分隔
ip=10.0.0.149,
account=administrator
password=chindeo888

[android]
account=root
password=Chindeo

[http]
port=8001

```

##### 手动执行
http://localhost:8001/sync_log?sync_log=1 // 通讯录，设备
http://localhost:8001/sync_device_log?sync_log=1 // 设备日志

#### 问题
```text
运行一段时间后报错bind: An operation on a socket could not be performed because the system lacked sufficient buffer sp

上网搜查后确定问题源:代码连接端口的频次超出windows默认最大值

当然其中牵扯最大连接数量，起始中止端口号，释放连接资源时间（windows10默认120s）
```