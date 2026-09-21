<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, shallowRef, watch } from 'vue'
import BreedResultPanel from '../components/BreedResultPanel.vue'
import SpeciesDetailDialog from '../components/SpeciesDetailDialog.vue'
import { api, type EggGroup, type Medal, type MyPet, type Nature, type PetDeleteImpact, type Species } from '../api'
import { useBreedQuery } from '../composables/useBreedQuery'
import { useToast } from '../composables/useToast'
import { getEggGroups, getNatures, getSpeciesAll } from '../composables/useDictCache'

const { toast } = useToast()
const detailSpeciesId = ref<number | null>(null)
const list = ref<MyPet[]>([])
const species = shallowRef<Species[]>([])
const natures = shallowRef<Nature[]>([])
const eggGroups = shallowRef<EggGroup[]>([])
const medals = ref<Medal[]>([])
const medalTypes = ref<string[]>([])
const error = ref('')
const editing = ref<number | null>(null)
const formOpen = ref(false)
const saving = ref(false)

const breedOpen = ref(false)
const breedPet = ref<MyPet | null>(null)
const {
  mode: breedMode,
  queried: breedQueried,
  loading: breedLoading,
  error: breedError,
  displayItems: breedItems,
  itemScheme,
  schemeClass,
  schemeLabel,
  leftNatureNames,
  targetNatureNames,
  targetSpecies,
  reset: resetBreedQuery,
  runQuery: runBreedQuery,
} = useBreedQuery()

const form = reactive({
  speciesId: '' as number | '',
  gender: '公' as '公' | '母',
  natureId: '' as number | '',
  medalPick: {} as Record<string, number | ''>,
})

const speciesQuery = ref('')
const natureQuery = ref('')
/** null | 'species' | 'nature' | medalType */
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

const selectedSpecies = computed(() =>
  species.value.find((s) => s.id === form.speciesId) || null,
)

