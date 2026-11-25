import { locale, t } from './i18n.js'

function translatedLabels(labels) {
  const result = {}
  for (const [key, label] of Object.entries(labels)) {
    Object.defineProperty(result, key, { enumerable: true, get: () => t(...label) })
  }
  return result
}

export const pointTypes = translatedLabels({
  VIBRATION: ['振动', 'Vibration'],
  VIBRATION_3AXIS: ['三轴振动', 'Three-axis vibration'],
  TEMPERATURE: ['温度', 'Temperature'],
  HUMIDITY: ['湿度', 'Humidity'],
  PROCESS: ['工艺量', 'Process value'],
  RTU: ['RTU'],
  MODBUS_TCP: ['Modbus TCP'],
  PT100: ['PT100'],
})

export const channelTypes = translatedLabels({
  0: ['振动', 'Vibration'],
  1: ['转速', 'Rotational speed'],
  2: ['工艺量', 'Process value'],
  3: ['温度', 'Temperature'],
  4: ['音频', 'Audio'],
  10: ['湿度', 'Humidity'],
})

export const baseChannels = { 0: 'VIBRATION', 2: 'PROCESS', 3: 'TEMPERATURE', 10: 'HUMIDITY' }
export const directions = translatedLabels({
  0: ['单轴', 'Single axis'],
  1: ['Z 轴', 'Z axis'],
  2: ['X 轴', 'X axis'],
  3: ['Y 轴', 'Y axis'],
})
export const indicatorNames = translatedLabels({
  0: ['工艺量', 'Process value'],
  1: ['长波形', 'Long waveform'],
  11: ['高频加速度', 'High-frequency acceleration'],
  12: ['低频加速度', 'Low-frequency acceleration'],
  15: ['速度波形', 'Velocity waveform'],
  100: ['温度', 'Temperature'],
  101: ['低频总值', 'Overall low-frequency acceleration'],
  107: ['速度总值', 'Overall velocity'],
  110: ['湿度', 'Humidity'],
  112: ['高频总值', 'Overall high-frequency acceleration'],
  125: ['转速', 'Rotational speed'],
})

const statuses = translatedLabels({
  DRAFT: ['未下发', 'Draft'],
  PENDING: ['待发送', 'Pending'],
  NOTIFIED: ['已通知', 'Notified'],
  ACKNOWLEDGED: ['设备已应答', 'Acknowledged'],
  PULLED: ['设备已拉取', 'Fetched by collector'],
  TRANSFERRING: ['传输中', 'Transferring'],
  DOWNLOADED: ['设备已确认下载', 'Download confirmed'],
  RE_REGISTERED: ['设备已重新注册', 'Collector re-registered'],
  VERSION_MISMATCH: ['上报版本与目标不一致', 'Version mismatch'],
  FAILED: ['失败', 'Failed'],
  RUNNING: ['运行', 'Running'],
  STOPPED: ['停机', 'Stopped'],
  UNKNOWN: ['待判定', 'Unknown'],
  DISABLED: ['未启用', 'Disabled'],
  AT_OR_ABOVE: ['达到门限', 'At or above threshold'],
  BELOW: ['低于门限', 'Below threshold'],
  NO_DATA: ['暂无数据', 'No data'],
  UNSUPPORTED: ['不支持', 'Unsupported'],
})

export const defaults = {
  highFreq: 6250,
  highAccLength: 16384,
  longWaveFreq: 12500,
  longWaveLength: 131072,
  lowAccFreq: 3125,
  lowAccLength: 16384,
  plusLength: 1024,
  velocityLowFreq: 2,
  velocityHighFreq: 1000,
  velocityLength: 4096,
}

// SG500Z analysis frequencies and sample counts. The 2M Mars long-wave preset exceeds the demo receiver limit.
export const vibrationPresets = {
  highFreq: [1000, 5000, 6250, 10000, 12500, 20000],
  highAccLength: [0, 16384, 32768, 65536],
  longWaveFreq: [2000, 3125, 5000, 6250, 10000, 12500, 20000],
  longWaveLength: [0, 65536, 131072, 262144, 524288],
  lowAccFreq: [500, 1000, 1563, 2000, 3125],
  lowAccLength: [8192, 16384, 23552, 32768, 65536],
  plusLength: [0, 1024],
  velocityLowFreq: Array.from({ length: 101 }, (_, hz) => hz),
  velocityHighFreq: [500, 1000, 1562, 2000, 3125],
  velocityLength: [0, 4096, 8192, 16384],
}

export const systemDefaults = {
  indexPeriod: 60,
  wavePeriod: 300,
  longWavePeriod: 259200,
  longWaveOffset: 0,
  waveCompress: 1,
}

export function options(labels, numeric = true) {
  return Object.entries(labels).map(([value, label]) => ({ value: numeric ? Number(value) : value, label }))
}

export function number(key, label, min = 0, max, step = 1) {
  return { key, label, kind: 'number', min, max, step, required: true }
}

export function select(key, label, items, required = true) {
  return { key, label, kind: 'select', options: items, required }
}

export function statusText(value) {
  return statuses[value] || value || '—'
}

export function formatValue(value) {
  return typeof value === 'number'
    ? value.toLocaleString(locale.value === 'en' ? 'en-US' : 'zh-CN', { maximumFractionDigits: 6, useGrouping: false })
    : value ?? '—'
}

export function timeText(value) {
  return value == null ? '—' : String(value).replace('T', ' ')
}

export function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

export function nextNumber(rows, key, max) {
  for (let value = 1; value <= max; value++) {
    if (!rows.some((row) => Number(row[key]) === value)) return value
  }
  return null
}
