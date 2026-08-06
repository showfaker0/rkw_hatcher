<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, shallowRef } from 'vue'
import { api, type BreedingLine, type BreedingLineRun, type Medal, type MyPet, type Nature, type Species } from '../api'
import { buildBreedSchemes, schemeTitle } from '../composables/useBreedSchemes'
import { useToast } from '../composables/useToast'
import { getNatures, getSpeciesAll } from '../composables/useDictCache'

const { toast } = useToast()
const list = ref<BreedingLine[]>([])
const species = shallowRef<Species[]>([])
const natures = shallowRef<Nature[]>([])
const pets = ref<MyPet[]>([])
const medals = ref<Medal[]>([])
const medalTypes = ref<string[]>([])
const error = ref('')
const editing = ref<number | null>(null)
const formOpen = ref(false)
const saving = ref(false)
const pendingDeleteId = ref<number | null>(null)

const produceOpen = ref(false)
const produceLine = ref<BreedingLine | null>(null)
const pickStudId = ref<number | null>(null)
const pickDamId = ref<number | null>(null)
const produceBusy = ref(false)
const pendingStopRunId = ref<number | null>(null)

const form = reactive({
  targetSpeciesId: '' as number | '',
  expectedNatureId: '' as number | '',
  medalPick: {} as Record<string, number | ''>,
})
/** 编辑时保留原名称/状态/备注，弹窗内不展示 */
const editMeta = reactive({
  name: '',
  status: '进行中',
  stepsNote: '' as string | null,
})

const listQ = ref('')
const resetSpinning = ref(false)
let resetSpinTimer: ReturnType<typeof setTimeout> | null = null
const speciesQuery = ref('')
const natureQuery = ref('')
const openMenu = ref<string | null>(null)
const speciesSearchRef = ref<HTMLInputElement | null>(null)
const natureSearchRef = ref<HTMLInputElement | null>(null)

const medalsByType = computed(() => {
  const map: Record<string, Medal[]> = {}
  for (const t of medalTypes.value) map[t] = []
  for (const m of medals.value) {
    if (!map[m.type]) map[m.type] = []
    map[m.type].push(m)
  }
  return map
})

const speciesById = computed(() => {
  const m = new Map<number, Species>()
  for (const s of species.value) m.set(s.id, s)
  return m
})

const selectedSpecies = computed(() =>
  species.value.find((s) => s.id === form.targetSpeciesId) || null,
)
const selectedNature = computed(() =>
  natures.value.find((n) => n.id === form.expectedNatureId) || null,
)

const filteredSpecies = computed(() => {
  const q = speciesQuery.value.trim().toLowerCase()
  if (!q) return species.value
  return species.value.filter((s) => {
    if (s.name.toLowerCase().includes(q)) return true
    return (s.evoChain || []).some((n) => String(n).toLowerCase().includes(q))
  })
})

const filteredNatures = computed(() => {
  const q = natureQuery.value.trim().toLowerCase()
  if (!q) return natures.value
  return natures.value.filter((n) => {
    const blob = `${n.name}${n.boost}${n.penalty}`.toLowerCase()
    return blob.includes(q)
  })
})

const globalActiveRuns = computed(() =>
  list.value.reduce((n, b) => n + (b.runs?.length || 0), 0),
)

const produceRuns = computed(() => produceLine.value?.runs || [])

const produceInUseIds = computed(() => {
  const ids = new Set<number>()
  for (const r of produceRuns.value) {
    ids.add(r.studPetId)
    ids.add(r.damPetId)
  }
  return ids
})

const producePoolStuds = computed(() => {
  const line = produceLine.value
  if (!line) return [] as MyPet[]
  const seen = new Set<number>()
  const out: MyPet[] = []
  for (const sch of line.schemes || []) {
    for (const p of sch.studs || []) {
      if (seen.has(p.id) || produceInUseIds.value.has(p.id)) continue
      seen.add(p.id)
      out.push(freshPet(p))
    }
  }
  return out
})

const producePoolDams = computed(() => {
  const line = produceLine.value
  if (!line) return [] as MyPet[]
  const seen = new Set<number>()
  const out: MyPet[] = []
  for (const sch of line.schemes || []) {
    for (const p of sch.dams || []) {
      if (seen.has(p.id) || produceInUseIds.value.has(p.id)) continue
      seen.add(p.id)
      out.push(freshPet(p))
    }
  }
  return out
})

