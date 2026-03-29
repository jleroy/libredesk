<template>
  <AdminSplitLayout>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <div class="flex flex-col justify-evenly h-full gap-4">
        <div class="space-y-2">
          <p class="text-lg">{{ $t('globals.terms.email') }}</p>
          <p>{{ $t('admin.inbox.help.email') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/connecting-inboxes"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style"
          >
            <p>{{ $t('globals.messages.learnMore') }}</p>
          </a>
        </div>
        <div class="space-y-2">
          <p class="text-lg">{{ $t('globals.terms.liveChat') }}</p>
          <p>{{ $t('admin.inbox.help.livechat') }}</p>
        </div>
      </div>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
