<script setup>
import { computed, ref } from 'vue'
import { formatValue, timeText } from '../config.js'
import { t } from '../i18n.js'

const props = defineProps({
  records: { type: Array, required: true },
  loading: Boolean,
})

const kind = defineModel('kind', { type: String, required: true })
const selectedId = ref('')
const sensorMode = computed(() => kind.value === 'sensor-info')
const selected = computed(() => props.records.find((row) => row.id === selectedId.value) || props.records[0])
const identity = computed(() => sensorMode.value ? t('传感器序列号', 'Sensor serial') : t('电池编号', 'Battery ID'))

const sensorFields = [
  ['SN', '传感器序列号', 'Sensor serial'],
  ['AppVer', '控制器版本', 'Controller version'],
  ['AppBuildCode', '控制器构建号', 'Controller build'],
  ['HardVer', '硬件版本', 'Hardware version'],
  ['AccCoef', '加速度校准系数', 'Acceleration calibration'],
  ['DigitalFreq', '数传频段', 'Radio band'],
  ['SendPower', '发射功率', 'Transmit power', 'dBm'],
  ['StartState', '启动状态', 'Startup state'],
  ['Vol', '电压', 'Voltage', 'V'],
  ['Cur', '电流', 'Current', 'mA'],
  ['Temp', '温度', 'Temperature', '°C'],
  ['LQI', '链路质量 LQI', 'Link quality LQI'],
  ['RSSI', '信号强度 RSSI', 'Signal strength RSSI'],
  ['TransTime', '传输时间', 'Transmission time'],
  ['RetransCount', '重传次数', 'Retransmissions'],
  ['TotalPackets', '总包数', 'Total packets'],
]
const batteryFields = [
  ['ID', '电池编号', 'Battery ID'],
  ['Model', '电池型号', 'Battery model'],
  ['MenuTime', '出厂时间', 'Manufactured at'],
  ['Vol', '电压', 'Voltage', 'V'],
  ['Cur', '电流', 'Current', 'mA'],
  ['Res', '内阻', 'Internal resistance', 'Ω'],
  ['RSSI', '信号强度 RSSI', 'Signal strength RSSI'],
  ['EfrTemp', '芯片温度', 'Chip temperature', '°C'],
  ['BattHealth', '电池健康度', 'Battery health', '%'],
  ['BattState', '电池状态', 'Battery status'],
  ['BattRep', '电池更换标志', 'Battery replacement flag'],
  ['TotalmWh', '总功耗', 'Total energy', 'mWh'],
  ['SleepTime', '休眠时间', 'Sleep time', 's'],
  ['ActTime', '运行时间', 'Active time', 'ms'],
  ['UpTime', '累计上传时间', 'Upload time', 'ms'],
  ['GathTime', '累计采集时间', 'Acquisition time', 'ms'],
]
const detailFields = computed(() => (sensorMode.value ? sensorFields : batteryFields).filter(([key]) => {
  const value = selected.value?.data[key]
  return value != null && value !== ''
}))

function showValue(value) {
  return value == null || value === '' ? '—' : typeof value === 'number' ? formatValue(value)
    : typeof value === 'object' ? JSON.stringify(value) : String(value)
}

function fieldValue(key, unit) {
  const value = selected.value.data[key]
  if (key === 'BattState') {
    const status = {
      0: t('未识别电池', 'Battery not detected'),
      1: t('正常 DT 电池', 'Normal DT battery'),
      2: t('非 DT 电池', 'Non-DT battery'),
      3: t('电量不足', 'Battery low'),
    }
    return status[value] ?? showValue(value)
  }
  return unit ? showValue(value) + ' ' + unit : showValue(value)
}
</script>

