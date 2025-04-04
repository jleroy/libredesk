<template>
  <div class="min-h-screen flex flex-col bg-background">
    <main class="flex-1 flex items-center justify-center p-4">
      <div class="w-full max-w-[400px]">
        <Card class="bg-card border border-border shadow-xl rounded-xl">
          <CardContent class="p-8 space-y-6">
            <div class="space-y-2 text-center">
              <CardTitle class="text-3xl font-bold text-foreground">{{
                t('auth.resetPassword')
              }}</CardTitle>
              <p class="text-muted-foreground">{{ t('auth.enterEmailForReset') }}</p>
            </div>

            <form @submit.prevent="requestResetAction" class="space-y-4">
              <div class="space-y-2">
                <Label for="email" class="text-sm font-medium text-foreground">{{
                  t('globals.terms.email')
                }}</Label>
                <Input
                  id="email"
                  type="email"
                  :placeholder="t('auth.enterEmail')"
                  v-model.trim="resetForm.email"
                  :class="{ 'border-destructive': emailHasError }"
                  class="w-full bg-card border-border text-foreground placeholder:text-muted-foreground rounded-lg py-2 px-3 focus:ring-2 focus:ring-ring focus:border-ring transition-all duration-200 ease-in-out"
                />
              </div>

              <Button
                class="w-full bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg py-2 transition-all duration-200 ease-in-out transform hover:scale-105"
                :disabled="isLoading"
                type="submit"
              >
                <span v-if="isLoading" class="flex items-center justify-center">
                  <svg
                    class="animate-spin -ml-1 mr-3 h-5 w-5 text-primary-foreground"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      class="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      stroke-width="4"
                    ></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  {{ t('auth.sending') }}
                </span>
                <span v-else>{{ t('auth.sendResetLink') }}</span>
              </Button>
            </form>

            <Error
              v-if="errorMessage"
              :errorMessage="errorMessage"
              :border="true"
              class="w-full bg-destructive/10 text-destructive border-destructive/20 p-3 rounded-lg text-sm"
            />

            <div class="text-center">
              <router-link
                to="/"
                class="text-sm text-primary hover:text-primary/80 transition-all duration-200 ease-in-out"
              >
                {{ t('auth.backToLogin') }}
              </router-link>
            </div>
          </CardContent>
        </Card>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'
import { validateEmail } from '@/utils/strings'
import { useTemporaryClass } from '@/composables/useTemporaryClass'
import { Button } from '@/components/ui/button'
import { Error } from '@/components/ui/error'
import { Card, CardContent, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { Label } from '@/components/ui/label'
import { useI18n } from 'vue-i18n'

const errorMessage = ref('')
const { t } = useI18n()
const isLoading = ref(false)
const emitter = useEmitter()
const router = useRouter()
const resetForm = ref({
  email: ''
})

const validateForm = () => {
  if (!validateEmail(resetForm.value.email)) {
    errorMessage.value = 'Invalid email address.'
    useTemporaryClass('reset-password-container', 'animate-shake')
    return false
  }
  return true
}

const requestResetAction = async () => {
  if (!validateForm()) return

  errorMessage.value = ''
  isLoading.value = true

  try {
    await api.resetPassword({
      email: resetForm.value.email
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Reset link sent',
      description: 'Please check your email for the reset link.'
    })
    router.push({ name: 'login' })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Reset link sent',
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
    errorMessage.value = handleHTTPError(err).message
    useTemporaryClass('reset-password-container', 'animate-shake')
  } finally {
    isLoading.value = false
  }
}

const emailHasError = computed(() => {
  return !validateEmail(resetForm.value.email) && resetForm.value.email !== ''
})
</script>
