<template>
  <CommandDialog
    :open="open"
    v-model:search-term="searchTerm"
    :filter-function="isMacroMode ? passThroughFilter : undefined"
    @update:open="toggleOpen"
    :class="[
      'z-[51] !top-[44%] !w-[calc(100%-1.5rem)] !min-w-0 gap-0 rounded-lg border-border/80 bg-popover shadow-lg [&>button]:right-3 [&>button]:top-3 [&>button]:rounded-md [&>button]:bg-muted/70 [&>button]:opacity-60 [&>button]:hover:opacity-100',
      isMacroMode ? '!max-w-5xl' : '!max-w-2xl'
    ]"
    command-class="rounded-lg bg-popover [&_[cmdk-input-wrapper]]:h-14 [&_[cmdk-input-wrapper]]:border-border/70 [&_[cmdk-input-wrapper]]:px-4 [&_[cmdk-input-wrapper]_svg]:text-muted-foreground [&_[cmdk-input]]:h-14 [&_[cmdk-input]]:pr-10 [&_[cmdk-input]]:text-base [&_[cmdk-group]]:py-2 [&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:pb-2 [&_[cmdk-group-heading]]:pt-1 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-item]]:mx-0.5 [&_[cmdk-item]]:rounded-md [&_[cmdk-item]]:px-3 [&_[cmdk-item]]:py-2 [&_[cmdk-item]]:transition-colors [&_[cmdk-item]]:duration-150"
  >
    <CommandInput :placeholder="t('command.typeCmdOrSearch')" @keydown="onInputKeydown" />
    <CommandList
      :class="[
        isMacroMode
          ? 'h-[50vh] min-h-[50vh] min-w-[50vw] overflow-hidden [&>div]:h-full'
          : 'h-auto min-h-[220px] max-h-[min(52vh,440px)]',
        { 'overflow-hidden': nestedCommand === 'apply-macro' }
      ]"
    >
      <CommandEmpty v-if="!isMacroMode || !macroStore.searchLoading">
        <p class="text-sm text-muted-foreground">{{ $t('command.noCommandAvailable') }}</p>
      </CommandEmpty>

      <!-- Snooze Options -->
      <CommandGroup v-if="nestedCommand === 'snooze'" :heading="t('command.snoozeFor')">
        <CommandItem value="1 hour" @select="handleSnooze(60)">
          1 {{ $t('globals.terms.hour') }}
        </CommandItem>
        <CommandItem value="3 hours" @select="handleSnooze(180)">
          3 {{ $t('globals.terms.hour', 2) }}
        </CommandItem>
        <CommandItem value="6 hours" @select="handleSnooze(360)">
          6 {{ $t('globals.terms.hour', 2) }}
        </CommandItem>
        <CommandItem value="12 hours" @select="handleSnooze(720)">
          12 {{ $t('globals.terms.hour', 2) }}
        </CommandItem>
        <CommandItem value="1 day" @select="handleSnooze(1440)">
          1 {{ $t('globals.terms.day') }}
        </CommandItem>
        <CommandItem value="2 days" @select="handleSnooze(2880)">
          2 {{ $t('globals.terms.day', 2) }}
        </CommandItem>
        <CommandItem value="3 days" @select="handleSnooze(4320)">
          3 {{ $t('globals.terms.day', 2) }}
        </CommandItem>
        <CommandItem value="1 week" @select="handleSnooze(10080)">
          1 {{ $t('globals.terms.week') }}
        </CommandItem>
        <CommandItem value="pick date & time" @select="showCustomDialog">
          {{ $t('globals.messages.pickDateAndTime') }}
        </CommandItem>
      </CommandGroup>

      <!-- Macros -->
      <div v-if="isMacroMode" class="h-full">
        <!-- Mounted only with results: radix's group counts its items once on mount and stays hidden when they arrive async. -->
        <CommandGroup
          v-if="visibleMacros.length"
          class="flex h-full min-h-0 flex-col"
        >
          <div class="min-h-0 flex-1">
            <div class="grid h-full min-h-0 grid-cols-12">
              <div
                ref="macroListRef"
                class="col-span-4 h-full min-h-0 overflow-y-auto border-r border-border/70 pr-2"
              >
                <CommandItem
                  v-for="(macro, index) in visibleMacros"
                  :key="macro.value"
                  :value="macro.label + '|' + index"
                  :data-index="index"
                  @select="handleApplyMacro(macro)"
                  @pointerenter="highlightedMacro = macro"
                  class="cursor-pointer"
                >
                  <div class="flex min-w-0 items-center">
                    <span class="text-sm w-full break-words whitespace-normal">{{
                      macro.label
                    }}</span>
                  </div>
                </CommandItem>
              </div>

              <div class="col-span-8 h-full min-h-0 overflow-y-auto px-5 pb-5 pt-2">
                <div class="flex min-h-full flex-col space-y-4 text-sm">
                  <div v-if="contentPending" class="flex flex-1 items-center justify-center">
                    <Spinner :absolute="false" />
                  </div>
                  <div v-else-if="replyContent" class="space-y-2">
                    <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {{ $t('command.replyPreview') }}
                    </p>
                    <Letter
                      :key="highlightedMacro?.value"
                      :html="replyContent"
                      :allowedSchemas="['cid', 'https', 'http', 'mailto']"
                      class="native-html min-h-[120px] w-full overflow-auto rounded-lg border bg-background p-4 shadow-sm"
                    />
                  </div>

                  <div v-if="otherActions.length > 0" class="space-y-2">
                    <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {{ $t('globals.terms.action', 2) }}
                    </p>
                    <div class="grid gap-2 sm:grid-cols-2">
                      <div
                        v-for="action in otherActions"
                        :key="action.type"
                        class="flex min-w-0 items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-sm"
                      >
                        <div
                          class="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-background text-muted-foreground shadow-sm"
                        >
                          <User v-if="action.type === 'assign_user'" :size="14" class="shrink-0" />
                          <Users
                            v-else-if="action.type === 'assign_team'"
                            :size="14"
                            class="shrink-0"
                          />
                          <Pin
                            v-else-if="action.type === 'set_status'"
                            :size="14"
                            class="shrink-0"
                          />
                          <Rocket
                            v-else-if="action.type === 'set_priority'"
                            :size="14"
                            class="shrink-0"
                          />
                          <Tags v-else :size="14" class="shrink-0" />
                        </div>
                        <span class="truncate">{{ getActionLabel(action) }}</span>
                      </div>
                    </div>
                  </div>

                  <div
                    v-if="!contentPending && !replyContent && otherActions.length === 0"
                    class="flex min-h-40 flex-1 flex-col items-center justify-center gap-2 rounded-lg border border-dashed bg-muted/20"
                  >
                    <span class="grid h-9 w-9 place-items-center rounded-lg border bg-background">
                      <Zap :size="16" class="text-muted-foreground" />
                    </span>
                    <p class="text-sm text-muted-foreground">
                      {{ $t('command.selectAMacro') }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CommandGroup>
      </div>

      <!-- Commands requiring a conversation to be open -->
      <CommandGroup
        :heading="t('globals.terms.conversation', 2)"
        value="conversations"
        v-else-if="conversationStore.isConversationOpen && !nestedCommand"
      >
        <CommandItem
          value="apply-macro"
          @select="setNestedCommand('apply-macro-to-existing-conversation')"
        >
          {{ $t('actions.applyMacro') }}
        </CommandItem>
        <CommandItem value="conv-snooze" @select="setNestedCommand('snooze')">
          {{ $t('globals.terms.snooze') }}
        </CommandItem>
        <CommandItem value="conv-resolve" @select="resolveConversation">
          {{ $t('globals.terms.resolve') }}
        </CommandItem>
      </CommandGroup>
    </CommandList>

    <!-- Navigation -->
    <div
      class="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-border/70 bg-muted/30 px-4 py-2"
    >
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd
          :class="KBD_CLASS"
          >Enter</kbd
        >
        {{ $t('globals.terms.select') }}
      </span>
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd
          :class="KBD_CLASS"
          >&uarr;&darr;</kbd
        >
        {{ $t('command.navigate') }}
      </span>
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd
          :class="KBD_CLASS"
          >Esc</kbd
        >
        {{ $t('globals.messages.close') }}
      </span>
      <span v-if="nestedCommand" class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd
          :class="KBD_CLASS"
          >Backspace</kbd
        >
        {{ $t('globals.messages.back') }}
      </span>
    </div>
  </CommandDialog>

  <!-- Date Picker for Custom Snooze -->
  <Dialog :open="showDatePicker" @update:open="closeDatePicker">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>{{ $t('command.pickSnoozeTime') }}</DialogTitle>
        <DialogDescription />
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <Popover :open="datePickerOpen" @update:open="datePickerOpen = $event">
          <PopoverTrigger as-child>
            <Button variant="outline" class="w-full justify-start text-left font-normal">
              <CalendarIcon class="mr-2 h-4 w-4" />
              {{ selectedDate ? selectedDate : t('globals.terms.pickDate') }}
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto p-0">
            <Calendar
              mode="single"
              v-model="selectedDate"
              @update:model-value="datePickerOpen = false"
            />
          </PopoverContent>
        </Popover>
        <div class="grid gap-2">
          <Label>{{ $t('globals.terms.time') }}</Label>
          <Input type="time" v-model="selectedTime" />
        </div>
      </div>
      <DialogFooter>
        <Button @click="handleCustomSnooze">{{ $t('globals.terms.snooze') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
const KBD_CLASS =
  'inline-flex h-5 items-center rounded-sm border bg-background px-1.5 font-mono text-xs font-medium text-foreground shadow-sm'

import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { useMagicKeys, useDebounceFn } from '@vueuse/core'
import { CalendarIcon } from 'lucide-vue-next'
import { useConversationStore } from '@main/stores/conversation'
import { useMacroStore } from '@main/stores/macro'
import { CONVERSATION_DEFAULT_STATUSES, MACRO_CONTEXT } from '@main/constants/conversation'
import { Users, User, Pin, Rocket, Tags, Zap } from 'lucide-vue-next'
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem
} from '@shared-ui/components/ui/command'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { Button } from '@shared-ui/components/ui/button'
import { Calendar } from '@shared-ui/components/ui/calendar'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import { Letter } from 'vue-letter'

const conversationStore = useConversationStore()
const macroStore = useMacroStore()
const { t } = useI18n()
const open = ref(false)
const emitter = useEmitter()
const nestedCommand = ref(null)
const showDatePicker = ref(false)
const datePickerOpen = ref(false)
const selectedDate = ref(null)
const selectedTime = ref('12:00')
const searchTerm = ref('')
const macroListRef = ref(null)

const passThroughFilter = (items) => items

const isMacroMode = computed(
  () =>
    nestedCommand.value === 'apply-macro-to-existing-conversation' ||
    nestedCommand.value === 'apply-macro-to-new-conversation'
)

const visibleMacros = computed(() => macroStore.macroOptions)

const searchMacrosDebounced = useDebounceFn(() => {
  macroStore.searchMacros(searchTerm.value?.trim() || '')
}, 300)

watch(isMacroMode, (on) => {
  if (on) macroStore.searchMacros(searchTerm.value?.trim() || '')
})

watch(searchTerm, () => {
  if (isMacroMode.value) searchMacrosDebounced()
})

function preventDefaultOnHotkey(key) {
  return (e) => {
    if (e.key === key && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
    }
  }
}

const { Meta_K, Ctrl_K } = useMagicKeys({
  passive: false,
  onEventFired: preventDefaultOnHotkey('k')
})

watch([Meta_K, Ctrl_K], ([mac, win]) => {
  if (mac || win) {
    if (nestedCommand.value !== 'apply-macro-to-new-conversation') setNestedCommand(null)
    toggleOpen()
  }
})

const { Meta_M, Ctrl_M } = useMagicKeys({
  passive: false,
  onEventFired: preventDefaultOnHotkey('m')
})

watch([Meta_M, Ctrl_M], ([mac, win]) => {
  if (mac || win) {
    if (nestedCommand.value !== 'apply-macro-to-new-conversation') {
      setNestedCommand('apply-macro-to-existing-conversation')
    }
    toggleOpen()
  }
})

const highlightedMacro = ref(null)

// New search results can drop the highlighted macro without any highlight mutation firing.
watch(visibleMacros, (macros) => {
  if (highlightedMacro.value && !macros.some((m) => m.value === highlightedMacro.value.value)) {
    highlightedMacro.value = null
  }
})

async function handleApplyMacro(macro) {
  let messageContent = ''
  if (macro.has_message_content) {
    try {
      messageContent = await macroStore.fetchMacroContent(macro.id)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      return
    }
  }
  // Create a deep copy.
  const plainMacro = JSON.parse(JSON.stringify(macro))
  plainMacro.message_content = messageContent
  if (nestedCommand.value === 'apply-macro-to-new-conversation') {
    conversationStore.setMacro(plainMacro, MACRO_CONTEXT.NEW_CONVERSATION)
  } else {
    conversationStore.setMacro(plainMacro, MACRO_CONTEXT.REPLY)
  }
  toggleOpen()
}

const getActionLabel = computed(() => (action) => {
  const prefixes = {
    assign_user: t('actions.assignAgent'),
    assign_team: t('actions.assignTeam'),
    set_status: t('actions.setStatus'),
    set_priority: t('actions.setPriority'),
    add_tags: t('actions.addTags'),
    set_tags: t('actions.setTags'),
    remove_tags: t('actions.removeTags')
  }
  return `${prefixes[action.type]}: ${action.display_value.length > 0 ? action.display_value.join(', ') : action.value.join(', ')}`
})

const replyContent = computed(() => {
  const macro = highlightedMacro.value
  return macro ? macroStore.macroContents[macro.id] || '' : ''
})

const failedContentIDs = ref(new Set())

const contentPending = computed(() => {
  const macro = highlightedMacro.value
  if (!macro?.has_message_content) return false
  return !(macro.id in macroStore.macroContents) && !failedContentIDs.value.has(macro.id)
})

let contentFetchTimer = null

watch(highlightedMacro, (macro) => {
  clearTimeout(contentFetchTimer)
  if (!macro?.has_message_content || macro.id in macroStore.macroContents) return
  failedContentIDs.value.delete(macro.id)
  contentFetchTimer = setTimeout(() => {
    macroStore.fetchMacroContent(macro.id).catch(() => failedContentIDs.value.add(macro.id))
  }, 150)
})

const otherActions = computed(
  () =>
    highlightedMacro.value?.actions?.filter(
      (a) => a.type !== 'send_private_note' && a.type !== 'send_reply'
    ) || []
)

function toggleOpen() {
  open.value = !open.value
}

function setNestedCommand(command) {
  nestedCommand.value = command
}

function formatDuration(minutes) {
  return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h`
}

async function handleSnooze(minutes) {
  await conversationStore.snoozeConversation(formatDuration(minutes))
  toggleOpen()
}

async function resolveConversation() {
  await conversationStore.updateStatus(CONVERSATION_DEFAULT_STATUSES.RESOLVED)
  toggleOpen()
}

function showCustomDialog() {
  toggleOpen()
  showDatePicker.value = true
}

function closeDatePicker() {
  showDatePicker.value = false
}

function handleCustomSnooze() {
  const [hours, minutes] = selectedTime.value.split(':')
  const snoozeDate = new Date(selectedDate.value)
  snoozeDate.setHours(parseInt(hours), parseInt(minutes))
  const diffMinutes = Math.floor((snoozeDate - new Date()) / (1000 * 60))

  if (diffMinutes <= 0) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.selectAFutureTime')
    })
    return
  }
  handleSnooze(diffMinutes)
  closeDatePicker()
  toggleOpen()
}

function onInputKeydown(e) {
  if (e.key === 'Backspace') {
    const inputVal = e.target.value || ''
    if (!inputVal && nestedCommand.value !== null) {
      e.preventDefault()
      nestedCommand.value = null
    }
  }
}

const nestedCommandHandler = (data) => {
  setNestedCommand(data.command)
  open.value = data.open
}

let highlightObserver = null

watch(macroListRef, (el) => {
  highlightObserver?.disconnect()
  highlightObserver = null
  highlightedMacro.value = null
  if (!el) return
  highlightObserver = new MutationObserver(() => {
    const idx = el.querySelector('[data-highlighted]')?.getAttribute('data-index')
    if (idx != null) highlightedMacro.value = visibleMacros.value[idx]
  })
  highlightObserver.observe(el, {
    attributes: true,
    attributeFilter: ['data-highlighted'],
    subtree: true
  })
})

onMounted(() => {
  emitter.on(EMITTER_EVENTS.SET_NESTED_COMMAND, nestedCommandHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SET_NESTED_COMMAND, nestedCommandHandler)
  highlightObserver?.disconnect()
  clearTimeout(contentFetchTimer)
})
</script>
