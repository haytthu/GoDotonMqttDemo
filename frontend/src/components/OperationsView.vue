<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api, query } from '../api.js'
import { formatValue, statusText, timeText } from '../config.js'
import { locale, t, translateError, translateMessage } from '../i18n.js'
import { clearFieldError, getFormError, markFieldError } from '../validation.js'

const props = defineProps({
  mode: { type: String, required: true },
  collectors: { type: Array, required: true },
  initialCollectorId: { type: String, default: '' },
})

const collectorId = ref(props.initialCollectorId)
const logTab = ref('files')
const uploadedLogs = ref([])
const messages = ref([])
const firmware = ref([])
const firmwareIds = ref([])
const pendingUploadId = ref('')
const upgradeHistory = ref([])
const selectedFiles = ref([])
const targetVersion = ref('')
const fileInput = ref(null)
const status = ref(null)
const loading = ref(false)
const working = ref(false)
const error = ref('')
const note = ref('')
let refreshTimer
let generation = 0
const errorId = 'operation-error'
const maxFirmwareBytes = 64 * 1024 * 1024
const maxFirmwareFiles = 10
const uploadsById = computed(() => new Map(firmware.value.map((item) => [item.id, item])))
const planFiles = computed(() => firmwareIds.value.map((id) => uploadsById.value.get(id)).filter(Boolean))
const pendingUploads = computed(() => firmware.value.filter((item) => !firmwareIds.value.includes(item.id)))

async function refresh() {
  const current = ++generation
  loading.value = true
  try {
    if (props.mode === 'logs') {
      const filesSelected = logTab.value === 'files'
      const target = filesSelected ? uploadedLogs : messages
      const rows = await api((filesSelected ? '/log-files' : '/logs') + query({
        collectorId: collectorId.value, limit: 250,
      }))
      if (generation === current) target.value = rows || []
      return
    }
    const [files, history, deviceStatus] = await Promise.all([
      api('/firmware'),
      api('/firmware/history'),
      collectorId.value ? api('/collectors/' + encodeURIComponent(collectorId.value) + '/status') : null,
    ])
    if (generation !== current) return
    firmware.value = files || []
    upgradeHistory.value = history || []
    status.value = deviceStatus
    firmwareIds.value = firmwareIds.value.filter((id) => uploadsById.value.has(id))
    if (!uploadsById.value.has(pendingUploadId.value)) pendingUploadId.value = ''
  } catch (e) {
    if (generation === current) error.value = e.message
  } finally {
    if (generation === current) loading.value = false
  }
}

async function upload(event) {
  if (working.value) return
  note.value = ''
  error.value = getFormError(event.currentTarget, errorId) || ''
  if (error.value) return
  if (planFiles.value.length + selectedFiles.value.length > maxFirmwareFiles) {
    markFieldError(fileInput.value, errorId)
    error.value = 'Select between 1 and 10 firmware files'
    return
  }
  if (selectedFiles.value.some((file) => file.size === 0 || file.size > maxFirmwareBytes)) {
    markFieldError(fileInput.value, errorId)
    error.value = 'Upload a nonempty firmware file of up to 64 MB'
    return
  }
  working.value = true
  generation++
  try {
    for (const file of selectedFiles.value) {
      const body = new FormData()
      body.append('file', file)
      const uploaded = await api('/firmware', { method: 'POST', body })
      firmware.value.push(uploaded)
      addPlanFile(uploaded)
      selectedFiles.value = selectedFiles.value.slice(1)
    }
    if (fileInput.value) fileInput.value.value = ''
    note.value = 'upload'
  } catch (e) {
    error.value = e.message
  } finally {
    await refresh()
    working.value = false
  }
}

function addPlanFile(upload) {
  if (planFiles.value.some((item) => item.md5 === upload.md5)) {
    error.value = 'The upgrade contains duplicate firmware files'
    return
  }
  firmwareIds.value.push(upload.id)
}

