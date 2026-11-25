<script setup>
import { computed, ref, watch } from 'vue'
import { api } from '../api.js'
import { directions, formatValue, indicatorNames, number, options, pointTypes, select, statusText, timeText } from '../config.js'
import { t, translateError } from '../i18n'
import { axisLabel, modelName } from '../modelTree.js'
import { getAlarmError } from '../validation.js'
import FormFields from './FormFields.vue'

const props = defineProps({
  kind: { type: String, required: true },
  draft: { type: Object, required: true },
  organizations: { type: Array, required: true },
  devices: { type: Array, required: true },
  points: { type: Array, required: true },
  activeDirection: { type: Number, default: 0 },
})

const measurements = {
  VIBRATION: [101, 107, 112],
  VIBRATION_3AXIS: [101, 107, 112],
  TEMPERATURE: [100],
  HUMIDITY: [110],
  PROCESS: [0],
  RTU: [0],
  MODBUS_TCP: [0],
  PT100: [0],
}

const isTriaxial = computed(() => props.kind === 'points' && props.draft.type === 'VIBRATION_3AXIS')
const alarmError = computed(() => props.kind === 'points' ? getAlarmError(props.draft) : null)
const operation = ref(null)
const operationError = ref('')

watch(isTriaxial, (triaxial) => {
  if (triaxial) {
    props.draft.axisNames ||= {}
    for (const alarm of props.draft.alarms) alarm.direction ??= 0
  }
}, { immediate: true })

watch([() => props.kind, () => props.draft.id, () => props.draft.stopEnabled], ([kind, id, enabled], _, onCleanup) => {
  operation.value = null
  operationError.value = ''
  if (kind !== 'devices' || !id || !enabled) return
  let active = true
  let loading = false
  async function refresh() {
    if (loading) return
    loading = true
    try {
      const result = await api('/devices/' + encodeURIComponent(id) + '/operation-status')
      if (active) {
        operation.value = result
        operationError.value = ''
      }
    } catch (error) {
      if (active) operationError.value = error.message
    } finally {
      loading = false
    }
  }
  refresh()
  const timer = setInterval(refresh, 5000)
  onCleanup(() => {
    active = false
    clearInterval(timer)
  })
}, { immediate: true })

const fields = computed(() => {
  const name = { key: 'name', label: t('名称', 'Name'), required: true, maxlength: 100 }
  if (props.kind !== 'points' || props.draft.axisGroupId) return [name]
  const types = options(pointTypes, false).filter((type) => !props.draft.id || type.value !== 'VIBRATION_3AXIS')
  return [name, select('type', t('测点类型', 'Point type'), types)]
})

const parent = computed(() => {
  if (props.kind === 'points') {
    const device = props.devices.find((row) => row.id === props.draft.deviceId)
    return { label: t('所属设备', 'Parent device'), name: device?.name }
  }
  const organization = props.organizations.find((row) => row.id === (props.draft.parentId || props.draft.organizationId))
  return {
    label: props.kind === 'devices' ? t('所属组织', 'Parent organization') : t('上级组织', 'Parent organization'),
    name: modelName('organizations', organization),
  }
})

const devicePoints = computed(() => props.points.filter((point) => point.deviceId === props.draft.id))
const stopPoints = computed(() => devicePoints.value.filter(eligibleStopPoint))

function stopPoint(id) {
  return devicePoints.value.find((point) => point.id === id)
}

function eligibleStopPoint(point) {
  return point?.type === 'VIBRATION' && (!point.axisGroupId || Number(point.direction) === 1)
}

function stopPointDirection(point) {
  return point?.axisGroupId ? Number(point.direction) : 0
}

function unsupportedStopRule(row) {
  if (!row.pointId) return false
  const point = stopPoint(row.pointId)
  const direction = Number(row.direction ?? 0)
  return !eligibleStopPoint(point) || (direction !== 0 && direction !== stopPointDirection(point))
}

function stopRuleDirectionLabel(row) {
  if (!row.pointId) return '—'
  const point = stopPoint(row.pointId)
  const configuredDirection = Number(row.direction ?? point?.direction ?? 0)
  const direction = configuredDirection === 0 && eligibleStopPoint(point) ? stopPointDirection(point) : configuredDirection
  if (direction === 0 && point?.type !== 'VIBRATION') return t('无方向', 'No axis')
  return directions[direction] ?? String(direction)
}

