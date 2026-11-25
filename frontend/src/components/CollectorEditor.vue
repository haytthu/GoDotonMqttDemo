<script setup>
import { computed } from 'vue'
import { baseChannels, defaults, directions, nextNumber, number, options, pointTypes, select, systemDefaults, vibrationPresets } from '../config.js'
import { t } from '../i18n.js'
import FormFields from './FormFields.vue'

const props = defineProps({
  collector: { type: Object, required: true },
  points: { type: Array, required: true },
  codeLocked: Boolean,
})

const tab = defineModel('activeTab', { type: String, required: true })
const tabs = computed(() => [
  { id: 'system', label: t('采集参数', 'Acquisition settings') },
  { id: 'channels', label: t('基础通道', 'Base channels') },
])

const baseFields = computed(() => [
  { key: 'name', label: t('采集器名称', 'Collector name'), required: true, maxlength: 100 },
  { key: 'code', label: t('采集器编号', 'Collector ID'), required: true, maxlength: 64, readonly: props.codeLocked,
    hint: t('填写设备实际使用的 MQTT 编号', 'Use the collector ID from its MQTT topics') },
])

const systemFields = computed(() => [
  number('indexPeriod', t('指标采集间隔（秒）', 'Indicator interval (s)')),
  number('wavePeriod', t('波形采集间隔（秒）', 'Waveform interval (s)')),
  number('longWavePeriod', t('长波形采集间隔（秒）', 'Long waveform interval (s)')),
  number('longWaveOffset', t('长波形时间偏移（秒）', 'Long waveform offset (s)')),
  { key: 'collectType', label: t('采集模式', 'Collection mode'), displayValue: t('定时采集', 'Scheduled') },
  select('waveCompress', t('波形压缩', 'Waveform compression'), options({ 0: t('关闭', 'Off'), 1: 'bzip2' })),
])

const vibrationLabels = computed(() => [
  ['highFreq', t('高频加速度频率（Hz）', 'High-frequency acceleration (Hz)')],
  ['highAccLength', t('高频加速度点数', 'High-frequency acceleration samples')],
  ['longWaveFreq', t('长波形频率（Hz）', 'Long waveform frequency (Hz)')],
  ['longWaveLength', t('长波形点数', 'Long waveform samples')],
  ['lowAccFreq', t('低频加速度频率（Hz）', 'Low-frequency acceleration (Hz)')],
  ['lowAccLength', t('低频加速度点数', 'Low-frequency acceleration samples')],
  ['plusLength', t('低频加速度附加长度（点）', 'Extra low-frequency acceleration samples')],
  ['velocityLowFreq', t('速度下限频率（Hz）', 'Velocity low cutoff (Hz)')],
  ['velocityHighFreq', t('速度上限频率（Hz）', 'Velocity high cutoff (Hz)')],
  ['velocityLength', t('速度目标点数', 'Target velocity samples')],
])

function vibrationFields(model, inherited) {
  const zeroLabels = {
    highAccLength: t('不采集（0）', 'Off (0)'),
    longWaveLength: t('不采集（0）', 'Off (0)'),
    plusLength: t('无附加点（0）', 'No extra samples (0)'),
    velocityLowFreq: t('0（自动，2 Hz）', '0 (auto, 2 Hz)'),
    velocityLength: t('不计算（0）', 'Off (0)'),
  }
  return vibrationLabels.value.map(([key, label]) => {
    const values = vibrationPresets[key]
    const items = values.map((value) => ({
      value,
      label: value === 0 ? zeroLabels[key] : String(value),
    }))
    const current = model[key]
    const usesDefault = current == null
    if (usesDefault) {
      const value = inherited[key] ?? defaults[key]
      items.unshift({ value: current, label: t('默认（' + value + '）', 'Default (' + value + ')') })
    } else if (current !== '' && !values.includes(Number(current))) {
      items.unshift({ value: current, label: t(current + '（当前值）', current + ' (current value)') })
    }
    return select(key, label, items, !usesDefault)
  })
}

const bindingFields = computed(() => ({
  types: options(Object.fromEntries(Object.entries(baseChannels).map(([type, label]) => [type, pointTypes[label]]))),
  points: Object.fromEntries(Object.entries(baseChannels).map(([type, name]) => [
    type,
    props.points.filter((point) => point.type === name && (!point.axisGroupId || point.id === point.axisGroupId))
      .map((point) => ({
        value: point.id,
        label: point.axisGroupId
          ? `${point.groupLabel || point.axisGroupName || point.label || point.name} · ${pointTypes.VIBRATION_3AXIS}`
          : point.label || point.name,
      })),
  ])),
}))

function threeAxis(channel) {
  return Number(channel.channelType) === 0
    && props.points.some((point) => point.id === channel.pointId && point.axisGroupId === point.id)
}

