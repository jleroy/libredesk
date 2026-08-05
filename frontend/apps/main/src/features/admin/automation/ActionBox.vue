<template>
  <div class="space-y-6" :class="{ 'box p-5': actions.length > 0 }">
    <div class="space-y-6">
      <div v-for="(action, index) in actions" :key="index" class="space-y-6">
        <div v-if="index > 0">
          <hr class="border-t-2 border-dotted border-border" />
        </div>

        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="flex gap-5">
              <div class="w-48">
                <!-- Type -->
                <Select
                  v-model="action.type"
                  @update:modelValue="(value) => handleFieldChange(value, index)"
                >
                  <SelectTrigger class="m-auto">
                    <SelectValue :placeholder="t('placeholders.selectAction')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem
                        v-for="(actionConfig, key) in conversationActions"
                        :key="key"
                        :value="key"
                      >
                        {{ actionConfig.label }}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              <!-- Value -->
              <div
                v-if="action.type && conversationActions[action.type]?.type === 'tag'"
                class="w-full"
              >
                <SelectTag
                  v-model="action.value"
                  :items="tagsStore.tagNames.map((tag) => ({ label: tag, value: tag }))"
                  :placeholder="t('placeholders.selectTags')"
                />
              </div>

              <div
                v-if="action.type && conversationActions[action.type]?.type === 'recipients'"
                class="w-full"
              >
                <TagsInput
                  :modelValue="action.value || []"
                  @update:modelValue="(value) => handleNotifyChange(value, index)"
                  :addOnBlur="true"
                  :addOnTab="true"
                  :addOnPaste="true"
                >
                  <TagsInputItem
                    v-for="item in action.value || []"
                    :key="item"
                    :value="item"
                  >
                    <TagsInputItemText />
                    <TagsInputItemDelete />
                  </TagsInputItem>
                  <TagsInputInput :placeholder="t('placeholders.notifyRecipient')" />
                </TagsInput>
                <p class="text-xs text-muted-foreground mt-1">
                  {{ $t('admin.automation.notifyRecipientHint') }}
                </p>
              </div>

              <div
                class="w-48"
                v-if="action.type && conversationActions[action.type]?.type === 'select'"
              >
                <SelectComboBox
                  v-model="action.value[0]"
                  :items="conversationActions[action.type]?.options"
                  :placeholder="t('placeholders.selectValue')"
                  @select="handleValueChange($event, index)"
                  :type="action.type === 'assign_team' ? 'team' : 'user'"
                />
              </div>

              <div
                class="flex gap-3"
                v-if="action.type && conversationActions[action.type]?.type === 'webhook'"
              >
                <div class="w-48">
                  <SelectComboBox
                    v-model="action.value[0]"
                    :items="webhookStore.options"
                    :placeholder="t('placeholders.selectWebhook')"
                    @select="handleWebhookChange($event, index)"
                  />
                </div>
                <div class="w-48">
                  <Input
                    type="text"
                    :placeholder="t('placeholders.webhookEventName')"
                    :modelValue="action.value[1] || ''"
                    @update:modelValue="(value) => handleWebhookEventChange(value, index)"
                  />
                  <p class="text-xs text-muted-foreground mt-1">
                    {{ $t('admin.automation.webhookEventNameHint') }}
                  </p>
                </div>
              </div>

              <div
                class="w-48"
                v-if="action.type && conversationActions[action.type]?.type === 'text'"
              >
                <Input
                  type="text"
                  :placeholder="placeholderForText(action.type)"
                  :modelValue="action.value[0] || ''"
                  @update:modelValue="(value) => handleTextChange(value, index)"
                />
              </div>
            </div>

            <CloseButton :onClose="() => removeAction(index)" />
          </div>

          <div
            class="box p-2 h-96 min-h-96"
            v-if="action.type && conversationActions[action.type]?.type === 'richtext'"
          >
            <Editor
              :autoFocus="false"
              v-model:htmlContent="action.value[0]"
              @update:htmlContent="(value) => handleEditorChange(value, index)"
              :placeholder="t('editor.newLine')"
            />
          </div>
        </div>
      </div>
    </div>
    <div>
      <Button type="button" variant="outline" @click.prevent="addAction">{{
        $t('actions.addAction')
      }}</Button>
    </div>
  </div>
</template>

<script setup>
import { toRefs } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import CloseButton from '@main/components/button/CloseButton.vue'
import { useTagStore } from '@main/stores/tag'
import { useWebhookStore } from '@main/stores/webhook'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { SelectTag } from '@shared-ui/components/ui/select'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@shared-ui/components/ui/tags-input'
import { useConversationFilters } from '@main/composables/useConversationFilters'
import { getTextFromHTML } from '@shared-ui/utils/string'
import { useI18n } from 'vue-i18n'
import Editor from '@main/components/editor/TextEditor.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'

const props = defineProps({
  actions: {
    type: Array,
    required: true
  }
})

const { actions } = toRefs(props)
const { t } = useI18n()
const emit = defineEmits(['update-actions', 'add-action', 'remove-action'])
const tagsStore = useTagStore()
const webhookStore = useWebhookStore()
const { conversationActions } = useConversationFilters()

webhookStore.fetchWebhooks()

const handleNotifyChange = (value, index) => {
  actions.value[index].value = value || []
  emitUpdate(index)
}

const handleTextChange = (value, index) => {
  actions.value[index].value = [value]
  emitUpdate(index)
}

const handleWebhookChange = (value, index) => {
  if (typeof value === 'object') {
    value = value.value
  }
  const current = actions.value[index].value || []
  actions.value[index].value = [value || '', current[1] || '']
  emitUpdate(index)
}

const handleWebhookEventChange = (value, index) => {
  const current = actions.value[index].value || []
  actions.value[index].value = [current[0] || '', value]
  emitUpdate(index)
}

const placeholderForText = (type) => {
  if (type === 'snooze') return t('placeholders.snoozeDuration')
  return t('actions.setValue')
}

const handleFieldChange = (value, index) => {
  actions.value[index].value = []
  actions.value[index].type = value
  emitUpdate(index)
}

const handleValueChange = (value, index) => {
  if (typeof value === 'object') {
    value = value.value
  }
  actions.value[index].value = [value]
  emitUpdate(index)
}

const handleEditorChange = (value, index) => {
  // If text is empty, set HTML to empty string
  const textContent = getTextFromHTML(value)
  if (textContent.length === 0) {
    value = ''
  }
  actions.value[index].value = [value]
  emitUpdate(index)
}

const removeAction = (index) => {
  emit('remove-action', index)
}

const addAction = () => {
  emit('add-action')
}

const emitUpdate = (index) => {
  emit('update-actions', actions, index)
}
</script>
