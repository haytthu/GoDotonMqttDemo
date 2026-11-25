import { directions } from './config.js'
import { t } from './i18n'

export const ROOT_ID = 'root'
export const ROOT_KEY = 'organizations/' + ROOT_ID

export function modelKindLabel(kind) {
  return {
    organizations: t('组织', 'Organization'),
    devices: t('设备', 'Device'),
    points: t('测点', 'Point'),
  }[kind]
}

export function modelName(kind, record) {
  return kind === 'organizations' && record?.id === ROOT_ID
    ? t('根组织', 'Root organization')
    : record?.name || ''
}

export function modelKey(kind, id) {
  return kind + '/' + id
}

export function axisLabel(point, direction) {
  const axis = directions[direction]
  const name = point?.axisNames?.[direction]
  return name && name !== axis ? name + ' · ' + axis : axis || ''
}
