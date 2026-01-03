<template>
  <div>
    <Button @click="showDialog = true">Import Agents</Button>

    <Dialog v-model:open="showDialog">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Import Agents</DialogTitle>
        </DialogHeader>

        <div class="space-y-4 py-4">
          <!-- File Upload Section -->
          <div v-if="!importing && !complete" class="space-y-4">
            <div class="space-y-2">
              <label class="text-sm font-medium">Select CSV File</label>
              <Input 
                type="file" 
                accept=".csv"
                @change="onFileSelect"
                ref="fileInput"
              />
            </div>

            <Card v-if="file" class="p-3">
              <p class="text-sm"><strong>File:</strong> {{ file.name }}</p>
            </Card>

            <Alert>
              <AlertTitle>Required CSV Format</AlertTitle>
              <AlertDescription>
                <code class="text-xs">first_name, last_name, email, roles, teams</code>
                <p class="text-xs mt-1">Roles and teams must match exactly (case-sensitive)</p>
              </AlertDescription>
            </Alert>

            <Button @click="startImport" :disabled="!file" class="w-full">
              Start Import
            </Button>
          </div>

          <!-- Progress Section -->
          <div v-if="status" class="space-y-4">
            <div class="flex gap-4 justify-center">
              <div class="text-center">
                <p class="text-xs text-muted-foreground mb-1">Total</p>
                <Badge variant="outline" class="text-lg px-3 py-1">{{ status.total }}</Badge>
              </div>
              <div class="text-center">
                <p class="text-xs text-muted-foreground mb-1">Success</p>
                <Badge class="text-lg px-3 py-1 bg-green-500">{{ status.success }}</Badge>
              </div>
              <div class="text-center">
                <p class="text-xs text-muted-foreground mb-1">Errors</p>
                <Badge variant="destructive" class="text-lg px-3 py-1">{{ status.errors }}</Badge>
              </div>
            </div>

            <Separator />

            <div v-if="importing" class="flex items-center gap-2">
              <Spinner class="h-4 w-4" />
              <span class="text-sm">Importing agents...</span>
            </div>

            <Alert v-if="complete" class="bg-green-50 dark:bg-green-950 border-green-200">
              <AlertTitle class="text-green-600">Success!</AlertTitle>
              <AlertDescription class="text-green-600">Import completed successfully</AlertDescription>
            </Alert>

            <!-- Logs -->
            <div>
              <p class="text-sm font-medium mb-2">Import Logs</p>
              <Card class="p-3">
                <div class="bg-black text-white p-3 rounded-md text-xs font-mono max-h-60 overflow-y-auto space-y-1">
                  <div v-for="(log, idx) in status.logs" :key="idx">
                    {{ log }}
                  </div>
                </div>
              </Card>
            </div>
          </div>

          <Alert v-if="error" variant="destructive">
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>
        </div>

        <DialogFooter>
          <Button v-if="complete" @click="resetAndClose">Done</Button>
          <Button variant="outline" @click="closeDialog" :disabled="importing">
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, onBeforeUnmount } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Separator } from '@/components/ui/separator'
import { Spinner } from '@/components/ui/spinner'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import axios from 'axios'

const showDialog = ref(false)
const file = ref(null)
const importing = ref(false)
const complete = ref(false)
const status = ref(null)
const error = ref('')
const pollInterval = ref(null)

const emit = defineEmits(['import-complete'])

const getCSRFToken = () => {
  const name = 'csrf_token='
  const cookies = document.cookie.split(';')
  for (let i = 0; i < cookies.length; i++) {
    let c = cookies[i].trim()
    if (c.indexOf(name) === 0) {
      return c.substring(name.length, c.length)
    }
  }
  return ''
}

const onFileSelect = (e) => {
  file.value = e.target.files[0]
  error.value = ''
}

const startImport = async () => {
  if (!file.value) return

  error.value = ''
  importing.value = true

  const formData = new FormData()
  formData.append('file', file.value)

  try {
    await axios.post('/api/v1/agents/import', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
        'X-CSRFTOKEN': getCSRFToken()
      }
    })

    startPolling()
  } catch (err) {
    error.value = err.response?.data?.message || err.message || 'Upload failed'
    importing.value = false
  }
}

const startPolling = () => {
  pollInterval.value = setInterval(fetchStatus, 1000)
}

const fetchStatus = async () => {
  try {
    const res = await axios.get('/api/v1/agents/import/status')
    status.value = res.data.data

    if (!status.value.running) {
      stopPolling()
      importing.value = false
      complete.value = true
    }
  } catch (err) {
    if (err.response?.status !== 404) {
      console.error('Poll error:', err)
    }
  }
}

const stopPolling = () => {
  if (pollInterval.value) {
    clearInterval(pollInterval.value)
    pollInterval.value = null
  }
}

const closeDialog = () => {
  if (importing.value && !confirm('Import in progress. Close?')) return
  stopPolling()
  resetState()
  showDialog.value = false
}

const resetAndClose = () => {
  stopPolling()
  resetState()
  showDialog.value = false
  emit('import-complete')
}

const resetState = () => {
  file.value = null
  importing.value = false
  complete.value = false
  status.value = null
  error.value = ''
}

onBeforeUnmount(() => {
  stopPolling()
})
</script>