<template>
  <div class="flex flex-col h-full">
    <Tabs :modelValue="widgetStore.currentView" @update:modelValue="handleTabChange" class="flex flex-col h-full">
      <div class="flex-1 min-h-0">
        <TabsContent value="home" class="h-full mt-0">
          <HomeView />
        </TabsContent>
        <TabsContent value="messages" class="h-full mt-0">
          <ConversationsView v-if="!widgetStore.isChatView" />
          <ChatView v-else />
        </TabsContent>
      </div>
      <TabsList class="grid grid-cols-2 h-auto bg-background border-t rounded-none p-0">
        <TabsTrigger value="home" class="flex items-center justify-center gap-2 py-2.5 rounded-none shadow-none data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:text-primary text-muted-foreground">
          <House class="w-[18px] h-[18px]" />
          <span class="text-sm">{{ $t('globals.terms.home') }}</span>
        </TabsTrigger>
        <TabsTrigger value="messages" class="flex items-center justify-center gap-2 py-2.5 rounded-none shadow-none data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:text-primary text-muted-foreground">
          <MessagesSquare class="w-[18px] h-[18px]" />
          <span class="text-sm">{{ $t('globals.terms.message', 2) }}</span>
        </TabsTrigger>
      </TabsList>
      <div
        v-if="widgetStore.config?.show_powered_by !== false"
        class="text-center flex items-center justify-center"
      >
        <span class="text-[10px] text-muted-foreground"
          >{{ $t('globals.messages.poweredBy') }} <a href="https://libredesk.io" target="_blank">Libredesk</a></span
        >
      </div>

      <!-- Network Connection Banner -->
      <ConnectionBanner />
    </Tabs>
  </div>
</template>

<script setup>
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@shared-ui/components/ui/tabs'
import HomeView from '@widget/views/HomeView.vue'
import { House, MessagesSquare } from 'lucide-vue-next'
import ChatView from '@widget/views/ChatView.vue'
import ConversationsView from '@widget/views/ConversationsView.vue'
import ConnectionBanner from '@widget/components/ConnectionBanner.vue'
import { useWidgetStore } from '@widget/store/widget.js'

const widgetStore = useWidgetStore()

const handleTabChange = (value) => {
  if (value === 'home') {
    widgetStore.navigateToHome()
  } else if (value === 'messages') {
    widgetStore.navigateToMessages()
  }
}
</script>
