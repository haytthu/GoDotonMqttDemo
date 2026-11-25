import { ref } from 'vue'

const storageKey = 'sg500z.locale'
const savedLocale = localStorage.getItem(storageKey)
export const locale = ref(savedLocale === 'en' ? 'en' : 'zh')

function updateDocument() {
  document.documentElement.lang = locale.value === 'en' ? 'en' : 'zh-CN'
  document.title = t('SG500Z 采集管理', 'SG500Z Data Acquisition')
}

export function setLocale(value) {
  if (value === 'zh' || value === 'en') {
    locale.value = value
    localStorage.setItem(storageKey, value)
    updateDocument()
  }
}

export function t(chinese, english = chinese) {
  return locale.value === 'en' ? english : chinese
}

const messages = {
  '请求无效': 'Invalid request',
  '请求必须是 JSON 对象': 'The request must be a JSON object',
  '后端返回了无法识别的数据': 'The backend returned an unreadable response',
  '无法连接后端': 'Unable to reach the backend',
  '无法读取后端响应': 'Unable to read the backend response',
  '记录不存在': 'Record not found',
  '波形不存在': 'Waveform not found',
  '音频记录不存在': 'Audio recording not found',
  '日志文件不存在': 'Log file not found',
  '日志文件分包无效': 'Invalid log file packet',
  '日志分包接收缓存已满': 'The log packet buffer is full',
  '日志文件分包的文件名或总数不一致': 'Log packets have differing file names or packet counts',
  '日志文件超过接收缓冲区大小': 'The log file exceeds the receive buffer',
  '日志文件没有有效的 Base64 内容': 'The log file has no valid Base64 content',
  '日志文件 MD5 校验失败': 'Log file MD5 verification failed',
  '日志文件 MD5 无效': 'Invalid log file MD5',
  '请选择固件': 'Select a firmware file',
  '请为本次升级上传固件文件': 'Upload a firmware file for this upgrade',
  '请选择 1 至 10 个固件文件': 'Select between 1 and 10 firmware files',
  '本次升级包含重复的固件文件': 'The upgrade contains duplicate firmware files',
  '目标版本必须是文本': 'The target version must be text',
  '采集器尚未在线': 'The collector is offline',
  '名称不能为空，且最多 150 个字符': 'Enter a name of up to 150 characters',
  '采集器编号不能为空，且最多 150 个字符': 'Enter a collector ID of up to 150 characters',
  '采集器编号包含无效字符': 'The collector ID contains invalid characters',
  '采集器编号已存在': 'This collector ID already exists',
  '采集器编号未创建': 'This collector ID has not been configured',
  '已下发过参数的采集器编号不能修改': 'A collector ID is locked once settings have been published',
  '采集和上传周期必须为非负秒数': 'Collection and upload intervals must be zero or more seconds',
  '波形压缩方式无效': 'Invalid waveform compression mode',
  '扩展通道绑定测点不存在': 'The point bound to the extension channel could not be found',
  '扩展通道只能绑定同类型测点': 'Select a point that matches the extension channel type',
  '组织下还有子组织或设备': 'Remove the child organizations and devices first',
  '设备下还有测点': 'Remove the measurement points for this device first',
  '测点已绑定通道或启停机规则': 'This point is assigned to a channel or start/stop rule',
  '三轴测点中还有测点绑定了通道或启停机规则': 'Remove the channel bindings and start/stop rules for these axes first',
  '三轴测点只能在新建时一次创建': 'Create the three axis points together as a new measurement group',
  '三轴测点的所属设备和类型不能单独修改': 'The device and point type are fixed for each member of a three-axis group',
  '不能将组织放到自己或后代组织下': 'An organization cannot be its own ancestor',
  '根组织下只能创建组织': 'Only organizations can be added to the root organization',
  '根组织不能修改或删除': 'The root organization cannot be changed or deleted',
  '已有设备的组织下不能创建子组织': 'Move the devices before adding a child organization',
  '已有子组织的组织下不能创建设备': 'Add devices to an organization without child organizations',
  '上级组织不存在': 'Parent organization not found',
  '请选择有效组织': 'Select an existing organization',
  '请选择有效设备': 'Select an existing device',
  '请选择有效的测点类型': 'Select a valid point type',
  '请先解除测点绑定和启停机规则再更改设备或类型': 'Remove channel bindings and start/stop rules before changing the device or point type',
  '启停机测点不存在': 'Start/stop point not found',
  '启停机只能使用三轴 Z 轴或单轴振动测点': 'Start/stop detection requires a single-axis vibration point or the Z axis of a three-axis point',
  '启停机门限必须绑定本设备的测点和有效指标': 'Select a valid indicator and point on this device for the start/stop rule',
  '同一采集器通道和指标只能下发一个启停机门限': 'Set one start/stop threshold per collector channel and indicator',
  '报警门限的指标、等级、类型和数值必须有效且唯一': 'Enter a valid threshold for each unique indicator, level and alarm type',
  '报警门限方向无效': 'Invalid direction for the alarm threshold',
  '报警门限的指标、方向、等级和数值必须有效': 'Enter a valid metric, axis, level and alarm threshold',
  '同一方向和指标的报警等级不能重复': 'Use each alarm level once per point direction and metric',
  '同一方向和指标的报警门限必须随等级严格递增': 'Alarm thresholds must increase strictly by level for each point direction and metric',
  '通道号、方向和绑定测点类型必须匹配': 'The channel number, direction and point type must match',
  '有重复的通道和方向': 'A channel and direction are assigned more than once',
  '同一振动通道不能同时绑定单轴和多轴': 'Use a single-axis or a multi-axis binding per vibration channel',
  '三轴测点只需绑定一次物理通道': 'Bind each three-axis point to a single physical channel',
  '三轴测点缺少方向，无法下发参数': 'Cannot publish parameters because a three-axis measurement group is missing an axis',
  '轴名称最多 80 个字符': 'Axis names can have up to 80 characters',
  '请检查低频加速度和速度的频带、采样长度及附加长度': 'Check the bands, sample counts and extra samples for low-frequency acceleration and velocity',
  '等待设备下次请求参数': 'Waiting for the collector to request settings',
  '已发送获取参数通知，等待设备拉取': 'Settings notification sent. Waiting for the collector to fetch the settings',
  '设备已获取采集参数': 'The collector fetched the acquisition settings',
  '设备已确认通知': 'The collector acknowledged the notification',
  '等待设备回复超时，可在设备上线后重新发送': 'The collector did not respond in time. Retry when it is online',
  'MQTT Broker 未连接': 'The MQTT broker is disconnected',
  'MQTT 消息发送失败': 'The MQTT message could not be sent',
  '固件文件必须非空且不超过 64 MB': 'Upload a nonempty firmware file of up to 64 MB',
  '固件文件名无效': 'Invalid firmware file name',
  '固件文件超过 64 MB': 'The firmware file exceeds 64 MB',
  '固件文件不存在': 'Firmware file not found',
  '等待设备请求升级文件': 'Waiting for the collector to request the firmware file',
  '设备正在下载固件': 'The collector is downloading firmware',
  '已发送升级通知，等待设备请求文件': 'Upgrade notification sent. Waiting for the collector to request the firmware file',
  '设备已确认下载，等待重启上报版本': 'The collector confirmed the download. Waiting for the version after a restart',
  '固件包序号或每包大小无效': 'Invalid firmware packet index or size',
  '该设备没有此固件的升级任务': 'No upgrade task for this collector and firmware file',
  '无法计算固件 MD5': 'Unable to calculate the firmware MD5',
  '日志文件存储位置无效': 'Invalid collector log file storage location',
  'SQLite 操作失败': 'SQLite operation failed',
  'Modbus TCP 主机模式需要 IP 地址': 'Enter an IP address for Modbus TCP master mode',
  'RTU 量程下限不能超过上限': 'The lower range limit must not exceed the upper limit',
  'RtuData 缺少采集时间': 'RtuData is missing the acquisition timestamp',
  'RtuData 缺少 Values': 'RtuData is missing Values',
  '请选择扩展通道关联测点': 'Select a measurement point for the extension channel',
  '功能码必须是 1–255 的整数（支持十六进制）': 'The function code must be an integer from 1 to 255. Hexadecimal is supported',
  '波形至少需要两个有效采样点且采样率必须为正': 'A waveform needs at least two valid samples and a positive sampling rate',
  '波形含有无效采样值': 'The waveform contains invalid samples',
  '指标时间和数值的数量不一致': 'The indicator timestamp and value counts differ',
  '波形采集时间无效': 'Invalid waveform timestamp',
  '波形分包序号或总数无效': 'Invalid waveform packet index or count',
  '波形转换系数无效': 'Invalid waveform scaling factor',
  '缺少波形报文': 'Missing waveform message',
  '波形分包超出接收缓冲区': 'The waveform exceeds the packet buffer',
  '波形确认的分包总数与索引不一致': 'The confirmation and index report different waveform packet counts',
  '波形分包缺少 Base64 数据': 'Waveform packets must contain Base64 data',
  '波形文件标识或采样点数无效': 'Invalid waveform ID or sample count',
  '波形文件大小与索引中的点数不一致': 'The waveform file size does not match the sample count',
  '波形文件含有额外数据': 'The waveform file contains extra data',
  '低频加速度波形点数或参数无效': 'Invalid low-frequency acceleration sample count or waveform parameters',
  '速度波形的积分频带无效': 'Invalid integration band for the velocity waveform',
  '按实际采集频率计算的积分频带无效': 'Invalid integration band for the actual acquisition frequency',
  '低频加速度点数不足以生成所配置的速度波形': 'Not enough low-frequency acceleration samples for the configured velocity waveform',
}

