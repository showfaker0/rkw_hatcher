<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import BreedResultPanel from '../components/BreedResultPanel.vue'
import SpeciesDetailDialog from '../components/SpeciesDetailDialog.vue'
import { type Nature, type Species } from '../api'
import { useBreedQuery } from '../composables/useBreedQuery'
import { useToast } from '../composables/useToast'
import { getNatures, getSpeciesAll } from '../composables/useDictCache'

const { toast } = useToast()
const natures = shallowRef<Nature[]>([])
const species = shallowRef<Species[]>([])
const speciesId = ref<number | ''>('')
const detailSpeciesId = ref<number | null>(null)
const gender = ref<'' | '公' | '母'>('')
const natureId = ref<number | ''>('')

const {
  mode,
  queried,
  loading,
  error,
  displayItems,
  itemScheme,
  schemeClass,
  schemeLabel,
  leftNatureNames,
  targetNatureNames,
  targetSpecies,
  runQuery,
} = useBreedQuery()

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

async function query() {
  if (!canQuery.value) {
    toast('请完整选择精灵、性别、性格', { type: 'warning' })
    return
  }
  try {
    await runQuery({
      speciesId: Number(speciesId.value),
      gender: gender.value as '公' | '母',
      natureId: Number(natureId.value),
      speciesSnap: selectedSpecies.value,
      natureSnap: selectedNature.value,
    })
  } catch (e) {
    toast((e as Error).message, { type: 'error' })
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
      <BreedResultPanel
        :loading="loading"
        :items="displayItems"
        :mode="mode"
        :item-scheme="itemScheme"
        :scheme-class="schemeClass"
        :scheme-label="schemeLabel"
        :left-nature-names="leftNatureNames"
        :target-nature-names="targetNatureNames"
        :target-species="targetSpecies"
        @open-species="detailSpeciesId = $event"
      />
    </section>
    <SpeciesDetailDialog :species-id="detailSpeciesId" @close="detailSpeciesId = null" />
  </div>
</template>
