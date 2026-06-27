// 文件正文在 proto 中是 `bytes` 字段（open 响应 info、edit/create 的 content），
// 经 JSON 传输时实际是 base64 字符串，ts_proto 仅把类型标成 Uint8Array。
// 这里集中处理 base64 <-> 字节 <-> 文本 的桥接，UI 层不直接接触 base64。

const BINARY_SNIFF_LENGTH = 8000

// 字节数组编码为 base64（分块避免超长 spread 触发调用栈溢出）。
export const encodeBytesToBase64 = (bytes: Uint8Array): string => {
  let binary = ''
  const chunkSize = 0x8000
  for (let i = 0; i < bytes.length; i += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize))
  }
  return btoa(binary)
}

// base64 解码为字节数组。
export const decodeBase64ToBytes = (base64: string): Uint8Array => {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}

// UTF-8 文本编码为 base64（edit/create 写入用）。
export const encodeTextToBase64 = (text: string): string =>
  encodeBytesToBase64(new TextEncoder().encode(text))

// 字节内容是否疑似二进制：含 NUL 即判定（编辑器对其给“无法打开”遮罩）。
export const isBinaryBytes = (bytes: Uint8Array): boolean => {
  const limit = Math.min(bytes.length, BINARY_SNIFF_LENGTH)
  for (let i = 0; i < limit; i += 1) {
    if (bytes[i] === 0) return true
  }
  return false
}

// open 响应（运行时为 base64 字符串，类型标注是 Uint8Array）解码结果。
export interface DecodedFileContent {
  // 解码后的 UTF-8 文本（二进制时为空串）
  text: string
  // 是否疑似二进制 / 非法 UTF-8
  isBinary: boolean
}

// 解码 open 返回的文件正文：先转字节，判二进制，再尝试 UTF-8 解码。
export const decodeFileContent = (info: Uint8Array | string | undefined | null): DecodedFileContent => {
  if (!info) return { text: '', isBinary: false }

  const bytes = typeof info === 'string' ? decodeBase64ToBytes(info) : info
  if (isBinaryBytes(bytes)) {
    return { text: '', isBinary: true }
  }

  try {
    const text = new TextDecoder('utf-8', { fatal: true }).decode(bytes)
    return { text, isBinary: false }
  } catch {
    // 非法 UTF-8 → 当作二进制，避免乱码写回损坏文件
    return { text: '', isBinary: true }
  }
}
