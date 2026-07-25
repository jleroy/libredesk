import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useConnectionStore = defineStore('connection', () => {
  const connected = ref(false)
  const connecting = ref(false)
  const connectionFailed = ref(false)

  const setConnected = (value) => {
    connected.value = value
  }

  const setConnecting = (value) => {
    connecting.value = value
  }

  const setConnectionFailed = (failed) => {
    connectionFailed.value = failed
  }

  return {
    connected,
    connecting,
    connectionFailed,

    setConnected,
    setConnecting,
    setConnectionFailed
  }
})