const stopFields = computed(() => [
  {
    ...select('pointId', t('判断测点', 'Detection point'), stopPoints.value.map((point) => ({
      value: point.id, label: point.name + ' · ' + directions[stopPointDirection(point)],
    }))),
    placeholder: t('选择本设备测点', 'Choose a point on this device'),
  },
  number('code', t('指标编码', 'Metric code')),
  number('limit', t('启停机门限', 'Start/stop threshold'), null, undefined, 'any'),
])

watch(
  () => (props.draft.stopLimits || []).map((rule) => ({ rule, pointId: rule.pointId, point: stopPoint(rule.pointId) })),
  (rules, previous = []) => {
    const previousPoints = new Map(previous.map(({ rule, pointId }) => [rule, pointId]))
    for (const { rule, pointId, point } of rules) {
      if (!eligibleStopPoint(point)) continue
      const switched = previousPoints.has(rule) && previousPoints.get(rule) !== pointId
      if (switched || Number(rule.direction ?? 0) === 0) rule.direction = stopPointDirection(point)
    }
  },
  { immediate: true, flush: 'sync' },
)

function replaceStopPoint(index) {
  const rule = props.draft.stopLimits[index]
  props.draft.stopLimits[index] = { ...rule, pointId: '', direction: 0 }
}

const alarmFields = computed(() => {
  const rows = [
    number('code', t('指标编码', 'Metric code'), 0),
    select('level', t('报警级别', 'Alarm level'),
      [1, 2, 3, 4].map((level) => ({ value: level, label: t(level + ' 级', 'Level ' + level) }))),
    number('limit', t('上限', 'Upper limit'), null, undefined, 'any'),
  ]
  if (isTriaxial.value) {
    const axisOptions = [{ value: 0, label: t('全部方向', 'All axes') }]
    for (const direction of [1, 2, 3]) axisOptions.push({ value: direction, label: axisLabel(props.draft, direction) })
    rows.splice(1, 0, select('direction', t('方向', 'Axis'), axisOptions))
  }
  return rows
})

function addAlarm() {
  const code = measurements[props.draft.type]?.[0] ?? 0
  const direction = isTriaxial.value ? props.activeDirection : 0
  const used = props.draft.alarms.filter((alarm) => Number(alarm.code) === code && (
    Number(alarm.direction || 0) === direction || !direction || !Number(alarm.direction || 0)
  )).map((alarm) => Number(alarm.level))
  const level = [1, 2, 3, 4].find((value) => !used.includes(value)) || 1
  const alarm = { code, level, type: 0, limit: '' }
  if (isTriaxial.value) alarm.direction = direction
  props.draft.alarms.push(alarm)
}

function addStopLimit() {
  props.draft.stopLimits.push({ pointId: '', code: 107, direction: 0, limit: '' })
}
</script>

