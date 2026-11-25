<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { formatValue } from '../config.js'
import { locale, t } from '../i18n.js'

const props = defineProps({
  values: { type: Array, required: true },
  timestamps: { type: Array, default: null },
  selectedTime: { type: Number, default: null },
  selectable: Boolean,
  step: { type: Number, default: 1 },
  xUnit: { type: String, default: 's' },
  unit: { type: String, default: '' },
  label: { type: String, default: '' },
  detail: { type: String, default: '' },
  emptyText: { type: String, default: '' },
})

const emit = defineEmits(['select-time'])
const chartLabel = computed(() => props.label || t('数据曲线', 'Data chart'))
const timeAxis = computed(() => props.timestamps?.length && props.timestamps.length === props.values.length)
const canvas = ref(null)
const pointer = ref('')
let observer

function plotBounds(width, height) {
  return { left: width < 360 ? 58 : 72, top: 16, right: width - 19, bottom: height - 31 }
}

function xDomain() {
  if (!timeAxis.value) return [0, Math.max(props.values.length - 1, 1) * props.step]
  const first = props.timestamps[0]
  const last = props.timestamps[props.timestamps.length - 1]
  return Number.isFinite(props.selectedTime)
    ? [Math.min(first, props.selectedTime), Math.max(last, props.selectedTime)]
    : [first, last]
}

function timeLabel(time, detailed) {
  const date = new Date(time).toISOString().replace('T', ' ')
  return detailed ? date.slice(0, 23) : date.slice(5, 19)
}

function draw() {
  if (!canvas.value) return
  const { width, height } = canvas.value.getBoundingClientRect()
  if (!width || !height) return
  const scale = window.devicePixelRatio || 1
  canvas.value.width = width * scale
  canvas.value.height = height * scale
  const ctx = canvas.value.getContext('2d')
  ctx.scale(scale, scale)
  const plot = plotBounds(width, height)
  const plotWidth = plot.right - plot.left
  const plotHeight = plot.bottom - plot.top
  if (!props.values.length || plotWidth < 1) return

  let min = Infinity
  let max = -Infinity
  for (const value of props.values) {
    if (!Number.isFinite(value)) continue
    min = Math.min(min, value)
    max = Math.max(max, value)
  }
  if (min === Infinity) return
  const padding = max === min ? Math.max(Math.abs(min) * 0.1, 0.01) : (max - min) * 0.07
  min -= padding
  max += padding
  const y = (value) => plot.bottom - plotHeight * (value - min) / (max - min)
  const [first, last] = xDomain()
  const x = (value) => plot.left + plotWidth * (last === first ? 0.5 : (value - first) / (last - first))

  ctx.font = '11px system-ui'
  ctx.fillStyle = '#586580'
  ctx.strokeStyle = '#e0e7f1'
  ctx.textBaseline = 'middle'

  for (let i = 0; i <= 4; i++) {
    const position = plot.top + i * plotHeight / 4
    ctx.beginPath()
    ctx.moveTo(plot.left, position)
    ctx.lineTo(plot.right, position)
    ctx.stroke()
    ctx.textAlign = 'right'
    ctx.fillText(formatValue(max - (max - min) * i / 4), plot.left - 8, position)
  }
  const tickSpacing = timeAxis.value ? 148 : 110
  const ticks = last === first ? 0 : Math.min(4, Math.floor(plotWidth / tickSpacing))
  for (let i = 0; i <= ticks; i++) {
    const fraction = ticks ? i / ticks : 0.5
    const value = first + (last - first) * fraction
    const label = timeAxis.value ? timeLabel(value, false) : formatValue(value) + ' ' + props.xUnit
    ctx.textAlign = fraction === 0 ? 'left' : fraction === 1 ? 'right' : 'center'
    ctx.fillText(label, plot.left + plotWidth * fraction, plot.bottom + 16)
  }

  if (timeAxis.value && Number.isFinite(props.selectedTime)) {
    ctx.save()
    ctx.strokeStyle = '#8290bf'
    ctx.setLineDash([4, 4])
    ctx.beginPath()
    ctx.moveTo(x(props.selectedTime), plot.top)
    ctx.lineTo(x(props.selectedTime), plot.bottom)
    ctx.stroke()
    ctx.restore()
  }
  ctx.strokeStyle = '#4e57cd'
  ctx.fillStyle = '#4e57cd'
  ctx.lineWidth = 1.25
  ctx.beginPath()
  if (!timeAxis.value && props.values.length > plotWidth * 2) {
    const columns = Math.floor(plotWidth)
    for (let col = 0; col < columns; col++) {
      const start = Math.floor(col * props.values.length / columns)
      const end = Math.floor((col + 1) * props.values.length / columns)
      let lo = Infinity
      let hi = -Infinity
      for (let i = start; i < end; i++) {
        const value = props.values[i]
        if (!Number.isFinite(value)) continue
        lo = Math.min(lo, value)
        hi = Math.max(hi, value)
      }
      if (lo === Infinity) continue
      ctx.moveTo(plot.left + col, y(hi))
      ctx.lineTo(plot.left + col, y(lo))
    }
  } else {
    let begun = false
    for (let i = 0; i < props.values.length; i++) {
      if (!Number.isFinite(props.values[i])) {
        begun = false
        continue
      }
      const position = x(timeAxis.value ? props.timestamps[i] : i * props.step)
      if (begun) ctx.lineTo(position, y(props.values[i]))
      else ctx.moveTo(position, y(props.values[i]))
      begun = true
    }
  }
  ctx.stroke()
  if (props.values.length === 1 && Number.isFinite(props.values[0])) {
    ctx.beginPath()
    ctx.arc(x(first), y(props.values[0]), 3, 0, Math.PI * 2)
    ctx.fill()
  }
}

