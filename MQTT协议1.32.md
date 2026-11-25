# 






| 日期 | 修订内容 | 修订人 |
| --- | --- | --- |
| 2023/09/26 | 首次制定 | Xu |
| 2023/10/08 | 采集参数振动通道添加转频的转差率、极对数两个参数 | Xu |
| 2023/10/09 | 1.1添加同步指标上传策略(仅MS8XX使用) | Xu |
| 2023/10/25 | 1.2未注册情况下直接请求其他的命令将返回错误码-7 | Xu |
| 2023/11/16 | 1.2增加MS810儒海定制的部分参数 | Hu |
| 2023/12/11 | 1.2删除RH部分定制参数，采集参数中增加工艺量报警门限 | Hu |
| 2023/12/18 | 1.3 添加4G卡号信息，添加波形压缩 | Xu |
| 2023/12/29 | 1.4 添加低限报警门限 | Xu |
| 2024/01/23 | 1.5添加压缩波形解析异常返回错误码 | Xu |
| 2024/01/23 | 1.6增加MS810在上传波形数据时，增加波形报警等级的上传 | Hu |
| 2024/02/06 | 1.7触发采集参数增加峭度冲击门限 | Hu |
| 2024/02/23 | 1.8针对MS810增加无线传感器的指标间隔以及波形间隔的配置 | Hu |
| 2024/03/25 | 1.9 添加设备事件日志。 | Xu |
| 2024/05/20 | 1.10添加注指器相关支持 | Xu |
| 2024/5/27 | 1.11添加4G报警触发逻辑 | Xu |
| 2024/6/4 | 1.12 添加轴承健康指数支持 | Xu |
| 2024/6/21 | 1.13 添加800触发采集参数 | Xu |
| 2024/8/22 | 1.14 智能电表数据传输协议 | Xu |
| 2024/8/28 | 1.15 添加了 启停机状态 | Xu |
| 2024/9/6 | 1.16 停机门限添加变化率 | Xu |
| 2024/9/19 | 1.16.1 添加湿度通道 | Xu |
| 2024/10/17 | 1.16.2 MS810增加开关量触发采集 | Hu |
| 2024/12/30 | 3.3 采集参数中振动通道中数据增加注脂器SN | Hu |
| 2025/04/09 | 1.20 采集参数添加采集圈数和每圈采样点数 | Xu |
| 2024/04/25 | 1.21 转速通道添加采样圈数、转速报警门限下发 | Xu |
| 2024/05/12 | 1.22 系统参数增加工作时间配置，采集参数增加通道中控制传感器是否运行 | Hu |
| 2024/05/23 | 1.23 添加是否分包传输控制 | Xu |
| 2024/05/29 | 1.23 添加润滑联动控制 | Hu |
| 2024/06/21 | 1.24 添加多通道工艺量 | Xu |
| 2024/07/02 | 1.25 添加MS810工艺板卡参数 | Hu |
| 2025/10/29 | 1.27 添加压力测点配置支持，当使用压力传感器时，采集参数中增加灵敏度系数及修正项下发 | Hu |
| 2025/12/10 | 1.28 系统参数中增加偏置电压数据检查，不赋值及不等于2000上传偏置电压异常数据，赋值2000时，不上传偏执电压异常数据（目前仅MS810使用） | Hu |
| 2026/1/6 | 增工况传感器类型，控制下位机是否传输转频数据 | Hu |
| 2026/1/8 | 增加音频传感器类型 | Hu |
| 2026/1/12 | MS810增加定时等待模式，等待上位机下发开始或停止控制指令 | Hu |
| 2026/3/4 | MS810增加RTU数据触发采集 | Hu |


# 1、概述
本协议基于MQTT3.1.1设计。

采集器数据用于监测设备运行，为了严格确保数据不丢失，本协议采用**请求响应模式**。

