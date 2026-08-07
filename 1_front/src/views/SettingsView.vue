<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api'

const health = ref('')
const msg = ref('')
const error = ref('')

onMounted(async () => {
  try {
    await api.get('/health')
    health.value = '后端已连接'
  } catch {
    health.value = '后端未连接（请先启动 2_back）'
  }
})

async function exportSpecies() {
  try {
    await api.download('/export/species', 'species.xlsx')
    msg.value = '图鉴已导出'
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function exportPets() {
  try {
    await api.download('/export/pets', 'pets.xlsx')
    msg.value = '宠物已导出'
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function onImport(kind: 'species' | 'pets', ev: Event) {
  error.value = ''
  msg.value = ''
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const res = await api.upload<{ imported: number; failed: string[] }>(`/import/${kind}`, file)
    msg.value = `导入成功 ${res.imported} 条` + (res.failed?.length ? `；失败 ${res.failed.length}` : '')
    if (res.failed?.length) error.value = res.failed.join('；')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    input.value = ''
  }
}
</script>

<template>
  <section class="card">
    <h2>设置 / 导入导出</h2>
    <p :class="health.includes('已连接') ? 'muted' : 'error'">{{ health }}</p>
    <p class="muted">蛋组 / 性格 / 奖章字典存在本地 SQLite（首次启动自动种子），页面不可配置。</p>
    <div class="row">
      <button @click="exportSpecies">导出图鉴 Excel</button>
      <label class="btn" style="display:inline-flex;align-items:center">
        导入图鉴
        <input type="file" accept=".xlsx" hidden @change="onImport('species', $event)" />
      </label>
    </div>
    <div class="row">
      <button @click="exportPets">导出宠物 Excel</button>
      <label class="btn" style="display:inline-flex;align-items:center">
        导入宠物
        <input type="file" accept=".xlsx" hidden @change="onImport('pets', $event)" />
      </label>
    </div>
    <p v-if="msg" class="muted">{{ msg }}</p>
    <p v-if="error" class="error">{{ error }}</p>
  </section>

  <section class="card">
    <h2>启动说明</h2>
    <ol class="muted">
      <li>可选：复制 <code>2_back/.env.example</code> 为 <code>.env</code>（默认端口 <code>:3070</code>）</li>
      <li>后端：<code>cd 2_back && go run ./cmd/server</code>（自动创建/使用 SQLite）</li>
      <li>前端：<code>cd 1_front && npm run dev</code></li>
      <li>或双击 <code>release/rkw_hatcher.exe</code> 一键使用</li>
      <li>手机同 WiFi 访问电脑局域网 IP 的前端端口</li>
    </ol>
  </section>
</template>