const patterns = [
  [/^请求失败（(\d+)）$/, (_, status) => `Request failed (HTTP ${status})`],
  [/^设备返回状态 (.+)$/, (_, status) => `Collector returned status ${status}`],
  [/^已响应设备请求，第 (\d+) \/ (\d+) 包$/, (_, nth, total) => `Sent packet ${nth} of ${total}`],
  [/^重启后上报版本 ([^，]+)(?:，期望版本 (.+))?$/, (_, version, expected) =>
    `Version reported after restart: ${version === '未知' ? 'unknown' : version}${expected ? `. Expected: ${expected}` : ''}`],
  [/^(\S+) 通道 (\d+) 的(?:子通道|数据项) (\d+) 重复$/, (_, kind, channel, slot) =>
    `${channelLabel(kind)} channel ${channel}, position ${slot} is assigned more than once`],
  [/^(\S+) 通道 (\d+) 重复$/, (_, kind, channel) => `${channelLabel(kind)} channel ${channel} is configured more than once`],
  [/^(\S+) 通道 (\d+) 需要至少一个数据项$/, (_, kind, channel) => `Add a data position to ${channelLabel(kind)} channel ${channel}`],
  [/^(\S+) 必须是 (-?[\d.]+)–(-?[\d.]+) 范围内的整数$/, (_, field, min, max) => `${field} must be an integer from ${min} to ${max}`],
  [/^(\S+) 必须是有效数值$/, (_, field) => `${field} must be a valid number`],
  [/^(\S+) 必须是文本$/, (_, field) => `${field} must be text`],
  [/^(\S+) 必须是数组$/, (_, field) => `${field} must be an array`],
  [/^(\S+) 必须包含配置对象$/, (_, field) => `${field} must contain configuration objects`],
]