function addPendingUpload() {
  const upload = uploadsById.value.get(pendingUploadId.value)
  if (!upload) return
  error.value = ''
  addPlanFile(upload)
  pendingUploadId.value = ''
}

function movePlanFile(index, offset) {
  const [id] = firmwareIds.value.splice(index, 1)
  firmwareIds.value.splice(index + offset, 0, id)
}

async function upgrade(event) {
  if (working.value || !planFiles.value.length) return
  note.value = ''
  error.value = getFormError(event.currentTarget, errorId) || ''
  if (error.value) return
  working.value = true
  const uploads = [...firmwareIds.value]
  try {
    status.value = await api('/collectors/' + encodeURIComponent(collectorId.value) + '/upgrade', {
      method: 'POST', body: { firmwareIds: uploads, targetVersion: targetVersion.value },
    })
    firmware.value = firmware.value.filter((item) => !uploads.includes(item.id))
    firmwareIds.value = []
    targetVersion.value = ''
    note.value = 'upgrade'
    await refresh()
  } catch (e) {
    error.value = e.message
    await refresh()
  } finally {
    working.value = false
  }
}

function upgradeCollector(record) {
  const collector = props.collectors.find((item) => item.id === record.collectorId)
  const code = record.collectorCode || collector?.code
  const name = record.collectorName || collector?.name
  return [code, name].filter(Boolean).join(' · ') || record.collectorId || '—'
}

function logCollector(log) {
  return log.code || props.collectors.find((collector) => collector.id === log.collectorId)?.code || '—'
}

function logDirection(value) {
  if (value === 'RX' || value === 'IN') return t('接收', 'Received')
  if (value === 'TX' || value === 'OUT') return t('发送', 'Sent')
  if (value === 'SYSTEM' || value === 'INTERNAL') return t('系统', 'System')
  return value || '—'
}

function fileStatus(value) {
  switch (value) {
    case 'VERIFIED': return t('MD5 已校验', 'MD5 verified')
    case 'UNVERIFIED': return t('未提供 MD5', 'No MD5 supplied')
    case 'LEGACY': return t('历史文件', 'Legacy file')
    default: return value || '—'
  }
}

function downloadUrl(id) {
  return '/api/log-files/' + encodeURIComponent(id) + '/download'
}

function clearInputError(event) {
  if (event.target.dataset.formErrorId === errorId) {
    clearFieldError(event)
    error.value = ''
  }
}

watch(locale, () => {
  error.value = ''
})

watch([collectorId, logTab], () => {
  status.value = null
  uploadedLogs.value = []
  messages.value = []
  error.value = ''
  refresh()
})
onMounted(() => {
  refresh()
  refreshTimer = setInterval(() => {
    if (!document.hidden && !working.value && !loading.value) refresh()
  }, 5000)
})
onBeforeUnmount(() => clearInterval(refreshTimer))
</script>

