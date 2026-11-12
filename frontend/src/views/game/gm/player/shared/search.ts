// 玩家数据页的名称搜索：关键字先解析成 ID（前端名称表的当前语言译名与游戏原文名、服务端目录的原文名），
// 再把 ID 交给服务端筛选分页；不读出玩家这一类数据的全部条目。
// 列表接口自己支持关键字（偶像卡、支援卡、记忆）时，解析不出 ID 的关键字原样交给服务端模糊匹配（ID 与游戏原文名）。
import { computed, ref, shallowRef, watch } from 'vue'
import { searchGameNames } from '@/games/names'
import type { CatalogEntry, CatalogQuery } from '@/games/mgp'
import { getCurrentLocale } from '@/locales'
import { getRequestErrorMessage } from '@/utils/request'
import type { GmContext } from '../../context'

const DEBOUNCE = 250
const LIMIT = 500

export function useIdSearch(resolve: (q: string) => Promise<string[]>, options: { serverQuery?: boolean } = {}) {
  const keyword = ref('')
  /** 关键字解析出的 ID；没有关键字时为 null */
  const ids = shallowRef<string[] | null>(null)
  /** 解析不出 ID 时交给服务端模糊匹配的关键字（options.serverQuery 时才有） */
  const q = ref('')
  const loading = ref(false)
  const error = ref('')
  let seq = 0
  let timer: ReturnType<typeof setTimeout> | undefined

  watch(keyword, (value) => {
    clearTimeout(timer)
    const text = value.trim()
    const id = ++seq
    error.value = ''
    if (!text) {
      loading.value = false
      q.value = ''
      ids.value = null
      return
    }
    loading.value = true
    timer = setTimeout(async () => {
      try {
        const out = [...new Set(await resolve(text))]
        if (id === seq) {
          q.value = !out.length && options.serverQuery ? text : ''
          ids.value = out
        }
      } catch (e) {
        if (id === seq) {
          error.value = getRequestErrorMessage(e)
          q.value = ''
          ids.value = []
        }
      } finally {
        if (id === seq) loading.value = false
      }
    }, DEBOUNCE)
  })

  const active = computed(() => ids.value !== null)
  /** 有关键字但没有解析出任何 ID、也不交给服务端匹配：不必请求服务端 */
  const none = computed(() => ids.value !== null && ids.value.length === 0 && !q.value)

  return { keyword, ids, q, active, none, loading, error }
}

/** 名称表里命中关键字的 ID（没有目录接口的种类，如偶像卡）。 */
export async function nameIds(ctx: GmContext, kind: string, q: string): Promise<string[]> {
  return [...(await searchGameNames(ctx.game.value, getCurrentLocale(), kind, q))]
}

/** 目录里命中查询的条目（关键字经名称表解析后由服务端筛选）。 */
export async function catalogEntries(ctx: GmContext, kind: string, query: CatalogQuery): Promise<CatalogEntry[]> {
  return (await ctx.catalog(kind, { limit: LIMIT, ...query })).items
}

/** 关键字命中的偶像卡：卡名、皮肤名或角色名。 */
export async function idolCardIds(ctx: GmContext, q: string): Promise<string[]> {
  const [cards, skins, charas] = await Promise.all([
    nameIds(ctx, 'idolCard', q),
    catalogEntries(ctx, 'idolCardSkin', { q }),
    catalogEntries(ctx, 'character', { q }),
  ])
  const out = new Set(cards)
  for (const s of skins) if (s.parentId) out.add(s.parentId)
  const groups = await Promise.all(charas.map((c) => catalogEntries(ctx, 'idolCardSkin', { characterId: c.id })))
  for (const g of groups) for (const s of g) if (s.parentId) out.add(s.parentId)
  return [...out]
}

/** 可作为奖励的目录（货币的 resourceId 为空串，不参与按 ID 筛选）。 */
const REWARD_KINDS = ['item', 'supportCard', 'idolCardSkin', 'music', 'costume', 'costumeHead']

/** 关键字命中的奖励资源 ID（道具、支援卡、偶像卡皮肤、乐曲、服装）。 */
export async function rewardIds(ctx: GmContext, q: string): Promise<string[]> {
  const groups = await Promise.all(REWARD_KINDS.map((kind) => catalogEntries(ctx, kind, { q })))
  return groups.flat().map((e) => e.id)
}

/** 关键字命中的卡池可抽出的卡（支援卡、偶像卡及其皮肤）。 */
export async function gashaCardIds(ctx: GmContext, q: string): Promise<string[]> {
  const [support, idol] = await Promise.all([catalogEntries(ctx, 'supportCard', { q }), idolCardIds(ctx, q)])
  return [...support.map((e) => e.id), ...idol]
}
