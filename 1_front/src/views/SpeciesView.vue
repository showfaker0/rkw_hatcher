<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, shallowRef, ref, watch } from 'vue'
import SpeciesDetailDialog from '../components/SpeciesDetailDialog.vue'
import { api, type EggGroup, type Nature, type Species, type SpeciesDeleteImpact } from '../api'
import { useToast } from '../composables/useToast'
import {
  getEggGroups,
  getNatures,
  invalidateSpeciesCache,
  setSpeciesAllCache,
} from '../composables/useDictCache'

const { toast } = useToast()

type SyncCandidate = {
  no: number
  name: string
  evoChain: string[]
  eggGroupIds: number[]
  eggGroupNames: string[]
  iconUrl: string
  wikiPetId?: string
}

const list = shallowRef<Species[]>([])
const eggGroups = shallowRef<EggGroup[]>([])
const natures = shallowRef<Nature[]>([])
const q = ref('')
const filterEggIds = ref<number[]>([])
/** true=并集，false=交集(默认) */
const unionMode = ref(false)
const loading = ref(false)
const saving = ref(false)

const formOpen = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({
  name: '',
  iconUrl: '',
  no: null as number | null,
  bestPvpNatureIds: [] as number[],
})

const syncLoading = ref(false)
const pickOpen = ref(false)
const candidates = shallowRef<SyncCandidate[]>([])
const selected = ref<Record<string, boolean>>({})
const syncMeta = ref({ totalFetched: 0, existing: 0, newCount: 0 })
const committing = ref(false)
const resetSpinning = ref(false)
const detailSpeciesId = ref<number | null>(null)
const deleteOpen = ref(false)
const deleting = ref(false)
const deleteImpact = ref<SpeciesDeleteImpact | null>(null)

const boostOrder = ['生命', '物攻', '魔攻', '物防', '魔防', '速度']
const naturesByBoost = computed(() => {
  const map = new Map<string, Nature[]>()
  for (const n of natures.value) {
    const key = n.boost || '其他'
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(n)
  }
  const keys = [
    ...boostOrder.filter((k) => map.has(k)),
    ...[...map.keys()].filter((k) => !boostOrder.includes(k)),
  ]
  return keys.map((boost) => ({ boost, items: map.get(boost) || [] }))
})

let searchTimer: ReturnType<typeof setTimeout> | null = null
let resetSpinTimer: ReturnType<typeof setTimeout> | null = null
let listAbort: AbortController | null = null
let listSeq = 0

async function loadMeta() {
  const [eggs, nats] = await Promise.all([getEggGroups(), getNatures()])
  eggGroups.value = eggs
  natures.value = nats
}

async function loadList() {
  if (listAbort) listAbort.abort()
  const ctrl = new AbortController()
  listAbort = ctrl
  const seq = ++listSeq
  loading.value = true
  try {
    const params = new URLSearchParams()
    if (q.value.trim()) params.set('q', q.value.trim())
    if (filterEggIds.value.length) {
      params.set('eggGroupIds', filterEggIds.value.join(','))
      params.set('eggMode', unionMode.value ? 'union' : 'intersect')
    }
    const qs = params.toString()
    const data = await api.get<Species[]>('/species' + (qs ? `?${qs}` : ''), {
      signal: ctrl.signal,
    })
    if (seq !== listSeq) return
    list.value = data
    if (!qs) setSpeciesAllCache(data)
  } catch (e) {
    if ((e as Error).name === 'AbortError') return
    toast((e as Error).message || '加载失败', { type: 'error' })
  } finally {
    if (seq === listSeq) loading.value = false
  }
}

function scheduleSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    void loadList()
  }, 280)
}

onMounted(async () => {
  try {
    await loadMeta()
    await loadList()
  } catch (e) {
    toast((e as Error).message || '初始化失败', { type: 'error' })
  }
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
  if (resetSpinTimer) clearTimeout(resetSpinTimer)
  if (listAbort) listAbort.abort()
})

watch(q, () => {
  filterEggIds.value = []
  scheduleSearch()
})

watch(unionMode, () => {
  void loadList()
})

function toggleFilterEgg(id: number) {
  const i = filterEggIds.value.indexOf(id)
  if (i >= 0) filterEggIds.value.splice(i, 1)
  else filterEggIds.value.push(id)
  void loadList()
}

function toggleEggMode() {
  unionMode.value = !unionMode.value
}

function resetFilters() {
  resetSpinning.value = false
  requestAnimationFrame(() => {
    resetSpinning.value = true
  })
  if (resetSpinTimer) clearTimeout(resetSpinTimer)
  resetSpinTimer = setTimeout(() => {
    resetSpinning.value = false
  }, 560)

  q.value = ''
  filterEggIds.value = []
  unionMode.value = false
  if (searchTimer) clearTimeout(searchTimer)
  void loadList()
}

