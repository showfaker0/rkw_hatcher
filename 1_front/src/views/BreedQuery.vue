<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import { api, type MyPet, type Nature, type Species } from '../api'
import { useToast } from '../composables/useToast'
import { getNatures, getSpeciesAll } from '../composables/useDictCache'

type BreedMode = 'stud' | 'dam'
type BreedQueryRow = { species: Species; pets: MyPet[] }
type ResultItem =
  | { kind: 'pet'; pet: MyPet; species: Species }
  | { kind: 'species'; species: Species }

const { toast } = useToast()
const natures = shallowRef<Nature[]>([])
const species = shallowRef<Species[]>([])
const speciesId = ref<number | ''>('')
const gender = ref<'' | '公' | '母'>('')
const natureId = ref<number | ''>('')
const mode = ref<BreedMode | null>(null)
const results = ref<BreedQueryRow[]>([])
const queried = ref(false)
const loading = ref(false)
const error = ref('')
/** 仅在点击「查询」成功后写入；结果区只读这份快照，不跟表单实时联动 */
const queriedSpecies = shallowRef<Species | null>(null)
const queriedNature = shallowRef<Nature | null>(null)
const queriedNatureId = ref<number | null>(null)
const queriedSpeciesId = ref<number | null>(null)

const speciesQuery = ref('')
const natureQuery = ref('')
const openMenu = ref<string | null>(null)
const speciesSearchRef = ref<HTMLInputElement | null>(null)
const natureSearchRef = ref<HTMLInputElement | null>(null)

const selectedSpecies = computed(() =>
  species.value.find((s) => s.id === speciesId.value) || null,
)
const selectedNature = computed(() =>
  natures.value.find((n) => n.id === natureId.value) || null,
)

