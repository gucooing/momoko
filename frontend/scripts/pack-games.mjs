// 把 public/games/ 下所有游戏复制到 dist-games/<游戏>/…
// 用法：node scripts/pack-games.mjs [输出目录]（默认 dist-games）
import { promises as fs } from 'node:fs'
import path from 'node:path'

const srcDir = path.resolve('public/games')
const outDir = path.resolve(process.argv[2] || 'dist-games')

const walk = async (dir) => {
  const entries = await fs.readdir(dir, { withFileTypes: true })
  const nested = await Promise.all(
    entries.map(async (e) => {
      const p = path.join(dir, e.name)
      if (e.isDirectory()) return walk(p)
      return [(await fs.stat(p)).size]
    }),
  )
  return nested.flat()
}

const mb = (n) => `${(n / 1024 / 1024).toFixed(1)} MB`

const games = (await fs.readdir(srcDir, { withFileTypes: true }).catch(() => []))
  .filter((e) => e.isDirectory())
  .map((e) => e.name)
  .sort()

await fs.rm(outDir, { recursive: true, force: true })
await fs.mkdir(outDir, { recursive: true })

let total = 0
for (const game of games) {
  await fs.cp(path.join(srcDir, game), path.join(outDir, game), { recursive: true })
  const sizes = await walk(path.join(outDir, game))
  const bytes = sizes.reduce((a, b) => a + b, 0)
  total += bytes
  console.log(`  ${game}: ${sizes.length} files, ${mb(bytes)}`)
}
console.log(`game assets → ${path.relative(process.cwd(), outDir)}/ (${games.length} games, ${mb(total)})`)