function resetForm() {
  editingId.value = null
  form.name = ''
  form.iconUrl = ''
  form.no = null
  form.bestPvpNatureIds = []
}

function openEdit(s: Species) {
  editingId.value = s.id
  form.name = s.name
  form.iconUrl = s.iconUrl || ''
  form.no = s.no ?? null
  form.bestPvpNatureIds = [...(s.bestPvpNatureIds || [])]
  formOpen.value = true
}

function closeForm() {
  if (deleting.value) return
  formOpen.value = false
  deleteOpen.value = false
  deleteImpact.value = null
  resetForm()
}

async function askDelete() {
  if (!editingId.value || saving.value || deleting.value) return
  try {
    deleteImpact.value = await api.get<SpeciesDeleteImpact>(`/species/${editingId.value}/delete-impact`)
    deleteOpen.value = true
  } catch (e) {
    toast((e as Error).message || '无法删除', { type: 'error' })
  }
}

function cancelDelete() {
  if (deleting.value) return
  deleteOpen.value = false
  deleteImpact.value = null
}

async function confirmDelete() {
  if (!editingId.value) return
  deleting.value = true
  try {
    await api.del(`/species/${editingId.value}`)
    toast('已删除', { type: 'success' })
    deleteOpen.value = false
    deleteImpact.value = null
    formOpen.value = false
    resetForm()
    invalidateSpeciesCache()
    await loadList()
  } catch (e) {
    toast((e as Error).message || '删除失败', { type: 'error' })
  } finally {
    deleting.value = false
  }
}

function toggleFormNature(id: number) {
  const i = form.bestPvpNatureIds.indexOf(id)
  if (i >= 0) form.bestPvpNatureIds.splice(i, 1)
  else form.bestPvpNatureIds.push(id)
}

async function save() {
  if (!editingId.value) return
  saving.value = true
  try {
    if (!form.bestPvpNatureIds.length) throw new Error('至少勾选一个最佳性格')
    await api.put(`/species/${editingId.value}`, {
      bestPvpNatureIds: form.bestPvpNatureIds,
    })
    toast('编辑成功', { type: 'success' })
    closeForm()
    invalidateSpeciesCache()
    await loadList()
  } catch (e) {
    toast((e as Error).message || '编辑失败', { type: 'error' })
  } finally {
    saving.value = false
  }
}

async function runSync() {
  syncLoading.value = true
  try {
    const res = await api.post<{
      totalFetched: number
      existing: number
      newCount: number
      candidates: SyncCandidate[]
    }>('/species/sync/preview', {})
    syncMeta.value = {
      totalFetched: res.totalFetched,
      existing: res.existing,
      newCount: res.newCount,
    }
    candidates.value = res.candidates || []
    selected.value = {}
    for (const c of candidates.value) selected.value[c.name] = true
    if (!candidates.value.length) {
      toast('没有可新增的精灵（库中已有或工具箱无新数据）', { type: 'info' })
      return
    }
    pickOpen.value = true
  } catch (e) {
    toast((e as Error).message || '更新数据失败', { type: 'error' })
  } finally {
    syncLoading.value = false
  }
}

function toggleCandidate(name: string) {
  selected.value[name] = !selected.value[name]
}

function hideBrokenIcon(ev: Event) {
  const el = ev.target as HTMLImageElement | null
  if (el) el.style.display = 'none'
}

function selectAll(on: boolean) {
  const next: Record<string, boolean> = {}
  for (const c of candidates.value) next[c.name] = on
  selected.value = next
}

async function commitSelected() {
  const items = candidates.value.filter((c) => selected.value[c.name])
  if (!items.length) {
    toast('请至少勾选一只精灵', { type: 'warning' })
    return
  }
  committing.value = true
  try {
    const res = await api.post<{ inserted: number; skipped: number }>('/species/sync/commit', {
      items: items.map((c) => ({
        no: c.no,
        name: c.name,
        evoChain: c.evoChain,
        eggGroupIds: c.eggGroupIds,
        iconUrl: c.iconUrl,
      })),
    })
    toast(`新增 ${res.inserted} 条` + (res.skipped ? `，跳过 ${res.skipped}` : ''), {
      type: 'success',
    })
    pickOpen.value = false
    invalidateSpeciesCache()
    await loadList()
  } catch (e) {
    toast((e as Error).message || '写入失败', { type: 'error' })
  } finally {
    committing.value = false
  }
}

function formatNo(no?: number | null) {
  if (no == null || !Number.isFinite(no) || no <= 0) return '—'
  return String(Math.trunc(no)).padStart(3, '0')
}
</script>

