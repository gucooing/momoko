// MGP 密钥在前端生成：key_id 为 mgp_ + 6 位字母数字，secret 为 32 字节随机数的 base64url（无填充），
// 与游戏服务端 `mgp keygen` 的格式一致。随机数来自 crypto.getRandomValues。

export interface MgpKey {
  keyId: string
  secret: string
}

const ALPHABET = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'
// 62 × 4 = 248：大于等于它的字节丢弃，避免取模偏差。
const UNBIASED_LIMIT = 248

const randomBytes = (n: number) => {
  const buf = new Uint8Array(n)
  crypto.getRandomValues(buf)
  return buf
}

const base64Url = (bytes: Uint8Array) => {
  let bin = ''
  for (const b of bytes) bin += String.fromCharCode(b)
  return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

export const generateMgpKey = (): MgpKey => {
  let id = ''
  while (id.length < 6) {
    for (const b of randomBytes(16)) {
      if (b >= UNBIASED_LIMIT) continue
      id += ALPHABET[b % ALPHABET.length]
      if (id.length === 6) break
    }
  }
  return { keyId: `mgp_${id}`, secret: base64Url(randomBytes(32)) }
}
