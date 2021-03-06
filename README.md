# LogSync

查询日志信息
- 注意事项，注册为服务执行 tasklist 会报错用户名或密码错误
#### 注册/卸载服务/启动/停止/重启

```shell script
LogSync.exe -action version 查看版本号

```

#### 配置
- 见 config.example.yaml 

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

3.程序以服务形式运行，执行 tasklist 命令提示账号密码错误

```text
 windows系统下，程序以服务形式运行，执行 tasklist 命令提示账号密码错误，直接运行程序能够正常执行。
 经过测试后发现，需要在服务注冊后设置登录账号和密码，才能正常执行 tasklist pscp 等命令。

 更新应用后需要设置账号密码，（容易忘记）

 同时，tasklist 等命令返回的错误信息编码为 gbk 需要转码才能显示
```

4.warning: remote host tried to write to a file called '2021-02-26' when we requested a file called ''. If this is a wildcard, consider upgrading to SSH-2 or using the '-unsafe' option. Renaming of this file has been disallowed.
```text
 ssh-1 版本情况下复制多个文件的时候会提示警告，需要增加 -unsafe 选项解决。或者升级为 ssh-2 版本。
```

#### 热重启
- 暂时无法热重启，推荐方案endless,grace;都是用于http 服务热更新。
- 热重启无法自动设置 windows 账号密码。
- 可选方案：下载 update.exe 文件并执行，然后由 update.exe 完成程序更新（停止-删除文件-下载文件-启动）。

#### 编译
-w 忽略DWARFv3调试信息，使用该选项后将无法使用gdb进行调试。
-s 忽略符号表和调试信息。

```shell script
go build -ldflags "-w -s -X main.Version=v1.9"  -o ./cmd/LogSync.exe main.go

GOOS=windows GOARCH=amd64 CC=/usr/local/bin/x86_64-w64-mingw32-gcc CXX=/usr/local/bin/x86_64-w64-mingw32-g+ go build -ldflags " -w -s -X main.Version=v2.27"   -o ./cmd/LogSync.exe
```

#### 模拟日志
- del  删除日志文件
- device 设备日志时间和服务器时间不一致
- plugin 插件故障
- real 正常

```shell script
go run ./mocklog/main.go ./mocklog/path.go ./mocklog/fault.go  -action del
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
- v2.21 修复设备推送数据跳过无设备目录，PDA 故障状态不更新问题
- v2.22 修改日志超时判断，去除本地存储数据，根据远程日志记录时间比对，增加 ip ping 方式检查设备
- v2.23 增加日志文件时间字段，增加监控设备是否过滤判断
- v2.24 重构日志同步监控程序
- v2.25 修复护理大屏日志文件解析报错 增加 interface.log 统计  ，interf code -1 不报故障 ，大屏 mqtt 断开增加ping操作
- v2.26 修复日志超时插件状态还是正常问题，设备日志超时后置空日志插件内容，优化程序内存使用减少变量重复创建摧毁
- v2.27 增加 cmd 命令执行超时逻辑，防止程序进程阻塞,合并了执行 cmd 命令逻辑为一个方法，增加单元测试，修改同步服务返回数据显示
- v2.28 修复 interf 插件 code -1 还会报故障问题,优化超时扫描日志逻辑
- v2.29 去除 time.Ticker 定时处理，改为 time sleep。
     增加 3 点钟，重启服务，跳过 30 分钟（改为后台运维时间控制报警）。
     去除注冊服务依赖包，改用 nssm.exe 注冊 windows 服务。增加插件状态码code参数值。
- v2.30 简化日志错误消息，调整ip检查位置到最开始


