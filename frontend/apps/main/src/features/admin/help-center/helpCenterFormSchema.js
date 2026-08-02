import * as z from 'zod'

const localeRe = /^[a-z]{2,3}(-[A-Za-z0-9]{2,8})?$/

// Mirrors assetURLRe on the backend.
const urlRe = /^(?:https?:\/\/|\/)[^"'()\s\\<>;{}]+$/

export const createHelpCenterBasicsSchema = (t) =>
  createHelpCenterFormSchema(t).pick({ name: true, slug: true, page_title: true })

export const createHelpCenterFormSchema = (t) => {
  const optionalURL = z
    .string()
    .refine((v) => !v || urlRe.test(v), t('helpCenter.invalidURL'))
    .optional()

  const linkArray = z
    .array(
      z.object({
        label: z.string().min(1, t('globals.messages.required')),
        url: z
          .string()
          .min(1, t('globals.messages.required'))
          .regex(urlRe, t('helpCenter.invalidURL'))
      })
    )
    .optional()

  return z.object({
    name: z.string().min(1, t('globals.messages.required')),
    slug: z
      .string()
      .min(1, t('globals.messages.required'))
      .max(200, t('helpCenter.invalidSlug'))
      .regex(/^[a-z0-9_-]+$/, t('helpCenter.invalidSlug')),
    page_title: z.string().min(1, t('globals.messages.required')),
    header_text: z.string().optional(),
    meta_description: z.string().optional(),
    logo_url: optionalURL,
    color: z.string().optional(),
    nav_links: linkArray,
    custom_css: z.string().optional(),
    custom_js: z.string().optional(),
    default_locale: z
      .string()
      .min(1, t('globals.messages.required'))
      .regex(localeRe, t('helpCenter.invalidLocale'))
      .default('en'),
    allowed_locales: z
      .array(z.string().min(1, t('globals.messages.required')).regex(localeRe, t('helpCenter.invalidLocale')))
      .min(1, t('globals.messages.required'))
      .default(['en']),
    theme: z
      .object({
        favicon: optionalURL,
        tagline: z.string().optional(),
        header: z
          .object({
            background_type: z.string().optional(),
            background_color: z.string().optional(),
            gradient_from: z.string().optional(),
            gradient_to: z.string().optional(),
            background_image: optionalURL,
            text_color: z.string().optional()
          })
          .optional(),
        layout: z
          .object({
            collections: z.string().optional(),
            columns: z.coerce.number().optional()
          })
          .optional(),
        cards: z
          .object({
            hide_description: z.boolean().optional(),
            hide_count: z.boolean().optional(),
            show_authors: z.boolean().optional(),
            icon_position: z.enum(['inline', 'top', 'center']).optional()
          })
          .optional(),
        footer: z
          .object({
            background_color: z.string().optional(),
            text_color: z.string().optional(),
            tagline: z.string().optional()
          })
          .optional(),
        footer_links: linkArray,
        social_links: z
          .array(
            z.object({
              platform: z.string().min(1, t('globals.messages.required')),
              url: z
                .string()
                .min(1, t('globals.messages.required'))
                .regex(urlRe, t('helpCenter.invalidURL'))
            })
          )
          .optional(),
        article: z
          .object({
            hide_toc: z.boolean().optional(),
            hide_related: z.boolean().optional(),
            show_author_avatar: z.boolean().optional()
          })
          .optional()
      })
      .optional()
  })
}