<template>
  <div class="species-page">
    <section class="species-filter card">
      <div class="search-row">
        <div class="search-pill">
          <svg class="search-icon" viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="11" cy="11" r="6.5" fill="none" stroke="currentColor" stroke-width="1.8" />
            <path d="M16.2 16.2L20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
          </svg>
          <input
            v-model="q"
            type="text"
            placeholder="输入精灵名字、图鉴或进化链名称进行搜索"
          />
        </div>
        <button
          type="button"
          class="icon-btn"
          title="重置筛选"
          aria-label="重置筛选"
          @click="resetFilters"
        >
          <svg
            class="reset-icon"
            :class="{ spin: resetSpinning }"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              d="M20 12a8 8 0 1 1-2.2-5.5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
            />
            <path
              d="M20 5v5h-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>

      <div class="egg-grid">
        <button
          v-for="g in eggGroups"
          :key="g.id"
          type="button"
          class="egg-chip"
          :class="{ active: filterEggIds.includes(g.id) }"
          @click="toggleFilterEgg(g.id)"
        >
          {{ g.name }}
        </button>
      </div>

      <div class="filter-actions">
          <button
            type="button"
            class="egg-mode"
            :class="{ 'is-union': unionMode }"
            :title="unionMode ? '当前并集，点击切换为交集' : '当前交集，点击切换为并集'"
            :aria-label="unionMode ? '当前并集，点击切换为交集' : '当前交集，点击切换为并集'"
            @click="toggleEggMode"
          >
            <span class="egg-mode-thumb" aria-hidden="true"></span>
            <span class="egg-mode-btn" :class="{ active: !unionMode }">交集</span>
            <span class="egg-mode-btn" :class="{ active: unionMode }">并集</span>
          </button>
        <div class="list-count">当前共 {{ list.length }} 条数据</div>
        <div class="filter-actions-end">
          <button type="button" :disabled="syncLoading" @click="runSync">
            {{ syncLoading ? '更新中…' : '更新数据' }}
          </button>
        </div>
      </div>
    </section>

    <section class="species-list-wrap soft-scroll">
      <div v-if="loading && !list.length" class="list-empty muted">加载中…</div>
      <div v-else-if="!list.length" class="list-empty muted">暂无图鉴，点击「更新数据」从工具箱同步</div>
      <div v-else class="species-list">
        <article
          v-for="s in list"
          :key="s.id"
          v-memo="[s.id, s.no, s.name, s.iconUrl, s.evoChain, s.eggGroupNames, s.bestPvpNatureNames]"
          class="species-item card"
        >
          <span class="species-no">{{ formatNo(s.no) }}</span>
          <button
            type="button"
            class="species-icon-wrap species-icon-hit"
            :aria-label="'查看 ' + s.name + ' 种族值'"
            @click="detailSpeciesId = s.id"
          >
            <img
              v-if="s.iconUrl"
              class="species-icon"
              :src="s.iconUrl"
              alt=""
              referrerpolicy="no-referrer" @error="hideBrokenIcon"
            />
          </button>
          <div class="species-name">{{ s.name }}</div>
          <div class="species-info">
            <div class="meta-tags">
              <template v-for="(n, i) in (s.evoChain || [])" :key="'e' + n + i">
                <span v-if="i" class="meta-arrow">→</span>
                <span class="meta-tag">{{ n }}</span>
              </template>
              <span v-if="!(s.evoChain || []).length" class="muted">—</span>
            </div>
            <div class="meta-tags">
              <span v-for="g in (s.eggGroupNames || [])" :key="g" class="meta-tag egg">{{ g }}</span>
              <span v-if="!(s.eggGroupNames || []).length" class="muted">—</span>
            </div>
          </div>
          <div class="species-natures meta-tags">
            <span
              v-for="n in (s.bestPvpNatureNames || [])"
              :key="n"
              class="meta-tag nature"
            >{{ n }}</span>
            <span v-if="!(s.bestPvpNatureNames || []).length" class="muted">—</span>
          </div>
          <div class="actions">
            <button type="button" class="secondary" @click="openEdit(s)">编辑</button>
          </div>
        </article>
      </div>
    </section>

    <div v-if="formOpen" class="dialog-mask" @click.self="closeForm">
      <div class="dialog-card form-dialog edit-nature-dialog">
        <p class="dialog-title">编辑推荐性格</p>
        <div class="edit-species-head">
          <span class="species-no">{{ formatNo(form.no) }}</span>
          <span class="species-icon-wrap" aria-hidden="true">
            <img
              v-if="form.iconUrl"
              class="species-icon"
              :src="form.iconUrl"
              alt=""
              referrerpolicy="no-referrer" @error="hideBrokenIcon"
            />
          </span>
          <div class="edit-species-name">{{ form.name }}</div>
          <button type="button" class="danger" :disabled="saving || deleting" @click="askDelete">删除</button>
        </div>
        <p class="muted nature-pick-hint">推荐性格</p>
        <div class="nature-pick">
          <div v-for="row in naturesByBoost" :key="row.boost" class="nature-pick-row">
            <div class="nature-pick-grid">
              <button
                v-for="n in row.items"
                :key="n.id"
                type="button"
                class="nature-chip"
                :class="{ active: form.bestPvpNatureIds.includes(n.id) }"
                @click="toggleFormNature(n.id)"
              >
                <span class="nature-chip-name">{{ n.name }}</span>
                <span class="nature-chip-stats">
                  <span class="nature-boost">+{{ n.boost }}</span><span class="nature-sep">/</span><span class="nature-penalty">-{{ n.penalty }}</span>
                </span>
              </button>
            </div>
          </div>
        </div>
        <div class="dialog-actions">
          <button type="button" class="secondary" :disabled="saving || deleting" @click="closeForm">取消</button>
          <button type="button" :disabled="saving || deleting" @click="save">{{ saving ? '保存中…' : '保存' }}</button>
        </div>
      </div>
    </div>

    <div v-if="deleteOpen" class="dialog-mask" @click.self="cancelDelete">
      <div class="dialog-card">
        <p class="dialog-title">确认删除「{{ deleteImpact?.name || form.name }}」？</p>
        <div class="delete-impact">
          <p class="delete-impact-lead">
            将同时删除该图鉴在「我的精灵」中的
            <strong>{{ deleteImpact?.petCount ?? 0 }}</strong>
            只，以及以它为目标的
            <strong>{{ deleteImpact?.lineCount ?? 0 }}</strong>
            条产线，且不可恢复。
          </p>
          <p v-if="deleteImpact?.lineNames?.length" class="delete-impact-desc muted">
            产线：{{ deleteImpact.lineNames.join('、') }}
          </p>
        </div>
        <div class="dialog-actions">
          <button type="button" class="secondary" :disabled="deleting" @click="cancelDelete">取消</button>
          <button type="button" class="danger" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? '删除中…' : '删除' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="syncLoading" class="dialog-mask">
      <div class="dialog-card">
        <p class="dialog-title">正在从工具箱更新数据…</p>
        <p class="muted" style="text-align:center;margin:0">拉取最终体图鉴 / 进化链 / 蛋组</p>
        <div class="sync-loading-bar" aria-hidden="true" />
      </div>
    </div>

    <div v-if="pickOpen" class="dialog-mask" @click.self="pickOpen = false">
      <div class="dialog-card form-dialog pick-dialog">
        <p class="dialog-title">选择要新增的精灵</p>
        <p class="muted" style="margin-top:-8px;margin-bottom:12px;text-align:center">
          工具箱共 {{ syncMeta.totalFetched }} 只最终体，库中已有 {{ syncMeta.existing }}，
          可新增 {{ syncMeta.newCount }}（不会删除或覆盖已有）
        </p>
        <div class="actions" style="margin-bottom:10px">
          <button type="button" class="secondary" @click="selectAll(true)">全选</button>
          <button type="button" class="secondary" @click="selectAll(false)">全不选</button>
        </div>
        <div class="pick-list">
          <label
            v-for="c in candidates"
            :key="c.name"
            class="pick-item"
            :class="{ selected: !!selected[c.name] }"
          >
            <span class="pick-check">
              <input type="checkbox" :checked="!!selected[c.name]" @change="toggleCandidate(c.name)" />
              <span class="pick-check-ui" aria-hidden="true" />
            </span>
            <span class="pick-no">{{ formatNo(c.no) }}</span>
            <span class="pick-icon" aria-hidden="true">
              <img
                v-if="c.iconUrl"
                :src="c.iconUrl"
                alt=""
                referrerpolicy="no-referrer" @error="hideBrokenIcon"
              />
            </span>
            <div class="pick-body">
              <div class="pick-name">{{ c.name }}</div>
              <div class="pick-info">
                <div class="meta-tags">
                  <template v-for="(n, i) in (c.evoChain || [])" :key="n + i">
                    <span v-if="i" class="meta-arrow">→</span>
                    <span class="meta-tag">{{ n }}</span>
                  </template>
                  <span v-if="!(c.evoChain || []).length" class="muted">—</span>
                </div>
                <div class="meta-tags">
                  <span v-for="g in (c.eggGroupNames || [])" :key="g" class="meta-tag egg">{{ g }}</span>
                  <span v-if="!(c.eggGroupNames || []).length" class="muted">—</span>
                </div>
              </div>
            </div>
          </label>
        </div>
        <div class="dialog-actions">
          <button type="button" class="secondary" :disabled="committing" @click="pickOpen = false">取消</button>
          <button type="button" :disabled="committing" @click="commitSelected">
            {{ committing ? '写入中…' : '确认新增' }}
          </button>
        </div>
      </div>
    </div>

    <SpeciesDetailDialog :species-id="detailSpeciesId" @close="detailSpeciesId = null" />
  </div>
</template>
