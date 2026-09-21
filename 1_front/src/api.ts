const API_BASE = import.meta.env.VITE_API_BASE || 'http://127.0.0.1:3070/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, init)
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error((data as { error?: string }).error || res.statusText)
  return data as T
}

export const api = {
  get: <T>(path: string, init?: RequestInit) => request<T>(path, init),
  post: <T>(path: string, body: unknown, init?: RequestInit) =>
    request<T>(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      ...init,
    }),
  put: <T>(path: string, body: unknown, init?: RequestInit) =>
    request<T>(path, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      ...init,
    }),
  del: (path: string, init?: RequestInit) => request<void>(path, { method: 'DELETE', ...init }),
  upload: <T>(path: string, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<T>(path, { method: 'POST', body: fd })
  },
  download: async (path: string, filename: string) => {
    const res = await fetch(`${API_BASE}${path}`)
    if (!res.ok) throw new Error('下载失败')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  },
}

export type EggGroup = { id: number; name: string }
export type Nature = { id: number; name: string; boost: string; penalty: string }
export type Medal = { id: number; type: string; name: string }
export type Species = {
  id: number
  no?: number | null
  name: string
  iconUrl?: string | null
  evoChain: string[]
  bestPvpNatureIds: number[]
  bestPvpNatureNames?: string[]
  notes?: string | null
  eggGroupIds: number[]
  eggGroupNames?: string[]
}
export type PetMedalPick = { medalType: string; medalId: number; medalName?: string }
export type MyPet = {
  id: number
  speciesId: number
  speciesName?: string
  gender: '公' | '母'
  natureId: number
  natureName?: string
  /** 空闲中 | 忙碌中 */
  status?: string
  medals: PetMedalPick[]
}
export type BreedingScheme = {
  schemeType: number
  studs: MyPet[]
  dams: MyPet[]
}
export type BreedingLineRun = {
  id: number
  lineId: number
  studPetId: number
  damPetId: number
  stud?: MyPet
  dam?: MyPet
  createdAt?: string
}
export type BreedingLine = {
  id: number
  name: string
  targetSpeciesId: number
  targetSpeciesName?: string
  targetSpeciesIcon?: string
  expectedNatureId: number
  expectedNatureName?: string
  status: string
  stepsNote?: string | null
  medals?: PetMedalPick[]
  schemes?: BreedingScheme[]
  runs?: BreedingLineRun[]
  maxSlots?: number
}
export type BreedQueryResult = {
  species: Species
  pets: MyPet[]
}
export type PetDeleteLineImpact = {
  lineId: number
  lineName: string
  targetSpeciesName?: string
  expectedNatureName?: string
  willRemove: boolean
}
export type PetDeleteImpact = {
  blocked: boolean
  message?: string
  inProduction?: boolean
  lines: PetDeleteLineImpact[]
}
export type SpeciesDeleteImpact = {
  name: string
  petCount: number
  lineCount: number
  lineNames: string[]
}
export type TypeTag = { key: string; label: string }
export type StatRow = { key: string; label: string; value: number }
export type TypeRelations = {
  counter: TypeTag[]
  counteredBy: TypeTag[]
  resist: TypeTag[]
  resistedBy: TypeTag[]
}
export type SpeciesForm = {
  key: string
  name: string
  selectorLabel: string
  catalogNo?: number | null
  iconUrl: string
  types: TypeTag[]
  stage: number
  stageLabel: string
  isFinal: boolean
  isLeader: boolean
  stats: StatRow[]
  statMax: number
  typeRelations: TypeRelations
}
export type SpeciesDetail = {
  name: string
  defaultFormKey: string
  forms: SpeciesForm[]
}
