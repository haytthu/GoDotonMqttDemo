<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { api, query } from '../api.js'
import { channelTypes, formatValue, indicatorNames, timeText } from '../config.js'
import { t, translateError } from '../i18n'
import { axisLabel } from '../modelTree.js'
import DeviceReports from './DeviceReports.vue'
import ModelTree from './ModelTree.vue'
import SensorChart from './SensorChart.vue'

const props = defineProps({
  organizations: { type: Array, required: true },
  devices: { type: Array, required: true },
  collectors: { type: Array, required: true },
  points: { type: Array, required: true },
  initialCollectorId: { type: String, default: '' },
})

const tab = ref('metrics')
const reportKind = ref('sensor-info')
const collectorId = ref(props.initialCollectorId)
const selection = ref(null)
const waveType = ref(null)
const records = ref([])
const metricTrends = ref([])
const wave = ref(null)
const spectrum = ref(null)
const selectedWave = ref('')
const loading = ref(false)
const loadingChart = ref(false)
const error = ref('')
const audioError = ref(false)
let listRequest = 0
let chartRequest = 0

const vibrationKinds = [
  { waveCode: 11, metricCode: 112, unit: 'm/s²', labels: ['高频加速度', 'High-frequency acceleration'], switchLabels: ['高频', 'HF'] },
  { waveCode: 12, metricCode: 101, unit: 'm/s²', labels: ['低频加速度', 'Low-frequency acceleration'], switchLabels: ['低频', 'LF'] },
  { waveCode: 15, metricCode: 107, unit: 'mm/s', labels: ['速度', 'Velocity'], switchLabels: ['速度', 'Velocity'] },
]
const waveKinds = [
  ...vibrationKinds,
  { waveCode: 1, unit: 'm/s²', labels: ['长波形', 'Long waveform'], switchLabels: ['长波形', 'Long wave'] },
]
const audioKind = { unit: 'PCM', labels: ['音频', 'Audio'] }
const temperatureKind = { metricCode: 100, unit: '°C', labels: ['温度', 'Temperature'] }
const extensionPointTypes = new Set(['RTU', 'MODBUS_TCP', 'PT100'])
const visibleMetricTypes = Object.keys(channelTypes).join(',')
const automaticWaveType = ref(vibrationKinds[0].waveCode)
const recordLimit = 250
const pointId = computed(() => selection.value?.record.id || '')
const audioMode = computed(() => tab.value === 'audio')
const waveMode = computed(() => tab.value === 'waves' || audioMode.value)
const showModel = computed(() => tab.value === 'metrics' || tab.value === 'waves')
const dataPoints = computed(() => props.points.filter((point) => !extensionPointTypes.has(point.type)))
const metricKinds = computed(() => {
  if (selection.value?.record.type === 'VIBRATION') return [...vibrationKinds, temperatureKind]
  if (selection.value?.record.type === 'TEMPERATURE') return [temperatureKind]
  return []
})
const activeWave = computed(() => records.value.find((row) => row.id === selectedWave.value))
const activeWaveKind = computed(() => audioMode.value ? audioKind
  : waveKinds.find((kind) => kind.waveCode === (waveType.value ?? automaticWaveType.value)))
const waveTrend = computed(() => timeline(records.value, 'rms'))
const heading = computed(() => {
  if (audioMode.value) return t('音频记录', 'Audio recordings')
  if (tab.value === 'reports') return t('采集器上报', 'Collector reports')
  return selection.value ? pointName(pointId.value) : t('全部测点', 'All points')
})

function pointById(id) {
  return props.points.find((point) => point.id === id)
}
function pointName(id) {
  const point = pointById(id)
  return point?.label || point?.name || id || t('未绑定', 'Unassigned')
}
function directionName(id, direction) {
  return axisLabel(pointById(id), direction) || '—'
}
function metricChannel(record) {
  const number = record.subChannel > 0 ? record.channelNo + '/' + record.subChannel : record.channelNo
  return (channelTypes[record.channelType] || record.channelType) + ' · ' + number
}
function collectorName(id) {
  return props.collectors.find((collector) => collector.id === id)?.code || id || '—'
}
function indicatorName(code) {
  return indicatorNames[code] || code
}

function sampleTimestamp(value) {
  // The API returns wall-clock times in the configured collection timezone.
  return Date.parse(value ? value.replace(' ', 'T') + 'Z' : '')
}

