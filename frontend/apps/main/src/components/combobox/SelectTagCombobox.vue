<template>
  <SelectTag
    v-if="multiple"
    v-bind="$attrs"
    :items="items"
    :placeholder="placeholderText"
    :search="tagStore.searchTags"
    :searching="tagStore.searching"
  />
  <SelectComboBox
    v-else
    v-bind="$attrs"
    :items="items"
    :placeholder="placeholderText"
    :search="tagStore.searchTags"
    :searching="tagStore.searching"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, useAttrs, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTagStore } from '@/stores/tag'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { SelectTag } from '@shared-ui/components/ui/select'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  placeholder: {
    type: String,
    default: ''
  },
  multiple: {
    type: Boolean,
    default: false
  },
  // Conversations store tags as names; filters store them as ids.
  valueField: {
    type: String,
    default: 'name',
    validator: (value) => ['name', 'id'].includes(value)
  }
})

const { t } = useI18n()
const tagStore = useTagStore()
const attrs = useAttrs()

const placeholderText = computed(() => props.placeholder || t('placeholders.selectTags'))

const items = computed(() =>
  props.valueField === 'id'
    ? tagStore.tagOptions
    : tagStore.tagNames.map((name) => ({ label: name, value: name }))
)

onMounted(tagStore.fetchTags)

watch(
  () => attrs.modelValue,
  (value) => {
    if (props.valueField !== 'id') return
    const ids = Array.isArray(value) ? value : [value]
    tagStore.ensureTagIDs(ids)
  },
  { immediate: true }
)
</script>
