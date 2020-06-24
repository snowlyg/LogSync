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

```txt

[config]

;exts 后缀
exts=.log,.txt

;根目录名称，默认 log
root=log

;外网服务器配置
host=fyxt.t.chindeo.com:80
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
sync_log=s
;同步数据到外网
sync_data=s

[web]
; 大屏ip 多个大屏用 , 分隔
ip=10.0.0.149,
account=administrator
password=chindeo888

[android]
account=root
password=Chindeo

```

##### 手动执行
http://localhost:8001/synclog?sync_log=1