const produceBlockReason = computed(() => {
  const line = produceLine.value
  if (!line) return '产线不存在'
  if (produceBusy.value) return '处理中，请稍候'
  if (pickStudId.value == null || pickDamId.value == null) return '请先完成选择期望投入生产的精灵'
  const stud = producePoolStuds.value.find((p) => p.id === pickStudId.value)
  const dam = producePoolDams.value.find((p) => p.id === pickDamId.value)
  if (!stud || !dam) return '请先完成选择期望投入生产的精灵'
  if (globalActiveRuns.value >= 5) return '全局生产槽已满(5/5)，请先关闭其他产线的生产'
  if ((line.runs?.length || 0) >= (line.maxSlots || 0)) return '本产线生产槽已满，请先关闭一队再生产'
  if ((stud.status || '空闲中') === '忙碌中' || (dam.status || '空闲中') === '忙碌中') {
    return '所选精灵忙碌中，无法加入生产'
  }
  return ''
})

const canAddProduce = computed(() => !produceBlockReason.value)

const produceTitle = computed(() => {
  const line = produceLine.value
  if (!line) return '产线'
  const name = line.targetSpeciesName || line.name || '未命名'
  const nature = line.expectedNatureName || ''
  return nature ? `产线：${name}·${nature}` : `产线：${name}`
})

function defaultMedalId(type: string): number | '' {
  const name = type === '体型' ? '大块头' : type === '声音' ? '婉转音' : ''
  if (!name) return ''
  return medalsByType.value[type]?.find((m) => m.name === name)?.id ?? ''
}

function medalLabel(type: string) {
  const id = form.medalPick[type]
  if (!id) return '无'
  return medalsByType.value[type]?.find((m) => m.id === id)?.name || '无'
}

function buildMedals() {
  return medalTypes.value
    .filter((t) => form.medalPick[t])
    .map((t) => ({ medalType: t, medalId: Number(form.medalPick[t]) }))
}

const previewSchemes = computed(() => {
  if (!form.targetSpeciesId || !form.expectedNatureId) return []
  return buildBreedSchemes({
    targetSpeciesId: Number(form.targetSpeciesId),
    natureId: Number(form.expectedNatureId),
    medals: buildMedals(),
    pets: pets.value,
    speciesById: speciesById.value,
  })
})

const canSave = computed(() =>
  !!form.targetSpeciesId && !!form.expectedNatureId && previewSchemes.value.length > 0,
)

const displayedList = computed(() => {
  const q = listQ.value.trim().toLowerCase()
  const base = !q
    ? list.value.slice()
    : list.value.filter((b) => {
        const sp = speciesById.value.get(b.targetSpeciesId)
        const blob = [
          b.targetSpeciesName,
          ...(sp?.evoChain || []).map((n) => String(n)),
          b.expectedNatureName,
        ].join(' ').toLowerCase()
        return blob.includes(q)
      })
	return base.sort((a, b) => {
    const aActive = (a.runs?.length || 0) > 0 ? 1 : 0
    const bActive = (b.runs?.length || 0) > 0 ? 1 : 0
    if (aActive !== bActive) return bActive - aActive
    const rank = (x: BreedingLine) => {
      if ((x.schemes || []).some((s) => s.schemeType === 1)) return 3
      if ((x.schemes || []).some((s) => s.schemeType === 2 || s.schemeType === 3)) return 2
      if ((x.schemes || []).some((s) => s.schemeType === 4)) return 1
      return 0
    }
    const ra = rank(a)
    const rb = rank(b)
    if (ra !== rb) return rb - ra
    return b.id - a.id
  })
})

function freshPet(p: MyPet): MyPet {
  const live = pets.value.find((x) => x.id === p.id)
  return live ? { ...p, status: live.status || p.status } : p
}

function lineMaxSlots(b: BreedingLine) {
  if (typeof b.maxSlots === 'number') return b.maxSlots
  const studs = new Set<number>()
  const dams = new Set<number>()
  for (const sch of b.schemes || []) {
    for (const p of sch.studs || []) studs.add(p.id)
    for (const p of sch.dams || []) dams.add(p.id)
  }
  return Math.min(studs.size, dams.size, 5)
}

function lineActiveCount(b: BreedingLine) {
  return b.runs?.length || 0
}

function isStrongScheme(b: BreedingLine) {
  return (b.schemes || []).some((s) => s.schemeType === 1)
}

