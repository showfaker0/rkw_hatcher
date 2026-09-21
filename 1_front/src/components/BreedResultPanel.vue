<script setup lang="ts">
import type { BreedMode, BreedResultItem } from '../composables/useBreedQuery'

const props = withDefaults(defineProps<{
  loading: boolean
  items: BreedResultItem[]
  mode: BreedMode | null
  itemScheme: (item: BreedResultItem) => 1 | 2 | 3
  schemeClass: (s: 1 | 2 | 3) => string
  schemeLabel: (s: 1 | 2 | 3) => string
  leftNatureNames: (item: BreedResultItem) => string[]
  targetNatureNames: (item: BreedResultItem) => string[]
  targetSpecies: (item: BreedResultItem) => { id?: number; iconUrl?: string | null } | null
  /** 弹窗内为 false，避免叠开详情 */
  allowDetail?: boolean
}>(), { allowDetail: true })

const emit = defineEmits<{
  openSpecies: [id: number]
}>()

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

function openSpecies(id?: number | null) {
  if (props.allowDetail && id) emit('openSpecies', id)
}
</script>

<template>
  <div class="breed-result-wrap soft-scroll">
    <div v-if="loading" class="list-empty muted">查询中…</div>
    <div v-else-if="!items.length" class="list-empty muted">暂无匹配推荐</div>
    <div v-else class="breed-result-list">
      <article
        v-for="(item, idx) in items"
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

          <span
            class="species-icon-wrap scheme-pet-icon"
            :class="{ 'species-icon-hit': allowDetail }"
            :role="allowDetail ? 'button' : undefined"
            :aria-label="allowDetail ? '查看 ' + item.species.name + ' 种族值' : undefined"
            @click="openSpecies(item.species.id)"
          >
            <img
              v-if="item.species.iconUrl"
              class="species-icon"
              :src="item.species.iconUrl"
              alt=""
              referrerpolicy="no-referrer" @error="hideBrokenIcon"
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
          <span
            class="species-icon-wrap scheme-pet-icon"
            :class="{ 'species-icon-hit': allowDetail }"
            :role="allowDetail ? 'button' : undefined"
            :aria-label="allowDetail ? '查看目标精灵种族值' : undefined"
            @click="openSpecies(targetSpecies(item)?.id)"
          >
            <img
              v-if="targetSpecies(item)?.iconUrl"
              class="species-icon"
              :src="targetSpecies(item)?.iconUrl || ''"
              alt=""
              referrerpolicy="no-referrer" @error="hideBrokenIcon"
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
</template>
