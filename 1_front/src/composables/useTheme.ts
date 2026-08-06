import { ref, watch } from 'vue'

const THEME_KEY = 'rkw_theme'

export type ThemeMode = 'day' | 'night'

const theme = ref<ThemeMode>('day')

function applyTheme(mode: ThemeMode) {
  document.documentElement.setAttribute('data-theme', mode)
}

function initTheme() {
  const saved = localStorage.getItem(THEME_KEY) as ThemeMode | null
  theme.value = saved === 'night' ? 'night' : 'day'
  applyTheme(theme.value)
}

function toggleTheme() {
  theme.value = theme.value === 'day' ? 'night' : 'day'
}

watch(theme, (v) => {
  localStorage.setItem(THEME_KEY, v)
  applyTheme(v)
})

initTheme()

export function useTheme() {
  return { theme, toggleTheme }
}
