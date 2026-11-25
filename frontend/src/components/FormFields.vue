<script setup>
import { useId } from 'vue'

defineProps({
  model: { type: Object, required: true },
  fields: { type: Array, required: true },
})
const emit = defineEmits(['field-change'])
const formId = useId()
</script>

<template>
  <div class="field-grid">
    <label v-for="field in fields" :key="field.key" class="field" :title="field.hint">
      <span :id="formId + '-' + field.key">{{ field.label }}</span>
      <select v-if="field.kind === 'select'" v-model="model[field.key]" :required="field.required"
        :data-field-label="field.label" :aria-labelledby="formId + '-' + field.key"
        @change="emit('field-change', field.key)">
        <option v-if="field.placeholder" value="">{{ field.placeholder }}</option>
        <option v-for="item in field.options" :key="item.value" :value="item.value">
          {{ item.label }}
        </option>
      </select>
      <input
        v-else-if="field.kind === 'number'" v-model.number="model[field.key]"
        type="number" :min="field.min" :max="field.max" :step="field.step ?? 1"
        :required="field.required" :placeholder="field.placeholder"
        :data-field-label="field.label" :aria-labelledby="formId + '-' + field.key"
      />
      <input
        v-else-if="field.displayValue !== undefined" :value="field.displayValue" type="text" readonly
        :aria-labelledby="formId + '-' + field.key"
      />
      <input
        v-else v-model.trim="model[field.key]" type="text"
        :required="field.required" :placeholder="field.placeholder" :maxlength="field.maxlength" :readonly="field.readonly"
        :data-field-label="field.label" :aria-labelledby="formId + '-' + field.key"
      />
      <small v-if="field.hint" class="muted">{{ field.hint }}</small>
    </label>
  </div>
</template>

<style scoped>
[aria-invalid="true"] { border-color: #a63535; }
</style>