function cursorFraction(event) {
  const { width, height } = canvas.value.getBoundingClientRect()
  const { left, right } = plotBounds(width, height)
  return (event.offsetX - left) / (right - left)
}

function inspect(event) {
  const fraction = cursorFraction(event)
  if (!(fraction >= 0 && fraction <= 1) || !props.values.length) {
    pointer.value = ''
    return
  }
  let index = Math.round(fraction * (props.values.length - 1))
  let label = formatValue(index * props.step) + ' ' + props.xUnit
  if (timeAxis.value) {
    const [first, last] = xDomain()
    const time = first + fraction * (last - first)
    index = 0
    for (let i = 1; i < props.timestamps.length; i++) {
      if (Math.abs(props.timestamps[i] - time) < Math.abs(props.timestamps[index] - time)) index = i
    }
    label = timeLabel(props.timestamps[index], true)
  }
  pointer.value = label
    + ' · ' + formatValue(props.values[index]) + (props.unit ? ' ' + props.unit : '')
}

function selectSample(event) {
  if (!props.selectable || !timeAxis.value) return
  const fraction = cursorFraction(event)
  if (!(fraction >= 0 && fraction <= 1)) return
  const [first, last] = xDomain()
  emit('select-time', first + fraction * (last - first))
}

onMounted(() => {
  observer = new ResizeObserver(draw)
  observer.observe(canvas.value)
  draw()
})
onBeforeUnmount(() => observer?.disconnect())
watch([() => props.values, () => props.timestamps, () => props.step, () => props.xUnit, locale], () => {
  pointer.value = ''
  draw()
}, { flush: 'post' })
watch(() => props.selectedTime, draw, { flush: 'post' })
</script>

<template>
  <div class="chart">
    <div class="actions split chart-header">
      <strong>{{ chartLabel }} <span v-if="detail" class="muted">· {{ detail }}</span>
        <span v-if="unit" class="muted chart-unit">{{ t(`（${unit}）`, `(${unit})`) }}</span></strong>
      <output aria-live="off">{{ pointer }}</output>
    </div>
    <canvas v-show="values.length" ref="canvas" role="img" :aria-label="chartLabel" :class="{ selectable }"
      @pointermove="inspect" @pointerleave="pointer = ''" @click="selectSample"></canvas>
    <p v-if="!values.length" class="empty">{{ emptyText || t('暂无数据', 'No data') }}</p>
  </div>
</template>

<style scoped>
canvas.selectable { cursor: crosshair; }
.chart-unit { margin-inline-start: 0.25em; }
</style>

