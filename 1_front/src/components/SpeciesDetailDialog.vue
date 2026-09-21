<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api, type SpeciesDetail, type SpeciesForm, type StatRow } from '../api'

const props = defineProps<{ speciesId: number | null }>()
const emit = defineEmits<{ close: [] }>()

const loading = ref(false)
const error = ref('')
const detail = ref<SpeciesDetail | null>(null)
const formKey = ref('')

const STAT_ORDER = ['生命', '物攻', '魔攻', '物防', '魔防', '速度']
const RADAR_ORDER = ['生命', '魔攻', '魔防', '速度', '物防', '物攻']
const RADAR_MAX = 180

watch(
  () => props.speciesId,
  async (id) => {
    detail.value = null
    error.value = ''
    formKey.value = ''
    if (id == null) return
    loading.value = true
    try {
      const d = await api.get<SpeciesDetail>(`/species/${id}/detail`)
      detail.value = d
      formKey.value = d.defaultFormKey || d.forms[0]?.key || ''
    } catch (e) {
      error.value = (e as Error).message || '加载失败'
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

const current = computed(() => {
  const forms = detail.value?.forms || []
  return forms.find((f) => f.key === formKey.value) || forms[0] || null
})

function orderedStats(f: SpeciesForm, order: string[]): StatRow[] {
  const map = new Map(f.stats.map((s) => [s.label, s]))
  const out: StatRow[] = []
  for (const label of order) {
    const s = map.get(label)
    if (s) out.push(s)
  }
  return out
}

const stats = computed(() => (current.value ? orderedStats(current.value, STAT_ORDER) : []))

const radar = computed(() => {
  const f = current.value
  if (!f) return null
  const rows = orderedStats(f, RADAR_ORDER)
  if (!rows.length) return null
  const cx = 120
  const cy = 120
  const r = 78
  const pts = rows.map((s, i) => {
    const ang = -Math.PI / 2 + (i * Math.PI * 2) / rows.length
    const rr = r * (s.value / RADAR_MAX)
    return {
      x: cx + rr * Math.cos(ang),
      y: cy + rr * Math.sin(ang),
      lx: cx + (r + 22) * Math.cos(ang),
      ly: cy + (r + 22) * Math.sin(ang),
      ax: cx + r * Math.cos(ang),
      ay: cy + r * Math.sin(ang),
      label: s.label,
      value: s.value,
    }
  })
  const rings = [0.25, 0.5, 0.75, 1].map((t) =>
    rows
      .map((_, i) => {
        const ang = -Math.PI / 2 + (i * Math.PI * 2) / rows.length
        return `${cx + r * t * Math.cos(ang)},${cy + r * t * Math.sin(ang)}`
      })
      .join(' '),
  )
  return { cx, cy, r, pts, rings, poly: pts.map((p) => `${p.x},${p.y}`).join(' ') }
})

function formatNo(no?: number | null) {
  if (no == null || !Number.isFinite(no) || no <= 0) return '—'
  return String(Math.trunc(no)).padStart(3, '0')
}

function hideBrokenIcon(ev: Event) {
  const el = ev.target as HTMLImageElement | null
  if (el) el.style.display = 'none'
}

function close() {
  emit('close')
}

const matchBoxes = computed(() => {
  const rel = current.value?.typeRelations
  return [
    { key: 'counter', title: '克制', items: rel?.counter || [] },
    { key: 'countered', title: '被克制', items: rel?.counteredBy || [] },
    { key: 'resist', title: '抵抗', items: rel?.resist || [] },
    { key: 'resisted', title: '被抵抗', items: rel?.resistedBy || [] },
  ]
})
</script>

<template>
  <div v-if="speciesId != null" class="dialog-mask species-detail-mask" @click.self="close">
    <div class="dialog-card species-detail-dialog">
      <button type="button" class="sd-close" aria-label="关闭" @click="close">×</button>

      <div v-if="loading" class="sd-state muted">加载中…</div>
      <div v-else-if="error" class="sd-state error">{{ error }}</div>
      <template v-else-if="current">
        <div class="sd-head">
          <span class="species-icon-wrap sd-avatar" aria-hidden="true">
            <img
              v-if="current.iconUrl"
              :key="current.iconUrl"
              class="species-icon"
              :src="current.iconUrl"
              alt=""
              referrerpolicy="no-referrer" @error="hideBrokenIcon"
            />
          </span>
          <div class="sd-head-main">
            <div class="sd-name-row">
              <span class="sd-name">{{ current.name }}</span>
              <span class="sd-no">No.{{ formatNo(current.catalogNo) }}</span>
            </div>
            <div class="sd-tags">
              <span
                v-for="t in current.types"
                :key="t.key"
                class="sd-type"
                :data-key="t.key"
              >{{ t.label }}</span>
              <span v-if="current.stageLabel" class="sd-badge">{{ current.stageLabel }}</span>
              <span v-if="current.isFinal" class="sd-badge gold">最终形态</span>
              <span v-if="current.isLeader" class="sd-badge gold">首领化</span>
            </div>
          </div>
        </div>

        <label class="sd-form-row">
          <span>切换形态</span>
          <select v-model="formKey">
            <option v-for="f in detail?.forms || []" :key="f.key" :value="f.key">
              {{ f.name }}
            </option>
          </select>
        </label>

        <div class="sd-stats">
          <div class="sd-bars">
            <div v-for="s in stats" :key="s.key || s.label" class="sd-bar-row">
              <span class="sd-bar-label">{{ s.label }}</span>
              <span class="sd-bar-track">
                <span
                  class="sd-bar-fill"
                  :style="{ width: Math.min(100, (s.value / Math.max(current.statMax || 1, 1)) * 100) + '%' }"
                />
              </span>
              <span class="sd-bar-val">{{ s.value }}</span>
            </div>
          </div>
          <svg v-if="radar" class="sd-radar" viewBox="0 0 240 240" overflow="visible" aria-hidden="true">
            <polygon
              v-for="(ring, i) in radar.rings"
              :key="'r' + i"
              :points="ring"
              class="sd-radar-grid"
            />
            <line
              v-for="(p, i) in radar.pts"
              :key="'a' + i"
              :x1="radar.cx"
              :y1="radar.cy"
              :x2="p.ax"
              :y2="p.ay"
              class="sd-radar-axis"
            />
            <polygon :points="radar.poly" class="sd-radar-area" />
            <circle
              v-for="(p, i) in radar.pts"
              :key="'d' + i"
              :cx="p.x"
              :cy="p.y"
              r="3.2"
              class="sd-radar-dot"
            />
            <text
              v-for="(p, i) in radar.pts"
              :key="'l' + i"
              :x="p.lx"
              :y="p.ly"
              class="sd-radar-label"
              text-anchor="middle"
              dominant-baseline="middle"
            >{{ p.label }}</text>
          </svg>
        </div>

        <div class="sd-match">
          <p class="sd-match-title">属性克制</p>
          <div class="sd-match-grid">
            <div v-for="box in matchBoxes" :key="box.key" class="sd-match-box">
              <div class="sd-match-head">{{ box.title }}</div>
              <div class="sd-match-body">
                <span
                  v-for="t in box.items"
                  :key="t.key"
                  class="sd-type"
                  :data-key="t.key"
                >{{ t.label }}</span>
                <span v-if="!box.items.length" class="muted">—</span>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
