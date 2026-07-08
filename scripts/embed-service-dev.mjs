import { execSync, execFileSync, spawn } from "child_process"
import { resolve, dirname } from "path"
import { fileURLToPath } from "url"

// Builds the apps/embed-service image, ensures a long-lived `embed-service`
// container is up (potion-base-8M, 256-dim embeddings on :4107), waits for the
// model to load, then runs tender-worker once against it. Mirrors
// scripts/tender-worker-dev.mjs (which used the now-retired ollama service).
const root = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const NAME = "embed-service"

execSync(`docker build -t ${NAME} apps/embed-service`, { cwd: root, stdio: "inherit" })

// Reuse if already running, else start detached. Weights are baked into the
// image, so no persistent volume is needed.
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
    `docker run -d --name ${NAME} -p 4107:4107 ${NAME}`,
    { cwd: root, stdio: "inherit" }
  )
}

// "container up" isn't enough — the service loads the model at startup, so wait
// until /api/embed/health reports ok. ~60s covers a cold container start.
process.stdout.write("waiting for embed-service")
const deadline = Date.now() + 60_000
while (Date.now() < deadline) {
  try {
    const res = await fetch("http://localhost:4107/api/embed/health")
    const { status } = await res.json()
    if (status === "ok") {
      console.log(" ready")
      break
    }
  } catch {}
  if (Date.now() >= deadline) {
    console.error("\nembed-service was not healthy within 60s")
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
