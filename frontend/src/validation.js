import { t } from './i18n.js'

function controlLabel(input) {
  return input.dataset.fieldLabel || input.closest('label')?.querySelector('span')?.textContent.trim()
    || t('此字段', 'This field')
}

function fieldMessage(input) {
  const label = controlLabel(input)
  const validity = input.validity
  if (validity.badInput) return t(`${label}必须是有效数字`, `${label} must be a valid number`)
  if (validity.valueMissing) {
    if (input.type === 'file' || input.tagName === 'SELECT') {
      return t(`请选择${label}`, `Select a value for ${label}`)
    }
    return t(`请输入${label}`, `Enter a value for ${label}`)
  }
  if (validity.rangeUnderflow) return t(`${label}不能小于 ${input.min}`, `${label} must be at least ${input.min}`)
  if (validity.rangeOverflow) return t(`${label}不能大于 ${input.max}`, `${label} must be no more than ${input.max}`)
  if (validity.stepMismatch) {
    const step = input.step || '1'
    return t(`${label}的数值步长为 ${step}`, `${label} must use increments of ${step}`)
  }
  if (validity.tooLong) return t(`${label}最多 ${input.maxLength} 个字符`, `${label} can have up to ${input.maxLength} characters`)
  if (validity.tooShort) return t(`${label}至少 ${input.minLength} 个字符`, `${label} must have at least ${input.minLength} characters`)
  return t(`请检查${label}的格式`, `Check the format of ${label}`)
}

export function markFieldError(input, errorId) {
  input.setAttribute('aria-invalid', 'true')
  if (errorId) {
    input.dataset.formErrorId = errorId
    const descriptions = new Set((input.getAttribute('aria-describedby') || '').split(/\s+/).filter(Boolean))
    descriptions.add(errorId)
    input.setAttribute('aria-describedby', [...descriptions].join(' '))
  }
  input.focus()
}

export function clearFieldError(eventOrInput) {
  const input = eventOrInput.target || eventOrInput
  const errorId = input.dataset?.formErrorId
  if (!errorId) return
  input.removeAttribute('aria-invalid')
  delete input.dataset.formErrorId
  const descriptions = (input.getAttribute('aria-describedby') || '').split(/\s+/).filter((id) => id && id !== errorId)
  if (descriptions.length) input.setAttribute('aria-describedby', descriptions.join(' '))
  else input.removeAttribute('aria-describedby')
}

export function getFormError(form, errorId) {
  if (!form) return null
  const elements = Array.from(form.elements)
  for (const input of elements) clearFieldError(input)
  const invalid = elements.find((input) => input.willValidate && !input.validity.valid)
  if (!invalid) return null
  markFieldError(invalid, errorId)
  return fieldMessage(invalid)
}

export function getAlarmError(point) {
  const groups = new Map()
  const triaxialDraft = point.type === 'VIBRATION_3AXIS'
  for (const alarm of point.alarms || []) {
    if ([alarm.code, alarm.level, alarm.limit].some((value) => value === '' || value == null)) continue
    const code = Number(alarm.code)
    const level = Number(alarm.level)
    const limit = Number(alarm.limit)
    const direction = triaxialDraft ? Number(alarm.direction || 0) : Number(point.direction || 0)
    if (!Number.isInteger(code) || code < 0 || ![1, 2, 3, 4].includes(level) || !Number.isFinite(limit)
      || ![0, 1, 2, 3].includes(direction)) {
      return t('请填写有效的指标编码、报警级别、方向和上限。', 'Enter a valid metric code, alarm level, axis and upper limit.')
    }
    const effectiveAxes = triaxialDraft && direction === 0 ? [1, 2, 3] : [direction]
    for (const axis of effectiveAxes) {
      const key = `${code}/${axis}`
      if (!groups.has(key)) groups.set(key, new Map())
      const thresholds = groups.get(key)
      if (thresholds.has(level)) {
        return t('同一测点方向和指标的报警级别不能重复。', 'Use each alarm level once per point direction and metric.')
      }
      thresholds.set(level, limit)
    }
  }
  for (const thresholds of groups.values()) {
    let lowerLimit = -Infinity
    for (const [, limit] of [...thresholds].sort(([a], [b]) => a - b)) {
      if (limit <= lowerLimit) {
        return t('同一测点方向和指标的报警门限必须随级别严格递增。', 'Alarm thresholds must increase strictly by level for each point direction and metric.')
      }
      lowerLimit = limit
    }
  }
  return null
}