const chineseMessages = new Map(Object.entries(messages).map(([chinese, english]) => [english, chinese]))

const chineseErrorPatterns = [
  [/^Confirmed (\d+) of (\d+) firmware downloads$/, (_, count, total) => `已确认下载 ${count} / ${total} 个固件文件`],
  [/^Request failed \(HTTP (\d+)\)$/, (_, status) => `请求失败（${status}）`],
  [/^(RTU|Modbus TCP|PT100) channel (\d+), position (\d+) is assigned more than once$/, (_, kind, channel, slot) =>
    `${kind} 通道 ${channel} 的${kind === 'PT100' ? '子通道' : '数据项'} ${slot} 重复`],
  [/^(RTU|Modbus TCP) channel (\d+) is configured more than once$/, (_, kind, channel) => `${kind} 通道 ${channel} 重复`],
  [/^Add a data position to (RTU|Modbus TCP) channel (\d+)$/, (_, kind, channel) => `${kind} 通道 ${channel} 需要至少一个数据项`],
  [/^(\S+) must be an integer from (-?\d+) to (-?\d+)$/, (_, field, min, max) => `${field} 必须是 ${min}–${max} 范围内的整数`],
  [/^(\S+) must be a valid number$/, (_, field) => `${field} 必须是有效数值`],
  [/^(\S+) must be text$/, (_, field) => `${field} 必须是文本`],
  [/^(\S+) must be an array$/, (_, field) => `${field} 必须是数组`],
  [/^(\S+) must contain configuration objects$/, (_, field) => `${field} 必须包含配置对象`],
]

function channelLabel(value) {
  return { rtu: 'RTU', modbusTcp: 'Modbus TCP', pt100: 'PT100' }[value] || value
}

export function translateMessage(value) {
  const message = String(value ?? '')
  if (locale.value === 'zh') return message
  if (Object.hasOwn(messages, message)) return messages[message]
  return translatePattern(message, patterns)
}

export function translateError(value) {
  if (locale.value === 'en') return translateMessage(value)
  const message = String(value ?? '')
  return chineseMessages.get(message) ?? translatePattern(message, chineseErrorPatterns)
}

function translatePattern(message, candidates) {
  for (const [pattern, replace] of candidates) {
    if (pattern.test(message)) return message.replace(pattern, replace)
  }
  return message
}

updateDocument()