function timeline(rows, valueKey = 'value') {
  const samples = rows.map((row) => ({ time: sampleTimestamp(row.time), value: row[valueKey] }))
    .filter(({ time, value }) => Number.isFinite(time) && Number.isFinite(value))
    .sort((a, b) => a.time - b.time)
  return { timestamps: samples.map((sample) => sample.time), values: samples.map((sample) => sample.value) }
}

async function loadMetricTrends(filters) {
  const groups = await Promise.all(metricKinds.value.map(async (kind) => {
    const types = kind === temperatureKind ? '0,3' : '0'
    const rows = await api('/metrics' + query({ ...filters, channelTypes: types, code: kind.metricCode }))
    const byChannel = new Map()
    for (const row of rows || []) {
      const channel = [row.collectorId, row.channelType, row.channelNo, row.direction].join('/')
      if (!byChannel.has(channel)) byChannel.set(channel, [])
      byChannel.get(channel).push(row)
    }
    return [...byChannel.entries()].map(([channel, samples]) => ({
      ...timeline(samples),
      key: kind.metricCode + '/' + channel,
      kind,
      channel: byChannel.size > 1 ? samples[0] : null,
    })).filter((series) => series.values.length)
  }))
  return groups.flat()
}

function selectPoint(node) {
  if (node.record.id !== pointId.value) waveType.value = null
  selection.value = node
}

async function refresh() {
  const request = ++listRequest
  const previousWave = selectedWave.value
  loading.value = true
  records.value = []
  metricTrends.value = []
  error.value = ''
  selectedWave.value = ''
  wave.value = null
  spectrum.value = null
  audioError.value = false
  chartRequest++
  loadingChart.value = false
  try {
    const filters = {
      collectorId: collectorId.value,
      pointId: showModel.value ? pointId.value : '',
      limit: recordLimit,
    }
    if (tab.value === 'metrics' && metricKinds.value.length) {
      const trends = await loadMetricTrends(filters)
      if (request === listRequest) metricTrends.value = trends
      return
    }
    if (tab.value === 'reports') {
      const rows = await api('/' + reportKind.value + query(filters))
      if (request === listRequest) records.value = rows || []
      return
    }
    if (waveMode.value) {
      const audio = audioMode.value
      let code = audio ? 0 : waveType.value
      if (!audio && code == null) {
        const latest = await api('/waves' + query({ ...filters, limit: 1, audio: false,
          code: waveKinds.map((kind) => kind.waveCode).join(',') }))
        if (request !== listRequest) return
        code = latest?.length ? Number(latest[0].code) : vibrationKinds[0].waveCode
        automaticWaveType.value = code
      }
      const rows = await api('/waves' + query({ ...filters, code, audio }))
      if (request !== listRequest) return
      records.value = audio ? (rows || []).filter((row) => row.code === 0) : rows || []
      const selected = records.value.find((row) => row.id === previousWave) || records.value[0]
      if (selected) viewWave(selected)
      return
    }
    const rows = await api('/metrics' + query({ ...filters, channelTypes: visibleMetricTypes }))
    if (request !== listRequest) return
    records.value = rows || []
  } catch (e) {
    if (request === listRequest) error.value = e.message
  } finally {
    if (request === listRequest) loading.value = false
  }
}

async function viewWave(record) {
  const id = record.id
  const request = ++chartRequest
  selectedWave.value = id
  wave.value = null
  spectrum.value = null
  loadingChart.value = true
  error.value = ''
  audioError.value = false
  try {
    const result = await Promise.allSettled([
      api('/waves/' + encodeURIComponent(id)),
      record.length > 1 ? api('/waves/' + encodeURIComponent(id) + '/spectrum') : Promise.resolve(null),
    ])
    if (request !== chartRequest) return
    if (result[0].status === 'fulfilled') wave.value = result[0].value
    if (result[1].status === 'fulfilled') spectrum.value = result[1].value
    const failure = result.find((item) => item.status === 'rejected')
    if (failure) error.value = failure.reason.message
  } finally {
    if (request === chartRequest) loadingChart.value = false
  }
}

