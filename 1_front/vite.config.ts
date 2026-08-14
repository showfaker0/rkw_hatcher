import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { spawn, type ChildProcess, execFile } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)
const __dirname = path.dirname(fileURLToPath(import.meta.url))
const BACK_DIR = path.resolve(__dirname, '../2_back')
const HEALTH = 'http://127.0.0.1:3070/api/health'
const GO_BIN = 'C:\\Program Files\\Go\\bin'
const PORT = 3070

let backendProc: ChildProcess | null = null

async function isBackendUp(): Promise<boolean> {
  try {
    const res = await fetch(HEALTH)
    return res.ok
  } catch {
    return false
  }
}

function ensureBackend(): void {
  if (backendProc && !backendProc.killed) return
  const env = { ...process.env }
  env.Path = `${GO_BIN};${env.Path || ''}`
  env.PATH = `${GO_BIN};${env.PATH || ''}`
  if (!env.GOPROXY) env.GOPROXY = 'https://goproxy.cn,direct'
  env.RKW_NO_BROWSER = '1'
  const outFd = fs.openSync(path.join(BACK_DIR, 'run_out.txt'), 'w')
  const errFd = fs.openSync(path.join(BACK_DIR, 'run_err.txt'), 'w')
  backendProc = spawn('go', ['run', './cmd/server'], {
    cwd: BACK_DIR,
    env,
    shell: true,
    windowsHide: true,
    stdio: ['ignore', outFd, errFd],
  })
  backendProc.on('exit', () => {
    backendProc = null
  })
}

async function killTree(pid: number): Promise<void> {
  try {
    await execFileAsync('taskkill', ['/PID', String(pid), '/T', '/F'], { windowsHide: true })
  } catch {
    // process may already exit
  }
}

async function killPortListeners(port: number): Promise<void> {
  try {
    const { stdout } = await execFileAsync(
      'powershell.exe',
      [
        '-NoProfile',
        '-Command',
        `Get-NetTCPConnection -LocalPort ${port} -State Listen -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -Unique`,
      ],
      { windowsHide: true },
    )
    const pids = stdout
      .split(/\r?\n/)
      .map((s) => s.trim())
      .filter(Boolean)
      .map(Number)
      .filter((n) => Number.isFinite(n) && n > 0)
    for (const pid of pids) {
      await killTree(pid)
    }
  } catch {
    // ignore
  }
}

async function stopBackend(): Promise<{ ok: boolean; detail: string }> {
  const pid = backendProc?.pid
  backendProc = null
  if (pid) await killTree(pid)
  await killPortListeners(PORT)
  // 等端口释放
  await new Promise((r) => setTimeout(r, 400))
  const stillUp = await isBackendUp()
  return stillUp
    ? { ok: false, detail: '端口仍被占用，关闭失败' }
    : { ok: true, detail: '已关闭' }
}

function json(res: import('http').ServerResponse, code: number, body: unknown) {
  res.statusCode = code
  res.setHeader('Content-Type', 'application/json')
  res.end(JSON.stringify(body))
}

function backendLauncher(): Plugin {
  return {
    name: 'backend-launcher',
    configureServer(server) {
      server.middlewares.use(async (req, res, next) => {
        const url = req.url?.split('?')[0]
        if (url === '/__boot_backend' && (req.method === 'POST' || req.method === 'GET')) {
          if (await isBackendUp()) {
            json(res, 200, { ok: true, already: true })
            return
          }
          try {
            ensureBackend()
            json(res, 200, { ok: true, started: true })
          } catch (e) {
            json(res, 500, { ok: false, error: String(e) })
          }
          return
        }
        if (url === '/__stop_backend' && (req.method === 'POST' || req.method === 'GET')) {
          try {
            const result = await stopBackend()
            json(res, result.ok ? 200 : 500, result)
          } catch (e) {
            json(res, 500, { ok: false, detail: String(e) })
          }
          return
        }
        next()
      })
    },
  }
}

export default defineConfig({
  plugins: [vue(), backendLauncher()],
  server: {
    host: true,
    port: 5173,
  },
})