const filteredSpecies = computed(() => {
  const q = speciesQuery.value.trim().toLowerCase()
  if (!q) return species.value
  return species.value.filter((s) => {
    if (s.name.toLowerCase().includes(q)) return true
    if (s.no != null && String(s.no).includes(q)) return true
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

const canQuery = computed(
  () => !!speciesId.value && !!gender.value && !!natureId.value,
)

/** 母查且查询时性格已是该物种最佳性格之一 → 专推同性格父母培育 */
const damNatureAlreadyBest = computed(() => {
  if (mode.value !== 'dam' || queriedNatureId.value == null || !queriedSpecies.value) return false
  return (queriedSpecies.value.bestPvpNatureIds || []).includes(queriedNatureId.value)
})

/** 方案 1/2/3（对齐产线：性格对=落在目标性格上）
 * 子代随母；目标性格 = 母物种最佳 PVP（母已是最佳之一时，目标就是查询性格）
 * 方案1：公对 + 母对；方案2：公错 + 母对；方案3：公对 + 母错
 * 母查询性格不在最佳里时，母恒为「错」→ 只可能方案3，绝无方案1
 */
function itemScheme(item: ResultItem): 1 | 2 | 3 {
  const formN = queriedNatureId.value
  if (formN == null) return 3

  if (mode.value === 'stud') {
    // 公查：查询公性格即目标性格；图鉴/母宠侧看母是否已具备该性格
    if (item.kind === 'species') {
      return (item.species.bestPvpNatureIds || []).includes(formN) ? 1 : 3
    }
    return item.pet.natureId === formN ? 1 : 3
  }

  // 母查
  const motherOK = damNatureAlreadyBest.value
  if (item.kind === 'species') {
    // 图鉴公：视为「性格可对」的种公来源；母不对则只能方案3
    return motherOK ? 1 : 3
  }
  const studOK = motherOK
    ? item.pet.natureId === formN
    : (queriedSpecies.value?.bestPvpNatureIds || []).includes(item.pet.natureId)
  if (motherOK && studOK) return 1
  if (motherOK && !studOK) return 2
  if (!motherOK && studOK) return 3
  return 3
}

function schemeClass(s: 1 | 2 | 3) {
  return s === 1 ? 'strong' : 'weak'
}

function schemeLabel(s: 1 | 2 | 3) {
  if (s === 1) return '推荐方案1 · 公对 / 母对（强推荐）'
  if (s === 2) return '推荐方案2 · 公错 / 母对'
  return '推荐方案3 · 公对 / 母错'
}

/** 左侧展示该推荐行自己的性格：个体=宠物性格；图鉴=该物种最佳PVP */
function leftNatureNames(item: ResultItem): string[] {
  if (item.kind === 'pet') {
    return item.pet.natureName ? [item.pet.natureName] : []
  }
  return item.species.bestPvpNatureNames || []
}

/** 右侧目标性格：子代期望性格（仅用查询快照） */
function targetNatureNames(item: ResultItem): string[] {
  if (mode.value === 'stud') {
    // 公查：子代种=母；目标性格 ⊆ 母种最佳PVP
    // 公性格（查询）必在最佳内才会出结果；若母个体性格也在最佳内，一并展示
    const bestNames = item.species.bestPvpNatureNames || []
    const bestSet = new Set(bestNames)
    const out: string[] = []
    if (queriedNature.value?.name && bestSet.has(queriedNature.value.name)) {
      out.push(queriedNature.value.name)
    } else if (queriedNature.value?.name) {
      out.push(queriedNature.value.name)
    }
    if (item.kind === 'pet' && item.pet.natureName && bestSet.has(item.pet.natureName)) {
      if (!out.includes(item.pet.natureName)) out.push(item.pet.natureName)
    }
    return out
  }
  // 母已是最佳性格：目标就是该性格（父母同性格培育）
  if (damNatureAlreadyBest.value) {
    return queriedNature.value ? [queriedNature.value.name] : []
  }
  if (item.kind === 'pet') {
    return item.pet.natureName ? [item.pet.natureName] : []
  }
  return queriedSpecies.value?.bestPvpNatureNames || []
}

/** 右侧目标精灵：子代物种=母；母查→查询时母精灵；公查→推荐的母侧图鉴 */
function targetSpecies(item: ResultItem): Species | null {
  if (mode.value === 'dam') return queriedSpecies.value
  return item.species
}

/** 我的精灵在前（同物种优先、再按方案等级），图鉴在后 */
const displayItems = computed<ResultItem[]>(() => {
  const pets: ResultItem[] = []
  const specs: ResultItem[] = []
  const sid = queriedSpeciesId.value
  for (const r of results.value) {
    for (const p of r.pets || []) {
      pets.push({ kind: 'pet', pet: p, species: r.species })
    }
    specs.push({ kind: 'species', species: r.species })
  }
  pets.sort((a, b) => {
    const aSame = a.kind === 'pet' && sid != null && a.species.id === sid ? 0 : 1
    const bSame = b.kind === 'pet' && sid != null && b.species.id === sid ? 0 : 1
    if (aSame !== bSame) return aSame - bSame
    const d = itemScheme(a) - itemScheme(b)
    if (d) return d
    return (a.kind === 'pet' ? a.pet.id : 0) - (b.kind === 'pet' ? b.pet.id : 0)
  })
  specs.sort((a, b) => {
    const aSame = a.kind === 'species' && sid != null && a.species.id === sid ? 0 : 1
    const bSame = b.kind === 'species' && sid != null && b.species.id === sid ? 0 : 1
    return aSame - bSame
  })
  return [...pets, ...specs]
})

onMounted(async () => {
  try {
    const [sp, nat] = await Promise.all([getSpeciesAll(), getNatures()])
    species.value = sp
    natures.value = nat
  } catch (e) {
    toast((e as Error).message || '初始化失败', { type: 'error' })
  }
  document.addEventListener('mousedown', onDocDown)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', onDocDown)
})

function onDocDown(e: MouseEvent) {
  const t = e.target as HTMLElement | null
  if (!t?.closest('.pet-combo')) openMenu.value = null
}

async function toggleMenu(key: string) {
  if (openMenu.value === key) {
    openMenu.value = null
    return
  }
  openMenu.value = key
  await nextTick()
  if (key === 'species') speciesSearchRef.value?.focus()
  if (key === 'nature') natureSearchRef.value?.focus()
}

function pickSpecies(s: Species) {
  speciesId.value = s.id
  speciesQuery.value = ''
  openMenu.value = null
}

function pickNature(n: Nature) {
  natureId.value = n.id
  natureQuery.value = ''
  openMenu.value = null
}

function formatPetId(id: number) {
  return String(id).padStart(3, '0')
}

function formatNo(no?: number | null) {
  if (no == null || !Number.isFinite(no) || no <= 0) return '—'
  return String(Math.trunc(no)).padStart(3, '0')
}

function hideBrokenIcon(e: Event) {
  const el = e.target as HTMLImageElement
  el.style.display = 'none'
}

async function query() {
  error.value = ''
  if (!canQuery.value) {
    toast('请完整选择精灵、性别、性格', { type: 'warning' })
    return
  }
  loading.value = true
  queried.value = true
  try {
    const params = new URLSearchParams({
      speciesId: String(speciesId.value),
      gender: String(gender.value),
      natureId: String(natureId.value),
    })
    const data = await api.get<{ mode: BreedMode; results: BreedQueryRow[] }>(
      '/breed/query?' + params.toString(),
    )
    mode.value = data.mode
    results.value = data.results || []
    queriedSpecies.value = selectedSpecies.value
    queriedNature.value = selectedNature.value
    queriedSpeciesId.value = Number(speciesId.value)
    queriedNatureId.value = Number(natureId.value)
  } catch (e) {
    error.value = (e as Error).message
    results.value = []
    mode.value = null
    queriedSpecies.value = null
    queriedNature.value = null
    queriedSpeciesId.value = null
    queriedNatureId.value = null
    toast((e as Error).message, { type: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="breed-page">
    <section class="card breed-query-card">
      <h2>最佳配种查询</h2>

      <div class="breed-query-form">
        <div class="pet-field">
          <span>精灵（图鉴）</span>
          <div class="pet-combo" :class="{ open: openMenu === 'species' }">
            <button
              type="button"
              class="pet-combo-trigger"
              :class="{ placeholder: !speciesId }"
              @click="toggleMenu('species')"
            >{{ selectedSpecies?.name || '请选择' }}</button>
            <span class="pet-combo-arrow" aria-hidden="true" />
            <div v-if="openMenu === 'species'" class="pet-combo-menu">
              <div class="pet-combo-search" @mousedown.stop>
                <input
                  ref="speciesSearchRef"
                  v-model="speciesQuery"
                  type="text"
                  placeholder="搜索名称、图鉴或进化链…"
                  autocomplete="off"
                />
              </div>
              <div class="pet-combo-list soft-scroll">
                <button
                  v-for="s in filteredSpecies"
                  :key="s.id"
                  type="button"
                  class="pet-combo-option"
                  :class="{ active: speciesId === s.id }"
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
              :class="{ active: gender === '公' }"
              title="公"
              @click="gender = '公'"
            >♂</button>
            <button
              type="button"
              class="gender-btn female"
              :class="{ active: gender === '母' }"
              title="母"
              @click="gender = '母'"
            >♀</button>
          </div>
        </div>

        <div class="pet-field">
          <span>性格</span>
          <div class="pet-combo" :class="{ open: openMenu === 'nature' }">
            <button
              type="button"
              class="pet-combo-trigger"
              :class="{ placeholder: !natureId }"
              @click="toggleMenu('nature')"
            >{{ selectedNature ? `${selectedNature.name} (+${selectedNature.boost}/-${selectedNature.penalty})` : '请选择' }}</button>
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
                  :class="{ active: natureId === n.id }"
                  @mousedown.prevent="pickNature(n)"
                >{{ n.name }} (+{{ n.boost }}/-{{ n.penalty }})</button>
                <div v-if="!filteredNatures.length" class="muted pet-combo-empty">无匹配性格</div>
              </div>
            </div>
          </div>
        </div>

        <div class="breed-query-actions">
          <button type="button" :disabled="loading" @click="query">
            {{ loading ? '查询中…' : '查询' }}
          </button>
        </div>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section v-if="queried" class="card breed-result-card">
      <h2>
        推荐结果
        <span v-if="!loading" class="muted breed-result-meta">· {{ displayItems.length }} 项</span>
      </h2>
      <div class="breed-result-wrap soft-scroll">
        <div v-if="loading" class="list-empty muted">查询中…</div>
        <div v-else-if="!displayItems.length" class="list-empty muted">暂无匹配推荐</div>
        <div v-else class="breed-result-list">
          <article
            v-for="(item, idx) in displayItems"
            :key="item.kind === 'pet' ? 'p-' + item.pet.id : 's-' + item.species.id + '-' + idx"
            class="breed-result-item"
            :class="{ 'breed-result-species': item.kind === 'species' }"
          >
            <div class="breed-result-main card">
              <div class="breed-result-id">
                <template v-if="item.kind === 'pet'">
                  <span
                    class="pet-status-dot"
                    :class="(item.pet.status || '空闲中') === '忙碌中' ? 'busy' : 'idle'"
                    :title="item.pet.status || '空闲中'"
                  />
                  <span class="scheme-pet-id">{{ formatPetId(item.pet.id) }}</span>
                </template>
                <span v-else class="species-no">{{ formatNo(item.species.no) }}</span>
              </div>

              <span class="species-icon-wrap scheme-pet-icon" aria-hidden="true">
                <img
                  v-if="item.species.iconUrl"
                  class="species-icon"
                  :src="item.species.iconUrl"
                  alt=""
                  @error="hideBrokenIcon"
                />
              </span>

              <div class="breed-result-name">
                {{ item.kind === 'pet' ? (item.pet.speciesName || item.species.name) : item.species.name }}
              </div>

              <span
                class="gender-mark breed-result-gender"
                :class="(item.kind === 'pet' ? item.pet.gender : (mode === 'stud' ? '母' : '公')) === '公' ? 'male' : 'female'"
              >{{ (item.kind === 'pet' ? item.pet.gender : (mode === 'stud' ? '母' : '公')) === '公' ? '♂' : '♀' }}</span>

              <div class="meta-tags breed-result-eggs">
                <span
                  v-for="g in (item.species.eggGroupNames || [])"
                  :key="g"
                  class="meta-tag egg"
                >{{ g }}</span>
                <span v-if="!(item.species.eggGroupNames || []).length" class="muted">—</span>
              </div>

              <div class="meta-tags breed-result-nature">
                <template v-if="leftNatureNames(item).length">
                  <span
                    v-for="n in leftNatureNames(item)"
                    :key="'need-' + n"
                    class="meta-tag nature"
                  >{{ n }}</span>
                </template>
                <span v-else class="muted">—</span>
              </div>

              <span
                class="line-scheme-strength"
                :class="schemeClass(itemScheme(item))"
                :title="schemeLabel(itemScheme(item))"
                :aria-label="schemeLabel(itemScheme(item))"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <path fill="currentColor" d="M12 3.2l2.4 4.86 5.36.78-3.88 3.78.92 5.34L12 15.7l-4.8 2.52.92-5.34L4.24 8.84l5.36-.78L12 3.2z" />
                </svg>
              </span>
            </div>

            <span class="breed-result-arrow meta-arrow" aria-hidden="true">→</span>

            <div class="breed-result-goal card">
              <span class="species-icon-wrap scheme-pet-icon" aria-hidden="true">
                <img
                  v-if="targetSpecies(item)?.iconUrl"
                  class="species-icon"
                  :src="targetSpecies(item)?.iconUrl || ''"
                  alt=""
                  @error="hideBrokenIcon"
                />
              </span>
              <div class="meta-tags breed-result-target">
                <span
                  v-for="n in targetNatureNames(item)"
                  :key="'t-' + n"
                  class="meta-tag nature"
                >{{ n }}</span>
                <span v-if="!targetNatureNames(item).length" class="muted">—</span>
              </div>
            </div>
          </article>
        </div>
      </div>
    </section>
  </div>
</template>