## 1.2、设备启动流程说明
<!-- 这是一张图片，ocr 内容为： -->
![](https://cdn.nlark.com/yuque/0/2023/png/282059/1695783322096-e34a4103-e2a6-4bc7-be97-ea9f8dcd899e.png)

### 1.2.1参数请求流程：
设备正常开机后会请求系统参数，通过判断系统参数中的参数时间（与上一次的参数时间对比）是否变化，如果有变化则会请求采集参数（如果没有变化，则不会请求采集参数），请求到采集参数后会重启设备，重复此流程。



### 1.2.2 升级文件请求流程
设备开机后会请求升级文件列表，如果列表内有升级文件（请求到的数组不为空数组），则认为需要升级，会依次请求升级文件，升级结束后会自动重启设备，重复此流程。

**<font style="color:#DF2A3F;">注：如果每一次参数时间不一样，就会陷入每次都需要请求参数的死循环，每次都有升级文件也是如此。</font>**

# 2、主题说明
### 2.1.变量
**前缀**：灵活配置的路径前缀，方便统一订阅，也可以根据用户需求进行设置。下文用[Prefix]表示

**采集器编码**：采集器唯一标识，下文用[<font style="color:black;">Code</font>]表示

**<font style="color:black;">业务关键字</font>**<font style="color:black;">：标识某一类命令的字符串代号，如</font><font style="color:black;">Register</font>

**<font style="color:black;">通讯</font>****<font style="color:black;">Id</font>****<font style="color:black;">：</font>**<font style="color:black;">标识单次应答过程的唯一</font><font style="color:black;">Id</font><font style="color:black;">，下文用</font><font style="color:black;">[MessageId]</font><font style="color:black;">表示</font>

**<font style="color:black;">通讯回复状态</font>**<font style="color:black;">：标识本次执行是否按照预期执行,0-失败，1-成功，特殊失败错误需要返回的详见错误编码表，下文用[Status]表示</font>

### <font style="color:black;">2.2.错误编码表</font>
| <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- |
| <font style="color:black;">1</font> | <font style="color:black;">成功</font> |
| <font style="color:black;">0</font> | <font style="color:black;">失败</font> |
| <font style="color:black;">-1</font> | <font style="color:black;">CRC</font><font style="color:black;">检验失败</font> |
| <font style="color:black;">-2</font> | <font style="color:black;">未找到通道对应测点</font> |
| <font style="color:black;">-3</font> | <font style="color:black;">Json</font><font style="color:black;">解析出错</font> |
| <font style="color:black;">-4</font> | <font style="color:black;">测点的采集配置异常</font> |
| <font style="color:black;">-5</font> | <font style="color:black;">快速波形传输</font><font style="color:black;">包不完整</font> |
| <font style="color:black;">-6</font> | <font style="color:black;">波形传输系数为</font><font style="color:black;">0</font> |
| -7 | 未请求注册指令 |
| -8 | 波形压缩数据异常 |


### 2.3.规则
采集站订阅消息主题：<font style="color:black;">[Prefix]/[Code]/</font><font style="color:black;">业务关键字</font><font style="color:black;">/[MessageId]/[Status]</font>

<font style="color:black;">采集站建议统一订阅：</font><font style="color:black;">[Prefix]/[Code]/#</font>

<font style="color:black;"></font>

上位机订阅消息主题：<font style="color:black;">[Prefix]/Online/[Code]/</font><font style="color:black;">业务关键字</font><font style="color:black;">/[MessageId]/[Status]</font>

<font style="color:black;">上位机建议统一订阅：[Prefix]/Online/#</font>

<font style="color:black;"></font>

# 3、协议
## 3.1.注册
### 3.1.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/Register/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">RegisterInfo</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组，</font>**<font style="color:black;">编码方式为</font>****<font style="color:black;">UTF-8</font>****<font style="color:black;">，下同</font>**<font style="color:black;">。</font> |




### 3.1.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/Register/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |




_RegisterInfo __注册信息_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| <font style="color:black;">Ver</font> | <font style="color:black;">string</font> | <font style="color:black;">协议版本号</font> |
| <font style="color:black;">Code</font> | <font style="color:black;">string</font> | <font style="color:black;">采集站编码</font><font style="color:red;"></font> |
| <font style="color:black;">Mac</font> | <font style="color:black;">string</font> | <font style="color:black;">Mac</font><font style="color:black;">地址</font> |
| <font style="color:black;">SN</font> | <font style="color:black;">string</font> | <font style="color:black;">采集器</font><font style="color:black;">SN</font> |
| <font style="color:black;">ConnType</font> | <font style="color:black;">int</font> | <font style="color:black;">采集器的通信模式（</font><font style="color:black;">0-</font><font style="color:black;">未知，</font><font style="color:black;">1-4g</font><font style="color:black;">，</font><font style="color:black;">2-wifi</font><font style="color:black;">，</font><font style="color:black;">3-</font><font style="color:black;">以太网）</font> |
| <font style="color:black;">DasType</font> | <font style="color:black;">Int</font> | <font style="color:black;">采集器类型（0-无线SG500Z；20-有线MS800；21-设备云盒 ；22-电机云盒；  100-NB101； 120-LoraSG500L； 24-MS810； 101-Cat1一代； 102-Cat1二代； 103-Cat1一三代； 106-CT610M； 25-EC710； 26-WT81M； 27-SI500智能注指器  ）</font> |
| <font style="color:black;">BoardInfoList</font> | <font style="color:black;">BoardInfo[]</font> | <font style="color:black;">板卡信息列表</font> |
| LocalIp | string | 本机IP |
| ServerIp | String | 服务器IP（APP配置） |
| GateWay | string | 网关地址 |
| SubnetMask | String | 子网掩码 |
| ServerPort | Int | 服务器端口（APP） |
| RSSI | Int | 信号强度 |
|  |  |  |
| LteInfo | LteInfo | 4、5G信息 |


_LteInfo 4、5G信息_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | --- |
| Revision | string | 模块版本 |
| ICCID | string | ICCID |
| IMEI | string | 模块IMEI号 |
| RSRP | Int |  |
| RSRQ | Int |  |
| RSSI | Int |  |
| SNR | Int | 信噪比 |
| Latitude | float | 维度 |
| Longitude | float | 经度 |


_BoardInfo __板卡信息_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Code | int32 | 板卡编码 |
| AppVer | string | 软件版本 |
| HardVer | string | 硬件版本 |
| IMEI | string | 模块IMEI号 |


### 3.1.3采集器请求示例
```json
{
  "Ver": "V1.0.0",
  "Code": "80001002",
  "Mac": "00:58:22:10:10:10",
  "SN": "23020037",
  "LocalIp": "192.168.0.71",
  "ServerIp": "192.0.2.10",
  "GateWay": "192.168.0.1",
  "SubnetMask": "255.255.255.0",
  "ServerPort": 1883,
  "ConnType": 2,
  "DasType": 20,
  "RSSI": -35,
  "BoardInfoList": [
    {
      "Code": 1,
      "AppVer": "8.16.50.28P25",
      "HardVer": ""
    },
    {
      "Code": 5,
      "AppVer": "1.P09ST",
      "HardVer": ""
    },
    {
      "Code": 6,
      "AppVer": "1.14",
      "HardVer": ""
    },
    {
      "Code": 33,
      "AppVer": "1.11",
      "HardVer": ""
    },
    {
      "Code": 98,
      "AppVer": "V3.0.1",
      "HardVer": ""
    }
  ]
}
```



## 3.2.请求策略参数
### 3.2.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/SysPara/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


### 3.2.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/SysPara/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">SysPara</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


_SysPara 策略参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| <font style="color:black;">SysTime</font> | <font style="color:black;">Datetime</font> | <font style="color:black;">系统时间</font> |
| <font style="color:black;">ParaTime</font> | <font style="color:black;">Datetime</font> | <font style="color:black;">参数时间</font> |
| <font style="color:black;">IndexPeriod</font> | <font style="color:black;">int</font> | <font style="color:black;">指标间隔</font><font style="color:black;">(s)</font> |
| <font style="color:black;">WavePeriod</font> | <font style="color:black;">int</font> | <font style="color:black;">波形间隔</font><font style="color:black;">(s)</font> |
| <font style="color:black;">LongWavePeriod</font> | <font style="color:black;">int</font> | <font style="color:black;">长波形间隔</font><font style="color:black;">(s)</font> |
| <font style="color:black;">LongWaveOffset</font> | <font style="color:black;">int</font> | <font style="color:black;">长波形时间偏移(s)</font> |
| <font style="color:black;">CollectType</font> | <font style="color:black;">int</font> | <font style="color:black;">MS8XX：采集类型(</font><font style="color:red;">1-定时，2-连续</font><font style="color:black;">)  6- MS810触发采集 7-定时等待模式（获取策略参数后等待上位机下发采集指令）</font><br/><font style="color:black;">CS20M：温度唤醒模式：1 -- 常规模式 3--秒级温度唤醒  4 --奎光云定制（仅CS20M有效） 5--秒级触发 </font> |
| <font style="color:black;">OutType485</font> | <font style="color:black;">int</font> | <font style="color:black;">485</font><font style="color:black;">输出类型（</font><font style="color:black;">0-</font><font style="color:black;">高频；</font><font style="color:black;">1-</font><font style="color:black;">低频；</font><font style="color:black;">2</font><font style="color:black;">速度）</font> |
| TechPeriod | int | 工艺数据采集间隔（单位：s）（MS810使用这个字段控制温度及转速数据间隔） |
| RpmPeriod | int | 转速数据采集间隔（单位：s） |
| RtuPeriod | int | Rtu数据轮询间隔（单位：s） |
| UploadPeriod | int | 数据上传周期（单位：s） |
| AlarmCount | int | 连续报警次数 <font style="color:#DF2A3F;">（定制版本属性，请忽略）</font> |
| WaveCompress | int | 是否使用压缩波形传输；0--不支持压缩  1 支持压缩 |
| UserCustom | int | 用户自定义字段 一般传 0 |
| WlessIndexPeriod | int | MS810使用，无线传感器的指标间隔 |
| WlessWavePeriod | int | MS810使用，无线传感器的波形间隔 |
| CircleCount | int | 采集圈数（定制版本） |
| CirclePointCount | int | 每圈采样点数（定制版本） |
| WorkTime | string | 仅包含时分秒，开始工作时间（81M） |
| SleepTime | string | 仅包含时分秒，休眠时间（81M） |
| UploadSegment | int | 0:不分包传输(4G传感器强制分包，不生效)  1：分包传输 |
| IsLub | int | 是否控制注脂器0:表示不控制，1:表示控制 |
| BiasCheck | int | 是否检查偏置电压，2000时不上传偏置电压异常数据，不为2000以及不赋值不生效 |


### 3.2.3 上位机返回参数
```json
{
  "SysTime": "2023-09-27T11:49:35",
  "ParaTime": "2023-09-26T14:01:07",
  "IndexPeriod": 5,
  "TechPeriod": 5,
  "WavePeriod": 1800,
  "LongWavePeriod": 86400,
  "LongWaveOffset": 3600,
  "CollectType": 1,
  "OutType485": 0,
  "RpmPeriod": 5,
  "RtuPeriod": 5,
  "UserCustom": 0,
  "UploadPeriod": 7200,
  "WlessIndexPeriod":600,
  "WlessWavePeriod":7200
}
```

__

## 3.3.请求采集参数
### 3.3.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/CollectPara/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


### 3.3.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/CollectPara/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">Byte[]</font> | <font style="color:black;">CollectPara</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |




_CollectPara __采集参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| VibChannelList | VibChannalPara[] | 振动/音频通道列表 |
| TechChannelList | TechChannelPara[] | 工艺通道列表（有线） |
| SpeedChannelList | SpeedChannelPara[] | 转速通道列表（有线） |
| RtuChannelList | RtuChannelPara[] | Modbus Rtu通道列表 |
| TriggerList | TriggerPara[] | 触发采集参数列表（有线及4G的主动探测） |
| StopLimitList | StopLimitInfo[] | 启停机门限 |
| ModbusTCPChannelList | ModbusTCPChannelPara[] | ModbusTCP通道 |
|  |  |  |


_VibChannalPara __振动通道_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| ChNo | int | 振动通道号 |
| SpeedNo | int | 转速通道号（振动关联转速）有线 |
| SensType | int | 传感器类型（0-加速度； 1-位移;2-压力，3-速度，4-工况，5-音频） |
| SensCoef | float | 灵敏度系数（有线） |
| CnstCurSrcState | int | 控制恒流源状态（0-关闭；1-打开）（有线） |
| VibParaList | VibPara[ ] | 振动采集参数列表 |
| LimitParaList | LimitPara[ ] | 报警门限参数（如果不涉及报警加采可传空数组） |
| DevId | Int | 设备id（有线） |
| IsTem | Int | 是否测温（0 --不测温 1--测温） |
| SpeedDiffRate | float | 转差率 （用于磁力计计算转频使用） |
| PolePairsNum | int | 极对数（用于磁力计计算转频使用） |
| LubSn | string | 绑定润滑的SN号 |
| IsRunning | int | 0-表示关闭，1-表示运行 |
| Sensitivity | float | 灵敏度系数-压力测点使用 |
| Amendment | float | 修正项-压力测点使用 |




_VibPara __振动采集参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Code | int | 采集定义编码<br/>0-音频波形；1-长波形；11-高频波形；12-低频波形；<br/>15-速度波形 |
| LowFreq | int | 下限分析频率（HZ） |
| HighFreq | int | 上限分析频率（HZ）-该频率用于采集 |
| Length | int | 采集长度 |
| PlusLength | int | 附加长度 |
| Unit | string | 工程单位（m/s^2） |
| Dir | int | 方向（0-单轴；1-Z轴；2-X轴；3-Y轴） |




_LimitPara __报警门限参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| Code | int | 指标类型（100-温度101-低频总值107-速度总值112-高频总值，202-峰值206-均值217-标准差） |
| Level | int | 报警等级 |
| Limit | float | 报警门限 |
| Dir | Int | 方向（0-单轴；1-Z轴；2-X轴；3-Y轴） |
| Type | int | 门限类型,字段，0-超限，1-低限，2-门外，3-门内。4-相对。 当前默认为0 |






_TechChannelPara 工艺通道参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">ChNo</font> | <font style="color:black;">int</font> | <font style="color:black;">工艺通道号</font> |
| <font style="color:black;">SensCoef</font> | <font style="color:black;">float</font> | <font style="color:black;">灵敏度系数</font> |
| <font style="color:black;">Amendment</font> | <font style="color:black;">float</font> | <font style="color:black;">修正项</font> |
| <font style="color:black;">DevId</font> | <font style="color:black;">Int</font> | <font style="color:black;">设备</font><font style="color:black;">id</font> |
| LimitParaList | LimitPara[ ] | 报警门限参数（如果不涉及报警加采可传空数组） |


_LimitPara __报警门限参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Code | int | 指标类型（0-工艺量默认类型） |
| Level | int | 报警等级 |
| Limit | float | 报警门限 |
| Dir | Int | 方向（0-默认方向） |




_SpeedChannelPara __转速通道参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">ChNo</font> | <font style="color:black;">int</font> | <font style="color:black;">转速通道号</font> |
| <font style="color:black;">TriggerCount</font> | <font style="color:black;">int</font> | <font style="color:black;">每圈触发数</font> |
| <font style="color:black;">CircleCount</font> | Int | 采样圈数（扭振） |
| <font style="color:black;">DevId</font> | <font style="color:black;">Int</font> | <font style="color:black;">设备</font><font style="color:black;">id</font> |
| LimitParaList | LimitPara[ ] | 报警门限参数（如果不涉及报警加采可传空数组，扭振） |




_RtuChannelPara Rtu通道参数_

| 属性 | 类型 | 说明 |
| :--- | :--- | :--- |
| ChNo | int | Rtu通道号 |
| RTUDeviceAddr | int | 设备地址 |
| FunCode | int | 功能码  |
| RegAddr | int | 寄存器起始地址 |
| DataLength | int | 数据长度 |
| DataType | List<int> | 数据类型列表（0-Bool，1-Short，2-Double，3-Int32，4-UInt32，5-Float） |
| DevId | Int | 设备id |




_TriggerPara __触发采集参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Freq | int | 分析频率(HZ) |
| Length | int | 采集长度 |
| TriggerGroup | int | 触发采集组数 |
| TriggerType | int | 触发类型  1-高频加速度；2-速度；3-转速 4-高频加速度\转速；5-速度\转速； |
| TriggerLimitList | TriggerLimitPara [] | 触发门限列表 |
| ActiveDetectionCycle | int | 主动检测周期 |
| AlarmInterval | int | 报警间隔 |
| AlarmSamplingFrequency | int | 报警采样次数 |


<font style="color:blue;"></font>

_TriggerLimitPara __触发门限_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| Channels | Int[] | 通道集合（这些通道同步触发,多设备不同参数） |
| AccLimit | float | 加速度门限值  |
| VerLimit | float | 速度门限值 |
| RpmLimit | float | 转速门限值 |
| ShockLimit | float | 加速度冲击门限值 |
| KurLimit | float | 峭度冲击门限值 |


StopLimitList 启停机门限

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| ChNo | Int | 通道号 |
| ChType | int | 通道类型（0=振动 1:转速 2:工艺,3-温度，4-音频，5-RTU）（**<font style="color:#DF2A3F;">目前仅支持振动通道</font>**） |
| Code | int | 指标类型合集（0-默认； 100-温度；101-低频总值；107-速度总值；112-高频总值；）（**<font style="color:#DF2A3F;">目前仅支持振动速度</font>**） |
| StopLimit | float | 停机门限值(高于这个门限则视为启机状态) |
| StopLimitChangeRate | float | 变化率(未启用) |


ModbusTCPChannelPara ModbusTcp通道参数

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| ChNo | Int | 通道号 |
| DeviceAddr | int | 设备地址 |
| IPAddr | string | Ip地址（做主机时需要知道对方IP地址） |
| Port | int | 端口 |
| Status | int | 读写模式0-读，1-写 |
| FunCode | int | 功能码 |
| Mode | int | 模式（主机、从机） |
| StartAddr | int | 起始地址 |
| ModbusTCPParaList | ModbusTCPPara[] | 参数列表 |




ModbusTCPPara

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| DataType | Int | 数据类型0-bool，1-short，2-double，3-int，4-uint，5-float |
| RegAddr | int | 寄存器地址 |
| Coef_K | float | 转换系数 |
| Coef_B | float | 修正项 |


### 3.3.3 上位机返回参数（有线采集站单一通道demo）
```json
{
    "VibChannelList":[
        {
            "ChNo":1,
            "SpeedNo":0,
            "SensType":0,
            "CnstCurSrcState":0,
            "SensCoef":10,
            "DevId":1,
            "VibParaList":[
                {
                    "LowFreq":0,
                    "HighFreq":20000,
                    "Length":1048576,
                    "Unit":"m/s^2",
                    "PlusLength":0,
                    "Dir":0,
                    "Code":1
                },
                {
                    "LowFreq":0,
                    "HighFreq":20000,
                    "Length":131072,
                    "Unit":"m/s^2",
                    "PlusLength":0,
                    "Dir":0,
                    "Code":11
                },
                {
                    "LowFreq":0,
                    "HighFreq":2000,
                    "Length":16384,
                    "Unit":"m/s^2",
                    "PlusLength":8192,
                    "Dir":0,
                    "Code":12
                },
                {
                    "LowFreq":0,
                    "HighFreq":1000,
                    "Length":8192,
                    "Unit":"m/s^2",
                    "PlusLength":8192,
                    "Dir":0,
                    "Code":15
                }
            ],
            "LimitParaList":[

            ]
        }
    ],
    "TechChannelList":[

    ],
    "SpeedChannelList":[

    ],
    "TriggerList":[
      
    ],
    "RtuChannelList":[

    ],
}
```

### 3.3.4 上位机返回参数（无线网关单一通道参数）
```json
{
    "VibChannelList":[
        {
            "ChNo":1,
            "SpeedNo":0,
            "SensType":0,
            "CnstCurSrcState":0,
            "SensCoef":10,
            "DevId":1,
            "VibParaList":[
                {
                    "LowFreq":0,
                    "HighFreq":2000,
                    "Length":8192,
                    "Unit":"m/s^2",
                    "PlusLength":1024,
                    "Dir":2,
                    "Code":12
                },
                {
                    "LowFreq":0,
                    "HighFreq":1000,
                    "Length":4096,
                    "Unit":"m/s^2",
                    "PlusLength":1024,
                    "Dir":2,
                    "Code":15
                },
                {
                    "LowFreq":0,
                    "HighFreq":2000,
                    "Length":8192,
                    "Unit":"m/s^2",
                    "PlusLength":1024,
                    "Dir":3,
                    "Code":12
                },
                {
                    "LowFreq":0,
                    "HighFreq":1000,
                    "Length":4096,
                    "Unit":"m/s^2",
                    "PlusLength":1024,
                    "Dir":3,
                    "Code":15
                },
                {
                    "LowFreq":0,
                    "HighFreq":20000,
                    "Length":131072,
                    "Unit":"m/s^2",
                    "PlusLength":0,
                    "Dir":1,
                    "Code":1
                },
                {
                    "LowFreq":0,
                    "HighFreq":20000,
                    "Length":16384,
                    "Unit":"m/s^2",
                    "PlusLength":0,
                    "Dir":1,
                    "Code":11
                },
                {
                    "LowFreq":0,
                    "HighFreq":2000,
                    "Length":8192,
                    "Unit":"m/s^2",
                    "PlusLength":1024,
                    "Dir":1,
                    "Code":12
                },
                {
                    "LowFreq":0,
                    "HighFreq":1000,
                    "Length":4096,
                    "Unit":"m/s^2",
                    "PlusLength":1024,
                    "Dir":1,
                    "Code":15
                }
            ],
            "LimitParaList":[

            ]
        }
    ],
    "TechChannelList":[

    ],
    "SpeedChannelList":[

    ],
    "TriggerList":[
        {
        }
    ],
    "RtuChannelList":[

    ],
}
```

**<font style="color:red;">注：采集参数只给出了单个振动采集通道的情况下的样例，如果是多通道的情况，仅复制VibChannelList数组内的内容，然后修改通道号即可</font>**

**<font style="color:red;"></font>**

## 3.3.A 获取触发采集参数（仅在MS810触发采集下有效）
### 3.3.A.1 采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/MS810TriggerParams/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;"></font> | <font style="color:black;"></font> |


### 3.3.A.2 上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/MS810TriggerParams/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">MS810TriggerParam[]序列化为json后转为字节数组</font> |


<font style="color:black;">MS810TriggerParam 810 触发采集参数</font>

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| DevId | <font style="color:black;">int</font> | 设备id |
| <font style="color:black;">AlarmCount</font> | <font style="color:black;">int</font> | <font style="color:black;">进入报警次数（1为默认值）</font> |
| <font style="color:black;">AlarmOutCount</font> | <font style="color:black;">int</font> | 取消报警次数<font style="color:black;">（1为默认值）</font> |
| <font style="color:black;">ProcessChTriggers</font> | <font style="color:black;">ProcessChTrigger[]</font> | 工艺量通道触发参数 |
| <font style="color:black;">SpeedChTrigger</font> | <font style="color:black;">SpeedChTrigger</font> | 转速触发通道参数 |
| <font style="color:black;">SwitchChTriggers</font> | <font style="color:black;">SwicthChTrigger[]</font> | 开关量触发通道参数 |


<font style="color:black;">SpeedChTrigger工艺量参数</font>

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| ChNo | <font style="color:black;">int</font> | <font style="color:black;">通道号</font> |
| <font style="color:black;">LowLimit</font> | <font style="color:black;">int</font> | <font style="color:black;">低值门限（-1为默认值）</font> |
| <font style="color:black;">HighLimit</font> | <font style="color:black;">int</font> | <font style="color:black;">高值门限（-1为默认值）</font> |
| <font style="color:black;">Type</font> | <font style="color:black;">int</font> | <font style="color:black;">0窗内报警，1高值报警，2低值报警</font> |


<font style="color:black;">ProcessChTrigger工艺量参数</font>

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| ChNo | <font style="color:black;">int</font> | 通道号 |
| <font style="color:black;">LowLimit</font> | <font style="color:black;">float</font> | <font style="color:black;">低值门限（-1为默认值）</font> |
| <font style="color:black;">HighLimit</font> | <font style="color:black;">float</font> | <font style="color:black;">高值门限（-1为默认值）</font> |
| <font style="color:black;">Type</font> | <font style="color:black;">int</font> | <font style="color:black;">0窗内报警，1高值报警，2低值报警</font> |


<font style="color:black;">SwitchChTrigger工艺量参数</font>

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| ChNo | <font style="color:black;">int</font> | 通道号 |
| <font style="color:black;">DataFlag</font> | <font style="color:black;">int</font> | <font style="color:black;">0-上升沿，1-下降沿</font> |
| <font style="color:black;">Delay</font> | <font style="color:black;">int</font> | <font style="color:black;">延迟时间（s）</font> |


## 
## 3.3.B 获取润滑门限参数（仅在IsLub启用时请求）
### 3.3.B.1 采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/LubParams/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;"></font> | <font style="color:black;"></font> |


### 3.3.B.2 上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/LubParams/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">LubParams[]序列化为json后转为字节数组</font> |


<font style="color:black;">LubParams 注脂门限参数</font>

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| SN | <font style="color:black;">string</font> | 注脂器SN号 |
| LQ | <font style="color:black;">float</font> | LQ阈值 |
| <font style="color:black;">AlarmType</font> | <font style="color:black;">int</font> | <font style="color:black;">1:固定阈值判断，2:趋势阈值判断</font> |
| AlarmPercent | <font style="color:black;">float</font> | 阈值倍率 |
| <font style="color:black;">AlarmPeriod</font> | <font style="color:black;">int</font> | 周期间隔（单位天） |
| AlarmCapacity | <font style="color:black;">int</font> | 报警注脂容量 |




## 3.3.C 获取MS810工艺板卡采集参数（仅在MS810识别到电能及应力办卡时请求生效）
### 3.3.C.1 采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/MS810TechBoardParams/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;"></font> | <font style="color:black;"></font> |


### 3.3.C.2 上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/MS810TechBoardParams/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">MS810TechBoardParams[]序列化为json后转为字节数组</font> |


<font style="color:black;">MS810TechBoardParams 工艺板卡采集参数</font>

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| ChNo | int | 振动通道号 |
| SensCoef | float | 灵敏度系数（有线） |
| VibParaList | CollectPara[ ] | 振动采集参数列表 |
| DevId | Int | 设备id（有线） |




_CollectPara 振动采集参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Code | int | 采集定义编码<br/>0-音频波形；1-长波形；2-低频长波形；11-高频波形；12-低频波形；<br/>15-速度波形 |
| LowFreq | int | 下限分析频率（HZ） |
| HighFreq | int | 上限分析频率（HZ）-该频率用于采集 |
| Length | int | 采集长度 |
| PlusLength | int | 附加长度 |
| Unit | string | 工程单位（m/s^2） |
| Dir | int | 方向（0-单轴；1-Z轴；2-X轴；3-Y轴） |


## 3.4.A.传输指标数据
### 3.4.A.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/IndexData/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">IndexData</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.4.A.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/IndexData/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


<font style="color:black;">IndexData</font>_指标数据_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">ChNo</font> | <font style="color:black;">int</font> | <font style="color:black;">通道号</font> |
| <font style="color:black;">ChType</font> | <font style="color:black;">int</font> | <font style="color:black;">通道类型（0=振动 1:转速 2:工艺,3-温度，4-音频，5-RTU，6-ModBus-TCP，7 - MS810 ZigBee，8-润滑，10 湿度通道 </font> 11-<font style="color:black;">电能应力板卡通道 ，12-</font>多通道工艺量<font style="color:black;">）</font> |
| <font style="color:black;">Content</font> | <font style="color:black;">IndexDataContent</font><font style="color:black;">[]</font> | <font style="color:black;">数值集合</font> |
| <font style="color:black;">Sn</font> | String | 传感器SN |
| <font style="color:black;">Time</font> | <font style="color:black;">Datetime[]</font> | <font style="color:black;">时间数组</font> |
| <font style="color:black;">SubChNo</font> | <font style="color:black;">int</font> | 子通道号 |


<font style="color:black;">IndexDataContent </font>_指标数据实体_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Dir | int | 方向（0-无方向；1-Z轴；2-X轴；3-Y轴） |
| Code | VibEnum | 详情见下表 |
| Values | Float[Time.Length] | 数值集合  |
| DataType | int | 数据类型（可空，只存在于多工艺量情况下的子通道） |




VibEnum

| Code | <font style="color:white;">说明</font> |
| --- | :--- |
| 0 | 默认 |
| 100 | 温度 |
| 101 | 低频总值 |
| 107 | 速度总值 |
| 112 | 高频总值                                        |
| 121 | 高频包络峰值 |
| 123 | 低频包络峰值 |
| 124 | 音频分贝 |
| 125 | 漏磁频率（<font style="color:#DF2A3F;">为避免误会，修改为漏磁频率，转速频率由漏磁频率计算得出，</font>**<font style="color:#DF2A3F;">（漏磁频率/极对数）x（1-转差率）</font>**如果需要转速的话，实际频率乘以60） |
| 126 | 漏磁RMS |
| 127 | 位移偏执电压 |
| 128 | 轴承健康指数（HI值） |
| 130 | 峰值 |
| 131 | 峰峰值 |
| 132 | 最大值 |
| 133 | 最小值 |
| 134 | 方根幅值 |
| 135 | 峭度指标 |
| 136 | 峰值指标 |
| 137 | 脉冲指标 |
| 138 | 波形指标  |
| 139 |  裕度指标  |
| 140 |  歪度指标  |
| 141 | 平均幅值 |
| 142 | 均值 |
| 300 | 油脂器状态(0：正常，1：空油舱，3：高回压，4：低回压，5：电流超限，6：低电量，7：定时正常) |
| 301 | 油脂器电压 |
| 302 | 油脂器油脂剩余量 |
| 15131 | 一次积分量峰峰值 |




### 3.4.A.3 下位机返回的样例
```json

[
  {
    "ChNo": 1,
    "ChType": 0,
    "Time": [
      "2023-09-27T11:30:01",
      "2023-09-27T12:30:01"
    ],
    "Content": [
      {
        "Dir": 1,
        "Code": 112,
        "Values": [
          0.07558,
          0.08046
        ]
      },
      {
        "Dir": 1,
        "Code": 101,
        "Values": [
          0.02205,
          0.02274
        ]
      },
      {
        "Dir": 1,
        "Code": 107,
        "Values": [
          0.04924,
          0.04217
        ]
      },
      {
        "Dir": 1,
        "Code": 100,
        "Values": [
          27,
          28
        ]
      },
      {
        "Dir": 2,
        "Code": 101,
        "Values": [
          0.0659,
          0.06446
        ]
      },
      {
        "Dir": 2,
        "Code": 107,
        "Values": [
          0.12168,
          0.09921
        ]
      },
      {
        "Dir": 3,
        "Code": 101,
        "Values": [
          0.07508,
          0.07084
        ]
      },
      {
        "Dir": 3,
        "Code": 107,
        "Values": [
          0.13429,
          0.09832
        ]
      }
    ]
  }
]
```

## 3.4.B.传输同步指标数据（目前仅MS8XX系列在用）
### 3.4.B.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/SyncIndexData/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">SyncIndexData序列化为json后转为字节数组</font> |


### 3.4.B.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/SyncIndexData/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


<font style="color:black;">SyncIndexData</font>_指标数据_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">ChNo</font> | <font style="color:black;">int[]</font> | <font style="color:black;">通道号</font> |
| <font style="color:black;">ChType</font> | <font style="color:black;">int</font> | <font style="color:black;">通道类型（</font><font style="color:black;">0=</font><font style="color:black;">振动</font><font style="color:black;"> 1:</font><font style="color:black;">转速</font><font style="color:black;"> 2:</font><font style="color:black;">工艺</font><font style="color:black;">,3-</font><font style="color:black;">温度，</font><font style="color:black;">4-</font><font style="color:black;">音频，</font><font style="color:black;">5-RTU</font><font style="color:black;">）</font> |
| <font style="color:black;">Content</font> | <font style="color:black;">IndexDataContent</font><font style="color:black;">[]</font> | <font style="color:black;">数值集合</font> |
| <font style="color:black;">Time</font> | <font style="color:black;">Datetime</font> | <font style="color:black;">时间数组</font> |


<font style="color:black;">IndexDataContent </font>_指标数据实体_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Dir | int | 方向（0-无方向；1-Z轴；2-X轴；3-Y轴） |
| Code | Int | 指标类型合集（0-默认； 100-温度；101-低频总值；107-速度总值；112-高频总值；121-高频包络峰值；123-低频包络峰值；124-音频分贝；125-漏磁频率；126-转速RMS；127-位移偏执电压） |
| Values | Float[<font style="color:black;">ChNo</font>.Length] | 数值集合 ,和通道号集合一一对应 |


### 3.4.B.3 下位机返回的样例
```json

[
  {
    "ChNo": [1,2,3,4,5,6,7,8,9,10,11,12],
    "ChType": 0,
    "Time": "2023-09-27T12:30:01",
    "Content": [
      {
        "Dir": 1,
        "Code": 112,
        "Values": [
          0.07558,
          0.08046,
          0.07558,
          0.08046,
          0.07558,
          0.08046,
          0.07558,
          0.08046,
          0.07558,
          0.08046,
          0.07558,
          0.08046,
        ]
      },
      {
        "Dir": 1,
        "Code": 101,
        "Values": [
          0.02205,
          0.02274,
          0.02205,
          0.02274,
          0.02205,
          0.02274,
          0.02205,
          0.02274,
          0.02205,
          0.02274,
          0.02205,
          0.02274,
        ]
      },
      {
        "Dir": 1,
        "Code": 107,
        "Values": [
          0.04924,
          0.04217,
          0.04924,
          0.04217,
          0.04924,
          0.04217,
          0.04924,
          0.04217,
          0.04924,
          0.04217,
          0.04924,
          0.04217,
        ]
      },
      {
        "Dir": 1,
        "Code": 100,
        "Values": [
          27,
          28,
          27,
          28,
          27,
          28,
          27,
          28,
          27,
          28,
          27,
          28,
        ]
      },
      {
        "Dir": 2,
        "Code": 101,
        "Values": [
          0.0659,
          0.06446,
          0.0659,
          0.06446,
          0.0659,
          0.06446,
          0.0659,
          0.06446,
          0.0659,
          0.06446,
          0.0659,
          0.06446,
        ]
      },
    ]
  }
]
```

## 
## 3.4.C.传输智能电表指标数据
### 3.4.C.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/EIndexData/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">List<EIndexData>序列化为json后</font><br/><font style="color:black;">转为字节数组</font> |


### 3.4.C.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/EIndexData/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


<font style="color:black;">EIndexData</font>_指标数据_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| SinglePhaseIndex | SinglePhaseIndexData[] | 单相指标 |
| SynthesizeIndex | SynthesizeIndexData | 综合指标 |
| Time | DateTime | 数据时间 |


SinglePhaseIndexData

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| Phase | int | 0 :A相 1 ：B相 2：C相 |
| VRms | float | 电压有效值 |
| BaseVRms | float | 基波电压有效值 |
| Angle | float | 相位差（°） |
| VTHD | float | 电压波形的总谐波失真 |
| UnderDevuation | float | 低于主电压标称值的偏差(mV) |
| OverDevuation | float | 高于主电压标称值的偏差(mV) |
| IRms | float | 电流有效值 |
| BaseIRms | float | 基波电流有效值 |
| ITHD | float | 电流波形的总谐波失真 |
| ActivePower | int | 有功功率(uW) |
| BaseActivePower | int | 基波有功功率(uW) |
| ReactivePower | int | 无功功率(uW) |
| BaseReactivePower | int | 基波无功功率(uW) |
| ApparentPower | int | 视在功率(uw) |
| BaseApparentPower | int | 基波视在功率（uw） |
| VToIAngle | int | V to I角度 |
| MainFre | float | 主电源频率（Hz） |
| InputActiveEnergy | int | 输入有功能量(Wh) |
| OutputActiveEnergy | int | 输出有功能量(Wh) |
| BaseInputActiveEnergy | int | 基波输入有功能量(Wh) |
| BaseOutputActiveEnergy | int | 基波输出有功能量(Wh) |
| ReactiveEnergyFir | int | 第一象限无功能量(Wh) |
| ReactiveEnergySec | int | 第二象限无功能量(Wh) |
| ReactiveEnergyThr | int | 第三象限无功能量(Wh) |
| ReactiveEnergyFour | int | 第四象限无功能量(Wh) |
| BaseReactiveEnergyFir | int | 基波第一象限无功能量(Wh) |
| BaseReactiveEnergySec | int | 基波第二象限无功能量(Wh) |
| BaseReactiveEnergyThr | int | 基波第三象限无功能量(Wh) |
| BaseReactiveEnergyFour | int | 基波第四象限无功能量(Wh) |
| InputApparentEnergy | int | 输入视在能量（Wh） |
| OutputApparentEnergy | int | 输出视在能量（Wh） |


SynthesizeIndexData

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| --- | --- | --- |
| VRms | float | 电压有效值 |
| VRmsAB | float | AB线对线电压有效值 |
| VRmsBC | float | BC线对线电压有效值 |
| VRmsCA | float | CA线对线电压有效值 |
| BaseVRmsAB | float | AB线对线基波电压有效值 |
| BaseVRmsBC | float | BC线对线基波电压有效值 |
| BaseVRmsCA | float | CA线对线基波电压有效值 |
| ActivePower | int | 有功功率(uW)  |
| BaseActivePower | int | 基波有功功率(uW) |
| ReactivePower | int | 无功功率(uW) |
| BaseReactivePower | int | 基波无功功率(uW) |
| ApparentPower | int | 视在功率(uw) |
| BaseApparentPower | int | 基波视在功率（uw） |


参数示例：

```json
[
  {
  "SinglePhaseIndex": [
    {
      "Phase": 0,
      "VRms": 220,
      "BaseVRms": 221,
      "Angle": 1,
      "VTHD": 2,
      "UnderDevuation": 3,
      "OverDevuation": 4,
      "IRms": 5,
      "BaseIRms": 6,
      "ITHD": 7,
      "ActivePower": 8,
      "BaseActivePower": 9,
      "ReactivePower": 10,
      "BaseReactivePower": 11,
      "ApparentPower":12,
      "BaseApparentPower": 13,
      "VToIAngle": 14,
      "MainFre": 15,
      "InputActiveEnergy": 16,
      "OutputActiveEnergy": 17,
      "BaseInputActiveEnergy": 18,
      "ReactiveEnergyFir": 19,
      "ReactiveEnergySec": 20,
      "ReactiveEnergyThr": 21,
      "ReactiveEnergyFour": 22,
      "BaseReactiveEnergyFir": 23,
      "BaseReactiveEnergySec": 24,
      "BaseReactiveEnergyThr": 25,
      "BaseReactiveEnergyFour": 26,
      "InputApparentEnergy": 27,
      "OutputApparentEnergy": 28
    },
    {
      "Phase": 1,
      "VRms": 220,
      "BaseVRms": 221,
      "Angle": 1,
      "VTHD": 2,
      "UnderDevuation": 3,
      "OverDevuation": 4,
      "IRms": 5,
      "BaseIRms": 6,
      "ITHD": 7,
      "ActivePower": 8,
      "BaseActivePower": 9,
      "ReactivePower": 10,
      "BaseReactivePower": 11,
      "ApparentPower": 12,
      "BaseApparentPower": 13,
      "VToIAngle": 14,
      "MainFre": 15,
      "InputActiveEnergy": 16,
      "OutputActiveEnergy": 17,
      "BaseInputActiveEnergy": 18,
      "ReactiveEnergyFir": 19,
      "ReactiveEnergySec": 20,
      "ReactiveEnergyThr": 21,
      "ReactiveEnergyFour": 22,
      "BaseReactiveEnergyFir": 23,
      "BaseReactiveEnergySec": 24,
      "BaseReactiveEnergyThr": 25,
      "BaseReactiveEnergyFour": 26,
      "InputApparentEnergy": 27,
      "OutputApparentEnergy": 28
    },{
        "Phase": 2,
        "VRms": 220,
        "BaseVRms": 221,
        "Angle": 1,
        "VTHD": 2,
        "UnderDevuation": 3,
        "OverDevuation": 4,
        "IRms": 5,
        "BaseIRms": 6,
        "ITHD": 7,
        "ActivePower": 8,
        "BaseActivePower": 9,
        "ReactivePower": 10,
        "BaseReactivePower": 11,
        "ApparentPower":12,
        "BaseApparentPower": 13,
        "VToIAngle": 14,
        "MainFre": 15,
        "InputActiveEnergy": 16,
        "OutputActiveEnergy": 17,
        "BaseInputActiveEnergy": 18,
        "ReactiveEnergyFir": 19,
        "ReactiveEnergySec": 20,
        "ReactiveEnergyThr": 21,
        "ReactiveEnergyFour": 22,
        "BaseReactiveEnergyFir": 23,
        "BaseReactiveEnergySec": 24,
        "BaseReactiveEnergyThr": 25,
        "BaseReactiveEnergyFour": 26,
        "InputApparentEnergy": 27,
        "OutputApparentEnergy": 28
      }
  ],
  "SynthesizeIndex": {
    "VRms": 1,
    "VRmsAB": 2,
    "VRmsBC": 3,
    "VRmsCA": 4,
    "BaseVRmsAB": 5,
    "BaseVRmsBC": 6,
    "BaseVRmsCA": 7,
    "ActivePower": 8,
    "BaseActivePower": 9,
    "ReactivePower": 10,
    "BaseReactivePower": 11,
    "ApparentPower":12,
    "BaseApparentPower": 13
  },
  "Time": "2024-08-26T16:07:52"
},
{
  "SinglePhaseIndex": [
    {
      "Phase": 0,
      "VRms": 220,
      "BaseVRms": 221,
      "Angle": 1,
      "VTHD": 2,
      "UnderDevuation": 3,
      "OverDevuation": 4,
      "IRms": 5,
      "BaseIRms": 6,
      "ITHD": 7,
      "ActivePower": 8,
      "BaseActivePower": 9,
      "ReactivePower": 10,
      "BaseReactivePower": 11,
      "ApparentPower":12,
      "BaseApparentPower": 13,
      "VToIAngle": 14,
      "MainFre": 15,
      "InputActiveEnergy": 16,
      "OutputActiveEnergy": 17,
      "BaseInputActiveEnergy": 18,
      "ReactiveEnergyFir": 19,
      "ReactiveEnergySec": 20,
      "ReactiveEnergyThr": 21,
      "ReactiveEnergyFour": 22,
      "BaseReactiveEnergyFir": 23,
      "BaseReactiveEnergySec": 24,
      "BaseReactiveEnergyThr": 25,
      "BaseReactiveEnergyFour": 26,
      "InputApparentEnergy": 27,
      "OutputApparentEnergy": 28
    },
    {
      "Phase": 1,
      "VRms": 220,
      "BaseVRms": 221,
      "Angle": 1,
      "VTHD": 2,
      "UnderDevuation": 3,
      "OverDevuation": 4,
      "IRms": 5,
      "BaseIRms": 6,
      "ITHD": 7,
      "ActivePower": 8,
      "BaseActivePower": 9,
      "ReactivePower": 10,
      "BaseReactivePower": 11,
      "ApparentPower": 12,
      "BaseApparentPower": 13,
      "VToIAngle": 14,
      "MainFre": 15,
      "InputActiveEnergy": 16,
      "OutputActiveEnergy": 17,
      "BaseInputActiveEnergy": 18,
      "ReactiveEnergyFir": 19,
      "ReactiveEnergySec": 20,
      "ReactiveEnergyThr": 21,
      "ReactiveEnergyFour": 22,
      "BaseReactiveEnergyFir": 23,
      "BaseReactiveEnergySec": 24,
      "BaseReactiveEnergyThr": 25,
      "BaseReactiveEnergyFour": 26,
      "InputApparentEnergy": 27,
      "OutputApparentEnergy": 28
    },{
        "Phase": 2,
        "VRms": 220,
        "BaseVRms": 221,
        "Angle": 1,
        "VTHD": 2,
        "UnderDevuation": 3,
        "OverDevuation": 4,
        "IRms": 5,
        "BaseIRms": 6,
        "ITHD": 7,
        "ActivePower": 8,
        "BaseActivePower": 9,
        "ReactivePower": 10,
        "BaseReactivePower": 11,
        "ApparentPower":12,
        "BaseApparentPower": 13,
        "VToIAngle": 14,
        "MainFre": 15,
        "InputActiveEnergy": 16,
        "OutputActiveEnergy": 17,
        "BaseInputActiveEnergy": 18,
        "ReactiveEnergyFir": 19,
        "ReactiveEnergySec": 20,
        "ReactiveEnergyThr": 21,
        "ReactiveEnergyFour": 22,
        "BaseReactiveEnergyFir": 23,
        "BaseReactiveEnergySec": 24,
        "BaseReactiveEnergyThr": 25,
        "BaseReactiveEnergyFour": 26,
        "InputApparentEnergy": 27,
        "OutputApparentEnergy": 28
      }
  ],
  "SynthesizeIndex": {
    "VRms": 1,
    "VRmsAB": 2,
    "VRmsBC": 3,
    "VRmsCA": 4,
    "BaseVRmsAB": 5,
    "BaseVRmsBC": 6,
    "BaseVRmsCA": 7,
    "ActivePower": 8,
    "BaseActivePower": 9,
    "ReactivePower": 10,
    "BaseReactivePower": 11,
    "ApparentPower":12,
    "BaseApparentPower": 13
  },
  "Time": "2024-08-26T18:07:52"
}
]
```

## 3.5.传输波形数据
### 3.5.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/WaveData/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">WaveDataReq</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.5.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/WaveData/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">WaveDataResp</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |




_WaveDataReq __波形数据请求_

| 属性 | 类型 | 说明 |
| --- | --- | --- |
| Id | int | 波形唯一标识符，用于标识多个分包属于同一个波形 |
| PackCount | int | 总分包数 |
| PackIndex | int | 当前包序号（从0开始） |
| ChNo | int | 通道号 |
| ChType | int | 通道类型（0=振动 1:转速 2:工艺,3-温度，4-音频，5-RTU） |
| Dir | int | 方向（0-无方向；1-Z轴；2-X轴；3-Y轴） |
| Code | int | 指标类型（0-音频，1-长波形；2-低频长波形，11-高频加速度；12低频加速度；15-速度波形；30-漏磁波形） |
| Time | Datetime | 开始时间 |
| PlusLength | int | 附加长度 |
| Freq | int | **<font style="color:#DF2A3F;">分析</font>**频率（HZ） |
| RPM | int | 转速（RPM）（有线） |
| TimeMark | int | 关联的转速通道号（有线） |
| Coef | Float | 转换系数 |
| Src | Int | 1:默认，2:加速度触发，3：速度触发，4：转速触发，5：加速度/转速触发，6：速度/转速，7：冲击触发，8：启机触发，9：温度触发（有线） |
| IsSensor | Int | 0:未接传感器的空采数据，1：传感器数据 |
| Content | byte[] | 波形数据（两个字节<font style="color:#DF2A3F;">(高位在前</font>)转int16，再乘以Coef得到原始数值，如果是code为0的音频波形数据，两个字节需要是<font style="color:#DF2A3F;">低位在前</font>） |
| CRC | Int | 校验码，建议最后一包给出 CRC32 |
| ZipType | Int | 0—不压缩  1—zip2压缩（默认不压缩） |
| SendModel | Int | 0-普通模式  1-快速模式（默认普通模式） |
| AlarmLevel | Int | 儒海定制参数，波形传输中上传下位机计算的报警等级（目前上传3，4级等级） |
| Mean | float | 该波形的均值 |




_WaveDataResp __波形数据响应_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">Id</font> | <font style="color:black;">int</font> | <font style="color:black;">波形唯一标识符，用于标识多个分包属于同一个波形</font> |
| <font style="color:black;">PackIndex</font> | <font style="color:black;">int</font> | <font style="color:black;">已接收包序号</font> |


_WaveDataQuickRespDto__波形数据响应_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">UnRceivePack</font> | <font style="color:black;">Int[]</font> | <font style="color:black;">未接收包</font> |


### 3.5.3 下位机波形传输示例
```json
{
    "Id":1,
    "PackCount":16,
    "PackIndex":1,
    "ChNo":1,                                                 
    "Dir":0,
    "Code":11,
    "Time":"2023-09-09T13:29:45",
    "PlusLength":0,
    "Freq":20000,
    "RPM":0,
    "TimeMark":0,
    "Coef":0.000149200001033023,
    "Src":1,
    "IsSensor":0,
    "CRC":4042370163,
    "Content":"BJQA5QFQ/xEAXwD0/wAAXwHz/+D/Gf2oAL7+WwD1ARH/9P8O/z3aAEXACs="
}
```

### 3.5.4 上位机波形回复示例
```json
{
    "Id":1,
    "PackIndex":1
}
```

### 3.5.5 上位机波形回复示例（快速模式）
```json
{
    "UnRceivePack":[
      1,5
    ]
}
```

**<font style="color:#DF2A3F;">注：快速模式下不需要每一包都返回，只需要在接收到最后一包后返回未接收到的包，示例中表示丢失了包序号为1，5的包，下位机将会重发此两包，接收完全后，需要将成功结果返回，即返回一个空数组</font>**

```json
{
    "UnRceivePack":[
    ]
}
```

## 3.6.传输时标数据（有线）
### 3.6.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/TimeMarkData/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">TimeMarkDataReq</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.6.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/TimeMarkData/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">TimeMarkDataResp</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


_TimeMarkDataReq时标数据_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Id | int | 时标唯一标识符，用于标识多个分包属于同一个时标 |
| PackCount | int | 总分包数 |
| PackIndex | int | 当前包序号 |
| ChNo | int | 转速通道号 |
| Type | int | 时标类型（0-AD时标, 1-转速时标） |
| Time | Datetime | 采样时间 |
| Content | int[] | 分包时标数据，得到所有分包数据后，按照包序号整合 |
| TimeBase | int | 时基 |
| Code | int | 指标类型（11-高频加速度；12低频加速度） |
| Length | int | 数据长度 |
| CRC | byte[] | 校验码，建议最后一包给出 |




_TimeMarkDataResp __时标数据响应_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Id | int | 时标唯一标识符，用于标识多个分包属于同一个时标 |
| PackIndex | int | 已接受包序号 |


### 3.6.3 下位机发送数据示例
```json
{
    "Id":138,
    "PackCount":1,
    "PackIndex":1,
    "ChNo":0,
    "Type":0,
    "Time":"2023-09-27T12:00:03",
    "TimeBase":98304000,
    "Code":11,
    "Length":128,
    "Content":[
        1402733974,
        1404700054,
        1406666134,
        1408632214,
        1410598294,
        1412564374
    ]
}
```

### 3.6.4 上位机回复数据示例
```json
{"Id":138,"PackIndex":1}
```

## 3.7.传输自诊断数据（有线）
### 3.7.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/CheckData/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">CheckData[]</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.7.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/CheckData/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |




_CheckData__自诊断数据_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| CheckType | int | 自检项目类型（1-系统；2-偏执电压；3-转速通道；4-音频通道） |
| Id | int | 编号（通道号） |
| CheckValue | float | 值 |
| State | int | 状态（0-异常；1-正常） |


### 3.7.3 下位机发送数据示例
```json
[
    {
        "CheckType":1,
        "Id":1,
        "CheckValue":1,
        "State":1
    },
    {
        "CheckType":1,
        "Id":2,
        "CheckValue":0,
        "State":1
    },
  …………
    {
        "CheckType":2,
        "Id":16,
        "CheckValue":18.52393,
        "State":0
    }
]
```





## 3.8.传输传感器信息
### 3.8.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/SensorInfo/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">SensorInfo[]</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.8.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/SensorInfo/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


_SensorInfo__传感器信息_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">ChNo</font> | <font style="color:black;">int</font> | <font style="color:black;">通道号</font> |
| <font style="color:black;">SN</font> | <font style="color:black;">int</font> | <font style="color:black;">传感器序列号</font> |
| <font style="color:black;">AppVer</font> | <font style="color:black;">int</font> | <font style="color:black;">控制器版本</font> |
| <font style="color:black;">AppBuildCode</font> | <font style="color:black;">int</font> | <font style="color:black;">控制器</font><font style="color:black;">Build</font><font style="color:black;">号</font> |
| <font style="color:black;">HardVer</font> | <font style="color:black;">int</font> | <font style="color:black;">硬件版本信息</font> |
| <font style="color:black;">AccCoef</font> | <font style="color:black;">float</font> | <font style="color:black;">加速度校准系数</font> |
| <font style="color:black;">DigitalFreq</font> | <font style="color:black;">int</font> | <font style="color:black;">数传频段</font> |
| <font style="color:black;">SendPower</font> | <font style="color:black;">int</font> | <font style="color:black;">发射功率（</font><font style="color:black;">dbm</font><font style="color:black;">）</font> |
| <font style="color:black;">StartState</font> | <font style="color:black;">int</font> | <font style="color:black;">启动状态</font> |
| <font style="color:black;">Vol</font> | <font style="color:black;">float</font> | <font style="color:black;">电池电压（</font><font style="color:black;">V</font><font style="color:black;">）</font> |
| <font style="color:black;">Cur</font> | <font style="color:black;">int</font> | <font style="color:black;">电流（</font><font style="color:black;">mA</font><font style="color:black;">）</font> |
| <font style="color:black;">Temp</font> | <font style="color:black;">int</font> | <font style="color:black;">温度（摄氏度）</font> |
| <font style="color:black;">Clock</font> | <font style="color:black;">Datetime</font> | <font style="color:black;">传感器时钟</font> |
| <font style="color:black;">LQI</font> | <font style="color:black;">int</font> | <font style="color:black;">LQI</font> |
| <font style="color:black;">RSSI</font> | <font style="color:black;">int</font> | <font style="color:black;">RSSI</font> |
| <font style="color:black;">TransTime</font> | <font style="color:black;">int</font> | <font style="color:black;">传输时间</font> |
| <font style="color:black;">RetransCount</font> | <font style="color:black;">int</font> | <font style="color:black;">重传次数</font> |
| <font style="color:black;">TotalPackets</font> | <font style="color:black;">int</font> | <font style="color:black;">总包数</font> |


### 3.8.3 下位机数据请求示例
```json
[
    {
        "ChNo":3,
        "SN":35885,
        "AppVer":113,
        "AppBuildCode":9,
        "HardVer":0,
        "AccCoef":1,
        "DigitalFreq":15,
        "SendPower":0,
        "StartState":1,
        "Vol":3.53596,
        "Cur":23,
        "Temp":27,
        "Clock":"2023-09-27T09:59:58",
        "LQI":0,
        "RSSI":39,
        "TransTime":9608,
        "RetransCount":23,
        "TotalPackets":1281
    },
    {
        "ChNo":3,
        "SN":35885,
        "AppVer":113,
        "AppBuildCode":9,
        "HardVer":0,
        "AccCoef":1,
        "DigitalFreq":15,
        "SendPower":0,
        "StartState":1,
        "Vol":3.53115,
        "Cur":16,
        "Temp":27,
        "Clock":"2023-09-27T08:00:00",
        "LQI":0,
        "RSSI":39,
        "TransTime":9996,
        "RetransCount":3,
        "TotalPackets":1281
    }
]
```

## 3.9.RTU数据
### 3.9.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/RtuData/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">RtuDataReq[]</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.9.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/RtuData/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


__

<font style="color:black;">RtuDataReq RTU</font><font style="color:black;">通道数据</font>

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| ChNo | int | 通道号 |
| ChType | int | 通道类型 |
| Time | Datetime | 时间 |
| Values | Byte[] | 数据buffer |


### 3.9.3 下位机数据传输示例
```json
[
    {
        "ChNo":3,
        "ChType":5,
        "Time":"2023-09-27T08:00:00",
        "Values":"AAA=="
    }
]
```

## 3.10.传输电池信息
### 3.10.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/BatteryInfo/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">SensorInfo[]</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.10.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/BatteryInfo/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |




___BatteryInfo__电池信息_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| ChNo | int | 通道号 |
| Time | Datetime | 时间 |
| Vol | float | 电压（V） |
| Cur | float | 电流（mA） |
| Res | float | 内阻（Ω） |
| RSSI | float | 信号强度 |


### 3.10.3 下位机数据上传示例
```json
[
    {
        "ChNo":2,
        "Time":"2023-09-27T11:46:30",
        "Vol":3.51675,
        "Cur":33.26416,
        "Res":4.6575,
        "RSSI":7.7
    }
]
```

## 3.10.传输日志文件
### 3.11.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/LogFile/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">LogFileReq </font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.11.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/LogFile/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">LogFileResp </font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


_LogFileReq __日志文件请求_

| 属性 | 类型 | 说明 |
| :--- | :--- | --- |
| Md5Code | string | MD5码 |
| PackCount | int | 总分包数 |
| PackIndex | int | 当前包序号 |
| FileName | string | 文件名 |
| Content | byte[] | 内容（保存成文件即可） |


_LogFileResp__日志文件响应_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">Md5Code</font> | <font style="color:black;">string</font> | <font style="color:black;">MD5</font><font style="color:black;">码</font> |
| <font style="color:black;">PackIndex</font> | <font style="color:black;">int</font> | <font style="color:black;">已接受包序号</font> |


### 3.11.3 下位机请求示例
```json
{
    "Md5Code":"660fbfbb16167fb620f7c20c51329000",
    "FileName":"2023-09-27(114848).log",
    "PackCount":1,
    "PackIndex":1,
    "Content":"UgAAAAEAAAABFBcJGwstLEsAAAAAAAAAARQXCRsLLTCLAAAALQEAAAEUFwkbCy4BWgAAAEwAAAABFBcJGwsuAosAAACSAQAAARQXCRsLLgJaAAAA5QAAAAEUFwkbCy4CiwAAAC0BAAACFBcJGwsuHloAAAA0AAAAAhQXCRsLLh+LAAAAkgEAAAIUFwkbCy4fWgAAAJoAAAACFBcJGwsuH4sAAABnAAAAAhQXCRsLLiBaAAAANBwCAAIUFwkbCy4pYQEAAAAAAAABFBcJGwswKmEBAAAWBAAAAhQXCRsLMCphAQAAAAAAAAMUFwkbCzAqYQEAAAAAAAAEFBcJGwswK2ABAABzBAAAARQXCRsLMCtgAQAAFgQAAAIUFwkbCzArYAEAAPsHAAADFBcJGwswK2ABAAAuBAAABBQXCRsLMCsBAAAAD4YDAAEUFwkbCzAtAQAAAJ3AAQABFBcJGwswLQEAAAAPhgMAAhQXCRsLMC0BAAAAncABAAIUFwkbCzAuQAAAAAAAAAABFBcJGwswLg=="
}
```

### 3.11.4 上位机接收到数据回复
```json
{
    "Md5Code":"660fbfbb16167fb620f7c20c51329000",
    "PackIndex":1
}
```



## 3.12.请求升级文件列表
### 3.12.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/UpdateFileList/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


### 1.1.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/UpdateFileList/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">UpdateFile[]</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组（传</font><font style="color:black;">[],</font><font style="color:black;">默认没有升级）</font> |


_UpdateFile __升级文件_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">Name</font> | <font style="color:black;">string</font> | <font style="color:black;">文件名称</font> |
| <font style="color:black;">Size</font> | <font style="color:black;">int</font> | <font style="color:black;">字节数</font> |
| <font style="color:black;">Md5Code</font> | <font style="color:black;">string</font> | <font style="color:black;">MD5</font><font style="color:black;">码</font> |
| <font style="color:black;">CRC</font> | <font style="color:black;">Uint</font> | <font style="color:black;">整体的CRC32</font> |


### 3.12.3 上位机返回示例
```json
[
    {
        "Name":"ms_mcu_v8.16.50.51.bin",
        "Size":523888,
        "Md5Code":"6FFE4BB5408A3C8F7C2B890F0EF5FC71",
        "CRC":1978445735
    },
    {
        "Name":"ms_wifi_v3.0.1.bin",
        "Size":677060,
        "Md5Code":"805221A1A795D8E9DC045148247BE5BC",
        "CRC":3011529532
    }
]
```

## 3.13.请求升级文件
### 3.13.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/UpdateFile/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">UpdateFileReq</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.13.2.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/UpdateFile/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">UpdateFileResp</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


_UpdateFileReq __升级文件请求_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Md5Code | string | MD5码 |
| PackCount | int | 总分包数 |
| PackIndex | int | 当前包序号 |
| PackMaxLength | Int | 每包最大传输字节长度 |
| IsPlan | Int | 1：升级计划请求文件，<br/>0：上位机主动下发文件 |
| IsConfirm | Int | 0：常规请求包，<br/>1：最后的确认包，不回复 |


_UpdateFileResp __升级文件响应_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Md5Code | string | MD5码 |
| PackCount | int | 总分包数 |
| PackIndex | int | 当前包序号 |
| Content | byte[] | 内容 |


### 3.13.3 下位机请求
```json
{
    "Md5Code":"6FFE4BB5408A3C8F7C2B890F0EF5FC71",
    "PackCount":256,
    "PackIndex":0,
    "PackMaxLength":2048,
    "IsPlan":1,
    "IsConfirm":0
}
```

### 3.13.4 上位机返回数据
```json
{
    "Md5Code":"6FFE4BB5408A3C8F7C2B890F0EF5FC71",
    "PackCount":256,
    "PackIndex":2,
    "Content":"DP+vuWytoZAWHdhDk0UBwY5Sbce9InQLxMPZELDk5/a9864wzQQsUK2NYP1htgu+ud9e2vMcGpEy7RZ8z71l8Y="
}
```

## 3.14.通知获取参数
### 3.14.1.   上位机请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/IssuePara/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


### 3.14.2.   采集器返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/IssuePara/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |




## 3.15.通知获取升级文件
### 3.15.1.   上位机请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/IssueUpdateFile/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">UpdateFile</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


### 3.15.2.   采集器返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/IssueUpdateFile/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |




_UpdateFile __升级文件_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">Name</font> | <font style="color:black;">string</font> | <font style="color:black;">文件名称</font> |
| <font style="color:black;">Size</font> | <font style="color:black;">int</font> | <font style="color:black;">字节数</font> |
| <font style="color:black;">Md5Code</font> | <font style="color:black;">string</font> | <font style="color:black;">MD5</font><font style="color:black;">码</font> |


### 3.15.3 上位机请求示例
```json
{
        "Name":"ms_mcu_v8.16.50.51.bin",
        "Size":523888,
        "Md5Code":"6FFE4BB5408A3C8F7C2B890F0EF5FC71"
    },
```

## 3.16.下发网络参数（有线）
### 3.16.1.   上位机请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/IssueNetPara/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">Byte[]</font> | <font style="color:black;">NetParaDto</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |




### 3.16.2.   采集器返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/IssueNetPara/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">Byte[]</font> | <font style="color:black;">NetParaDto</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


<font style="color:black;">NetParaDto</font>_  __网络配置参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| LocalIp | string | 本机IP |
| ServerIp | String | 服务器IP |
| GateWay | string | 网关地址 |
| SubnetMask | String | 子网掩码 |
| ServerPort | Int | 服务器端口 |
| Mac | String | Mac地址 |


### 3.16.3 上位机请求示例
```json
{
  "LocalIp": "192.168.0.71",
  "ServerIp": "192.0.2.10",
  "GateWay": "192.168.0.1",
  "SubnetMask": "255.255.255.0",
  "ServerPort": 1883,
  "Mac": "00:58:22:10:10:10",
  "IsRead": 0
}
```



## 3.17.通知上传自诊断数据（有线）
### 3.17.1.   上位机请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/IssueCheckData/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


### 3.17.2.   采集器返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/IssueCheckData/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |




## 3.18.服务器启动
### 3.18.1.   服务器启动
| 字段 | 值 | 备注 |
| --- | --- | --- |
| 主题 | [Prefix]/ServerStart | 服务器启动指令，所有客户端再接收到该指令后，立马重新注册。 |
| 负载 | 空 |  |


## 3.19.通知重启
### 3.19.1.   上位机请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/IssueRestart/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte</font> | <font style="color:black;">重启方式</font><font style="color:black;">:0-</font><font style="color:black;">硬重启</font><font style="color:black;">,1-</font><font style="color:black;">软重启</font> |


### 3.19.2.   采集器返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/IssueRestart/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


## 3.20 设备事件
用于采集器报告设备事件，不需要回复

### 3.20.1 采集器事件
| 字段 | 值 | 备注 |
| --- | --- | --- |
| 主题 | [Prefix]/Online/[Code]/Event/<font style="color:black;">[MessageId]</font> |  |
| 负载 | <font style="color:black;">Byte[]</font> | <font style="color:black;">EventInfoDto[]序列化为json后转为字节数组</font> |


<font style="color:black;">EventInfoDto</font>_ 网络配置参数_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| Time | Datetime | 事件时间 |
| EventCode | int | 事件码 |
| Desc | string | 事件描述，可以传一段Json。 |


### 3.20.2 事件码
| <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- |
| 300 | 注脂执行状态 |
| 301 | 润滑不良（CS30M） |
| 302 | 电池电量不足（CS30M） |
|  |  |
|  |  |


### 3.20.3Desc的载体模型
| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| SN | string | 注脂器SN |
| TaskTime | string | 注脂任务时间 |
| TaskRunTime | string | 任务执行时间 |
| Status | int | 任务执行状态：2执行中，1成功，0失败 |


### 3.20.4 下位机请求示例
```json
{
  "Time": "2025-02-17T09:59:59",
  "EventCode": 300,
  "Desc": "{\"SN\":\"123456789\",\"TaskTime\":\"2025-02-17T09:59:58\",\"TaskTime\":\"2025-02-17T09:59:58\",\"Status\":2}"
}
```

## 3.21.请求注脂任务
### 3.21.1.   采集器请求
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/Online/[Code]/LubricateTaskList/[MessageId]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">空</font> | <font style="color:black;"></font> |


### 3.21.1.   上位机返回
| <font style="color:white;">字段</font> | <font style="color:white;">值</font> | <font style="color:white;">备注</font> |
| --- | --- | --- |
| <font style="color:black;">主题</font> | <font style="color:black;">[Prefix]/[Code]/LubricateTaskList/[MessageId]/[Status]</font> | <font style="color:black;"></font> |
| <font style="color:black;">负载</font> | <font style="color:black;">byte[]</font> | <font style="color:black;">LubricateTask[]序列化为json后转为字节数组（传[],默认没有升级）</font> |


<font style="color:black;">LubricateTask</font>_ 注脂任务_

| <font style="color:white;">属性</font> | <font style="color:white;">类型</font> | <font style="color:white;">说明</font> |
| :--- | :--- | :--- |
| <font style="color:black;">Sn</font> | <font style="color:black;">string</font> | <font style="color:black;">设备号</font> |
| <font style="color:black;">Amount </font> | <font style="color:black;">int</font> | 油量 |
| <font style="color:black;">Time</font> | DateTime | 计划时间 |
| <font style="color:black;">Type</font> | int | 0:常规任务；1：计划任务 |
| <font style="color:black;">Period</font> | int | 计划周期（秒） |
| <font style="color:black;">Enable</font> | int | 0 : 失效的 暂时不启用的  1：有效的 |


## 3.22.遗嘱
### 3.22.1.   采集器遗嘱
| 字段 | 值 | 备注 |
| --- | --- | --- |
| 主题 | [Prefix]/Online/[Code]/Disconnect |  |
| 负载 | 空 |  |




## 3.23.MS810开始采集
### 3.23.1.   控制开始采集
| 字段 | 值 | 备注 |
| --- | --- | --- |
| 主题 | [Prefix]/[Code]/StartCollect/<font style="color:black;">[MessageId]</font> |  |
| 负载 | <font style="color:black;">Byte[]</font> | <font style="color:black;">CollectPara</font><font style="color:black;">序列化为</font><font style="color:black;">json</font><font style="color:black;">后转为字节数组</font> |


采集参数结构与采集参数指令中返回数据结构一致，包含所有需要采集数据的通道参数，按照此命令提供的参数配置采集数据，然后数据间隔按照SysPara中给出的采集间隔

## 3.24.MS810停止采集
### 3.24.1.   控制停止采集
| 字段 | 值 | 备注 |
| --- | --- | --- |
| 主题 | [Prefix]/[Code]/StopCollect/<font style="color:black;">[MessageId]</font> |  |
| 负载 | 空 | <font style="color:black;"></font> |


## 3.25.MS810手动采集波形
### 3.25.1.   控制手动采集波形
| 字段 | 值 | 备注 |
| --- | --- | --- |
| 主题 | [Prefix]/[Code]/StartWave/<font style="color:black;">[MessageId]</font> |  |
| 负载 | 1 | <font style="color:black;"></font> |


