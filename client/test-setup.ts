import { spawn, type ChildProcess } from "child_process"
import { resolve } from "path"

const PROJECT_ROOT = resolve(import.meta.dirname, "..")
const SERVER_BIN = resolve(PROJECT_ROOT, "test-server")
const START_TIMEOUT = 30000

let serverProc: ChildProcess | null = null

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms))
}

export async function setup(): Promise<void> {
  await new Promise<void>((resolvePromise, reject) => {
    serverProc = spawn(SERVER_BIN, [], {
      cwd: PROJECT_ROOT,
      stdio: ["ignore", "pipe", "pipe"],
    })

    let started = false
    const onData = (data: Buffer) => {
      const text = data.toString().trim()
      if (!started && /^\d+$/.test(text)) {
        started = true
        resolvePromise()
      }
    }

    serverProc.stdout!.on("data", onData)
    serverProc.stderr!.on("data", () => {})

    serverProc.on("error", (err) => {
      if (!started) reject(err)
    })
    serverProc.on("exit", (code) => {
      if (!started) reject(new Error(`server exited with code ${code} before ready`))
    })

    setTimeout(() => {
      if (!started) reject(new Error("server start timeout"))
    }, START_TIMEOUT)
  })

  await sleep(1000)
}

export async function teardown(): Promise<void> {
  if (serverProc && !serverProc.killed) {
    serverProc.kill("SIGTERM")
    serverProc = null
  }
}
