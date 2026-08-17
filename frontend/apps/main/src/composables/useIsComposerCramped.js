import { useMediaQuery } from '@vueuse/core'

// The inline composer needs ~280px; a phone in landscape is as short of room as a phone in portrait is narrow.
export function useIsComposerCramped () {
  return useMediaQuery('(max-width: 767px), (max-height: 500px) and (pointer: coarse)')
}
