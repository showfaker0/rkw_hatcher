import type { MyPet, PetMedalPick, Species } from '../api'

export type BreedSchemePreview = {
  schemeType: 1 | 2 | 3 | 4
  studs: MyPet[]
  dams: MyPet[]
}

/** 与后端 buildBreedSchemes 规则一致；推荐不看忙碌/空闲 */
export function buildBreedSchemes(opts: {
  targetSpeciesId: number
  natureId: number
  medals: PetMedalPick[]
  pets: MyPet[]
  speciesById: Map<number, Species>
}): BreedSchemePreview[] {
  const { targetSpeciesId, natureId, medals, pets, speciesById } = opts
  const target = speciesById.get(targetSpeciesId)
  const targetEggs = new Set(target?.eggGroupIds || [])
  if (!targetEggs.size) return []

  const need = new Map<string, number>()
  for (const m of medals) {
    if (!m.medalId || !m.medalType) continue
    need.set(m.medalType, m.medalId)
  }

  const hasAll = (p: MyPet) => {
    if (!need.size) return true
    const got = new Map((p.medals || []).map((x) => [x.medalType, x.medalId]))
    for (const [t, id] of need) {
      if (got.get(t) !== id) return false
    }
    return true
  }

  const eggOk = (speciesId: number) => {
    const eggs = speciesById.get(speciesId)?.eggGroupIds || []
    return eggs.some((id) => targetEggs.has(id))
  }

  const studOK: MyPet[] = []
  const studBad: MyPet[] = []
  const damOK: MyPet[] = []
  const damBad: MyPet[] = []

  for (const p of pets) {
    if (!hasAll(p)) continue
    const natureOK = p.natureId === natureId
    if (p.gender === '母') {
      if (p.speciesId !== targetSpeciesId) continue
      ;(natureOK ? damOK : damBad).push(p)
      continue
    }
    if (p.gender !== '公') continue
    if (!eggOk(p.speciesId)) continue
    ;(natureOK ? studOK : studBad).push(p)
  }

  if (studOK.length && damOK.length) {
    return [{ schemeType: 1, studs: studOK, dams: damOK }]
  }
  const out: BreedSchemePreview[] = []
  if (studBad.length && damOK.length) {
    out.push({ schemeType: 2, studs: studBad, dams: damOK })
  }
  if (studOK.length && damBad.length) {
    out.push({ schemeType: 3, studs: studOK, dams: damBad })
  }
  if (!out.length && studBad.length && damBad.length) {
    out.push({ schemeType: 4, studs: studBad, dams: damBad })
  }
  return out
}

export function schemeTitle(t: number) {
  if (t === 1) return '强推荐 · 父母性格皆对'
  if (t === 2) return '弱推荐 · 公性格错 / 母性格对'
  if (t === 3) return '弱推荐 · 公性格对 / 母性格错'
  if (t === 4) return '极低效 · 父母性格皆错（仅蛋组匹配）'
  return `方案${t}`
}