<template>
  <section class="reports">
    <nav class="tabs" :aria-label="t('上报类型', 'Report type')">
      <button type="button" :class="{ active: sensorMode }" @click="kind = 'sensor-info'">
        {{ t('传感器信息', 'Sensor information') }}
      </button>
      <button type="button" :class="{ active: !sensorMode }" @click="kind = 'battery-info'">
        {{ t('电池信息', 'Battery information') }}
      </button>
    </nav>
    <div v-if="records.length" class="report-workspace">
      <div class="report-list table-wrap scroll-region" tabindex="0" :aria-label="t('上报记录', 'Report history')">
        <table>
          <thead><tr>
            <th scope="col">{{ t('采样时间', 'Sampled at') }}</th>
            <th scope="col">{{ t('采集器', 'Collector') }}</th>
            <th scope="col">{{ t('通道', 'Channel') }}</th>
            <th scope="col">{{ identity }}</th>
            <th scope="col">{{ t('电压（V）', 'Voltage (V)') }}</th>
            <th scope="col">{{ t('电流（mA）', 'Current (mA)') }}</th>
            <th scope="col">{{ sensorMode ? t('温度（°C）', 'Temperature (°C)') : t('内阻（Ω）', 'Resistance (Ω)') }}</th>
            <th scope="col">RSSI</th>
          </tr></thead>
          <tbody>
            <tr v-for="row in records" :key="row.id" :class="{ selected: selected?.id === row.id }" @click="selectedId = row.id">
              <td class="report-time">
                <button type="button" :aria-pressed="selected?.id === row.id" @click.stop="selectedId = row.id">
                  <time class="mono" :datetime="row.time">{{ timeText(row.time) }}</time>
                </button>
              </td>
              <td class="mono">{{ row.collectorCode }}</td><td class="mono">{{ row.channelNo }}</td>
              <td class="mono">{{ showValue(row.data[sensorMode ? 'SN' : 'ID']) }}</td>
              <td class="mono">{{ showValue(row.data.Vol) }}</td>
              <td class="mono">{{ showValue(row.data.Cur) }}</td>
              <td class="mono">{{ showValue(row.data[sensorMode ? 'Temp' : 'Res']) }}</td>
              <td class="mono">{{ showValue(row.data.RSSI) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <section v-if="selected" class="report-details scroll-region" :aria-label="t('上报详情', 'Report details')">
        <h3>{{ t('上报详情', 'Report details') }}</h3>
        <dl>
          <div><dt>{{ t('采集器', 'Collector') }}</dt><dd>{{ selected.collectorCode }} · {{ t('通道', 'Channel') }} {{ selected.channelNo }}</dd></div>
          <div><dt>{{ sensorMode ? t('传感器时钟', 'Sensor clock') : t('采样时间', 'Sampled at') }}</dt><dd>{{ timeText(selected.data[sensorMode ? 'Clock' : 'Time']) }}</dd></div>
          <div><dt>{{ t('接收时间', 'Received at') }}</dt><dd>{{ timeText(selected.receivedAt) }}</dd></div>
          <div v-for="[key, zh, en, unit] in detailFields" :key="key"><dt>{{ t(zh, en) }}</dt><dd>{{ fieldValue(key, unit) }}</dd></div>
        </dl>
      </section>
    </div>
    <p v-else class="empty">
      {{ loading ? t('读取数据中…', 'Loading data…') : t('暂无匹配的上报记录', 'No matching reports') }}
    </p>
  </section>
</template>

<style scoped>
.reports { display: flex; flex: 1; flex-direction: column; min-height: 0; min-width: 0; gap: 12px; }
.report-workspace { display: grid; grid-template-columns: minmax(0, 1fr) minmax(210px, 260px); gap: 12px; flex: 1; min-height: 0; }
.report-list { max-height: none; }
.report-list tbody tr { cursor: pointer; }
.report-list tbody tr.selected { background: var(--accent-soft); }
.report-list .report-time { padding: 0; }
.report-time button { border: 0; background: transparent; padding: 8px; width: 100%; text-align: start; }
.report-details { border: 1px solid var(--border); border-radius: 8px; padding: 12px; }
.report-details dl { margin: 10px 0 0; font-size: 13px; }
.report-details dl div { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 8px; padding: 6px 0; border-top: 1px solid var(--border); }
.report-details dt { color: var(--muted); }
.report-details dd { margin: 0; overflow-wrap: anywhere; }
@media (max-width: 950px) {
  .report-workspace { grid-template-columns: minmax(0, 1fr); grid-template-rows: minmax(145px, 1fr) minmax(140px, 1fr); }
}
</style>
