<template>
  <SelectTag
    v-if="multiple"
    v-bind="$attrs"
    :items="items"
    :placeholder="placeholderText"
    :search="usersStore.searchUsers"
    :searching="usersStore.searching"
  />
  <SelectComboBox
    v-else
    v-bind="$attrs"
    :items="items"
    :placeholder="placeholderText"
    :search="usersStore.searchUsers"
    :searching="usersStore.searching"
    type="user"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, useAttrs, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUsersStore } from '@/stores/users'
import { useUserStore } from '@/stores/user'
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
  includeNone: {
    type: Boolean,
    default: false
  },
  prependItems: {
    type: Array,
    default: () => []
  },
  excludeAiAssistants: {
    type: Boolean,
    default: false
  },
  currentUserFirst: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()
const usersStore = useUsersStore()
const userStore = useUserStore()
const attrs = useAttrs()

const placeholderText = computed(() => props.placeholder || t('placeholders.selectAgent'))

const items = computed(() => {
  let options = props.excludeAiAssistants
    ? usersStore.options.filter((option) => option.type !== 'ai_assistant')
    : usersStore.options
  if (props.currentUserFirst) {
    const isMe = (option) => String(option.value) === String(userStore.userID)
    const me = options.find(isMe)
    if (me) options = [me, ...options.filter((option) => !isMe(option))]
  }
  const prepended = props.includeNone
    ? [{ value: 'none', label: t('globals.terms.none') }, ...props.prependItems]
    : props.prependItems
  return [...prepended, ...options]
})

onMounted(usersStore.fetchUsers)

watch(
  () => attrs.modelValue,
  (value) => {
    const ids = Array.isArray(value) ? value : [value]
    usersStore.ensureUserIDs(ids)
  },
  { immediate: true }
)
</script>