<template>
  <div class="flow">
    <label v-if="parent.name" class="field model-parent">
      <span>{{ parent.label }}</span>
      <input :value="parent.name" readonly />
    </label>
    <FormFields :fields="fields" :model="draft" />
    <p v-if="kind === 'points' && draft.axisGroupId" class="muted">
      {{ t('测点方向', 'Point axis') }} · {{ directions[draft.direction] }}
    </p>
    <section v-if="isTriaxial" class="flow small divider">
      <h3>{{ t('三轴名称', 'Axis names') }}</h3>
      <div class="field-grid">
        <label v-for="direction in [1, 2, 3]" :key="direction" class="field">
          <span>{{ directions[direction] }}</span>
          <input v-model.trim="draft.axisNames[direction]" type="text" maxlength="80" :placeholder="directions[direction]" />
        </label>
      </div>
    </section>
    <section v-if="kind === 'devices'" class="flow small divider">
      <div class="actions split">
        <h3>{{ t('设备启停机门限', 'Device start/stop thresholds') }}</h3>
        <label class="check"><input v-model="draft.stopEnabled" type="checkbox" /> {{ t('启用', 'Enabled') }}</label>
      </div>
      <div v-if="draft.stopEnabled || draft.stopLimits.length" class="flow small">
        <p class="muted">{{ t('启停机使用本设备的单轴振动测点或三轴振动测点的 Z 轴。',
          'Start/stop detection uses a single-axis vibration point or the Z axis of a three-axis point on this device.') }}</p>
        <div v-for="(row, index) in draft.stopLimits" :key="index" class="flow small">
          <template v-if="unsupportedStopRule(row)">
            <p class="notice failure">{{ t('历史门限使用了不支持的测点或方向，当前不参与启停机判断。',
              'This legacy threshold uses an unsupported point or direction and is inactive.') }}</p>
            <div class="actions binding-fields">
              <p class="muted">
                {{ stopPoint(row.pointId)?.name || row.pointId }} · {{ stopRuleDirectionLabel(row) }}
                · {{ t('指标编码', 'Metric code') }} {{ row.code ?? 107 }}
                · {{ t('门限', 'Threshold') }} {{ formatValue(row.limit) }}
              </p>
              <button type="button" class="quiet" :disabled="!stopPoints.length" @click="replaceStopPoint(index)">
                {{ t('更换测点', 'Replace point') }}
              </button>
              <button type="button" class="quiet danger" @click="draft.stopLimits.splice(index, 1)">{{ t('移除', 'Remove') }}</button>
            </div>
          </template>
          <template v-else>
            <div class="actions binding-fields">
              <FormFields :fields="stopFields" :model="row" />
              <button type="button" class="quiet danger" @click="draft.stopLimits.splice(index, 1)">{{ t('移除', 'Remove') }}</button>
            </div>
            <p v-if="row.pointId" class="muted">{{ t('方向', 'Direction') }} · {{ stopRuleDirectionLabel(row) }}</p>
          </template>
        </div>
        <p v-if="!stopPoints.length" class="muted">{{ t('本设备暂无可用于启停机判断的测点。',
          'This device has no points eligible for start/stop detection.') }}</p>
        <button type="button" class="start" :disabled="!stopPoints.length" @click="addStopLimit">
          {{ t('添加门限', 'Add threshold') }}
        </button>
        <p v-if="draft.stopEnabled && operation?.state === 'DISABLED'" class="muted divider">
          {{ t('启停机配置尚未保存。保存后可查看判断状态。', 'Save the start/stop settings to see the operation state.') }}
        </p>
        <div v-else-if="operation" class="flow small divider">
          <p>
            <strong>{{ t('当前启停机状态', 'Current operation state') }}</strong>
            · {{ statusText(operation.state) }}
            <span v-if="operation.observedAt" class="muted"> · {{ timeText(operation.observedAt) }}</span>
          </p>
          <div v-if="operation.rules?.length" class="table-wrap">
            <table>
              <thead><tr>
                <th>{{ t('判断测点', 'Detection point') }}</th>
                <th>{{ t('方向', 'Direction') }}</th>
                <th>{{ t('指标编码', 'Metric code') }}</th>
                <th>{{ t('门限', 'Threshold') }}</th>
                <th>{{ t('最新值', 'Latest value') }}</th>
                <th>{{ t('状态', 'Status') }}</th>
              </tr></thead>
              <tbody>
                <tr v-for="rule in operation.rules" :key="rule.pointId + '/' + rule.code + '/' + rule.direction">
                  <td>{{ rule.pointName || rule.pointId }}</td><td>{{ stopRuleDirectionLabel(rule) }}</td>
                  <td>{{ rule.code }}</td><td>{{ formatValue(rule.limit) }}</td>
                  <td>{{ formatValue(rule.value) }}</td><td>{{ statusText(rule.state) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <p v-else-if="operationError" class="muted">{{ translateError(operationError) }}</p>
      </div>
    </section>
    <section v-if="kind === 'points'" class="flow small divider">
      <div class="actions split">
        <h3>{{ t('测点报警门限', 'Point alarm thresholds') }}</h3>
        <button type="button" @click="addAlarm">{{ t('添加报警门限', 'Add alarm threshold') }}</button>
      </div>
      <p class="muted">
        {{ t('指标编码', 'Metric codes:') }}
        {{ measurements[draft.type]?.map((code) => code + ' ' + indicatorNames[code]).join(t('、', ', ')) }}
      </p>
      <div v-for="(row, index) in draft.alarms" :key="index" class="actions binding-fields">
        <FormFields :fields="alarmFields" :model="row" />
        <button type="button" class="quiet danger" @click="draft.alarms.splice(index, 1)">{{ t('移除', 'Remove') }}</button>
      </div>
      <p v-if="alarmError" class="notice failure">{{ alarmError }}</p>
    </section>
  </div>
</template>

<style scoped>
.model-parent { width: min(100%, 360px); }
</style>
