<template>
  <div class="flex flex-col relative h-full">
    <div ref="threadEl" class="flex-1 overflow-y-auto" @scroll="handleScroll">
      <div class="min-h-full px-4 pb-10">
        <div
          class="text-center mt-3"
          v-if="
            conversationStore.currentConversationHasMoreMessages &&
            !conversationStore.messages.loading
          "
        >
          <Button
            size="sm"
            variant="outline"
            @click="conversationStore.fetchNextMessages"
            class="transition-all duration-200 hover:bg-gray-100 dark:hover:bg-gray-700 hover:scale-105 active:scale-95"
          >
            <RefreshCw size="17" class="mr-2" />
            {{ $t('globals.terms.loadMore') }}
          </Button>
        </div>

        <MessagesSkeleton :count="10" v-if="conversationStore.messages.loading" />

        <TransitionGroup v-else enter-active-class="animate-slide-in" tag="div" class="space-y-4">
          <div
            v-for="(message, index) in conversationStore.conversationMessages"
            :key="message.uuid"
            :data-message-uuid="message.uuid"
            :class="{
              'my-2': message.type === 'activity',
              'pt-4': index === 0
            }"
          >
            <div v-if="!message.private && message.type !== 'activity'">
              <MessageBubble :message="message" :direction="message.type" />
            </div>
            <div v-else-if="isPrivateNote(message)">
              <MessageBubble :message="message" direction="outgoing" />
            </div>
            <div v-else-if="message.type === 'activity'">
              <ActivityMessageBubble :message="message" />
            </div>
          </div>
        </TransitionGroup>
      </div>
    </div>

    <!-- Sticky container for the scroll arrow -->
    <Transition
      enter-active-class="transition ease-out duration-200"
      enter-from-class="opacity-0 translate-y-1"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition ease-in duration-150"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 translate-y-1"
    >
      <div v-show="!isAtBottom" class="absolute bottom-5 right-6 z-10">
        <button
          @click="handleScrollToBottom"
          class="w-10 h-10 rounded-full flex items-center justify-center shadow-lg border bg-background text-primary transition-colors duration-200 hover:bg-gray-100 dark:hover:bg-gray-700"
        >
          <ChevronDown size="18" />
        </button>
        <span
          v-if="unReadMessages > 0"
          class="absolute -top-1 -right-1 min-w-[20px] h-5 px-1.5 rounded-full bg-green-500 text-secondary text-xs font-medium flex items-center justify-center"
        >
          {{ unReadMessages }}
        </span>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import MessageBubble from './MessageBubble.vue'
import ActivityMessageBubble from './ActivityMessageBubble.vue'
import { useConversationStore } from '@/stores/conversation'
import { useUserStore } from '@/stores/user'
import { Button } from '@/components/ui/button'
import { RefreshCw, ChevronDown } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import MessagesSkeleton from './MessagesSkeleton.vue'

const route = useRoute()

const conversationStore = useConversationStore()
const userStore = useUserStore()
const threadEl = ref(null)
const emitter = useEmitter()
const isAtBottom = ref(true)
const unReadMessages = ref(0)
const currentConversationUUID = ref('')

const checkIfAtBottom = () => {
  const thread = threadEl.value
  if (thread) {
    const tolerance = 100
    const isBottom = thread.scrollHeight - thread.scrollTop - thread.clientHeight <= tolerance
    isAtBottom.value = isBottom
  }
}

const handleScroll = () => {
  checkIfAtBottom()
}

const handleScrollToBottom = () => {
  unReadMessages.value = 0
  scrollToBottom()
}

const scrollToBottom = () => {
  setTimeout(() => {
    const thread = threadEl.value
    if (thread) {
      thread.scrollTop = thread.scrollHeight
      checkIfAtBottom()
    }
  }, 50)
}

const scrollToMessage = (messageUUID) => {
  if (!messageUUID) {
    scrollToBottom()
    return
  }

  setTimeout(() => {
    const thread = threadEl.value
    const messageEl = thread?.querySelector(`[data-message-uuid="${messageUUID}"]`)
    if (messageEl && thread) {
      // Manual scroll calculation for reliability with variable-height messages
      const messageTop = messageEl.offsetTop
      const threadHeight = thread.clientHeight
      const messageHeight = messageEl.offsetHeight
      // Position message at ~1/3 from top of viewport for better visibility
      const targetScroll = messageTop - threadHeight / 3 + messageHeight / 2
      thread.scrollTop = Math.max(0, targetScroll)

      // Highlight the message briefly
      messageEl.classList.add('highlight-mention')
      setTimeout(() => messageEl.classList.remove('highlight-mention'), 2500)
    } else {
      // Message not found, scroll to bottom instead
      scrollToBottom()
    }
  }, 150)
}

onMounted(() => {
  checkIfAtBottom()
  handleNewMessage()
})

const handleNewMessage = () => {
  emitter.on(EMITTER_EVENTS.NEW_MESSAGE, (data) => {
    if (data.conversation_uuid === conversationStore.current.uuid) {
      // Agent's own message - always scroll to bottom
      if (data.message?.sender_id === userStore.userID) {
        scrollToBottom()
      }
      // Customer message - only scroll if already at bottom
      else if (isAtBottom.value) {
        scrollToBottom()
      }
      // Customer message but not at bottom - don't scroll, increment unread
      else {
        unReadMessages.value++
      }
    }
  })
}

watch(
  () => conversationStore.conversationMessages,
  (messages) => {
    // Scroll to bottom when conversation changes and there are new messages.
    // New messages on next db page should not scroll to bottom.
    if (
      messages.length > 0 &&
      conversationStore?.current?.uuid &&
      currentConversationUUID.value !== conversationStore.current.uuid
    ) {
      currentConversationUUID.value = conversationStore.current.uuid
      unReadMessages.value = 0

      // Check if this is a mentioned conversation
      const scrollToUUID = route.query.scrollTo
      if (scrollToUUID) {
        // Mentioned conversation - only scroll to message, NOT to bottom
        scrollToMessage(scrollToUUID)
      } else {
        // Normal conversation - scroll to bottom
        scrollToBottom()
      }
    }
  }
)

const isPrivateNote = (message) => {
  return message.type === 'outgoing' && message.private
}
</script>

<style scoped>
.highlight-mention {
  animation: highlightPulse 2.5s ease-out;
}

@keyframes highlightPulse {
  0% {
    background-color: rgb(251 191 36 / 0.35);
    border-radius: 0.5rem;
  }
  100% {
    background-color: transparent;
  }
}

/* Dark mode highlight - softer yellow */
:global(.dark) .highlight-mention {
  animation: highlightPulseDark 2.5s ease-out;
}

@keyframes highlightPulseDark {
  0% {
    background-color: rgb(250 204 21 / 0.2);
    border-radius: 0.5rem;
  }
  100% {
    background-color: transparent;
  }
}
</style>