function viewWaveAt(time) {
  let closest = null
  let distance = Infinity
  for (const row of records.value) {
    if (!Number.isFinite(row.rms)) continue
    const difference = Math.abs(sampleTimestamp(row.time) - time)
    if (difference < distance) {
      closest = row
      distance = difference
    }
  }
  if (closest && closest.id !== selectedWave.value) viewWave(closest)
}

watch([tab, collectorId, pointId, waveType, reportKind], refresh)
onMounted(refresh)
</script>

<template>
  <div class="management acquisition-layout" :class="{ 'no-model': !showModel }">
    <section v-if="showModel" class="panel registry viewport-panel">
      <div class="actions split">
        <h2>{{ t('设备模型', 'Device model') }}</h2>
        <button type="button" class="quiet" :disabled="!selection" @click="selection = null">{{ t('全部测点', 'All points') }}</button>
      </div>
      <div class="panel-body scroll-region" tabindex="0" :aria-label="t('设备树', 'Device tree')">
        <ModelTree :organizations="organizations" :devices="devices" :points="dataPoints"
          :selected-key="selection?.key" read-only @select="selectPoint" />
      </div>
    </section>

    <section class="panel viewport-panel data-panel">
      <h2>{{ heading }}</h2>
      <div class="actions split">
        <nav class="tabs data-tabs" :aria-label="t('数据类型', 'Data type')">
          <button type="button" :class="{ active: tab === 'metrics' }" @click="tab = 'metrics'">
            {{ t('指标数据', 'Metrics') }}
          </button>
          <button type="button" :class="{ active: tab === 'waves' }" @click="tab = 'waves'">
            {{ t('波形与频谱', 'Waveforms and spectrum') }}
          </button>
          <button type="button" :class="{ active: audioMode }" @click="tab = 'audio'">{{ t('音频', 'Audio') }}</button>
          <button type="button" :class="{ active: tab === 'reports' }" @click="tab = 'reports'">
            {{ t('传感器与电池', 'Sensors & batteries') }}
          </button>
        </nav>
        <div class="actions data-filters">
          <select v-model="collectorId" class="collector-filter" :aria-label="t('采集器', 'Collector')">
            <option value="">{{ t('全部采集器', 'All collectors') }}</option>
            <option v-for="collector in collectors" :key="collector.id" :value="collector.id">
              {{ collector.code }} · {{ collector.name }}
            </option>
          </select>
          <button type="button" :disabled="loading" @click="refresh">{{ loading ? t('查询中…', 'Loading…') : t('刷新', 'Refresh') }}</button>
        </div>
      </div>
      <p v-if="error" class="notice failure" role="alert">{{ translateError(error) }}</p>
      <div class="data-results panel-body">
        <p v-if="loading && tab === 'metrics'" class="empty">{{ t('读取数据中…', 'Loading data…') }}</p>
        <template v-else-if="tab === 'metrics' && metricKinds.length">
          <div v-if="metricTrends.length" class="flow chart-grid metric-charts scroll-region" tabindex="0" :aria-label="t('指标趋势', 'Metric trends')">
            <SensorChart v-for="series in metricTrends" :key="series.key" :label="t(...series.kind.labels)"
              :detail="series.channel ? collectorName(series.channel.collectorId) + ' · ' + metricChannel(series.channel) + ' · ' + directionName(series.channel.pointId, series.channel.direction) : ''"
              :values="series.values" :timestamps="series.timestamps" :unit="series.kind.unit" />
          </div>
          <p v-else class="empty">{{ t('该测点暂无指标数据', 'No metrics for this point') }}</p>
        </template>
        <DeviceReports v-else-if="tab === 'reports'" v-model:kind="reportKind" :records="records" :loading="loading" />
        <p v-else-if="audioMode && !records.length" class="empty">
          {{ loading ? t('读取数据中…', 'Loading data…') : t('暂无音频记录', 'No audio recordings') }}
        </p>
        <template v-else-if="waveMode">
          <div class="wave-workspace">
            <section class="wave-records" :aria-label="audioMode ? t('音频记录', 'Audio recordings') : t('波形记录', 'Waveform records')">
              <div v-if="!audioMode" class="wave-kind-tabs" role="group" :aria-label="t('波形类型', 'Waveform type')">
                <button v-for="kind in waveKinds" :key="kind.waveCode" type="button"
                  :aria-pressed="activeWaveKind?.waveCode === kind.waveCode" :title="t(...kind.labels)"
                  @click="waveType = kind.waveCode">{{ t(...kind.switchLabels) }}</button>
              </div>
              <div class="wave-record-table scroll-region" tabindex="0" :aria-label="t('波形记录', 'Waveform records')">
                <table>
                  <thead><tr><th scope="col">{{ t('时间', 'Time') }}</th>
                    <th scope="col">{{ audioMode ? t('通道', 'Channel') : t('有效值', 'RMS') }}</th></tr></thead>
                  <tbody>
                    <tr v-for="record in records" :key="record.id" :class="{ selected: selectedWave === record.id }" @click="viewWave(record)">
                      <td><button type="button" class="wave-record" :aria-pressed="selectedWave === record.id" @click.stop="viewWave(record)">
                        <time class="mono" :datetime="record.time">{{ timeText(record.time) }}</time>
                        <span v-if="audioMode" class="muted audio-source">{{ collectorName(record.collectorId) }}</span>
                      </button></td>
                      <td class="mono reading">{{ audioMode ? record.channelNo : formatValue(record.rms) }}</td>
                    </tr>
                  </tbody>
                </table>
                <p v-if="!records.length" class="empty compact">
                  {{ loading ? t('读取数据中…', 'Loading data…') : t('暂无匹配的波形记录', 'No matching waveform records') }}
                </p>
              </div>
            </section>
            <div class="flow chart-grid wave-charts scroll-region" :class="{ 'with-player': audioMode && activeWave }"
              tabindex="0" :aria-label="t('波形图谱', 'Waveform charts')">
              <section v-if="audioMode && activeWave" class="audio-player" :aria-label="t('音频播放', 'Audio playback')">
                <div class="actions split">
                  <strong>{{ t('音频播放', 'Audio playback') }}</strong>
                  <span class="muted">{{ collectorName(activeWave.collectorId) }} · {{ t('通道', 'Channel') }} {{ activeWave.channelNo }}</span>
                </div>
                <audio :key="activeWave.id" :src="'/api/waves/' + encodeURIComponent(activeWave.id) + '/audio'"
                  controls preload="metadata" :aria-label="t('播放音频', 'Play audio')" @error="audioError = true" />
                <p v-if="audioError" class="notice failure" role="alert">{{ t('无法播放此音频', 'This audio could not be played') }}</p>
              </section>
              <SensorChart :label="t('有效值趋势', 'RMS trend')" :detail="activeWaveKind ? t(...activeWaveKind.labels) : ''"
                :values="waveTrend.values" :timestamps="waveTrend.timestamps"
                :unit="activeWaveKind?.unit || ''" :selected-time="sampleTimestamp(activeWave?.time)" selectable
                :empty-text="loading ? t('读取有效值中…', 'Loading RMS values…') : t('暂无波形有效值', 'No waveform RMS values')"
                @select-time="viewWaveAt" />
              <SensorChart :label="t('时域波形', 'Time waveform')" :values="wave?.values || []"
                :step="wave?.sampleRate ? 1 / wave.sampleRate : 1" :unit="wave?.unit || ''"
                :detail="wave ? formatValue(wave.sampleRate) + ' Hz · ' + wave.values.length + ' ' + t('点', 'samples') : ''"
                :empty-text="loadingChart ? t('读取波形中…', 'Loading waveform…') : t('暂无波形数据', 'No waveform samples')" />
              <SensorChart :label="t('频谱', 'Spectrum')" :values="spectrum?.values || []"
                :step="spectrum?.frequencyStep || 1" x-unit="Hz" :unit="spectrum?.unit || ''"
                :empty-text="loadingChart ? t('计算频谱中…', 'Calculating spectrum…') : t('暂无频谱数据', 'No spectrum data')" />
            </div>
          </div>
        </template>
        <div v-else-if="records.length" class="table-wrap data-table scroll-region" tabindex="0" :aria-label="t('指标记录', 'Metric records')">
          <table>
            <thead>
              <tr>
                <th>{{ t('采集时间', 'Sampled at') }}</th><th>{{ t('测点', 'Point') }}</th><th>{{ t('采集器', 'Collector') }}</th>
                <th>{{ t('通道', 'Channel') }}</th><th>{{ t('方向', 'Axis') }}</th><th>{{ t('指标', 'Metric') }}</th><th>{{ t('数值', 'Value') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(record, index) in records" :key="index">
                <td class="mono">{{ timeText(record.time) }}</td>
                <td>{{ pointName(record.pointId) }}</td>
                <td>{{ collectorName(record.collectorId) }}</td>
                <td>{{ metricChannel(record) }}</td>
                <td>{{ Number(record.channelType) === 0 ? directionName(record.pointId, record.direction) : '—' }}</td>
                <td>{{ indicatorName(record.code) }}</td>
                <td class="mono reading">{{ formatValue(record.value) }}</td>
              </tr>
            </tbody>
          </table>
      </div>
      <p v-else class="empty">{{ t('暂无匹配的数据', 'No matching data') }}</p>
      </div>
    </section>
  </div>
</template>

<style scoped>
.acquisition-layout { grid-template-columns: 250px minmax(0, 1fr); }
.acquisition-layout.no-model { grid-template-columns: minmax(0, 1fr); }
.data-filters { flex: 1 1 240px; justify-content: flex-end; }
.collector-filter { flex: 0 1 230px; font-size: 14px; }
.data-panel > .notice { max-height: 20vh; overflow: auto; }
.data-results { display: flex; flex-direction: column; }
.data-results > .data-table, .metric-charts, .wave-workspace { flex: 1; min-width: 0; min-height: 0; max-height: none; }
.metric-charts { grid-template-columns: repeat(auto-fit, minmax(min(330px, 100%), 1fr)); grid-auto-rows: minmax(180px, 1fr); align-content: stretch; }
.wave-workspace { display: grid; grid-template: minmax(0, 1fr) / clamp(232px, 24%, 250px) minmax(0, 1fr); gap: 12px; }
.wave-records { display: flex; flex-direction: column; min-width: 0; min-height: 0; border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
.wave-kind-tabs { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 4px; padding: 7px; border-bottom: 1px solid var(--border); }
.wave-kind-tabs button { padding: 5px 3px; font-size: 12px; }
.wave-kind-tabs button[aria-pressed='true'] { background: var(--accent-soft); border-color: #dfe3fb; color: #313fb4; }
.wave-record-table { flex: 1; }
.wave-record-table table { table-layout: fixed; }
.wave-record-table th, .wave-record-table td { padding: 8px; vertical-align: middle; }
.wave-record-table th:last-child, .wave-record-table td:last-child { text-align: right; }
.wave-record-table th:last-child { width: 76px; }
.wave-record-table tbody tr { cursor: pointer; }
.wave-record-table tbody td:first-child { padding: 0; }
.wave-record-table tbody tr.selected td:first-child { box-shadow: inset 3px 0 #5459b7; }
.wave-record-table td.reading { font-size: 11px; white-space: normal; overflow-wrap: anywhere; }
.wave-record { display: block; width: 100%; text-align: left; border: 0; border-radius: 0; padding: 8px; background: transparent; font-size: 11px; }
.wave-charts { grid-template-rows: repeat(3, minmax(135px, 1fr)); }
.wave-charts.with-player { grid-template-rows: auto repeat(3, minmax(135px, 1fr)); }
.audio-source { display: block; margin-top: 3px; overflow-wrap: anywhere; }
.audio-player { display: grid; gap: 8px; padding: 10px 12px; border: 1px solid var(--border); border-radius: 8px; }
.audio-player audio { display: block; width: 100%; height: 44px; }
.data-results :deep(.chart) { display: flex; flex-direction: column; min-height: 0; }
.data-results :deep(.chart-header) { flex: none; }
.data-results :deep(.chart canvas) { flex: 1; height: 0; min-height: 0; }
@media (max-width: 1100px) {
  .acquisition-layout { grid-template-columns: 196px minmax(0, 1fr); }
}
@media (max-width: 900px) {
  .wave-workspace { display: block; overflow: auto; overscroll-behavior: contain; }
  .wave-records { max-height: 190px; margin-bottom: 12px; }
  .wave-charts { grid-template-rows: repeat(3, minmax(155px, 1fr)); overflow: visible; }
  .wave-charts.with-player { grid-template-rows: auto repeat(3, minmax(155px, 1fr)); }
}
@media (max-width: 760px) {
  .acquisition-layout { grid-template-columns: 1fr; }
}
</style>