function isVeryWeakScheme(b: BreedingLine) {
  const schemes = b.schemes || []
  if (!schemes.length) return false
  if (schemes.some((s) => s.schemeType === 1 || s.schemeType === 2 || s.schemeType === 3)) return false
  return schemes.some((s) => s.schemeType === 4)
}

function schemeStrengthLabel(b: BreedingLine) {
  if (isStrongScheme(b)) return '强推荐'
  if (isVeryWeakScheme(b)) return '极低效'
  return '弱推荐'
}

async function load() {
  const [lines, sp, nat, petList, med, types] = await Promise.all([
    api.get<BreedingLine[]>('/breeding-lines'),
    getSpeciesAll(),
    getNatures(),
    api.get<MyPet[]>('/pets'),
    api.get<Medal[]>('/medals'),
    api.get<string[]>('/medal-types'),
  ])
  list.value = lines
  species.value = sp
  natures.value = nat
  pets.value = petList
  medals.value = med
  medalTypes.value = types
  for (const t of medalTypes.value) {
    if (form.medalPick[t] === undefined) form.medalPick[t] = ''
  }
  if (produceLine.value) {
    const fresh = lines.find((x) => x.id === produceLine.value!.id) || null
    produceLine.value = fresh
    if (!fresh) {
      produceOpen.value = false
      pickStudId.value = null
      pickDamId.value = null
    }
  }
}

onMounted(() => {
  void load()
  document.addEventListener('mousedown', onDocDown)
})
onUnmounted(() => {
  document.removeEventListener('mousedown', onDocDown)
  if (resetSpinTimer) clearTimeout(resetSpinTimer)
})

function onDocDown(ev: MouseEvent) {
  const t = ev.target as HTMLElement | null
  if (!t?.closest('.pet-combo')) openMenu.value = null
}

function resetForm() {
  editing.value = null
  form.targetSpeciesId = ''
  form.expectedNatureId = ''
  form.medalPick = {}
  for (const t of medalTypes.value) form.medalPick[t] = defaultMedalId(t)
  editMeta.name = ''
  editMeta.status = '进行中'
  editMeta.stepsNote = null
  speciesQuery.value = ''
  natureQuery.value = ''
  openMenu.value = null
  error.value = ''
}

function openCreate() {
  resetForm()
  formOpen.value = true
}

function openEdit(b: BreedingLine) {
  editing.value = b.id
  form.targetSpeciesId = b.targetSpeciesId
  form.expectedNatureId = b.expectedNatureId
  form.medalPick = {}
  for (const t of medalTypes.value) form.medalPick[t] = ''
  for (const m of b.medals || []) form.medalPick[m.medalType] = m.medalId
  editMeta.name = b.name
  editMeta.status = b.status
  editMeta.stepsNote = b.stepsNote || null
  speciesQuery.value = ''
  natureQuery.value = ''
  openMenu.value = null
  error.value = ''
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
  resetForm()
}

function openProduce(b: BreedingLine) {
  produceLine.value = b
  pickStudId.value = null
  pickDamId.value = null
  pendingStopRunId.value = null
  produceOpen.value = true
}

function closeProduce() {
  produceOpen.value = false
  produceLine.value = null
  pickStudId.value = null
  pickDamId.value = null
  pendingStopRunId.value = null
}

function togglePickStud(p: MyPet) {
  if ((p.status || '空闲中') === '忙碌中') return
  pickStudId.value = pickStudId.value === p.id ? null : p.id
}

function togglePickDam(p: MyPet) {
  if ((p.status || '空闲中') === '忙碌中') return
  pickDamId.value = pickDamId.value === p.id ? null : p.id
}

async function addProduceRun() {
  const line = produceLine.value
  const reason = produceBlockReason.value
  if (!line || reason) {
    if (reason) toast(reason, { type: 'warning' })
    return
  }
  produceBusy.value = true
  try {
    await api.post(`/breeding-lines/${line.id}/runs`, {
      studPetId: pickStudId.value,
      damPetId: pickDamId.value,
    })
    toast('已加入生产', { type: 'success' })
    pickStudId.value = null
    pickDamId.value = null
    await load()
  } catch (e) {
    toast((e as Error).message, { type: 'error' })
  } finally {
    produceBusy.value = false
  }
}

function askStopRun(run: BreedingLineRun) {
  pendingStopRunId.value = run.id
}

