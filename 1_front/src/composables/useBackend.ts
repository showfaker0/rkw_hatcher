import { ref } from 'vue'
import { useToast } from './useToast'

/** disconnected=意外断开；closed=用户主动关闭 */
export type ConnStatus = 'disconnected' | 'closed' | 'loading' | 'connected'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://127.0.0.1:3070/api'
const HEARTBEAT_MS = 5000
const START_POLL_MS = 1000
const START_TIMEOUT_MS = 60_000
const CLOSED_KEY = 'rkw_backend_closed'

const status = ref<ConnStatus>('disconnected')
/** 心跳成功瞬间点亮呼吸高亮 */
const pulseBright = ref(false)

let heartbeatTimer: ReturnType<typeof setInterval> | null = null
let startPollTimer: ReturnType<typeof setInterval> | null = null
let failStreak = 0
let pulseTimer: ReturnType<typeof setTimeout> | null = null
let busy = false
let inited = false
/** 初始化完成后，红/灰→绿才允许整页强刷 */
let allowHardReload = false
/** 曾进入红/灰或主动启动流程，连上后需要强刷一次 */
let needsHardReload = false
let startPingBusy = false

const { toast } = useToast()

function markClosed() {
  try {
    sessionStorage.setItem(CLOSED_KEY, '1')
  } catch {
    /* ignore */
  }
}

function clearClosedMark() {
  try {
    sessionStorage.removeItem(CLOSED_KEY)
  } catch {
    /* ignore */
  }
}

function wasClosedByUser() {
  try {
    return sessionStorage.getItem(CLOSED_KEY) === '1'
  } catch {
    return false
  }
}

async function ping(): Promise<boolean> {
  const ctrl = new AbortController()
  const t = setTimeout(() => ctrl.abort(), 2200)
  try {
    const res = await fetch(`${API_BASE}/health`, { signal: ctrl.signal, cache: 'no-store' })
    return res.ok
  } catch {
    return false
  } finally {
    clearTimeout(t)
  }
}

function flashPulse() {
  // 先关掉再打开，保证连续心跳时也能重播 2s 动画
  pulseBright.value = false
  requestAnimationFrame(() => {
    pulseBright.value = true
    if (pulseTimer) clearTimeout(pulseTimer)
    pulseTimer = setTimeout(() => {
      pulseBright.value = false
    }, 2000)
  })
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
  failStreak = 0
}

function stopStartPoll() {
  if (startPollTimer) {
    clearInterval(startPollTimer)
    startPollTimer = null
  }
  startPingBusy = false
}

function startHeartbeat() {
  stopHeartbeat()
  heartbeatTimer = setInterval(() => {
    void runHeartbeat()
  }, HEARTBEAT_MS)
}

function hardReloadIfNeeded() {
  if (!allowHardReload || !needsHardReload) return
  needsHardReload = false
  // 整页强刷，避免后端恢复后各页仍显示旧态/空数据
  window.location.reload()
}

function becomeConnected() {
  const prev = status.value
  status.value = 'connected'
  flashPulse()
  if (prev !== 'connected') hardReloadIfNeeded()
}

function markUnhealthy() {
  needsHardReload = true
}

async function runHeartbeat() {
  if (status.value === 'disconnected' || status.value === 'closed') return
  if (busy && status.value === 'loading') return

  const ok = await ping()
  if (ok) {
    failStreak = 0
    clearClosedMark()
    if (status.value !== 'connected') {
      becomeConnected()
    } else {
      flashPulse()
    }
    return
  }

  // 第一次失败 → loading；连续第三次失败 → 红点并停心跳
  failStreak += 1
  if (failStreak === 1) {
    status.value = 'loading'
    return
  }
  if (failStreak >= 3) {
    markUnhealthy()
    status.value = 'disconnected'
    stopHeartbeat()
    toast('后端连接已断开', { type: 'warning' })
  }
}

/** 手动插一次心跳：不影响 5s 计时器节奏 */
async function manualHeartbeat() {
  if (status.value !== 'connected') return
  const ok = await ping()
  if (ok) {
    failStreak = 0
    flashPulse()
  } else {
    failStreak = Math.max(failStreak, 1)
    status.value = 'loading'
    // 交给后续正规心跳累计到 3 次
  }
}

async function bootBackendApi() {
  const res = await fetch('/__boot_backend', { method: 'POST' })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || '启动请求失败')
}

async function stopBackendApi() {
  const res = await fetch('/__stop_backend', { method: 'POST' })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { detail?: string }).detail || '关闭请求失败')
}

async function startServer() {
  if (busy) return
  busy = true
  stopHeartbeat()
  stopStartPoll()
  markUnhealthy()
  status.value = 'loading'
  failStreak = 0
  clearClosedMark()

  try {
    await bootBackendApi()
  } catch (e) {
    markUnhealthy()
    status.value = 'disconnected'
    busy = false
    toast((e as Error).message || '启动失败', { type: 'error' })
    return
  }

  const begin = Date.now()
  startPollTimer = setInterval(async () => {
    if (startPingBusy) return
    if (Date.now() - begin > START_TIMEOUT_MS) {
      stopStartPoll()
      markUnhealthy()
      status.value = 'disconnected'
      busy = false
      toast('启动失败：超时未响应', { type: 'error' })
      return
    }
    startPingBusy = true
    try {
      const ok = await ping()
      if (ok) {
        stopStartPoll()
        clearClosedMark()
        busy = false
        startHeartbeat()
        toast('后端已启动', { type: 'success' })
        becomeConnected()
      }
    } finally {
      startPingBusy = false
    }
  }, START_POLL_MS)
}

async function stopServer() {
  if (busy) return
  busy = true
  stopHeartbeat()
  stopStartPoll()
  status.value = 'loading'
  failStreak = 0

  try {
    await stopBackendApi()
    // 确认关掉：最多等几秒
    for (let i = 0; i < 8; i++) {
      if (!(await ping())) break
      await new Promise((r) => setTimeout(r, 400))
    }
    const still = await ping()
    busy = false
    if (still) {
      status.value = 'connected'
      startHeartbeat()
      toast('关闭失败，服务仍在运行', { type: 'warning' })
    } else {
      markClosed()
      markUnhealthy()
      status.value = 'closed'
      toast('后端已关闭', { type: 'info' })
    }
  } catch (e) {
    const up = await ping()
    busy = false
    if (up) {
      status.value = 'connected'
      startHeartbeat()
    } else {
      // 关停接口失败但服务已不在：仍按主动关闭处理
      markClosed()
      markUnhealthy()
      status.value = 'closed'
    }
    toast((e as Error).message || '关闭失败', { type: 'error' })
  }
}

async function initBackendMonitor() {
  if (inited) return
  inited = true
  const ok = await ping()
  if (ok) {
    clearClosedMark()
    status.value = 'connected'
    flashPulse()
    startHeartbeat()
  } else {
    status.value = wasClosedByUser() ? 'closed' : 'disconnected'
  }
  allowHardReload = true
}

function disposeBackendMonitor() {
  stopHeartbeat()
  stopStartPoll()
  if (pulseTimer) {
    clearTimeout(pulseTimer)
    pulseTimer = null
  }
  inited = false
  allowHardReload = false
  needsHardReload = false
  busy = false
}

initBackendMonitor()

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    disposeBackendMonitor()
  })
}

export function useBackend() {
  return {
    status,
    pulseBright,
    startServer,
    stopServer,
    manualHeartbeat,
  }
}
