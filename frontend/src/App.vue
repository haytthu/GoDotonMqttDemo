<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from './api.js'
import { clone, defaults, statusText, systemDefaults, timeText } from './config.js'
import { locale, setLocale, t, translateError } from './i18n'
import { axisLabel, modelKey, modelKindLabel, modelName, ROOT_ID } from './modelTree.js'
import CollectorEditor from './components/CollectorEditor.vue'
import DataView from './components/DataView.vue'
import ManagementEditor from './components/ManagementEditor.vue'
import ModelTree from './components/ModelTree.vue'
import OperationsView from './components/OperationsView.vue'
import { clearFieldError, getAlarmError, getFormError } from './validation.js'

const nav = computed(() => [
  { id: 'model', icon: '⌘', title: t('设备建模', 'Device modeling') },
  { id: 'collectors', icon: '▤', title: t('采集器', 'Collectors') },
  { id: 'data', icon: '∿', title: t('采集数据', 'Acquisition data') },
  { id: 'firmware', icon: '⇧', title: t('固件升级', 'Firmware upgrade') },
  { id: 'logs', icon: '≡', title: t('日志管理', 'Logs') },
])

const page = ref('model')
const catalog = ref({ collectors: [], organizations: [], devices: [], points: [] })
const selectedId = ref('')
const editorKind = ref('')
const modelFocus = ref(null)
const lastCollectorId = ref('')
const draft = ref(null)
const status = ref(null)
const editVersion = ref(0)
const editorForm = ref(null)
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const formError = ref('')
const note = ref('')
const collectorTab = ref('system')
let statusTimer
let statusRequest = 0

const heading = computed(() => page.value === 'collectors'
  ? t('SG500Z 采集器', 'SG500Z collectors')
  : nav.value.find((item) => item.id === page.value)?.title)
const modelSelection = computed(() => modelFocus.value
  ? modelKey(modelFocus.value.kind, modelFocus.value.id)
  : '')
const rootSelected = computed(() => editorKind.value === 'organizations' && selectedId.value === ROOT_ID)
const editorHeading = computed(() => {
  const label = editorKind.value === 'collectors' ? t('采集器', 'collector') : modelKindLabel(editorKind.value)
  if (rootSelected.value) return modelName('organizations', draft.value)
  const verb = selectedId.value ? t('编辑', 'Edit ') : t('新建', 'Add ')
  const axis = selectedId.value && editorKind.value === 'points' ? draft.value?.direction : 0
  return verb + label + (axis ? ' · ' + axisLabel(draft.value, axis) : '')
})
const editedPoints = computed(() => catalog.value.points.map((point) => {
  const device = record('devices', point.deviceId)?.name
  return {
    ...point,
    label: [device, point.name].filter(Boolean).join(' / '),
    groupLabel: point.axisGroupId ? [device, point.axisGroupName || point.name].filter(Boolean).join(' / ') : '',
  }
}))

function record(kind, id) {
  return catalog.value[kind].find((row) => row.id === id)
}

function fresh(kind) {
  switch (kind) {
    case 'organizations': return { name: '', parentId: '' }
    case 'devices': return { name: '', organizationId: '', stopEnabled: false, stopLimits: [] }
    case 'points': return { name: '', deviceId: '', type: 'VIBRATION', alarms: [] }
    case 'collectors': return {
      name: '', code: '', system: { ...systemDefaults }, vibrationDefaults: { ...defaults },
      channels: [],
    }
  }
  return null
}

function normalized(kind, row) {
  const basis = fresh(kind)
  const saved = { ...basis, ...clone(row) }
  if (kind === 'collectors') {
    saved.system = { ...basis.system, ...saved.system }
    saved.vibrationDefaults = { ...basis.vibrationDefaults, ...saved.vibrationDefaults }
    saved.channels ||= []
    for (const channel of saved.channels) {
      if (Number(channel.channelType) === 0) {
        const point = record('points', channel.pointId)
        channel.direction = point?.axisGroupId && point.id === point.axisGroupId ? 0 : channel.direction ?? 0
        channel.sensorType ??= 0
        channel.sensitivity ??= 10
      } else if (Number(channel.channelType) === 2) {
        channel.sensitivity ??= 1
        channel.amendment ??= 0
      }
    }
  }
  if (kind === 'organizations') saved.parentId ||= ''
  if (kind === 'devices') saved.stopLimits ||= []
  if (kind === 'points') {
    saved.alarms = (saved.alarms || []).filter((alarm) => Number(alarm.type || 0) === 0)
  }
  return saved
}