<template>
  <section class="panel viewport-panel operations-panel">
    <div class="actions split">
      <h2>{{ mode === 'logs' ? t('采集器日志', 'Collector logs') : t('固件升级', 'Firmware upgrade') }}</h2>
      <button type="button" :disabled="loading || working" @click="refresh">{{ t('刷新', 'Refresh') }}</button>
    </div>
    <p v-if="error" :id="errorId" class="notice failure" role="alert">{{ translateError(error) }}</p>
    <p v-if="note" class="notice success" role="status">
      {{ note === 'upload' ? t('固件文件已上传', 'Firmware files uploaded') : t('已创建升级任务', 'Upgrade task created') }}
    </p>
    <template v-if="mode === 'logs'">
      <nav class="tabs data-tabs" :aria-label="t('日志类型', 'Log type')">
        <button type="button" :class="{ active: logTab === 'files' }" @click="logTab = 'files'">
          {{ t('上传文件', 'Uploaded files') }}
        </button>
        <button type="button" :class="{ active: logTab === 'messages' }" @click="logTab = 'messages'">
          {{ t('MQTT 通信记录', 'MQTT messages') }}
        </button>
      </nav>
      <div class="filters">
        <label class="field">
          <span>{{ t('采集器', 'Collector') }}</span>
          <select v-model="collectorId">
            <option value="">{{ t('全部采集器', 'All collectors') }}</option>
            <option v-for="collector in collectors" :key="collector.id" :value="collector.id">
              {{ collector.code }} · {{ collector.name }}
            </option>
          </select>
        </label>
      </div>
      <template v-if="logTab === 'files'">
        <div v-if="uploadedLogs.length" class="table-wrap logs-table panel-body scroll-region" tabindex="0" :aria-label="t('上传文件', 'Uploaded files')">
          <table>
            <thead><tr>
              <th>{{ t('上传时间', 'Uploaded') }}</th><th>{{ t('采集器', 'Collector') }}</th>
              <th>{{ t('文件名', 'File name') }}</th><th>{{ t('大小（字节）', 'Size (bytes)') }}</th>
              <th>{{ t('状态', 'Status') }}</th><th>{{ t('操作', 'Action') }}</th>
            </tr></thead>
            <tbody>
              <tr v-for="log in uploadedLogs" :key="log.id">
                <td class="mono">{{ timeText(log.time) }}</td>
                <td>{{ logCollector(log) }}</td>
                <td>{{ log.name }}</td>
                <td class="mono">{{ formatValue(log.size) }}</td>
                <td>{{ fileStatus(log.status) }}</td>
                <td><a class="text-button" :href="downloadUrl(log.id)" :download="log.name">{{ t('下载', 'Download') }}</a></td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="empty">
          {{ loading ? t('加载日志中…', 'Loading logs…') : t('暂无采集器上传的日志文件', 'No uploaded log files yet') }}
        </p>
      </template>
      <template v-else>
        <div v-if="messages.length" class="table-wrap logs-table panel-body scroll-region" tabindex="0" :aria-label="t('MQTT 通信记录', 'MQTT messages')">
          <table>
            <thead><tr>
              <th>{{ t('时间', 'Time') }}</th><th>{{ t('采集器', 'Collector') }}</th>
              <th>{{ t('方向', 'Direction') }}</th><th>{{ t('命令', 'Command') }}</th>
              <th>{{ t('状态', 'Status') }}</th><th>{{ t('主题或详情', 'Topic or details') }}</th>
            </tr></thead>
            <tbody>
              <tr v-for="(log, index) in messages" :key="log.id || index">
                <td class="mono">{{ timeText(log.time) }}</td>
                <td>{{ logCollector(log) }}</td>
                <td>{{ logDirection(log.direction) }}</td>
                <td class="mono">{{ log.command || '—' }}</td>
                <td>{{ log.status ?? '—' }}</td>
                <td class="log-detail">
                  <span v-if="log.topic" class="mono">{{ log.topic }}</span>
                  <span v-if="log.message">{{ translateMessage(log.message) }}</span>
                  <span v-if="!log.topic && !log.message">—</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="empty">
          {{ loading ? t('加载日志中…', 'Loading logs…') : t('暂无 MQTT 通信记录', 'No MQTT messages yet') }}
        </p>
      </template>
    </template>

    <div v-else class="firmware-workspace panel-body">
      <div class="firmware-editor flow scroll-region" tabindex="0" :aria-label="t('升级配置', 'Upgrade configuration')">
        <form class="subpanel flow" novalidate @submit.prevent="upload" @input="clearInputError" @change="clearInputError">
          <h3>{{ t('上传固件文件', 'Upload firmware file') }}</h3>
          <div class="filters">
            <label class="field file-field">
              <span>{{ t('固件文件', 'Firmware file') }}</span>
              <input ref="fileInput" class="file-input" type="file" multiple required :disabled="working"
                :aria-label="t('固件文件', 'Firmware file')" :data-field-label="t('固件文件', 'Firmware file')"
                @change="selectedFiles = Array.from($event.target.files)" />
              <span class="file-picker" aria-hidden="true">
                <span class="file-trigger">{{ t('选择文件', 'Choose files') }}</span>
                <span class="file-name">{{ selectedFiles.map((file) => file.name).join(', ') || t('未选择文件', 'No files selected') }}</span>
              </span>
            </label>
          </div>
          <button type="submit" class="start" :disabled="working">{{ t('上传', 'Upload') }}</button>
        </form>
        <form class="flow divider" novalidate @submit.prevent="upgrade" @input="clearInputError" @change="clearInputError">
          <h3>{{ t('发起升级', 'Start upgrade') }}</h3>
          <div class="filters">
            <label class="field">
              <span>{{ t('目标采集器', 'Target collector') }}</span>
              <select v-model="collectorId" :disabled="working" required>
                <option value="">{{ t('选择采集器', 'Select a collector') }}</option>
                <option v-for="collector in collectors" :key="collector.id" :value="collector.id">
                  {{ collector.code }} · {{ collector.name }}
                </option>
              </select>
            </label>
            <label class="field">
              <span>{{ t('目标版本（可选）', 'Target version (optional)') }}</span>
              <input v-model.trim="targetVersion" type="text" :disabled="working"
                :placeholder="t('填写设备预期上报版本', 'Expected version reported by the collector')" />
            </label>
          </div>
          <div v-if="pendingUploads.length" class="filters">
            <label class="field">
              <span>{{ t('未使用的上传', 'Unused uploads') }}</span>
              <select v-model="pendingUploadId" :disabled="working || planFiles.length >= maxFirmwareFiles">
                <option value="">{{ t('选择固件文件', 'Select a firmware file') }}</option>
                <option v-for="item in pendingUploads" :key="item.id" :value="item.id">{{ item.name }}</option>
              </select>
            </label>
            <button type="button" :disabled="working || !pendingUploadId || planFiles.length >= maxFirmwareFiles" @click="addPendingUpload">
              {{ t('加入本次升级', 'Add to this upgrade') }}
            </button>
          </div>
          <div v-if="planFiles.length" class="table-wrap data-table">
            <table :aria-label="t('本次升级文件', 'Files for this upgrade')">
              <thead><tr>
                <th>{{ t('顺序', 'Order') }}</th><th>{{ t('升级文件', 'Firmware file') }}</th>
                <th>{{ t('大小（字节）', 'Size (bytes)') }}</th><th>MD5</th><th>{{ t('操作', 'Actions') }}</th>
              </tr></thead>
              <tbody>
                <tr v-for="(item, index) in planFiles" :key="item.id">
                  <td>{{ index + 1 }}</td><td>{{ item.name }}</td>
                  <td class="mono">{{ formatValue(item.size) }}</td><td class="mono">{{ item.md5 }}</td>
                  <td><div class="actions">
                    <button type="button" :disabled="working || index === 0" @click="movePlanFile(index, -1)">{{ t('上移', 'Move up') }}</button>
                    <button type="button" :disabled="working || index === planFiles.length - 1" @click="movePlanFile(index, 1)">{{ t('下移', 'Move down') }}</button>
                    <button type="button" class="danger" :disabled="working" @click="firmwareIds.splice(index, 1)">{{ t('移除', 'Remove') }}</button>
                  </div></td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else class="empty">{{ t('请添加本次升级的固件文件', 'Add the firmware files for this upgrade') }}</p>
          <button type="submit" class="primary start" :disabled="working || !planFiles.length">
            {{ t('发起升级', 'Start upgrade') }}
          </button>
          <dl v-if="status" class="detail-grid">
            <div><dt>{{ t('设备状态', 'Collector status') }}</dt><dd>{{ status.online ? t('在线', 'Online') : t('离线', 'Offline') }}</dd></div>
            <div><dt>{{ t('已上报固件版本', 'Reported firmware version') }}</dt><dd>{{ status.firmwareVersion || '—' }}</dd></div>
            <div><dt>{{ t('升级状态', 'Upgrade status') }}</dt><dd>{{ statusText(status.upgradeState) }}</dd></div>
            <div><dt>{{ t('传输进度', 'Transfer progress') }}</dt><dd>{{ status.upgradeProgress != null ? formatValue(status.upgradeProgress) + '%' : '—' }}</dd></div>
          </dl>
          <p v-if="status?.upgradeMessage" class="muted">{{ translateError(status.upgradeMessage) }}</p>
        </form>
      </div>
      <section class="firmware-history">
        <h3>{{ t('升级记录', 'Upgrade history') }}</h3>
        <div v-if="upgradeHistory.length" class="table-wrap scroll-region" tabindex="0" :aria-label="t('升级记录', 'Upgrade history')">
          <table>
            <thead><tr>
              <th>{{ t('发起时间', 'Started') }}</th><th>{{ t('采集器', 'Collector') }}</th>
              <th>{{ t('升级文件', 'Firmware files') }}</th><th>{{ t('目标版本', 'Target version') }}</th>
              <th>{{ t('最后状态', 'Last status') }}</th><th>{{ t('传输进度', 'Transfer progress') }}</th>
            </tr></thead>
            <tbody>
              <tr v-for="record in upgradeHistory" :key="record.id">
                <td class="mono">{{ timeText(record.startedAt) }}</td><td>{{ upgradeCollector(record) }}</td>
                <td>
                  <ol v-if="record.files?.length" class="upgrade-files">
                    <li v-for="item in record.files" :key="item.id || item.md5" :title="item.md5">
                      {{ item.name }}
                      <span class="muted">· {{ item.confirmed ? t('已确认', 'Confirmed') : formatValue(item.progress) + '%' }}</span>
                    </li>
                  </ol>
                  <span v-else>—</span>
                </td><td>{{ record.targetVersion || '—' }}</td>
                <td>{{ statusText(record.upgradeState) }}</td>
                <td>{{ record.upgradeProgress != null ? formatValue(record.upgradeProgress) + '%' : '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-else class="empty">
          {{ loading ? t('加载升级记录中…', 'Loading upgrade history…') : t('暂无升级记录', 'No upgrades yet') }}
        </p>
      </section>
    </div>
  </section>
</template>

<style scoped>
.operations-panel > .notice { max-height: 20vh; overflow: auto; }
.firmware-workspace { display: grid; grid-template: minmax(0, 1fr) / minmax(0, 1fr) minmax(0, 1fr); gap: 16px; }
.firmware-editor { grid-template-columns: minmax(0, 1fr); padding: 3px; }
.firmware-history { display: flex; flex-direction: column; min-width: 0; min-height: 0; gap: 10px; padding-inline-start: 16px; border-inline-start: 1px solid var(--border); }
.firmware-history .table-wrap { flex: 1; }
.upgrade-files { margin: 0; padding-inline-start: 1.5rem; }
.file-field { position: relative; }
.file-input { position: absolute; width: 1px; height: 1px; padding: 0; opacity: 0; }
.file-picker {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 3px 6px;
  border: 1px solid #d6dfed;
  border-radius: 6px;
  background: white;
  cursor: pointer;
}
.file-trigger {
  flex: none;
  padding: 4px 7px;
  border-radius: 4px;
  background: #edf1f8;
}
.file-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 400; }
.file-input:focus-visible + .file-picker { outline: 2px solid #5459b7; outline-offset: 2px; }
.file-input[aria-invalid="true"] + .file-picker, select[aria-invalid="true"] { border-color: #a63535; }
@media (max-width: 1050px) {
  .firmware-workspace { grid-template: minmax(0, 58fr) minmax(0, 42fr) / minmax(0, 1fr); gap: 12px; }
  .firmware-history { padding: 10px 0 0; border-inline-start: 0; border-top: 1px solid var(--border); }
}
</style>
