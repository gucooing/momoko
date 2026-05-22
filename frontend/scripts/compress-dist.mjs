import { brotliCompressSync, constants, gzipSync } from 'node:zlib'
import { promises as fs } from 'node:fs'
import path from 'node:path'

const distDir = path.resolve('dist')
const compressExtList = new Set([
  '.css',
  '.html',
  '.js',
  '.json',
  '.svg',
  '.txt',
  '.xml',
])

const walkDir = async (dir) => {
  const entries = await fs.readdir(dir, { withFileTypes: true })
  const files = await Promise.all(
    entries.map(async (entry) => {
      const fullPath = path.join(dir, entry.name)
      if (entry.isDirectory()) return walkDir(fullPath)
      return [fullPath]
    }),
  )

  return files.flat()
}

const writeCompressedFile = async (filePath, ext, buffer) => {
  const targetPath = `${filePath}${ext}`
  await fs.writeFile(targetPath, buffer)
}

const compressFile = async (filePath) => {
  const fileExt = path.extname(filePath)
  if (!compressExtList.has(fileExt)) return

  const sourceBuffer = await fs.readFile(filePath)
  if (!sourceBuffer.length) return

  const gzipBuffer = gzipSync(sourceBuffer, { level: 9 })
  if (gzipBuffer.length < sourceBuffer.length) {
    await writeCompressedFile(filePath, '.gz', gzipBuffer)
  }

  const brotliBuffer = brotliCompressSync(sourceBuffer, {
    params: {
      [constants.BROTLI_PARAM_QUALITY]: 11,
    },
  })
  if (brotliBuffer.length < sourceBuffer.length) {
    await writeCompressedFile(filePath, '.br', brotliBuffer)
  }
}

const buildCompressedAssets = async () => {
  const files = await walkDir(distDir)
  await Promise.all(files.map((filePath) => compressFile(filePath)))
}

buildCompressedAssets().catch((error) => {
  console.error('生成预压缩资源失败:', error)
  process.exitCode = 1
})
