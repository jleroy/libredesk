<template>
  <div>
    <Spinner v-if="isLoading" />
    <AdminSplitLayout>
      <template #content>
        <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
          <div class="flex justify-between mb-5">
            <div class="flex justify-end mb-4 w-full">
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child @click="newTag">
                  <Button class="ml-auto">{{
                    t('tag.new')
                  }}</Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle class="mb-1">
                      {{
                        isEditing
                          ? t('tag.edit')
                          : t('tag.new')
                      }}
                    </DialogTitle>
                    <DialogDescription>
                      {{
                        isEditing
                          ? t('admin.conversationTags.edit.description')
                          : t('admin.conversationTags.new.description')
                      }}
                    </DialogDescription>
                  </DialogHeader>
                  <TagsForm @submit.prevent="onSubmit">
                    <template #footer>
                      <DialogFooter class="mt-10">
                        <Button type="submit">{{ isEditing ? t('globals.messages.save') : t('globals.messages.create') }}</Button>
                      </DialogFooter>
                    </template>
                  </TagsForm>
                </DialogContent>
              </Dialog>
            </div>
          </div>
          <div>
            <DataTable :columns="createColumns(t, { onEdit: editTag })" :data="tags" :loading="isLoading" />
          </div>
        </div>
      </template>

      <template #help>
        <p>{{ $t('admin.tag.help') }}</p>
      </template>
    </AdminSplitLayout>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { Spinner } from '@shared-ui/components/ui/spinner/index.js'
import { createColumns } from '../../../features/admin/tags/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button/index.js'

import TagsForm from '@/features/admin/tags/TagsForm.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '../../../features/admin/tags/formSchema.js'
import { useEmitter } from '../../../composables/useEmitter.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api/index.js'

const { t } = useI18n()
const isLoading = ref(false)
const tags = ref([])
const emitter = useEmitter()
const dialogOpen = ref(false)
const isEditing = ref(false)
const editingId = ref(null)

onMounted(() => {
  getTags()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'tags') getTags()
  })
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, (data) => {
    if (data?.model === 'tags') {
      editTag(data.data)
    }
  })
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL)
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const editTag = (item) => {
  editingId.value = item.id
  form.setValues(item)
  form.setErrors({})
  isEditing.value = true
  dialogOpen.value = true
}

const newTag = () => {
  form.resetForm()
  form.setErrors({})
  isEditing.value = false
}

const getTags = async () => {
  isLoading.value = true
  const resp = await api.getTags()
  tags.value = resp.data.data
  isLoading.value = false
}

const onSubmit = form.handleSubmit(async (values) => {
  isLoading.value = true
  try {
    if (isEditing.value) {
      await api.updateTag(editingId.value, values)
    } else {
      await api.createTag(values)
    }
    dialogOpen.value = false
    getTags()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully'),
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})
</script>
