<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { useTheme } from './composables/useTheme'
import { useBackend } from './composables/useBackend'
import { useToast } from './composables/useToast'

const route = useRoute()
const { theme, toggleTheme } = useTheme()
const { status, pulseBright, startServer, stopServer, manualHeartbeat } = useBackend()
const { notices, closeNotice, setNoticeHeight } = useToast()

const menus = [
  { to: '/', label: '最佳配种查询' },
  { to: '/lines', label: '我的产线' },
  { to: '/pets', label: '我的精灵' },
  { to: '/species', label: '图鉴' },
]

const connTitle: Record<string, string> = {
  disconnected: '意外断开（点击启动）',
  closed: '已关闭（点击启动）',
  loading: '处理中…',
  connected: '已连接（单击检测 / 双击关闭）',
}

type DialogKind = 'start' | 'stop' | null
const dialog = ref<DialogKind>(null)
let clickTimer: ReturnType<typeof setTimeout> | null = null

function isActive(to: string) {
  return route.path === to
}

function onConnClick() {
  if (status.value === 'loading') return

  if (status.value === 'disconnected' || status.value === 'closed') {
    dialog.value = 'start'
    return
  }

  // 绿点：单击 / 双击分流
  if (clickTimer) {
    clearTimeout(clickTimer)
    clickTimer = null
    dialog.value = 'stop'
    return
  }
  clickTimer = setTimeout(() => {
    clickTimer = null
    void manualHeartbeat()
  }, 280)
}

async function confirmDialog() {
  const kind = dialog.value
  dialog.value = null
  if (kind === 'start') await startServer()
  if (kind === 'stop') await stopServer()
}

function cancelDialog() {
  dialog.value = null
}

function onNoticeMount(id: number, el: Element | null) {
  if (!(el instanceof HTMLElement)) return
  void nextTick(() => setNoticeHeight(id, el.offsetHeight))
}
</script>

<template>
  <div class="app-shell">
    <header class="top-bar">
      <div class="top-bar-main">
        <nav class="menu">
          <RouterLink
            v-for="m in menus"
            :key="m.to"
            :to="m.to"
            class="menu-item"
            :class="{ 'is-active': isActive(m.to) }"
          >
            {{ m.label }}
          </RouterLink>
        </nav>
      </div>

      <aside class="status-panel">
        <button
          type="button"
          class="status-btn theme-btn"
          :title="theme === 'day' ? '切换黑夜' : '切换白昼'"
          :aria-label="theme === 'day' ? '切换黑夜' : '切换白昼'"
          @click="toggleTheme"
        >
          <svg v-if="theme === 'day'" class="theme-icon" viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="12" cy="12" r="4" fill="currentColor" />
            <g stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
              <path d="M12 2v2.2M12 19.8V22M2 12h2.2M19.8 12H22M4.6 4.6l1.6 1.6M17.8 17.8l1.6 1.6M4.6 19.4l1.6-1.6M17.8 6.2l1.6-1.6" />
            </g>
          </svg>
          <svg v-else class="theme-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="currentColor"
              d="M16.4 2.1a1 1 0 0 1 1.1 1.4A8.5 8.5 0 1 0 20.5 15a1 1 0 0 1 1.5-1.2A10.5 10.5 0 1 1 16.4 2.1Z"
            />
          </svg>
        </button>

        <button
          type="button"
          class="status-btn conn-btn"
          :class="[status, { 'is-pulse': pulseBright }]"
          :title="connTitle[status]"
          :aria-label="connTitle[status]"
          @click="onConnClick"
        >
          <span v-if="status === 'loading'" class="conn-spinner" aria-hidden="true" />
          <span
            v-else
            class="conn-dot"
            :class="[status, { bright: pulseBright && status === 'connected' }]"
            aria-hidden="true"
          />
        </button>
      </aside>
    </header>

    <main class="page-body">
      <div class="page-inner">
        <RouterView />
      </div>
    </main>

    <div v-if="dialog" class="dialog-mask" @click.self="cancelDialog">
      <div class="dialog-card">
        <p class="dialog-title">
          {{ dialog === 'start' ? '是否启动服务端程序？' : '是否关闭服务端程序？' }}
        </p>
        <div class="dialog-actions">
          <button type="button" class="secondary" @click="cancelDialog">否</button>
          <button type="button" @click="confirmDialog">是</button>
        </div>
      </div>
    </div>

    <TransitionGroup name="notify">
      <div
        v-for="n in notices"
        :key="n.id"
        class="notify"
        :class="n.type ? `is-${n.type}` : ''"
        :style="{ top: `${n.offset}px` }"
        :ref="(el) => onNoticeMount(n.id, el as Element | null)"
      >
        <span v-if="n.type" class="notify-icon" aria-hidden="true">
          <svg v-if="n.type === 'success'" viewBox="0 0 24 24">
            <circle cx="12" cy="12" r="10" fill="currentColor" opacity="0.15" />
            <path d="M7.5 12.5l3 3 6-6.5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <svg v-else-if="n.type === 'warning'" viewBox="0 0 24 24">
            <circle cx="12" cy="12" r="10" fill="currentColor" opacity="0.15" />
            <path d="M12 7.5v5.5M12 16.2h.01" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" />
          </svg>
          <svg v-else-if="n.type === 'error'" viewBox="0 0 24 24">
            <circle cx="12" cy="12" r="10" fill="currentColor" opacity="0.15" />
            <path d="M9 9l6 6M15 9l-6 6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
          </svg>
          <svg v-else viewBox="0 0 24 24">
            <circle cx="12" cy="12" r="10" fill="currentColor" opacity="0.15" />
            <path d="M12 11v5.2M12 8.2h.01" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" />
          </svg>
        </span>
        <div class="notify-body">
          <div class="notify-title">{{ n.title }}</div>
          <div class="notify-msg">{{ n.message }}</div>
        </div>
        <button type="button" class="notify-close" aria-label="关闭" @click="closeNotice(n.id)">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M7 7l10 10M17 7L7 17" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
          </svg>
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>
