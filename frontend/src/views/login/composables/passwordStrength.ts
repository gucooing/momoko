/** 新密码最少位数（与后端 biz.MinPasswordLength 一致）。 */
export const MIN_PASSWORD_LENGTH = 8

/** 密码强度打分：0–4，与原型规则一致（长度 ≥8、大小写、数字、符号；不足 6 位最多 1 分）。 */
export const scorePassword = (pw: string) => {
  const rules = {
    length: pw.length >= 8,
    case: /[a-z]/.test(pw) && /[A-Z]/.test(pw),
    digit: /\d/.test(pw),
    symbol: /[^A-Za-z0-9]/.test(pw),
  }
  let score = Object.values(rules).filter(Boolean).length
  if (pw.length < 6) score = Math.min(score, 1)
  return { score, rules }
}
