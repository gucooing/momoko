// MGP 密钥：可以填写已有的（服务端配置里的 mgp.key_id / mgp.secret），也可以在前端生成。
// 生成的 key_id 为 mgp_ + 6 位字母数字，secret 为 32 字节随机数的 base64url（无填充），
// 与游戏服务端 `mgp keygen` 的格式一致。随机数来自 crypto.getRandomValues。

export interface MgpKey {
  keyId: string
  secret: string
}

// 与后端 pkg/mgpauth.ValidateKey 一致：key_id 4–64 位、secret 32–256 位，字符都限 [A-Za-z0-9_-]
const KEY_ID_RE = /^[A-Za-z0-9_-]{4,64}$/
const SECRET_RE = /^[A-Za-z0-9_-]{32,256}$/

/** 校验非空的那几项，返回第一个格式不对的字段；都没问题（或为空）返回空串。 */
export const validateMgpKey = (keyId: string, secret: string): '' | 'keyId' | 'secret' => {
  if (keyId && !KEY_ID_RE.test(keyId)) return 'keyId'
  if (secret && !SECRET_RE.test(secret)) return 'secret'
  return ''
}

// key_id / keyId / "key_id" 后面跟 : 或 =；前面不能紧挨字母或 _（避开 token_secret 之类）
const KEY_ID_FIELD_RE = /(?:^|[^A-Za-z_])key_?id["']?\s*[:=]\s*["']?([A-Za-z0-9_-]+)/i
const SECRET_FIELD_RE = /(?:^|[^A-Za-z_])secret["']?\s*[:=]\s*["']?([A-Za-z0-9_-]+)/i

/**
 * 从一段粘贴的文本里拆出 Key ID 与密钥：支持 `key_id: … / secret: …`（含 = 号、JSON、TOML、YAML 配置片段），
 * 或只有两段的输出（先 Key ID 后密钥）。拆不出完整且格式正确的一组时返回 null。
 */
export const parseMgpKeyText = (text: string): MgpKey | null => {
  const keyId = KEY_ID_FIELD_RE.exec(text)?.[1]
  const secret = SECRET_FIELD_RE.exec(text)?.[1]
  if (keyId && secret && !validateMgpKey(keyId, secret)) return { keyId, secret }
  const parts = text.trim().split(/\s+/)
  if (parts.length === 2 && KEY_ID_RE.test(parts[0]!) && SECRET_RE.test(parts[1]!)) return { keyId: parts[0]!, secret: parts[1]! }
  return null
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
