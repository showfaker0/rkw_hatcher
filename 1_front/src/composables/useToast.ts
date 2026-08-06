import { ref } from 'vue'

export type NotifyType = '' | 'primary' | 'success' | 'warning' | 'info' | 'error'

export interface NoticeItem {
  id: number
  title: string
  message: string
  type: NotifyType
  duration: number
  offset: number
  height: number
}

export type ToastOptions = {
  duration?: number
  title?: string
  type?: NotifyType
}

const GAP = 16
const START = 16
const DEFAULT_HEIGHT = 84

const notices = ref<NoticeItem[]>([])
let seed = 1
const timers = new Map<number, ReturnType<typeof setTimeout>>()

function defaultTitle(type: NotifyType) {
  switch (type) {
    case 'success':
      return '成功'
    case 'warning':
      return '警告'
    case 'error':
      return '错误'
    case 'info':
      return '提示'
    case 'primary':
      return '通知'
    default:
      return '提示'
  }
}

function recalcOffsets() {
  let top = START
  for (const n of notices.value) {
    n.offset = top
    top += (n.height || DEFAULT_HEIGHT) + GAP
  }
}

function closeNotice(id: number) {
  const i = notices.value.findIndex((n) => n.id === id)
  if (i < 0) return
  notices.value.splice(i, 1)
  const t = timers.get(id)
  if (t) clearTimeout(t)
  timers.delete(id)
  recalcOffsets()
}

function setNoticeHeight(id: number, height: number) {
  const n = notices.value.find((x) => x.id === id)
  if (!n || Math.abs(n.height - height) < 1) return
  n.height = height
  recalcOffsets()
}

export function useToast() {
  function toast(message: string, opts?: number | ToastOptions) {
    const options: ToastOptions = typeof opts === 'number' ? { duration: opts } : opts || {}
    const type = options.type ?? ''
    const id = seed++
    notices.value.push({
      id,
      title: options.title ?? defaultTitle(type),
      message,
      type,
      duration: options.duration ?? 4500,
      offset: 0,
      height: DEFAULT_HEIGHT,
    })
    recalcOffsets()

    const duration = options.duration ?? 4500
    if (duration > 0) {
      timers.set(
        id,
        setTimeout(() => closeNotice(id), duration),
      )
    }
  }

  return {
    notices,
    toast,
    closeNotice,
    setNoticeHeight,
  }
}