function cancelStopRun() {
  pendingStopRunId.value = null
}

async function confirmStopRun() {
  const line = produceLine.value
  const runId = pendingStopRunId.value
  if (!line || runId == null) return
  pendingStopRunId.value = null
  produceBusy.value = true
  try {
    await api.del(`/breeding-lines/${line.id}/runs/${runId}`)
    toast('已停止该队生产', { type: 'info' })
    await load()
  } catch (e) {
    toast((e as Error).message, { type: 'error' })
  } finally {
    produceBusy.value = false
  }
}

function toggleMenu(key: string) {
  if (openMenu.value === key) {
    openMenu.value = null
    return
  }
  openMenu.value = key
  if (key === 'species') {
    speciesQuery.value = ''
    void nextTick(() => speciesSearchRef.value?.focus())
  }
  if (key === 'nature') {
    natureQuery.value = ''
    void nextTick(() => natureSearchRef.value?.focus())
  }
}

function pickSpecies(s: Species) {
  form.targetSpeciesId = s.id
  speciesQuery.value = ''
  openMenu.value = null
}

function pickNature(n: Nature) {
  form.expectedNatureId = n.id
  natureQuery.value = ''
  openMenu.value = null
}

function pickMedal(type: string, id: number | '') {
  form.medalPick[type] = id
  openMenu.value = null
}

function hideBrokenIcon(ev: Event) {
  const el = ev.target as HTMLImageElement | null
  if (el) el.style.display = 'none'
}

function formatPetId(id: number) {
  return String(id).padStart(3, '0')
}

function resetListSearch() {
  resetSpinning.value = false
  requestAnimationFrame(() => {
    resetSpinning.value = true
  })
  if (resetSpinTimer) clearTimeout(resetSpinTimer)
  resetSpinTimer = setTimeout(() => {
    resetSpinning.value = false
  }, 560)
  listQ.value = ''
}

async function save() {
  error.value = ''
  if (!canSave.value) {
    error.value = '请选择目标精灵与性格，且需有推荐方案'
    return
  }
  saving.value = true
  try {
    const body = {
      name: editing.value ? editMeta.name : '',
      targetSpeciesId: Number(form.targetSpeciesId),
      expectedNatureId: Number(form.expectedNatureId),
      status: editing.value ? editMeta.status : '进行中',
      stepsNote: editing.value ? editMeta.stepsNote : null,
      medals: buildMedals(),
    }
    if (editing.value) await api.put(`/breeding-lines/${editing.value}`, body)
    else await api.post('/breeding-lines', body)
    toast('已保存产线', { type: 'success' })
    closeForm()
    await load()
  } catch (e) {
    error.value = (e as Error).message
    toast((e as Error).message, { type: 'error' })
  } finally {
    saving.value = false
  }
}

function remove(id: number) {
  pendingDeleteId.value = id
}

function cancelDelete() {
  pendingDeleteId.value = null
}

async function confirmDelete() {
  const id = pendingDeleteId.value
  if (!id) return
  pendingDeleteId.value = null
  await api.del(`/breeding-lines/${id}`)
  toast('已删除', { type: 'info' })
  await load()
}

function lineIcon(b: BreedingLine) {
  return b.targetSpeciesIcon || speciesById.value.get(b.targetSpeciesId)?.iconUrl || ''
}

function petIconUrl(p: MyPet) {
  return speciesById.value.get(p.speciesId)?.iconUrl || ''
}

/** 弱推荐：性格与目标不符的半透明显示；强推荐不降透明度 */
function produceNatureDim(p: MyPet) {
  const line = produceLine.value
  if (!line?.expectedNatureId) return false
  if ((line.schemes || []).some((s) => s.schemeType === 1)) return false
  return p.natureId !== line.expectedNatureId
}
</script>

