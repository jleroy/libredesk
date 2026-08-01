<template>
  <div
    class="editor-wrapper relative flex flex-col h-full min-h-0"
    :class="{ 'pointer-events-none': disabled }"
  >
    <EditorContent :editor="editor" class="native-html flex-1 min-h-0 overflow-y-auto pb-20" />

    <div
      v-if="editor"
      class="editor-toolbar absolute bottom-4 left-1/2 z-10 w-max max-w-[calc(100%-2rem)] -translate-x-1/2 rounded-xl border bg-background p-1 shadow-lg"
    >
      <EditorToolbar
        :editor="editor"
        show-article-tools
        :enable-inline-images="enableInlineImages"
        @open-link="linkDialog?.open()"
        @open-youtube="youtubeDialog?.open()"
        @open-image="imageInput?.click()"
      />
    </div>

    <input
      ref="imageInput"
      type="file"
      accept="image/*"
      multiple
      class="hidden"
      @change="onImageInputChange"
    />

    <EditorLinkDialog ref="linkDialog" :editor="editor" allow-button />
    <EditorYoutubeDialog ref="youtubeDialog" :editor="editor" />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { EditorContent } from '@tiptap/vue-3'
import EditorToolbar from './EditorToolbar.vue'
import EditorLinkDialog from './EditorLinkDialog.vue'
import EditorYoutubeDialog from './EditorYoutubeDialog.vue'
import { buildArticleExtensions } from './editorExtensions'
import { useTextEditor } from './useTextEditor'

const textContent = defineModel('textContent', { default: '' })
const htmlContent = defineModel('htmlContent', { default: '' })

const props = defineProps({
  placeholder: String,
  insertContent: String,
  autoFocus: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false },
  enableInlineImages: { type: Boolean, default: false },
  linkedModel: { type: String, default: 'messages' }
})

const emit = defineEmits(['send', 'filesDropped'])

const linkDialog = ref(null)
const youtubeDialog = ref(null)
const imageInput = ref(null)

const { editor, insertImages, focus } = useTextEditor({
  extensions: buildArticleExtensions({ getPlaceholder: () => props.placeholder }),
  htmlContent,
  textContent,
  autoFocus: props.autoFocus,
  editable: !props.disabled,
  insertContent: () => props.insertContent,
  isInlineEnabled: () => props.enableInlineImages,
  linkedModel: props.linkedModel,
  onSend: () => emit('send'),
  onOtherFiles: (files) => emit('filesDropped', files)
})

watch(
  () => props.disabled,
  (disabled) => editor.value?.setEditable(!disabled, false)
)

const onImageInputChange = (event) => {
  const files = Array.from(event.target.files || [])
  if (files.length > 0) insertImages(files)
  event.target.value = ''
}

defineExpose({ focus })
</script>

<style lang="scss" src="./editorStyles.scss"></style>

<style src="@public-static/article-content.css"></style>

<style lang="scss">
.tiptap {
  --hc-accent: hsl(var(--primary));
  --hc-border: hsl(var(--border));

  // Keep the body visible/editable regardless of the native open state.
  details.hc-details > .hc-details-content {
    display: block !important;
    margin-top: 0.5rem;
  }
}
</style>
