# 山特UPS-RS232协议使用TCP接入

这是一个用于ThingsPanel的协议插件，提供接入山特UPS-RS232的可能，现支持CKS系列WA及Q6指令对已知指令经行了解析。

## 特性

- 内置日志系统，支持文件轮转
- MQTT客户端集成
- 设备管理和缓存机制
- 表单配置管理
- HTTP服务支持
- 优雅的错误处理
- 配置文件管理

## 目录结构

```text
.
├── cmd/                  # 主程序入口
│   └── main.go           # 主程序
├── configs/              # 配置文件目录
│   └── config.yaml       # 主配置文件
├── internal/             # 内部包
│   ├── config/           # 配置结构定义
│   ├── form_json/        # 表单JSON定义
│   ├── handler/          # HTTP处理器
│   ├── tcpserver/        # TCP处理器
│   ├── pkg/              # 通用包
│   │   └── logger/       # 日志包
│   └── platform/         # 平台交互
└── go.mod                # Go模块文件
```

## 上报遥感数据

### 1. WA

```
"loadpower"              #负载功率
"loadvirtualpower"       #负载虚功率
"loadpercentage"         #负载百分比
"utilityfailstatus"      #实用程序失败状态
"batterylowstatus"       #低电池电压状态
"bypassstatus"           #旁路状态
"upsfailedstatus"        #UPS故障状态
"upstypestatus"          #备用状态
"testinprogressstatus"   #测试进行中状态
"shutdownstatus"         #关闭状态
```

### 2. Q6

```
"batterylevel"           #电池电量
"batterytemperature"     #电池温度
"outputvoltage"          #输出电压
"inputfrequency"         #输入频率
"outputfrequency"        #输出频率
"batteryvoltage"         #电池电压
"inputvoltage"           #输入电压
```

## 规范

- 官方插件开发说明文档

[http://thingspanel.io/zh-Hans/docs/system-development/eveloping-plug-in/customProtocol](http://thingspanel.io/zh-Hans/docs/system-development/eveloping-plug-in/customProtocol)
