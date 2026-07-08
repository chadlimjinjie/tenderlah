import { execSync, execFileSync, spawn } from "child_process"
import { resolve, dirname } from "path"
import { fileURLToPath } from "url"

// Builds the apps/ollama image, ensures a long-lived `midtone-ollama` container
// is up (all-minilm embeddings on :11434), waits for the model to be pulled,
// then runs tender-worker once against it. Mirrors scripts/apisix-dev.mjs.
const root = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const NAME = "midtone-ollama"

execSync(`docker build -t ${NAME} apps/ollama`, { cwd: root, stdio: "inherit" })

// Reuse if already running, else start detached with a persistent model volume.
const running = execFileSync("docker", ["ps", "-q", "-f", `name=^${NAME}$`]).toString().trim()
if (running) {
  console.log(`${NAME} already running, reusing it`)
} else {
  // ponytail: `docker run` fails if a *stopped* container with this name exists;
  // clear it first. Ignore the error when there's nothing to remove.
  try {
    execFileSync("docker", ["rm", "-f", NAME], { stdio: "ignore" })
  } catch {}
  execSync(
    `docker run -d --name ${NAME} -p 11434:11434 -v ollama-data:/root/.ollama ${NAME}`,
    { cwd: root, stdio: "inherit" }
  )
}

// "server up" isn't enough — the Dockerfile pulls all-minilm at runtime, so wait
// until it actually shows in /api/tags. ~120s covers a cold first-time pull.
process.stdout.write("waiting for all-minilm")
const deadline = Date.now() + 120_000
while (Date.now() < deadline) {
  try {
    const res = await fetch("http://localhost:11434/api/tags")
    const { models = [] } = await res.json()
    if (models.some((m) => m.name?.startsWith("all-minilm"))) {
      console.log(" ready")
      break
    }
  } catch {}
  if (Date.now() >= deadline) {
    console.error("\nollama did not have all-minilm ready within 120s")
    process.exit(1)
  }
  process.stdout.write(".")
  await new Promise((r) => setTimeout(r, 2000))
}

const worker = spawn("go", ["run", "./cmd/server", ...process.argv.slice(2)], {
  cwd: resolve(root, "apps", "tender-worker"),
  stdio: "inherit",
  shell: process.platform === "win32", // resolve `go` via PATH on Windows
})
worker.on("exit", (code) => {
  console.log(`\n${NAME} left running — stop it with: docker stop ${NAME}`)
  process.exit(code ?? 0)
})
