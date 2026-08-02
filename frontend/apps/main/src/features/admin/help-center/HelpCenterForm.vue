<template>
  <form ref="formEl" @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="slug">
      <FormItem>
        <FormLabel>{{ t('globals.terms.slug') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('helpCenter.slugHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="page_title">
      <FormItem>
        <FormLabel>{{ t('helpCenter.pageTitle') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="header_text">
      <FormItem>
        <FormLabel>{{ t('helpCenter.headerText') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="meta_description">
      <FormItem>
        <FormLabel>{{ t('helpCenter.metaDescription') }}</FormLabel>
        <FormControl>
          <Textarea :rows="2" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('helpCenter.homeMetaDescriptionHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="logo_url">
      <FormItem>
        <FormLabel>{{ t('globals.terms.logoUrl') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="color">
      <FormItem>
        <FormLabel>{{ t('globals.terms.primaryColor') }}</FormLabel>
        <FormControl>
          <Input type="color" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="theme.favicon">
      <FormItem>
        <FormLabel>{{ t('admin.general.faviconURL') }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="t('helpCenter.faviconHint')" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Header & hero -->
    <div class="border-t pt-6 space-y-4">
      <h3 class="text-sm font-semibold">{{ t('helpCenter.styling.header') }}</h3>

      <FormField v-slot="{ componentField }" name="theme.tagline">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.tagline') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="t('helpCenter.styling.taglineHint')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="theme.header.background_type">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.headerBackground') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="t('helpCenter.styling.bgDefault')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="default">{{ t('helpCenter.styling.bgDefault') }}</SelectItem>
                <SelectItem value="solid">{{ t('helpCenter.styling.bgSolid') }}</SelectItem>
                <SelectItem value="gradient">{{ t('globals.terms.gradient') }}</SelectItem>
                <SelectItem value="image">{{ t('globals.terms.image') }}</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
        </FormItem>
      </FormField>

      <FormField
        v-if="form.values.theme?.header?.background_type === 'solid'"
        v-slot="{ componentField }"
        name="theme.header.background_color"
      >
        <FormItem>
          <FormLabel>{{ t('globals.messages.backgroundColor') }}</FormLabel>
          <FormControl>
            <Input type="color" v-bind="componentField" />
          </FormControl>
        </FormItem>
      </FormField>

      <div v-if="form.values.theme?.header?.background_type === 'gradient'" class="flex gap-4">
        <FormField v-slot="{ componentField }" name="theme.header.gradient_from">
          <FormItem class="flex-1">
            <FormLabel>{{ t('globals.messages.gradientStart') }}</FormLabel>
            <FormControl>
              <Input type="color" v-bind="componentField" />
            </FormControl>
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="theme.header.gradient_to">
          <FormItem class="flex-1">
            <FormLabel>{{ t('globals.messages.gradientEnd') }}</FormLabel>
            <FormControl>
              <Input type="color" v-bind="componentField" />
            </FormControl>
          </FormItem>
        </FormField>
      </div>

      <FormField
        v-if="form.values.theme?.header?.background_type === 'image'"
        v-slot="{ componentField }"
        name="theme.header.background_image"
      >
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.headerImage') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              placeholder="https://example.com/header.jpg"
              v-bind="componentField"
            />
          </FormControl>
          <FormDescription>{{ t('helpCenter.styling.headerImageHint') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="theme.header.text_color">
        <FormItem>
          <FormLabel>{{ t('globals.terms.textColor') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="#ffffff" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ t('helpCenter.styling.textColorHint') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <!-- Landing page -->
    <div class="border-t pt-6 space-y-4">
      <h3 class="text-sm font-semibold">{{ t('helpCenter.styling.landingPage') }}</h3>

      <FormField v-slot="{ componentField }" name="theme.layout.collections">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.collectionLayout') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="t('helpCenter.styling.layoutGrid')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="grid">{{ t('helpCenter.styling.layoutGrid') }}</SelectItem>
                <SelectItem value="list">{{ t('helpCenter.styling.layoutList') }}</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
        </FormItem>
      </FormField>

      <FormField
        v-if="form.values.theme?.layout?.collections !== 'list'"
        v-slot="{ componentField }"
        name="theme.layout.columns"
      >
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.cardsPerRow') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="2">2</SelectItem>
                <SelectItem value="3">3</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="theme.cards.icon_position">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.cardIconPosition') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="inline">{{
                  t('helpCenter.styling.iconBesideTitle')
                }}</SelectItem>
                <SelectItem value="top">{{ t('helpCenter.styling.iconAboveTitle') }}</SelectItem>
                <SelectItem value="center">{{ t('helpCenter.styling.iconCentered') }}</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
        </FormItem>
      </FormField>

      <FormField v-slot="{ value, handleChange }" name="theme.cards.hide_description">
        <FormItem class="flex items-center gap-2 space-y-0">
          <FormControl>
            <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
          </FormControl>
          <FormLabel class="font-normal cursor-pointer">{{
            t('helpCenter.styling.showCardDescription')
          }}</FormLabel>
        </FormItem>
      </FormField>
      <FormField v-slot="{ value, handleChange }" name="theme.cards.hide_count">
        <FormItem class="flex items-center gap-2 space-y-0">
          <FormControl>
            <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
          </FormControl>
          <FormLabel class="font-normal cursor-pointer">{{
            t('helpCenter.styling.showCardCount')
          }}</FormLabel>
        </FormItem>
      </FormField>
      <FormField v-slot="{ value, handleChange }" name="theme.cards.show_authors">
        <FormItem class="flex items-center gap-2 space-y-0">
          <FormControl>
            <Checkbox :checked="value" @update:checked="handleChange" />
          </FormControl>
          <FormLabel class="font-normal cursor-pointer">{{
            t('helpCenter.styling.showCardAuthors')
          }}</FormLabel>
        </FormItem>
      </FormField>
    </div>

    <!-- Footer -->
    <div class="border-t pt-6 space-y-4">
      <h3 class="text-sm font-semibold">{{ t('helpCenter.styling.footer') }}</h3>

      <div class="flex gap-4">
        <FormField v-slot="{ componentField }" name="theme.footer.background_color">
          <FormItem class="flex-1">
            <FormLabel>{{ t('globals.messages.backgroundColor') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="#ffffff" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="theme.footer.text_color">
          <FormItem class="flex-1">
            <FormLabel>{{ t('globals.terms.textColor') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="#909aa5" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <FormField v-slot="{ componentField }" name="theme.footer.tagline">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.footerTagline') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="t('helpCenter.styling.footerTaglineHint')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <LinkListField name="theme.footer_links" :label="t('helpCenter.styling.footerLinks')" />

      <LinkListField
        name="theme.social_links"
        :label="t('helpCenter.styling.socialLinks')"
        :new-item="{ platform: 'website', url: '' }"
      >
        <template #leading="{ index }">
          <FormField v-slot="{ componentField }" :name="`theme.social_links[${index}].platform`">
            <FormItem class="w-40">
              <FormControl>
                <Select v-bind="componentField">
                  <SelectTrigger>
                    <SelectValue :placeholder="t('globals.terms.platform')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="p in socialPlatforms" :key="p" :value="p">{{
                      t(`helpCenter.social.${p}`)
                    }}</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </template>
      </LinkListField>
    </div>

    <!-- Article page -->
    <div class="border-t pt-6 space-y-3">
      <h3 class="text-sm font-semibold">{{ t('helpCenter.styling.articlePage') }}</h3>
      <FormField v-slot="{ value, handleChange }" name="theme.article.hide_toc">
        <FormItem class="flex items-center gap-2 space-y-0">
          <FormControl>
            <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
          </FormControl>
          <FormLabel class="font-normal cursor-pointer">{{
            t('helpCenter.styling.showToc')
          }}</FormLabel>
        </FormItem>
      </FormField>
      <FormField v-slot="{ value, handleChange }" name="theme.article.hide_related">
        <FormItem class="flex items-center gap-2 space-y-0">
          <FormControl>
            <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
          </FormControl>
          <FormLabel class="font-normal cursor-pointer">{{
            t('helpCenter.styling.showRelated')
          }}</FormLabel>
        </FormItem>
      </FormField>
      <FormField v-slot="{ value, handleChange }" name="theme.article.show_author_avatar">
        <FormItem class="flex items-center gap-2 space-y-0">
          <FormControl>
            <Checkbox :checked="value" @update:checked="handleChange" />
          </FormControl>
          <FormLabel class="font-normal cursor-pointer">{{
            t('helpCenter.styling.showAuthorAvatar')
          }}</FormLabel>
        </FormItem>
      </FormField>
    </div>

    <div class="border-t pt-6">
      <LinkListField name="nav_links" :label="t('helpCenter.navLinks')" />
    </div>

    <div class="space-y-2">
      <Label>{{ t('helpCenter.supportedLanguages') }}</Label>
      <p class="text-sm text-muted-foreground">{{ t('helpCenter.supportedLanguagesHint') }}</p>
      <div v-for="(field, index) in localeFields" :key="field.key" class="flex items-start gap-2">
        <FormField v-slot="{ componentField }" :name="`allowed_locales[${index}]`">
          <FormItem class="flex-1">
            <FormControl>
              <Input
                type="text"
                placeholder="en"
                list="hc-locale-suggestions"
                v-bind="componentField"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          :disabled="localeFields.length <= 1"
          @click="removeLocale(index)"
        >
          <X class="w-4 h-4" />
        </Button>
      </div>
      <Button type="button" variant="outline" size="sm" @click="pushLocale('')">
        {{ t('globals.messages.add') }}
      </Button>
      <datalist id="hc-locale-suggestions">
        <option v-for="lang in availableLanguages" :key="lang.code" :value="lang.code">
          {{ lang.name }}
        </option>
      </datalist>
    </div>

    <FormField v-slot="{ componentField }" name="default_locale">
      <FormItem>
        <FormLabel>{{ t('helpCenter.defaultLanguage') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="loc in localeOptions" :key="loc" :value="loc">{{
                loc
              }}</SelectItem>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ t('helpCenter.defaultLanguageHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="custom_css">
      <FormItem>
        <FormLabel>{{ t('helpCenter.customCSS') }}</FormLabel>
        <FormControl>
          <Textarea rows="4" class="font-mono" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="custom_js">
      <FormItem>
        <FormLabel>{{ t('helpCenter.customJS') }}</FormLabel>
        <FormControl>
          <Textarea rows="4" class="font-mono" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="flex justify-end space-x-2 pt-4">
      <Button type="button" variant="outline" @click="$emit('cancel')">
        {{ t('globals.messages.cancel') }}
      </Button>
      <Button type="submit" :isLoading="isLoading">
        {{ submitLabel }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { watch, computed, ref, onMounted, nextTick } from 'vue'
import { useForm, useFieldArray } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { X } from 'lucide-vue-next'
import LinkListField from './LinkListField.vue'
import { createHelpCenterFormSchema } from './helpCenterFormSchema.js'
import api from '@/api'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  helpCenter: {
    type: Object,
    default: null
  },
  submitForm: {
    type: Function,
    required: true
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['cancel', 'change'])

const { t } = useI18n()

const submitLabel = computed(() =>
  props.helpCenter ? t('globals.messages.update') : t('globals.messages.create')
)

const socialPlatforms = [
  'website',
  'twitter',
  'github',
  'linkedin',
  'facebook',
  'instagram',
  'youtube'
]

const toFormValues = (hc) => ({
  name: hc?.name || '',
  slug: hc?.slug || '',
  page_title: hc?.page_title || '',
  header_text: hc?.header_text || '',
  meta_description: hc?.meta_description || '',
  logo_url: hc?.logo_url || '',
  color: hc?.color || '#1f93ff',
  nav_links: Array.isArray(hc?.nav_links) ? hc.nav_links : [],
  custom_css: hc?.custom_css || '',
  custom_js: hc?.custom_js || '',
  default_locale: hc?.default_locale || 'en',
  allowed_locales:
    Array.isArray(hc?.allowed_locales) && hc.allowed_locales.length ? hc.allowed_locales : ['en'],
  theme: {
    favicon: hc?.theme?.favicon || '',
    tagline: hc?.theme?.tagline || '',
    header: {
      background_type: hc?.theme?.header?.background_type || 'default',
      background_color: hc?.theme?.header?.background_color || '#1f93ff',
      gradient_from: hc?.theme?.header?.gradient_from || '#1f93ff',
      gradient_to: hc?.theme?.header?.gradient_to || '#ffffff',
      background_image: hc?.theme?.header?.background_image || '',
      text_color: hc?.theme?.header?.text_color || ''
    },
    layout: {
      collections: hc?.theme?.layout?.collections || 'grid',
      columns: String(hc?.theme?.layout?.columns || 2)
    },
    cards: {
      hide_description: !!hc?.theme?.cards?.hide_description,
      hide_count: !!hc?.theme?.cards?.hide_count,
      show_authors: !!hc?.theme?.cards?.show_authors,
      icon_position: hc?.theme?.cards?.icon_position || 'inline'
    },
    footer: {
      background_color: hc?.theme?.footer?.background_color || '',
      text_color: hc?.theme?.footer?.text_color || '',
      tagline: hc?.theme?.footer?.tagline || ''
    },
    footer_links: Array.isArray(hc?.theme?.footer_links) ? hc.theme.footer_links : [],
    social_links: Array.isArray(hc?.theme?.social_links) ? hc.theme.social_links : [],
    article: {
      hide_toc: !!hc?.theme?.article?.hide_toc,
      hide_related: !!hc?.theme?.article?.hide_related,
      show_author_avatar: !!hc?.theme?.article?.show_author_avatar
    }
  }
})

const formEl = ref(null)

const form = useForm({
  validationSchema: toTypedSchema(createHelpCenterFormSchema(t)),
  initialValues: toFormValues(props.helpCenter)
})

const {
  fields: localeFields,
  push: pushLocale,
  remove: removeLocale
} = useFieldArray('allowed_locales')

const availableLanguages = ref([])

onMounted(async () => {
  try {
    const { data } = await api.getAvailableLanguages()
    availableLanguages.value = data.data || []
  } catch {
    availableLanguages.value = []
  }
})

const cleanLocales = (locales) => (locales || []).map((l) => (l || '').trim()).filter(Boolean)

const localeOptions = computed(() => cleanLocales(form.values.allowed_locales))

// The backend forces the default language back into the supported list, so the select is
// moved to a language that is actually still listed instead of showing a stale one.
watch(localeOptions, (locales) => {
  if (locales.length && !locales.includes(form.values.default_locale)) {
    form.setFieldValue('default_locale', locales[0], false)
  }
})

const toPayload = (values) => {
  const payload = JSON.parse(JSON.stringify(values))
  const allowed = cleanLocales(payload.allowed_locales)
  payload.nav_links = payload.nav_links || []
  payload.allowed_locales = allowed.length ? allowed : ['en']
  if (payload.theme?.layout) {
    payload.theme.layout.columns = Number(payload.theme.layout.columns) || 2
  }
  return payload
}

const onSubmit = form.handleSubmit(
  async (values) => {
    props.submitForm(toPayload(values))
  },
  async () => {
    await nextTick()
    formEl.value?.querySelector('[role="alert"]')?.scrollIntoView({ block: 'center' })
  }
)

watch(
  () => props.helpCenter,
  (newValues) => {
    if (newValues && Object.keys(newValues).length > 0) {
      form.setValues(toFormValues(newValues), false)
    }
  },
  { immediate: true }
)

// The select yields a string; the stored theme needs columns as a number.
watch(
  () => form.values,
  (values) => emit('change', toPayload(values)),
  { deep: true, immediate: true }
)
</script>