function channelFields(channel) {
  const fields = [
    number('channelNo', t('通道号', 'Channel number'), 1, 256),
    select('channelType', t('通道类型', 'Channel type'), bindingFields.value.types),
    {
      ...select('pointId', t('绑定测点', 'Measurement point'), bindingFields.value.points[channel.channelType] || []),
      placeholder: t('选择同类型测点', 'Select a compatible point'),
    },
  ]
  if (Number(channel.channelType) === 0) {
    if (!threeAxis(channel)) fields.push(select('direction', t('方向', 'Direction'), options(directions)))
    fields.push(select('sensorType', t('传感器类型', 'Sensor type'),
      options({ 0: t('通用', 'Generic'), 1: t('压电', 'Piezoelectric'), 2: 'MEMS' })))
    fields.push(number('sensitivity', t('灵敏系数', 'Sensitivity'), 0, undefined, 'any'))
  } else if (Number(channel.channelType) === 2) {
    fields.push(number('sensitivity', t('灵敏系数', 'Sensitivity'), null, undefined, 'any'))
    fields.push(number('amendment', t('修正值', 'Correction'), null, undefined, 'any'))
  }
  return fields
}

function hasOverride(channel) {
  return Object.keys(vibrationPresets).some((key) => key in channel)
}

function toggleOverride(channel) {
  if (hasOverride(channel)) {
    for (const key of Object.keys(vibrationPresets)) delete channel[key]
  } else {
    Object.assign(channel, defaults, props.collector.vibrationDefaults)
  }
}

function addChannel() {
  const no = nextNumber(props.collector.channels.filter((row) => Number(row.channelType) === 0), 'channelNo', 256)
  if (no != null) props.collector.channels.push({ channelNo: no, channelType: 0, direction: 0, pointId: '', sensorType: 0, sensitivity: 10, amendment: 0 })
}

function resetVibrationDefaults() {
  Object.assign(props.collector.vibrationDefaults, defaults)
  Object.assign(props.collector.system, systemDefaults)
}

function updateChannelBinding(channel, key) {
  if (key !== 'channelType' && key !== 'pointId') return
  const type = Number(channel.channelType)
  if (channel.pointId && !bindingFields.value.points[type]?.some((item) => item.value === channel.pointId)) {
    channel.pointId = ''
  }
  if (type !== 0 || threeAxis(channel)) channel.direction = 0
}
</script>

<template>
  <div class="flow">
    <div class="actions split">
      <FormFields :model="collector" :fields="baseFields" class="collector-identity" />
      <span class="tag model-tag">SG500Z</span>
    </div>
    <nav class="tabs" :aria-label="t('采集器设置', 'Collector settings')">
      <button v-for="item in tabs" :key="item.id" type="button"
        :class="{ active: tab === item.id }" :aria-selected="tab === item.id"
        @click="tab = item.id">
        {{ item.label }}
        <span v-if="item.id !== 'system' && collector[item.id]?.length" class="tab-count">
          {{ collector[item.id].length }}
        </span>
      </button>
    </nav>

    <template v-if="tab === 'system'">
      <section class="flow small">
        <div class="actions split">
          <h3>{{ t('系统参数', 'System settings') }}</h3>
          <button type="button" class="quiet" @click="resetVibrationDefaults">{{ t('恢复默认参数', 'Restore defaults') }}</button>
        </div>
        <FormFields :model="collector.system" :fields="systemFields" />
      </section>
      <section class="flow small divider">
        <h3>{{ t('默认振动参数', 'Default vibration settings') }}</h3>
        <FormFields :model="collector.vibrationDefaults" :fields="vibrationFields(collector.vibrationDefaults, defaults)" />
      </section>
    </template>

    <div v-else-if="tab === 'channels'" class="flow">
      <div class="actions split">
        <p class="muted">{{ t('选择通道类型、方向和对应测点。', 'Select a channel type, direction and measurement point.') }}</p>
        <button type="button" @click="addChannel">{{ t('添加基础通道', 'Add base channel') }}</button>
      </div>
      <p v-if="!collector.channels.length" class="empty compact">{{ t('尚未配置基础通道', 'No base channels configured') }}</p>
      <section v-for="(channel, index) in collector.channels" :key="index" class="subpanel flow">
        <div class="actions split">
          <h3>{{ t('通道', 'Channel') }} {{ channel.channelNo }}</h3>
          <div class="actions">
            <button v-if="Number(channel.channelType) === 0" type="button" class="quiet"
              @click="toggleOverride(channel)">
              {{ hasOverride(channel) ? t('使用默认参数', 'Use defaults') : t('单独配置振动参数', 'Override vibration settings') }}
            </button>
            <button type="button" class="quiet danger" @click="collector.channels.splice(index, 1)">{{ t('移除', 'Remove') }}</button>
          </div>
        </div>
        <FormFields :model="channel" :fields="channelFields(channel)" @field-change="updateChannelBinding(channel, $event)" />
        <p v-if="threeAxis(channel)" class="muted">
          {{ t('此测点自动绑定当前通道的 Z、X、Y 轴。', 'This point automatically uses the Z, X and Y axes of this channel.') }}
        </p>
        <FormFields v-if="Number(channel.channelType) === 0 && hasOverride(channel)"
          :model="channel" :fields="vibrationFields(channel, collector.vibrationDefaults)" />
      </section>
    </div>

  </div>
</template>
