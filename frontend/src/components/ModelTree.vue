<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { t } from '../i18n'
import { axisLabel, modelKey, modelKindLabel, modelName, ROOT_ID, ROOT_KEY } from '../modelTree.js'

const props = defineProps({
  organizations: { type: Array, required: true },
  devices: { type: Array, required: true },
  points: { type: Array, required: true },
  selectedKey: { type: String, default: '' },
  readOnly: Boolean,
  disabled: Boolean,
})

const emit = defineEmits(['select', 'action'])
const expanded = ref(new Set([ROOT_KEY]))
const menu = ref(null)
const menuElement = ref(null)
let menuTrigger = null

function childrenBy(rows, parentField) {
  const index = new Map()
  for (const row of rows) {
    const related = index.get(row[parentField]) || []
    related.push(row)
    index.set(row[parentField], related)
  }
  return index
}

const hierarchy = computed(() => {
  const root = props.organizations.find((row) => row.id === ROOT_ID)
  const organizations = childrenBy(props.organizations, 'parentId')
  const devices = childrenBy(props.devices, 'organizationId')
  const points = childrenBy(props.points, 'deviceId')
  const nodes = new Map()

  function add(kind, record, parentKey, depth) {
    const key = modelKey(kind, record.id)
    if (nodes.has(key)) return null
    const node = {
      kind, record, parentKey, depth, key,
      label: modelName(kind, record),
      axis: kind === 'points' && record.axisGroupId ? axisLabel(record, record.direction) : '',
      children: [],
    }
    nodes.set(key, node)
    if (kind === 'organizations') {
      node.children = [
        ...(organizations.get(record.id) || []).map((item) => add('organizations', item, key, depth + 1)),
        ...(devices.get(record.id) || []).map((item) => add('devices', item, key, depth + 1)),
      ].filter(Boolean)
    } else if (kind === 'devices') {
      node.children = (points.get(record.id) || []).map((item) => add('points', item, key, depth + 1)).filter(Boolean)
    }
    return node
  }

  return { root: root ? add('organizations', root, '', 0) : null, nodes }
})

const rows = computed(() => {
  const visible = []
  function append(node) {
    if (!node) return
    visible.push(node)
    if (expanded.value.has(node.key)) node.children.forEach(append)
  }
  append(hierarchy.value.root)
  return visible
})

watch([() => props.selectedKey, hierarchy], ([key]) => {
  let node = hierarchy.value.nodes.get(key)
  while (node?.parentKey) {
    expanded.value.add(node.parentKey)
    node = hierarchy.value.nodes.get(node.parentKey)
  }
}, { immediate: true })

function toggle(node) {
  if (expanded.value.has(node.key)) expanded.value.delete(node.key)
  else expanded.value.add(node.key)
}

function selectNode(node) {
  if (props.readOnly && node.kind !== 'points') {
    if (node.children.length) toggle(node)
    return
  }
  if (node.children.length) expanded.value.add(node.key)
  emit('select', node)
}

function actions(node) {
  let types = []
  if (node.kind === 'organizations') {
    const childrenKinds = new Set(node.children.map((child) => child.kind))
    if (node.key === ROOT_KEY) types = ['organizations']
    else if (!childrenKinds.size) types = ['organizations', 'devices']
    else if (childrenKinds.size === 1) types = [...childrenKinds]
  }
  if (node.kind === 'devices') types = ['points']
  const rows = types.map((kind) => ({
    id: kind, command: 'create',
    label: {
      organizations: t('新建组织', 'Add organization'),
      devices: t('新建设备', 'Add device'),
      points: t('新建测点', 'Add point'),
    }[kind],
  }))
  if (node.key !== ROOT_KEY) {
    rows.push({ id: node.kind, command: 'edit', label: t('编辑', 'Edit') })
    rows.push({ id: node.kind, command: 'remove', label: t('删除', 'Delete') })
  }
  return rows
}

async function openMenu(node, event, useAnchor = false) {
  if (props.readOnly || props.disabled) return
  event.preventDefault()
  const items = actions(node)
  if (!items.length) return
  menuTrigger = event.currentTarget
  const anchor = menuTrigger.getBoundingClientRect()
  const x = useAnchor ? anchor.left : event.clientX
  const y = useAnchor ? anchor.bottom + 3 : event.clientY
  menu.value = { node, items, x, y }
  await nextTick()
  if (!menu.value || !menuElement.value) return
  const bounds = menuElement.value.getBoundingClientRect()
  menu.value.x = Math.max(4, Math.min(x, window.innerWidth - bounds.width - 4))
  menu.value.y = Math.max(4, Math.min(y, window.innerHeight - bounds.height - 4))
  menuElement.value.querySelector('button')?.focus()
}

function closeMenu(restoreFocus = false) {
  if (!menu.value) return
  menu.value = null
  if (restoreFocus) menuTrigger?.focus()
  menuTrigger = null
}