function edit(kind, row, clearFeedback = true) {
  if (clearFeedback) {
    error.value = ''
    note.value = ''
  }
  formError.value = ''
  editorKind.value = kind
  selectedId.value = row?.id || ''
  draft.value = row ? normalized(kind, row) : fresh(kind)
  editVersion.value++
}

function openCollector(row, clearFeedback = true, preserveTab = false) {
  if (!preserveTab) collectorTab.value = 'system'
  status.value = null
  statusRequest++
  edit('collectors', row, clearFeedback)
  if (selectedId.value) {
    lastCollectorId.value = selectedId.value
    refreshStatus()
  }
}

function openModel(kind, row, clearFeedback = true) {
  if (!row) return
  modelFocus.value = { kind, id: row.id }
  edit(kind, row, clearFeedback)
}

function selectModel(node) {
  openModel(node.kind, node.record)
}

function openRememberedModel(clearFeedback = true) {
  const focus = modelFocus.value
  if (focus) {
    const row = record(focus.kind, focus.id)
    if (row) {
      openModel(focus.kind, row, clearFeedback)
      return
    }
  }
  openModel('organizations', record('organizations', ROOT_ID), clearFeedback)
}

function modelAction({ command, kind, node }) {
  if (busy.value) return
  if (command === 'remove') {
    selectModel(node)
    removeRecord()
    return
  }
  if (command === 'create') {
    modelFocus.value = { kind: node.kind, id: node.record.id }
    edit(kind)
    const relationship = { organizations: 'parentId', devices: 'organizationId', points: 'deviceId' }[kind]
    draft.value[relationship] = node.record.id
  } else {
    selectModel(node)
  }
  nextTick(() => editorForm.value?.querySelector('input:not([readonly])')?.focus())
}

function changePage(next) {
  if (page.value === next) return
  page.value = next
  error.value = ''
  formError.value = ''
  note.value = ''
  selectedId.value = ''
  editorKind.value = ''
  draft.value = null
  status.value = null
  statusRequest++
  if (next === 'collectors') {
    openCollector(record('collectors', lastCollectorId.value) || catalog.value.collectors[0])
  } else if (next === 'model') {
    openRememberedModel()
  }
}

async function refreshCatalog() {
  loading.value = true
  try {
    const updates = Object.keys(catalog.value).map(async (kind) => {
      catalog.value[kind] = await api('/' + kind) || []
    })
    const result = await Promise.allSettled(updates)
    if (!draft.value) {
      if (page.value === 'model') openRememberedModel(false)
      if (page.value === 'collectors') openCollector(catalog.value.collectors[0], false)
    }
    const failure = result.find((item) => item.status === 'rejected')
    if (failure) error.value = failure.reason.message
  } finally {
    loading.value = false
  }
}

async function refreshStatus(showError = false) {
  if (page.value !== 'collectors' || !selectedId.value) return
  const id = selectedId.value
  const current = ++statusRequest
  try {
    const result = await api('/collectors/' + encodeURIComponent(id) + '/status')
    if (current === statusRequest) status.value = result
  } catch (e) {
    if (showError && current === statusRequest) error.value = e.message
  }
}