<template>
  <div class="lines-page">
    <section class="card pets-toolbar">
      <div class="pets-toolbar-row lines-toolbar-row">
        <div class="pets-toolbar-left">
          <div class="search-row pets-search">
            <div class="search-pill">
              <svg class="search-icon" viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="11" cy="11" r="6.5" fill="none" stroke="currentColor" stroke-width="1.8" />
                <path d="M16.2 16.2L20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
              </svg>
              <input v-model="listQ" type="text" placeholder="搜索目标精灵（含进化链）、目标性格…" />
            </div>
            <button
              type="button"
              class="icon-btn"
              title="清空搜索"
              aria-label="清空搜索"
              @click="resetListSearch"
            >
              <svg class="reset-icon" :class="{ spin: resetSpinning }" viewBox="0 0 24 24" aria-hidden="true">
                <path d="M20 12a8 8 0 1 1-2.2-5.5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
                <path d="M20 5v5h-5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
          </div>
        </div>
        <div class="list-count">当前共 {{ displayedList.length }} 条数据</div>
        <div class="pets-toolbar-right">
          <button type="button" @click="openCreate">新增</button>
        </div>
      </div>
    </section>

    <section class="lines-list-wrap soft-scroll">
      <div v-if="!list.length" class="list-empty muted">暂无产线，点击「新增」添加</div>
      <div v-else-if="!displayedList.length" class="list-empty muted">无匹配产线</div>
      <div v-else class="lines-list">
        <article v-for="b in displayedList" :key="b.id" class="line-item card">
          <button
            type="button"
            class="line-slots"
            :title="`生产 ${lineActiveCount(b)}/${lineMaxSlots(b)}（全局 ${globalActiveRuns}/5）`"
            :aria-label="`打开生产，当前 ${lineActiveCount(b)}/${lineMaxSlots(b)}`"
            @click="openProduce(b)"
          >
            <span
              v-for="i in lineMaxSlots(b)"
              :key="i"
              class="line-slot-block"
              :class="i <= lineActiveCount(b) ? 'busy' : 'idle'"
            />
          </button>
          <span class="species-icon-wrap" aria-hidden="true">
            <img v-if="lineIcon(b)" class="species-icon" :src="lineIcon(b)" alt="" @error="hideBrokenIcon" />
          </span>
          <div class="line-name">{{ b.targetSpeciesName || b.name }}</div>
          <div class="meta-tags line-natures">
            <span v-if="b.expectedNatureName" class="meta-tag nature">{{ b.expectedNatureName }}</span>
            <span v-else class="muted">—</span>
          </div>
          <div class="meta-tags line-medals-stack">
            <span
              v-for="m in (b.medals || [])"
              :key="m.medalType + m.medalId"
              class="meta-tag medal"
              :class="m.medalType === '声音' ? 'medal-voice' : 'medal-body'"
            >{{ m.medalName }}</span>
            <span v-if="!(b.medals || []).length" class="muted">—</span>
          </div>
          <span
            class="line-scheme-strength"
            :class="{
              strong: isStrongScheme(b),
              weak: !isStrongScheme(b) && !isVeryWeakScheme(b),
              'very-weak': isVeryWeakScheme(b),
            }"
            :title="schemeStrengthLabel(b)"
            :aria-label="schemeStrengthLabel(b)"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path fill="currentColor" d="M12 3.2l2.4 4.86 5.36.78-3.88 3.78.92 5.34L12 15.7l-4.8 2.52.92-5.34L4.24 8.84l5.36-.78L12 3.2z" />
            </svg>
          </span>
          <div class="line-spacer" aria-hidden="true" />
          <div class="actions">
            <button type="button" class="secondary" @click="openEdit(b)">编辑</button>
            <button type="button" class="danger" @click="remove(b.id)">删除</button>
          </div>
        </article>
      </div>
    </section>

    <div v-if="formOpen" class="dialog-mask" @click.self="closeForm">
      <div class="dialog-card form-dialog line-form-dialog">
        <p class="dialog-title">{{ editing ? '编辑产线 #' + editing : '新增产线' }}</p>

        <div class="line-form-grid">
          <div class="pet-field species-field">
            <span>目标精灵</span>
            <div class="pet-combo" :class="{ open: openMenu === 'species' }">
              <button
                type="button"
                class="pet-combo-trigger"
                :class="{ placeholder: !form.targetSpeciesId }"
                @click="toggleMenu('species')"
              >{{ selectedSpecies?.name || '请选择' }}</button>
              <span class="pet-combo-arrow" aria-hidden="true" />
              <div v-if="openMenu === 'species'" class="pet-combo-menu">
                <div class="pet-combo-search" @mousedown.stop>
                  <input ref="speciesSearchRef" v-model="speciesQuery" type="text" placeholder="搜索名称或进化链…" autocomplete="off" />
                </div>
                <div class="pet-combo-list soft-scroll">
                  <button
                    v-for="s in filteredSpecies"
                    :key="s.id"
                    type="button"
                    class="pet-combo-option"
                    :class="{ active: form.targetSpeciesId === s.id }"
                    @mousedown.prevent="pickSpecies(s)"
                  >{{ s.name }}</button>
                </div>
              </div>
            </div>
          </div>

          <div class="pet-field">
            <span>期望性格</span>
            <div class="pet-combo" :class="{ open: openMenu === 'nature' }">
              <button
                type="button"
                class="pet-combo-trigger"
                :class="{ placeholder: !form.expectedNatureId }"
                @click="toggleMenu('nature')"
              >{{ selectedNature ? `${selectedNature.name} (+${selectedNature.boost}/-${selectedNature.penalty})` : '请选择' }}</button>
              <span class="pet-combo-arrow" aria-hidden="true" />
              <div v-if="openMenu === 'nature'" class="pet-combo-menu">
                <div class="pet-combo-search" @mousedown.stop>
                  <input ref="natureSearchRef" v-model="natureQuery" type="text" placeholder="搜索性格…" autocomplete="off" />
                </div>
                <div class="pet-combo-list soft-scroll">
                  <button
                    v-for="n in filteredNatures"
                    :key="n.id"
                    type="button"
                    class="pet-combo-option"
                    :class="{ active: form.expectedNatureId === n.id }"
                    @mousedown.prevent="pickNature(n)"
                  >{{ n.name }} (+{{ n.boost }}/-{{ n.penalty }})</button>
                </div>
              </div>
            </div>
          </div>

          <div v-for="t in medalTypes" :key="t" class="pet-field">
            <span>奖章·{{ t }}</span>
            <div class="pet-combo" :class="{ open: openMenu === 'medal:' + t }">
              <button
                type="button"
                class="pet-combo-trigger"
                :class="{ placeholder: !form.medalPick[t] }"
                @click="toggleMenu('medal:' + t)"
              >{{ medalLabel(t) }}</button>
              <span class="pet-combo-arrow" aria-hidden="true" />
              <div v-if="openMenu === 'medal:' + t" class="pet-combo-menu soft-scroll">
                <button type="button" class="pet-combo-option" :class="{ active: !form.medalPick[t] }" @mousedown.prevent="pickMedal(t, '')">无</button>
                <button
                  v-for="m in medalsByType[t] || []"
                  :key="m.id"
                  type="button"
                  class="pet-combo-option"
                  :class="{ active: form.medalPick[t] === m.id }"
                  @mousedown.prevent="pickMedal(t, m.id)"
                >{{ m.name }}</button>
              </div>
            </div>
          </div>
        </div>

        <div class="line-schemes-panel">
          <p class="line-schemes-title">配对方案</p>
          <div class="line-schemes-body">
            <p v-if="!form.targetSpeciesId || !form.expectedNatureId" class="line-schemes-empty muted">请先选择目标精灵与性格</p>
            <p v-else-if="!previewSchemes.length" class="line-schemes-empty muted">暂无推荐方案（需有可配公母且奖章/蛋组满足）</p>
            <div v-else class="line-scheme-list">
              <div
                v-for="sch in previewSchemes"
                :key="sch.schemeType"
                class="line-scheme-card"
                :class="{ strong: sch.schemeType === 1, weak: sch.schemeType !== 1 }"
              >
                <div class="line-scheme-head">{{ schemeTitle(sch.schemeType) }}</div>
                <div class="line-scheme-cols">
                  <div class="line-scheme-col">
                    <div class="line-scheme-col-title">
                      <span class="gender-mark male" aria-label="公">♂</span>
                    </div>
                    <div class="line-scheme-pets">
                      <div
                        v-for="p in sch.studs"
                        :key="'s' + p.id"
                        class="line-scheme-pet"
                        :class="{ dim: sch.schemeType === 2 || sch.schemeType === 4 }"
                      >
                        <span
                          class="pet-status-dot"
                          :class="(p.status || '空闲中') === '忙碌中' ? 'busy' : 'idle'"
                          :title="p.status || '空闲中'"
                        />
                        <span class="scheme-pet-id">{{ formatPetId(p.id) }}</span>
                        <span class="species-icon-wrap scheme-pet-icon" aria-hidden="true">
                          <img v-if="petIconUrl(p)" class="species-icon" :src="petIconUrl(p)" alt="" @error="hideBrokenIcon" />
                        </span>
                        <span class="scheme-pet-name">{{ p.speciesName }}</span>
                        <span class="scheme-pet-nature meta-tag nature">{{ p.natureName || '—' }}</span>
                      </div>
                    </div>
                  </div>
                  <div class="line-scheme-col">
                    <div class="line-scheme-col-title">
                      <span class="gender-mark female" aria-label="母">♀</span>
                    </div>
                    <div class="line-scheme-pets">
                      <div
                        v-for="p in sch.dams"
                        :key="'d' + p.id"
                        class="line-scheme-pet"
                        :class="{ dim: sch.schemeType === 3 || sch.schemeType === 4 }"
                      >
                        <span
                          class="pet-status-dot"
                          :class="(p.status || '空闲中') === '忙碌中' ? 'busy' : 'idle'"
                          :title="p.status || '空闲中'"
                        />
                        <span class="scheme-pet-id">{{ formatPetId(p.id) }}</span>
                        <span class="species-icon-wrap scheme-pet-icon" aria-hidden="true">
                          <img v-if="petIconUrl(p)" class="species-icon" :src="petIconUrl(p)" alt="" @error="hideBrokenIcon" />
                        </span>
                        <span class="scheme-pet-name">{{ p.speciesName }}</span>
                        <span class="scheme-pet-nature meta-tag nature">{{ p.natureName || '—' }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <p v-if="error" class="error">{{ error }}</p>
        <div class="dialog-actions">
          <button type="button" class="secondary" :disabled="saving" @click="closeForm">取消</button>
          <button type="button" :disabled="saving || !canSave" @click="save">
            {{ saving ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="produceOpen && produceLine" class="dialog-mask" @click.self="closeProduce">
      <div class="dialog-card form-dialog line-produce-dialog">
        <p class="dialog-title">
          {{ produceTitle }}
          <span class="muted produce-title-meta">
            {{ produceRuns.length }}/{{ lineMaxSlots(produceLine) }} · 全局 {{ globalActiveRuns }}/5
          </span>
        </p>

        <div class="produce-zone">
          <div class="produce-zone-head">生产中</div>
          <div v-if="!produceRuns.length" class="produce-empty muted">暂无生产队，从下方选择一公一母加入</div>
          <div v-else class="produce-pair-panel">
            <div class="produce-pair-row produce-pair-head">
              <span class="gender-mark male" aria-label="公">♂</span>
              <span class="gender-mark female" aria-label="母">♀</span>
            </div>
            <div v-for="r in produceRuns" :key="r.id" class="produce-pair-row produce-run-row">
              <div v-if="r.stud" class="line-scheme-pet produce-run-pet">
                <span class="pet-status-dot busy" title="忙碌中" />
                <span class="scheme-pet-id">{{ formatPetId(r.stud.id) }}</span>
                <span class="species-icon-wrap scheme-pet-icon" aria-hidden="true">
                  <img v-if="petIconUrl(r.stud)" class="species-icon" :src="petIconUrl(r.stud)" alt="" @error="hideBrokenIcon" />
                </span>
                <span class="scheme-pet-name">{{ r.stud.speciesName }}</span>
                <span class="scheme-pet-nature meta-tag nature">{{ r.stud.natureName || '—' }}</span>
              </div>
              <div v-else class="produce-slot-empty" />
              <div v-if="r.dam" class="line-scheme-pet produce-run-pet">
                <span class="pet-status-dot busy" title="忙碌中" />
                <span class="scheme-pet-id">{{ formatPetId(r.dam.id) }}</span>
                <span class="species-icon-wrap scheme-pet-icon" aria-hidden="true">
                  <img v-if="petIconUrl(r.dam)" class="species-icon" :src="petIconUrl(r.dam)" alt="" @error="hideBrokenIcon" />
                </span>
                <span class="scheme-pet-name">{{ r.dam.speciesName }}</span>
                <span class="scheme-pet-nature meta-tag nature">{{ r.dam.natureName || '—' }}</span>
              </div>
              <div v-else class="produce-slot-empty" />
              <button
                type="button"
                class="produce-run-close-icon"
                title="关闭该队生产"
                aria-label="关闭该队生产"
                :disabled="produceBusy"
                @click="askStopRun(r)"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M5 8h14M9.5 8V6.8c0-.7.5-1.3 1.2-1.3h2.6c.7 0 1.2.6 1.2 1.3V8M10 11v5.5M14 11v5.5M7.2 8l.7 11c0 .8.7 1.4 1.5 1.4h5.2c.8 0 1.5-.6 1.5-1.4l.7-11"
                  />
                </svg>
              </button>
            </div>
          </div>
        </div>

        <div class="produce-zone">
          <div class="produce-zone-head">待生产</div>
          <div class="produce-pair-panel">
            <div class="produce-pair-row produce-pair-head">
              <span class="gender-mark male" aria-label="公">♂</span>
              <span class="gender-mark female" aria-label="母">♀</span>
            </div>
            <div class="produce-pair-row produce-pool-body">
              <div class="line-scheme-pets">
                <p v-if="!producePoolStuds.length" class="muted produce-empty-inline">无可选公</p>
                <button
                  v-for="p in producePoolStuds"
                  :key="'ps' + p.id"
                  type="button"
                  class="line-scheme-pet produce-pick"
                  :class="{
                    selected: pickStudId === p.id,
                    disabled: (p.status || '空闲中') === '忙碌中',
                    dim: produceNatureDim(p),
                  }"
                  :disabled="(p.status || '空闲中') === '忙碌中'"
                  @click="togglePickStud(p)"
                >
                  <span
                    class="pet-status-dot"
                    :class="(p.status || '空闲中') === '忙碌中' ? 'busy' : 'idle'"
                    :title="p.status || '空闲中'"
                  />
                  <span class="scheme-pet-id">{{ formatPetId(p.id) }}</span>
                  <span class="species-icon-wrap scheme-pet-icon" aria-hidden="true">
                    <img v-if="petIconUrl(p)" class="species-icon" :src="petIconUrl(p)" alt="" @error="hideBrokenIcon" />
                  </span>
                  <span class="scheme-pet-name">{{ p.speciesName }}</span>
                  <span class="scheme-pet-nature meta-tag nature">{{ p.natureName || '—' }}</span>
                </button>
              </div>
              <div class="line-scheme-pets">
                <p v-if="!producePoolDams.length" class="muted produce-empty-inline">无可选母</p>
                <button
                  v-for="p in producePoolDams"
                  :key="'pd' + p.id"
                  type="button"
                  class="line-scheme-pet produce-pick"
                  :class="{
                    selected: pickDamId === p.id,
                    disabled: (p.status || '空闲中') === '忙碌中',
                    dim: produceNatureDim(p),
                  }"
                  :disabled="(p.status || '空闲中') === '忙碌中'"
                  @click="togglePickDam(p)"
                >
                  <span
                    class="pet-status-dot"
                    :class="(p.status || '空闲中') === '忙碌中' ? 'busy' : 'idle'"
                    :title="p.status || '空闲中'"
                  />
                  <span class="scheme-pet-id">{{ formatPetId(p.id) }}</span>
                  <span class="species-icon-wrap scheme-pet-icon" aria-hidden="true">
                    <img v-if="petIconUrl(p)" class="species-icon" :src="petIconUrl(p)" alt="" @error="hideBrokenIcon" />
                  </span>
                  <span class="scheme-pet-name">{{ p.speciesName }}</span>
                  <span class="scheme-pet-nature meta-tag nature">{{ p.natureName || '—' }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="dialog-actions">
          <button type="button" class="secondary" :disabled="produceBusy" @click="closeProduce">关闭</button>
          <button
            type="button"
            class="produce-add-btn"
            :class="{ 'is-blocked': !canAddProduce }"
            :disabled="produceBusy"
            :title="produceBlockReason || '加入生产'"
            @click="addProduceRun"
          >
            {{ produceBusy ? '处理中…' : '加入生产' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="pendingStopRunId != null" class="dialog-mask" @click.self="cancelStopRun">
      <div class="dialog-card">
        <p class="dialog-title">确认停止该队生产？</p>
        <div class="dialog-actions">
          <button type="button" class="secondary" @click="cancelStopRun">取消</button>
          <button type="button" class="danger" @click="confirmStopRun">停止</button>
        </div>
      </div>
    </div>

    <div v-if="pendingDeleteId != null" class="dialog-mask" @click.self="cancelDelete">
      <div class="dialog-card">
        <p class="dialog-title">确认删除这条产线？</p>
        <div class="dialog-actions">
          <button type="button" class="secondary" @click="cancelDelete">取消</button>
          <button type="button" class="danger" @click="confirmDelete">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>