function execute(item) {
  const node = menu.value.node
  closeMenu()
  if (item.command === 'create') expanded.value.add(node.key)
  emit('action', { command: item.command, kind: item.id, node })
}

function dismiss(event) {
  if (event.type === 'keydown' && event.key === 'Escape') closeMenu(true)
  if (event.type === 'pointerdown' && !menuElement.value?.contains(event.target)) closeMenu()
}

onMounted(() => {
  window.addEventListener('keydown', dismiss)
  window.addEventListener('pointerdown', dismiss)
  window.addEventListener('resize', hideMenu)
  window.addEventListener('scroll', hideMenu, true)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', dismiss)
  window.removeEventListener('pointerdown', dismiss)
  window.removeEventListener('resize', hideMenu)
  window.removeEventListener('scroll', hideMenu, true)
})
function hideMenu() { closeMenu() }
</script>

<template>
  <div class="model-tree" :aria-label="t('设备模型', 'Device model')">
    <p v-if="!rows.length" class="empty compact">{{ t('暂无设备模型', 'No device model') }}</p>
    <div v-for="node in rows" :key="node.key" class="tree-row"
      :class="{ selected: selectedKey === node.key }"
      :style="{ paddingInlineStart: 3 + node.depth * 14 + 'px' }"
      @contextmenu="openMenu(node, $event)">
      <button v-if="node.children.length" type="button" class="tree-toggle"
        :aria-label="(expanded.has(node.key) ? t('折叠', 'Collapse') : t('展开', 'Expand')) + ' ' + node.label"
        :aria-expanded="expanded.has(node.key)" :disabled="disabled" @click.stop="toggle(node)">
        <span aria-hidden="true">{{ expanded.has(node.key) ? '▾' : '▸' }}</span>
      </button>
      <span v-else class="tree-spacer" aria-hidden="true"></span>
      <button type="button" class="tree-label" :aria-current="selectedKey === node.key ? 'true' : undefined"
        :disabled="disabled" :title="node.label" @click="selectNode(node)"
        @keydown.shift.f10.prevent="openMenu(node, $event, true)">
        <span class="tree-icon" :class="node.kind" aria-hidden="true">
          {{ node.kind === 'organizations' ? '▰' : node.kind === 'devices' ? '◇' : '⊙' }}
        </span>
        <span class="tree-name">{{ node.label }}</span>
        <span v-if="node.axis" class="tree-axis">{{ node.axis }}</span>
        <span v-else class="sr-only">{{ modelKindLabel(node.kind) }}</span>
      </button>
      <button v-if="!readOnly" type="button" class="tree-action"
        :disabled="disabled" :aria-label="t('操作', 'Actions') + ' ' + node.label"
        @click.stop="openMenu(node, $event, true)">···</button>
    </div>
    <Teleport to="body">
      <div v-if="menu" ref="menuElement" class="tree-menu" :style="{ left: menu.x + 'px', top: menu.y + 'px' }"
        :aria-label="t('节点操作', 'Node actions')">
        <button v-for="item in menu.items" :key="item.command + item.id" type="button"
          :class="{ danger: item.command === 'remove' }" @click="execute(item)">{{ item.label }}</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.model-tree { min-width: 0; }
.tree-row {
  position: relative;
  min-height: 35px;
  display: flex;
  align-items: center;
  gap: 2px;
  border-radius: 6px;
  margin: 1px 0;
}
.tree-row:hover { background: #f5f7fc; }
.tree-row.selected { background: var(--accent-soft); color: #313fb4; }
.tree-toggle, .tree-spacer {
  width: 19px;
  min-height: 28px;
  padding: 0;
  flex: 0 0 19px;
}
.tree-toggle, .tree-label, .tree-action {
  background: transparent;
  border: 0;
  border-radius: 4px;
  color: inherit;
}
.tree-label {
  min-height: 33px;
  display: flex;
  flex: 1;
  min-width: 0;
  align-items: center;
  text-align: left;
  padding: 2px 3px;
  gap: 6px;
  font-size: 13px;
}
.tree-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tree-axis { flex: 0 0 auto; color: var(--muted); font-size: 11px; font-weight: normal; }
.tree-icon { color: #555db2; min-width: 14px; text-align: center; }
.tree-icon.organizations { color: #987337; }
.tree-icon.points { color: #237363; }
.tree-action { min-height: 28px; min-width: 25px; padding: 0 3px; font-size: 15px; }
.tree-menu {
  position: fixed;
  min-width: 150px;
  z-index: 10;
  display: grid;
  padding: 4px;
  border: 1px solid var(--border);
  background: #fff;
  border-radius: 7px;
  box-shadow: 0 6px 16px #29365920;
}
.tree-menu button { border: 0; min-height: 29px; text-align: left; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
</style>
