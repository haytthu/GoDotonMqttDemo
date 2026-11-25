# SG500Z MQTT Demo · Go 版

`backend/` 是 Go 后端，`frontend/` 是 Java demo 原有的 Vue 前端。MQTT 报文见 [MQTT 协议 1.32](MQTT%E5%8D%8F%E8%AE%AE1.32.md)。

[变更说明 / Changelog](CHANGELOG.md)

## 启动

使用 Go 1.23.4 或以上版本、Node.js 24，以及一个 Go 后端和采集器都能访问的 MQTT Broker。

在项目根目录打开 PowerShell，启动后端。

```powershell
Set-Location backend
$env:MQTT_BROKER = 'tcp://127.0.0.1:1883'
$env:MQTT_PREFIX = 'iesd'
go run .
```

另开一个 PowerShell 窗口，在项目根目录启动前端。

```powershell
Set-Location frontend
npm ci
npm run dev -- --host 127.0.0.1
```

打开 [前端页面](http://127.0.0.1:5173/)。[健康接口](http://127.0.0.1:8080/api/health) 的 `brokerConnected` 字段显示 Broker 连接状态。

### 环境变量

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `MQTT_BROKER` | `tcp://127.0.0.1:1883` | Broker 地址 |
| `MQTT_PREFIX` | `iesd` | 与采集器一致的 MQTT 主题前缀 |
| `MQTT_CLIENT_ID` | `sg500z-demo` | 后端 MQTT 客户端 ID |
| `MQTT_USERNAME`、`MQTT_PASSWORD` | 空 | Broker 登录凭据 |
| `MQTT_UPGRADE_LIST_PAYLOAD` | `true` | 默认发送 Mars 固件数组格式。采集器使用单对象协议时设为 `false` |
| `DEMO_BIND_ADDRESS` | `127.0.0.1` | 后端监听地址 |
| `DEMO_PORT` | `8080` | 后端端口 |
| `DEMO_API_URL` | `http://127.0.0.1:8080` | 前端开发和预览服务器的代理目标 |
| `DEMO_DATA_DIR` | `./data` | 后端数据目录，相对于 Go 进程的工作目录 |
| `DEMO_ZONE` | `Asia/Shanghai` | 采集和报文时间使用的时区 |

## 接入采集器

1. 在设备建模树上右键新建组织、设备和测点。新建三轴振动测点时会创建 Z、X、Y 三个测点，可以按轴配置报警并查询数据。
2. 配置 SG500Z，填写采集器在 MQTT 主题中使用的编号，把物理通道绑定到对应测点。支持基础通道、RTU、Modbus TCP 和 PT100。
3. 保存采集参数，供采集器下次请求时读取。对在线设备，可点击「下发参数」通知设备拉取。采集器注册后，页面可查看设备是否已拉取。
4. 在「采集数据」查看指标、波形、频谱、音频、传感器和电池上报。在「日志管理」查询通信记录并下载采集器上传的日志。
5. 上传与设备匹配的固件后，可发起升级并查看下载进度，以及设备重启后的版本核对结果。

## 数据迁移

从 `backend/` 启动时，默认使用 `backend/data/`。配置和数据索引存放于 `sg500z.sqlite`，波形、音频、固件、服务日志和采集器上传日志分别存放于 `waves/`、`audio/`、`firmware/`、`logs/` 和 `collector-logs/`。

如需沿用 Java demo 的数据，先停止 Java 后端，再将其完整的 `backend/data/` 复制到 Go 项目的 `backend/data/`，或者通过 `DEMO_DATA_DIR` 指向数据副本。