const selectedNature = computed(() =>
  natures.value.find((n) => n.id === form.natureId) || null,
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

function medalLabel(type: string) {
  const id = form.medalPick[type]
  if (!id) return '无'
  return medalsByType.value[type]?.find((m) => m.id === id)?.name || '无'
}

async function load() {
  const [sp, nat, eggs, med, types, pets] = await Promise.all([
    getSpeciesAll(),
    getNatures(),
    getEggGroups(),
    api.get<Medal[]>('/medals'),
    api.get<string[]>('/medal-types'),
    api.get<MyPet[]>('/pets'),
  ])
  species.value = sp
  natures.value = nat
  eggGroups.value = eggs
  medals.value = med
  medalTypes.value = types
  list.value = pets
  for (const t of medalTypes.value) {
    if (form.medalPick[t] === undefined) form.medalPick[t] = ''
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
  form.speciesId = ''
  form.gender = '公'
  form.natureId = ''
  form.medalPick = {}
  for (const t of medalTypes.value) form.medalPick[t] = defaultMedalId(t)
  speciesQuery.value = ''
  natureQuery.value = ''
  openMenu.value = null
  error.value = ''
}

function defaultMedalId(type: string): number | '' {
  const name = type === '体型' ? '大块头' : type === '声音' ? '婉转音' : ''
  if (!name) return ''
  return medalsByType.value[type]?.find((m) => m.name === name)?.id ?? ''
}

function openCreate() {
  resetForm()
  formOpen.value = true
}

function openEdit(p: MyPet) {
  editing.value = p.id
  form.speciesId = p.speciesId
  form.gender = p.gender
  form.natureId = p.natureId
  form.medalPick = {}
  for (const t of medalTypes.value) form.medalPick[t] = ''
  for (const m of p.medals || []) form.medalPick[m.medalType] = m.medalId
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

/** 等同于在最佳配种查询页选中该精灵的种/性别/性格后点查询 */
async function openBreedRecommend(p: MyPet) {
  breedPet.value = p
  breedOpen.value = true
  resetBreedQuery()
  try {
    await runBreedQuery({
      speciesId: p.speciesId,
      gender: p.gender,
      natureId: p.natureId,
      speciesSnap: species.value.find((s) => s.id === p.speciesId) || null,
      natureSnap: natures.value.find((n) => n.id === p.natureId) || null,
    })
  } catch (e) {
    toast((e as Error).message, { type: 'error' })
  }
}

function closeBreedRecommend() {
  breedOpen.value = false
  breedPet.value = null
  resetBreedQuery()
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
  form.speciesId = s.id
  speciesQuery.value = ''
  openMenu.value = null
}

function pickNature(n: Nature) {
  form.natureId = n.id
  natureQuery.value = ''
  openMenu.value = null
}

function pickMedal(type: string, id: number | '') {
  form.medalPick[type] = id
  openMenu.value = null
}

function buildMedals() {
  return medalTypes.value
    .filter((t) => form.medalPick[t])
    .map((t) => ({ medalType: t, medalId: Number(form.medalPick[t]) }))
}

async function save() {
  error.value = ''
  saving.value = true
  try {
    if (!form.speciesId || !form.natureId) throw new Error('请选择精灵与性格')
    const body = {
      speciesId: Number(form.speciesId),
      gender: form.gender,
      natureId: Number(form.natureId),
      medals: buildMedals(),
    }
    if (editing.value) await api.put(`/pets/${editing.value}`, body)
    else await api.post('/pets', body)
    toast(editing.value ? '编辑成功' : '新增成功', { type: 'success' })
    closeForm()
    await load()
  } catch (e) {
    error.value = (e as Error).message
    toast((e as Error).message, { type: 'error' })
  } finally {
    saving.value = false
  }
}

function formatDeleteLineLabel(l: { targetSpeciesName?: string; expectedNatureName?: string; lineName: string }) {
  const sp = l.targetSpeciesName || ''
  const nat = l.expectedNatureName || ''
  if (sp && nat) return `${sp}·${nat}`
  return sp || l.lineName || '未命名产线'
}

async function remove(id: number) {
  try {
    const impact = await api.get<PetDeleteImpact>(`/pets/${id}/delete-impact`)
    if (impact.blocked) {
      const names = (impact.lines || []).map(formatDeleteLineLabel).filter(Boolean)
      const tip = names.length
        ? `${impact.message || '无法删除'}（生产中产线：${names.join('、')}）`
        : (impact.message || '无法删除')
      toast(tip, { type: 'error' })
      return
    }
    pendingDeleteImpact.value = impact
    pendingDeleteId.value = id
  } catch (e) {
    toast((e as Error).message, { type: 'error' })
  }
}

async function confirmDelete() {
  const id = pendingDeleteId.value
  if (!id) return
  pendingDeleteId.value = null
  pendingDeleteImpact.value = null
  try {
    await api.del(`/pets/${id}`)
    toast('已删除', { type: 'info' })
    await load()
  } catch (e) {
    toast((e as Error).message, { type: 'error' })
  }
}

function cancelDelete() {
  pendingDeleteId.value = null
  pendingDeleteImpact.value = null
}

function hideBrokenIcon(ev: Event) {
  const el = ev.target as HTMLImageElement | null
  if (el) el.style.display = 'none'
}

function petIcon(p: MyPet) {
  return speciesById.value.get(p.speciesId)?.iconUrl || ''
}

function petEggs(p: MyPet) {
  return speciesById.value.get(p.speciesId)?.eggGroupNames || []
}

function formatPetId(id: number) {
  return String(id).padStart(3, '0')
}

const listQ = ref('')
const filterEggIds = ref<number[]>([])
/** true=并集，false=交集(默认) — 与图鉴页一致 */
const unionMode = ref(false)
const resetSpinning = ref(false)
const pendingDeleteId = ref<number | null>(null)
const pendingDeleteImpact = ref<PetDeleteImpact | null>(null)
/** none | female | male — 点击循环 */
const genderSort = ref<'none' | 'female' | 'male'>('none')
let resetSpinTimer: ReturnType<typeof setTimeout> | null = null

const speciesById = computed(() => {
  const m = new Map<number, Species>()
  for (const s of species.value) m.set(s.id, s)
  return m
})
const genderSortLabel = computed(() => {
  if (genderSort.value === 'female') return '♀优先'
  if (genderSort.value === 'male') return '♂优先'
  return '无排序'
})

function petMatchesEggs(p: MyPet) {
  if (!filterEggIds.value.length) return true
  const eggs = new Set(speciesById.value.get(p.speciesId)?.eggGroupIds || [])
  if (!eggs.size) return false
  if (unionMode.value) return filterEggIds.value.some((id) => eggs.has(id))
  return filterEggIds.value.every((id) => eggs.has(id))
}

const displayedList = computed(() => {
  const q = listQ.value.trim().toLowerCase()
  let rows = list.value
  if (q) {
    rows = rows.filter((p) => {
      if ((p.speciesName || '').toLowerCase().includes(q)) return true
      if ((p.natureName || '').toLowerCase().includes(q)) return true
      const sp = speciesById.value.get(p.speciesId)
      return (sp?.evoChain || []).some((n) => String(n).toLowerCase().includes(q))
    })
  }
  if (filterEggIds.value.length) {
    rows = rows.filter(petMatchesEggs)
  }
  if (genderSort.value === 'none') return rows
  const preferFemale = genderSort.value === 'female'
  return [...rows].sort((a, b) => {
    const aw = a.gender === '母' ? 0 : 1
    const bw = b.gender === '母' ? 0 : 1
    const d = preferFemale ? aw - bw : bw - aw
    return d || b.id - a.id
  })
})

watch(listQ, () => {
  filterEggIds.value = []
})

function toggleFilterEgg(id: number) {
  const i = filterEggIds.value.indexOf(id)
  if (i >= 0) filterEggIds.value.splice(i, 1)
  else filterEggIds.value.push(id)
}

function toggleEggMode() {
  unionMode.value = !unionMode.value
}

function cycleGenderSort() {
  genderSort.value =
    genderSort.value === 'none' ? 'female' : genderSort.value === 'female' ? 'male' : 'none'
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
  filterEggIds.value = []
  unionMode.value = false
}
</script>

<template>
  <div class="pets-page">
    <section class="species-filter card pets-filter">
      <div class="search-row">
        <div class="search-pill">
          <svg class="search-icon" viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="11" cy="11" r="6.5" fill="none" stroke="currentColor" stroke-width="1.8" />
            <path d="M16.2 16.2L20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
          </svg>
          <input
            v-model="listQ"
            type="text"
            placeholder="输入精灵名字、性格或进化链名称进行搜索"
          />
        </div>
        <button
          type="button"
          class="icon-btn"
          title="重置筛选"
          aria-label="重置筛选"
          @click="resetListSearch"
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
        <div class="pets-filter-start">
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
          <button
            type="button"
            class="secondary gender-sort-btn"
            :class="'is-' + genderSort"
            :title="'性别排序：' + genderSortLabel + '（点击切换）'"
            :aria-label="'性别排序：' + genderSortLabel"
            @click="cycleGenderSort"
          >
            <span v-if="genderSort === 'none'" class="gender-sort-icons" aria-hidden="true">
              <span class="gender-mark male">♂</span>
              <span class="gender-mark female">♀</span>
            </span>
            <span
              v-else
              class="gender-mark"
              :class="genderSort === 'female' ? 'female' : 'male'"
              aria-hidden="true"
            >{{ genderSort === 'female' ? '♀' : '♂' }}</span>
          </button>
        </div>
        <div class="list-count">当前共 {{ displayedList.length }} 条数据</div>
        <div class="filter-actions-end">
          <button type="button" @click="openCreate">新增</button>
        </div>
      </div>
    </section>

    <section class="pets-list-wrap soft-scroll">
      <div v-if="!list.length" class="list-empty muted">暂无精灵，点击「新增」添加</div>
      <div v-else-if="!displayedList.length" class="list-empty muted">无匹配精灵</div>
      <div v-else class="pets-list">
        <article v-for="p in displayedList" :key="p.id" class="pet-item card">
          <span class="pet-id" :title="(p.status || '空闲中')">
            <span
              class="pet-status-dot"
              :class="(p.status || '空闲中') === '忙碌中' ? 'busy' : 'idle'"
              aria-hidden="true"
            ></span>
            {{ formatPetId(p.id) }}
          </span>
          <button
            type="button"
            class="species-icon-wrap species-icon-hit"
            :aria-label="'查看 ' + (p.speciesName || '精灵') + ' 种族值'"
            @click="detailSpeciesId = p.speciesId"
          >
            <img
              v-if="petIcon(p)"
              class="species-icon"
              :src="petIcon(p)"
              alt=""
              referrerpolicy="no-referrer" @error="hideBrokenIcon"
            />
          </button>
          <div class="pet-name">{{ p.speciesName }}</div>
          <span class="gender-mark pet-gender" :class="p.gender === '公' ? 'male' : 'female'">
            {{ p.gender === '公' ? '♂' : '♀' }}
          </span>
          <div class="pet-eggs meta-tags">
            <span
              v-for="g in petEggs(p)"
              :key="g"
              class="meta-tag egg"
            >{{ g }}</span>
            <span v-if="!petEggs(p).length" class="muted">—</span>
          </div>
          <div class="pet-natures meta-tags">
            <span v-if="p.natureName" class="meta-tag nature">{{ p.natureName }}</span>
            <span v-else class="muted">—</span>
          </div>
          <div class="pet-medals meta-tags">
            <span
              v-for="m in (p.medals || [])"
              :key="m.medalType + m.medalId"
              class="meta-tag medal"
              :class="m.medalType === '声音' ? 'medal-voice' : 'medal-body'"
            >{{ m.medalName }}</span>
            <span v-if="!(p.medals || []).length" class="muted">—</span>
          </div>
          <div class="actions">
            <button type="button" class="pet-breed-rec" @click="openBreedRecommend(p)">配种推荐</button>
            <button type="button" class="secondary" @click="openEdit(p)">编辑</button>
            <button type="button" class="danger" @click="remove(p.id)">删除</button>
          </div>
        </article>
      </div>
    </section>

    <div v-if="formOpen" class="dialog-mask" @click.self="closeForm">
      <div class="dialog-card form-dialog pet-form-dialog">
        <p class="dialog-title">{{ editing ? '编辑精灵 #' + formatPetId(editing) : '新增精灵' }}</p>

        <div class="pet-form-grid">
          <div class="pet-field species-field">
            <span>精灵</span>
            <div class="pet-combo" :class="{ open: openMenu === 'species' }">
              <button
                type="button"
                class="pet-combo-trigger"
                :class="{ placeholder: !form.speciesId }"
                @click="toggleMenu('species')"
              >{{ selectedSpecies?.name || '请选择' }}</button>
              <span class="pet-combo-arrow" aria-hidden="true" />
              <div v-if="openMenu === 'species'" class="pet-combo-menu">
                <div class="pet-combo-search" @mousedown.stop>
                  <input
                    ref="speciesSearchRef"
                    v-model="speciesQuery"
                    type="text"
                    placeholder="搜索名称或进化链…"
                    autocomplete="off"
                  />
                </div>
                <div class="pet-combo-list soft-scroll">
                  <button
                    v-for="s in filteredSpecies"
                    :key="s.id"
                    type="button"
                    class="pet-combo-option"
                    :class="{ active: form.speciesId === s.id }"
                    @mousedown.prevent="pickSpecies(s)"
                  >{{ s.name }}</button>
                  <div v-if="!filteredSpecies.length" class="muted pet-combo-empty">无匹配精灵</div>
                </div>
              </div>
            </div>
          </div>

          <div class="pet-field">
            <span>性别</span>
            <div class="gender-pick" role="group" aria-label="性别">
              <button
                type="button"
                class="gender-btn male"
                :class="{ active: form.gender === '公' }"
                @click="form.gender = '公'"
                title="公"
              >♂</button>
              <button
                type="button"
                class="gender-btn female"
                :class="{ active: form.gender === '母' }"
                @click="form.gender = '母'"
                title="母"
              >♀</button>
            </div>
          </div>

          <div class="pet-field">
            <span>性格</span>
            <div class="pet-combo" :class="{ open: openMenu === 'nature' }">
              <button
                type="button"
                class="pet-combo-trigger"
                :class="{ placeholder: !form.natureId }"
                @click="toggleMenu('nature')"
              >
                <template v-if="selectedNature">
                  <span>{{ selectedNature.name }}</span>
                  <span class="nature-stat">（+{{ selectedNature.boost }}/-{{ selectedNature.penalty }}）</span>
                </template>
                <template v-else>请选择</template>
              </button>
              <span class="pet-combo-arrow" aria-hidden="true" />
              <div v-if="openMenu === 'nature'" class="pet-combo-menu">
                <div class="pet-combo-search" @mousedown.stop>
                  <input
                    ref="natureSearchRef"
                    v-model="natureQuery"
                    type="text"
                    placeholder="搜索性格…"
                    autocomplete="off"
                  />
                </div>
                <div class="pet-combo-list soft-scroll">
                  <button
                    v-for="n in filteredNatures"
                    :key="n.id"
                    type="button"
                    class="pet-combo-option"
                    :class="{ active: form.natureId === n.id }"
                    @mousedown.prevent="pickNature(n)"
                  >
                    <span>{{ n.name }}</span>
                    <span class="nature-stat">（+{{ n.boost }}/-{{ n.penalty }}）</span>
                  </button>
                  <div v-if="!filteredNatures.length" class="muted pet-combo-empty">无匹配性格</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="pet-medal-grid">
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
                <button
                  type="button"
                  class="pet-combo-option"
                  :class="{ active: !form.medalPick[t] }"
                  @mousedown.prevent="pickMedal(t, '')"
                >无</button>
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

        <p v-if="error" class="error">{{ error }}</p>
        <div class="dialog-actions">
          <button type="button" class="secondary" :disabled="saving" @click="closeForm">取消</button>
          <button type="button" :disabled="saving" @click="save">
            {{ saving ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="pendingDeleteId != null" class="dialog-mask" @click.self="cancelDelete">
      <div class="dialog-card">
        <p class="dialog-title">确认删除这只精灵？</p>
        <div v-if="pendingDeleteImpact?.lines?.length" class="delete-impact">
          <p class="delete-impact-lead">
            该精灵在以下产线规划中：
            <strong>{{ pendingDeleteImpact.lines.map(formatDeleteLineLabel).join('、') }}</strong>
          </p>
          <p class="delete-impact-desc muted">
            删除后将重算相关产线方案；若无法再配对，对应产线会被删除
            <template v-if="pendingDeleteImpact.lines.some((l) => l.willRemove)">
              （预计删除：{{ pendingDeleteImpact.lines.filter((l) => l.willRemove).map(formatDeleteLineLabel).join('、') }}）
            </template>
            。
          </p>
        </div>
        <div class="dialog-actions">
          <button type="button" class="secondary" @click="cancelDelete">取消</button>
          <button type="button" class="danger" @click="confirmDelete">删除</button>
        </div>
      </div>
    </div>

    <div v-if="breedOpen" class="dialog-mask" @click.self="closeBreedRecommend">
      <div class="dialog-card breed-recommend-dialog">
        <p class="dialog-title">
          配种推荐
          <template v-if="breedPet">
            · {{ breedPet.speciesName || '精灵' }}
            {{ breedPet.gender === '公' ? '♂' : '♀' }}
            <template v-if="breedPet.natureName"> · {{ breedPet.natureName }}</template>
          </template>
          <span v-if="breedQueried && !breedLoading" class="muted breed-result-meta">
            · {{ breedItems.length }} 项
          </span>
        </p>
        <p v-if="breedError" class="error">{{ breedError }}</p>
        <BreedResultPanel
          :loading="breedLoading"
          :items="breedItems"
          :mode="breedMode"
          :item-scheme="itemScheme"
          :scheme-class="schemeClass"
          :scheme-label="schemeLabel"
          :left-nature-names="leftNatureNames"
          :target-nature-names="targetNatureNames"
          :target-species="targetSpecies"
          :allow-detail="false"
        />
        <div class="dialog-actions">
          <button type="button" class="secondary" @click="closeBreedRecommend">关闭</button>
        </div>
      </div>
    </div>

    <SpeciesDetailDialog :species-id="detailSpeciesId" @close="detailSpeciesId = null" />
  </div>
</template>