async function run(action) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  note.value = ''
  try {
    await action()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

function toggleLanguage() {
  setLocale(locale.value === 'zh' ? 'en' : 'zh')
  if (formError.value) {
    for (const field of editorForm.value?.elements || []) clearFieldError(field)
    formError.value = ''
  }
}

function onFormInput(event) {
  clearFieldError(event)
  formError.value = ''
}

function saveRecord() {
  if (rootSelected.value) return
  const invalid = getFormError(editorForm.value, 'configuration-error')
    || (editorKind.value === 'points' && getAlarmError(draft.value))
  if (invalid) {
    formError.value = invalid
    return
  }
  formError.value = ''
  run(async () => {
    const kind = editorKind.value
    const id = selectedId.value
    const body = clone(draft.value)
    if (kind === 'points' && body.type !== 'VIBRATION_3AXIS') {
      delete body.axisNames
      for (const alarm of body.alarms) delete alarm.direction
    }
    const saved = await api('/' + kind + (id ? '/' + encodeURIComponent(id) : ''), {
      method: id ? 'PUT' : 'POST', body,
    })
    await refreshCatalog()
    const row = record(kind, saved?.id || id) || saved
    if (kind === 'collectors') openCollector(row, false, true)
    else if (row) openModel(kind, row, false)
    note.value = 'saved'
  })
}

function removeRecord() {
  if (rootSelected.value || !selectedId.value) return
  const name = draft.value.name
  const groupName = draft.value.axisGroupName || name
  const prompt = editorKind.value === 'points' && draft.value.axisGroupId
    ? t('删除“' + groupName + '”的全部三个轴向测点？', 'Delete all three axis points for “' + groupName + '”?')
    : t('删除“' + name + '”？', 'Delete “' + name + '”?')
  if (!window.confirm(prompt)) return
  run(async () => {
    const kind = editorKind.value
    const deleted = draft.value
    await api('/' + kind + '/' + encodeURIComponent(selectedId.value), { method: 'DELETE' })
    await refreshCatalog()
    if (kind === 'collectors') {
      openCollector(catalog.value.collectors[0], false)
    } else {
      const parentKind = kind === 'points' ? 'devices' : 'organizations'
      const parentId = kind === 'points' ? deleted.deviceId : kind === 'devices' ? deleted.organizationId : deleted.parentId
      const parent = record(parentKind, parentId)
      if (parent) openModel(parentKind, parent, false)
      else openModel('organizations', record('organizations', ROOT_ID), false)
    }
    note.value = 'deleted'
  })
}

function collectorAction(action) {
  run(async () => {
    const response = await api('/collectors/' + encodeURIComponent(selectedId.value) + '/' + action, { method: 'POST' })
    if (action === 'publish' && response) status.value = response
    await refreshStatus(true)
    note.value = action
  })
}

const dirty = computed(() => {
  if (!selectedId.value || !draft.value) return false
  const kind = editorKind.value
  const saved = record(kind, selectedId.value)
  return saved != null && JSON.stringify(normalized(kind, saved)) !== JSON.stringify(draft.value)
})

const message = computed(() => ({
  saved: t('已保存', 'Saved'),
  deleted: t('已删除', 'Deleted'),
  publish: t('已提交采集器通知请求', 'Collector notification requested'),
  restart: t('已提交重启指令', 'Restart command submitted'),
})[note.value])

watch(() => catalog.value.collectors, () => {
  if (lastCollectorId.value && !record('collectors', lastCollectorId.value)) lastCollectorId.value = ''
})

onMounted(() => {
  refreshCatalog()
  statusTimer = setInterval(() => {
    if (!document.hidden && !busy.value) refreshStatus()
  }, 5000)
})
onBeforeUnmount(() => clearInterval(statusTimer))
</script>

<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-icon" aria-hidden="true"><i></i><i></i><i></i></div>
        <div><strong>SG500Z</strong><span>{{ t('采集管理', 'Acquisition management') }}</span></div>
      </div>
      <nav class="main-nav" :aria-label="t('主导航', 'Main navigation')">
        <button v-for="item in nav" :key="item.id" type="button" :class="{ active: page === item.id }"
          :aria-current="page === item.id ? 'page' : undefined" :disabled="busy" @click="changePage(item.id)">
          <span class="nav-icon" aria-hidden="true">{{ item.icon }}</span>
          <span class="nav-label">{{ item.title }}</span>
        </button>
      </nav>
    </aside>

    <main>
      <header class="page-header actions split">
        <h1>{{ heading }}</h1>
        <div class="actions">
          <button v-if="page === 'model' || page === 'collectors'" type="button" :disabled="loading || busy" @click="refreshCatalog">
            {{ loading ? t('刷新中…', 'Refreshing…') : t('刷新列表', 'Refresh list') }}
          </button>
          <button type="button" class="language-toggle" :title="t('切换到英语', 'Switch to Chinese')"
            :aria-label="t('切换到英语', 'Switch to Chinese')" @click="toggleLanguage">
            <span class="language-icon" aria-hidden="true"><span>文</span><span>A</span></span>
          </button>
        </div>
      </header>

      <p v-if="formError || error" id="configuration-error" class="notice failure" role="alert" :class="{ 'form-error': formError }">
        {{ formError || translateError(error) }}
      </p>
      <p v-if="message" class="notice success" role="status">{{ message }}</p>

      <div class="page-content">
        <div v-if="page === 'model' || page === 'collectors'" class="management" :class="{ 'model-layout': page === 'model' }">
          <section v-if="page === 'model'" class="panel registry viewport-panel">
            <h2>{{ t('设备模型', 'Device model') }}</h2>
            <div class="panel-body scroll-region" tabindex="0" :aria-label="t('设备树', 'Device tree')">
              <p v-if="loading && !catalog.organizations.length" class="empty compact">{{ t('读取中…', 'Loading…') }}</p>
              <ModelTree v-else :organizations="catalog.organizations" :devices="catalog.devices" :points="catalog.points"
                :selected-key="modelSelection" :disabled="busy" @select="selectModel" @action="modelAction" />
            </div>
          </section>

          <section v-else class="panel registry viewport-panel">
            <div class="actions split">
              <h2>{{ t('采集器', 'Collectors') }}</h2>
              <button type="button" class="primary" :disabled="busy" @click="openCollector(undefined)">{{ t('新建', 'Add') }}</button>
            </div>
            <div class="panel-body scroll-region flow small" tabindex="0" :aria-label="t('采集器列表', 'Collector list')">
              <p v-if="!catalog.collectors.length" class="empty compact">
                {{ loading ? t('读取中…', 'Loading…') : t('暂无采集器', 'No collectors') }}
              </p>
              <button v-for="row in catalog.collectors" :key="row.id" class="registry-row" type="button"
                :class="{ active: selectedId === row.id }" :disabled="busy" @click="openCollector(row)">
                <strong>{{ row.name || row.code }}</strong>
                <span>{{ row.code }}</span>
              </button>
            </div>
          </section>

          <section v-if="draft" class="panel editor viewport-panel">
            <div class="actions split">
              <h2>{{ editorHeading }}</h2>
              <div v-if="page === 'collectors' && selectedId" class="actions">
                <span class="tag" :class="{ online: status?.online }">
                  {{ status?.online ? t('在线', 'Online') : t('离线', 'Offline') }}
                </span>
                <button type="button" @click="changePage('data')">{{ t('查看数据', 'View data') }}</button>
              </div>
            </div>

            <p v-if="rootSelected" class="muted">{{ t('从根组织的操作菜单新建下属组织。', 'Use the root organization’s actions to add an organization.') }}</p>

            <form v-else :key="editVersion" ref="editorForm" class="editor-form panel-body" novalidate
              @input="onFormInput" @change="onFormInput" @submit.prevent="saveRecord">
              <div class="editor-content scroll-region flow" tabindex="0" :aria-label="editorHeading">
                <dl v-if="page === 'collectors' && status" class="detail-grid">
                  <div><dt>Broker</dt><dd>{{ status.brokerConnected ? t('已连接', 'Connected') : t('未连接', 'Disconnected') }}</dd></div>
                  <div>
                    <dt>{{ t('参数状态', 'Parameter status') }}</dt>
                    <dd>{{ status.parameterState === 'PENDING' ? t('已保存，待拉取', 'Saved, awaiting device request') : statusText(status.parameterState) }}</dd>
                  </div>
                  <div><dt>{{ t('配置版本', 'Configuration revision') }}</dt><dd>{{ status.publishedVersion ?? '—' }}</dd></div>
                  <div><dt>{{ t('固件版本', 'Firmware version') }}</dt><dd>{{ status.firmwareVersion || '—' }}</dd></div>
                  <div><dt>{{ t('最后通信', 'Last communication') }}</dt><dd>{{ timeText(status.lastSeen) }}</dd></div>
                </dl>
                <p v-if="page === 'collectors' && status?.parameterState === 'DRAFT'" class="muted">
                  {{ t('保存后，采集器请求参数时即可获取配置。点击「下发参数」可主动通知采集器。',
                    'After saving, the collector can get the settings on request. Select Publish parameters to notify the collector manually.') }}
                </p>
                <p v-else-if="page === 'collectors' && status?.parameterState === 'PENDING'" class="muted">
                  {{ t('采集器请求参数时即可获取已保存的配置。点击「下发参数」可主动通知采集器。',
                    'The saved settings are available when the collector requests them. Select Publish parameters to notify the collector manually.') }}
                </p>
                <p v-if="page === 'collectors' && status?.parameterMessage" class="muted">{{ translateError(status.parameterMessage) }}</p>

                <CollectorEditor v-if="page === 'collectors'" v-model:active-tab="collectorTab" :collector="draft"
                  :points="editedPoints" :code-locked="Boolean(status?.publishedVersion)" />
                <ManagementEditor v-else :kind="editorKind" :draft="draft" :organizations="catalog.organizations"
                  :devices="catalog.devices" :points="editedPoints" :active-direction="Number(draft.direction || 0)" />
              </div>
              <div class="actions form-footer">
                <button class="primary" type="submit" :disabled="busy">{{ t('保存', 'Save') }}</button>
                <button v-if="page === 'collectors' && selectedId" type="button" :disabled="busy || dirty"
                  :title="dirty ? t('保存修改后可通知采集器', 'Save changes before notifying the collector') : ''" @click="collectorAction('publish')">
                  {{ t('下发参数', 'Publish parameters') }}
                </button>
                <button v-if="page === 'collectors' && selectedId" type="button" :disabled="busy || !status?.online"
                  @click="collectorAction('restart')">{{ t('重启设备', 'Restart device') }}</button>
                <button v-if="selectedId" type="button" class="quiet danger end" :disabled="busy" @click="removeRecord">
                  {{ t('删除', 'Delete') }}
                </button>
              </div>
            </form>
          </section>
        </div>

        <DataView v-else-if="page === 'data'" :organizations="catalog.organizations" :devices="catalog.devices"
          :collectors="catalog.collectors" :points="editedPoints" :initial-collector-id="lastCollectorId" />
        <OperationsView v-else :key="page" :mode="page" :collectors="catalog.collectors"
          :initial-collector-id="lastCollectorId" />
      </div>
    </main>
  </div>
</template>

<style scoped>
.language-toggle { display: grid; place-items: center; width: 38px; min-height: 36px; padding: 4px; }
.language-icon { display: block; position: relative; width: 23px; height: 21px; }
.language-icon > span {
  position: absolute;
  display: grid;
  place-items: center;
  width: 15px;
  height: 15px;
  border: 1px solid currentColor;
  border-radius: 3px;
  font-size: 10px;
  line-height: 1;
}
.language-icon > span:first-child { top: 0; inset-inline-start: 0; }
.language-icon > span:last-child { bottom: 0; inset-inline-end: 0; background: #fff; }
.editor-form { display: flex; flex-direction: column; gap: 10px; }
.editor-content { flex: 1; padding: 3px; }
.form-footer { flex: none; padding-top: 10px; }
.model-layout { grid-template-columns: 250px minmax(0, 1fr); }
@media (max-width: 900px) {
  .model-layout { grid-template-columns: 210px minmax(0, 1fr); }
}
@media (max-width: 760px) {
  .model-layout { grid-template-columns: 1fr; }
}
</style>
