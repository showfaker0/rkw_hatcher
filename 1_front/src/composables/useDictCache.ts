import { shallowRef } from 'vue'
import { api, type EggGroup, type Nature, type Species } from '../api'

const eggGroups = shallowRef<EggGroup[] | null>(null)
const natures = shallowRef<Nature[] | null>(null)
const speciesAll = shallowRef<Species[] | null>(null)

let eggPromise: Promise<EggGroup[]> | null = null
let naturePromise: Promise<Nature[]> | null = null
let speciesPromise: Promise<Species[]> | null = null

export async function getEggGroups(force = false): Promise<EggGroup[]> {
  if (!force && eggGroups.value) return eggGroups.value
  if (!eggPromise) {
    eggPromise = api
      .get<EggGroup[]>('/egg-groups')
      .then((d) => {
        eggGroups.value = d
        return d
      })
      .finally(() => {
        eggPromise = null
      })
  }
  return eggPromise
}

export async function getNatures(force = false): Promise<Nature[]> {
  if (!force && natures.value) return natures.value
  if (!naturePromise) {
    naturePromise = api
      .get<Nature[]>('/natures')
      .then((d) => {
        natures.value = d
        return d
      })
      .finally(() => {
        naturePromise = null
      })
  }
  return naturePromise
}

/** 无筛选条件的全量图鉴（供产线/精灵页下拉复用） */
export async function getSpeciesAll(force = false): Promise<Species[]> {
  if (!force && speciesAll.value) return speciesAll.value
  if (!speciesPromise) {
    speciesPromise = api
      .get<Species[]>('/species')
      .then((d) => {
        speciesAll.value = d
        return d
      })
      .finally(() => {
        speciesPromise = null
      })
  }
  return speciesPromise
}

export function invalidateSpeciesCache() {
  speciesAll.value = null
  speciesPromise = null
}

export function setSpeciesAllCache(data: Species[]) {
  speciesAll.value = data
  speciesPromise = null
}

export function invalidateDictCache() {
  eggGroups.value = null
  natures.value = null
  eggPromise = null
  naturePromise = null
  invalidateSpeciesCache()
}
