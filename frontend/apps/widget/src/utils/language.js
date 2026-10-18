const normalizeLanguage = (requested) => {
  try {
    return new Intl.Locale(String(requested || '').trim().replaceAll('_', '-')).baseName
  } catch {
    return ''
  }
}

export const matchLanguage = (requested, availableCodes) => {
  const wanted = normalizeLanguage(requested)
  if (!wanted || !Array.isArray(availableCodes)) return ''
  const exact = availableCodes.find((code) => normalizeLanguage(code) === wanted)
  if (exact) return exact
  const locale = new Intl.Locale(wanted).maximize()
  return availableCodes.find((code) => {
    const normalized = normalizeLanguage(code)
    if (!normalized) return false
    const candidate = new Intl.Locale(normalized).maximize()
    return candidate.language === locale.language && candidate.script === locale.script
  }) || ''
}

export const resolveLanguage = (config, hostLanguage, browserLanguages, availableCodes) => {
  const fallback = normalizeLanguage(config.fallback_language) || 'en-US'
  if (config.language !== 'auto') {
    const locale = normalizeLanguage(config.language) || fallback
    return { locale, requestedLocale: locale }
  }
  const preferences = [hostLanguage, ...browserLanguages, fallback]
    .map(normalizeLanguage)
    .filter(Boolean)
  const locale = [...preferences, 'en-US']
    .map((requested) => matchLanguage(requested, availableCodes))
    .find(Boolean) || 'en-US'
  return { locale, requestedLocale: preferences[0] }
}
