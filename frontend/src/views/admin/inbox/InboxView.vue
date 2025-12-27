<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <div class="native-html">
        <p>Connect your Google or Microsoft email account, or configure IMAP and SMTP settings manually.</p>
        <p>Each added inbox creates a new email channel for receiving customer emails.</p>
        <p>
          <a href="https://docs.libredesk.io/configuration/connecting-inboxes" target="_blank">
            Learn more
          </a>
        </p>
      </div>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
