import { computed, ref, shallowRef } from 'vue'
import { api, type MyPet, type Nature, type Species } from '../api'

export type BreedMode = 'stud' | 'dam'
export type BreedQueryRow = { species: Species; pets: MyPet[] }
export type BreedResultItem =
  | { kind: 'pet'; pet: MyPet; species: Species }
  | { kind: 'species'; species: Species }

/**
 * 最佳配种查询的结果态与展示规则（与 BreedQuery 页同源）。
 * 查询页 / 精灵页弹窗共用，避免两套逻辑分叉。
 */
export function useBreedQuery() {
  const mode = ref<BreedMode | null>(null)
  const results = ref<BreedQueryRow[]>([])
  const queried = ref(false)
  const loading = ref(false)
  const error = ref('')
  /** 仅在查询成功后写入；结果区只读这份快照，不跟表单实时联动 */
  const queriedSpecies = shallowRef<Species | null>(null)
  const queriedNature = shallowRef<Nature | null>(null)
  const queriedNatureId = ref<number | null>(null)
  const queriedSpeciesId = ref<number | null>(null)

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
  function itemScheme(item: BreedResultItem): 1 | 2 | 3 {
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
  function leftNatureNames(item: BreedResultItem): string[] {
    if (item.kind === 'pet') {
      return item.pet.natureName ? [item.pet.natureName] : []
    }
    return item.species.bestPvpNatureNames || []
  }

  /** 右侧目标性格：子代期望性格（仅用查询快照） */
  function targetNatureNames(item: BreedResultItem): string[] {
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
    // 母已是最佳性格之一
    if (damNatureAlreadyBest.value) {
      const out: string[] = []
      if (queriedNature.value?.name) out.push(queriedNature.value.name)
      // 我的精灵：公性格若也在母种最佳内且不同，一并展示（如母平和+公沉默 → 平和、沉默）
      // 图鉴行仍只标查询性格（同性格公来源），不在此追加
      if (item.kind === 'pet' && item.pet.natureName) {
        const best = queriedSpecies.value?.bestPvpNatureNames || []
        if (best.includes(item.pet.natureName) && !out.includes(item.pet.natureName)) {
          out.push(item.pet.natureName)
        }
      }
      return out
    }
    if (item.kind === 'pet') {
      return item.pet.natureName ? [item.pet.natureName] : []
    }
    // 图鉴公：右侧只展示「该公种最佳 ∩ 母种最佳」——公种不具备的性格无法由此繁育
    const studBest = new Set(item.species.bestPvpNatureNames || [])
    return (queriedSpecies.value?.bestPvpNatureNames || []).filter((n) => studBest.has(n))
  }

  /** 右侧目标精灵：子代物种=母；母查→查询时母精灵；公查→推荐的母侧图鉴 */
  function targetSpecies(item: BreedResultItem): Species | null {
    if (mode.value === 'dam') return queriedSpecies.value
    return item.species
  }

  /** 我的精灵在前（同物种优先、再按方案等级），图鉴在后 */
  const displayItems = computed<BreedResultItem[]>(() => {
    const pets: BreedResultItem[] = []
    const specs: BreedResultItem[] = []
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

  function reset() {
    mode.value = null
    results.value = []
    queried.value = false
    loading.value = false
    error.value = ''
    queriedSpecies.value = null
    queriedNature.value = null
    queriedSpeciesId.value = null
    queriedNatureId.value = null
  }

  /** 与查询页点「查询」同一接口、同一写快照时机 */
  async function runQuery(opts: {
    speciesId: number
    gender: '公' | '母'
    natureId: number
    speciesSnap: Species | null
    natureSnap: Nature | null
  }) {
    error.value = ''
    loading.value = true
    queried.value = true
    try {
      const params = new URLSearchParams({
        speciesId: String(opts.speciesId),
        gender: String(opts.gender),
        natureId: String(opts.natureId),
      })
      const data = await api.get<{ mode: BreedMode; results: BreedQueryRow[] }>(
        '/breed/query?' + params.toString(),
      )
      mode.value = data.mode
      results.value = data.results || []
      queriedSpecies.value = opts.speciesSnap
      queriedNature.value = opts.natureSnap
      queriedSpeciesId.value = opts.speciesId
      queriedNatureId.value = opts.natureId
    } catch (e) {
      error.value = (e as Error).message
      results.value = []
      mode.value = null
      queriedSpecies.value = null
      queriedNature.value = null
      queriedSpeciesId.value = null
      queriedNatureId.value = null
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    mode,
    results,
    queried,
    loading,
    error,
    queriedSpecies,
    queriedNature,
    queriedNatureId,
    queriedSpeciesId,
    damNatureAlreadyBest,
    displayItems,
    itemScheme,
    schemeClass,
    schemeLabel,
    leftNatureNames,
    targetNatureNames,
    targetSpecies,
    reset,
    runQuery,
  }
}
