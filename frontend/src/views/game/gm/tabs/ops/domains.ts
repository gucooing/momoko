// 运营管理用到的 7 份配置（游戏服务端的数据文件）：每份一个编辑状态（服务端内容、草稿、自动保存、校验、发布），
// 以及发布前检查出的问题与「查看改动」里的分组名称。运营管理页面提供，日历、详情抽屉与发布条注入后直接改草稿对象。
import { computed, inject, provide, type ComputedRef, type InjectionKey } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GakumasConfigDomain } from '@/api/gakumas'
import type {
  GashaConfig,
  HomeConfig,
  LoginBonusConfig,
  NoticeConfig,
  PvpRateConfig,
  ResearchConfig,
  StoryEventConfig,
} from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { homeEventProblems, researchProblems, storyProblems } from './events/events'
import { catalogKindOf, gashaChecks, rarityInPools } from './gasha/gasha'
import { useGashaLabels } from './gasha/labels'
import { eventProblems, normalProblems, selectProblems, totalProblems } from './login/login'
import { configProblems, noticeProblems } from './notices/notice'
import { gradeProblems, ruleProblems, seasonLabel, seasonProblems } from './pvp/pvp'
import { useCatalogLookup, type CatalogLookup } from './shared/catalogLookup'
import { useConfigDomain, type ConfigDomainState } from './shared/useConfigDomain'

export interface OpsDomain<T = unknown> {
  key: GakumasConfigDomain
  state: ConfigDomainState<T>
  /** 阻止发布的问题（每条带上是哪一项） */
  blockers: ComputedRef<string[]>
  /** 「查看改动」里一组改动的名称（路径前两段） */
  groupLabel: (path: string[]) => string
  /** 改动里引用的 ID 的名称 */
  refLabel?: (collection: string, id: string) => string
}

export interface OpsDomains {
  gasha: OpsDomain<GashaConfig>
  notice: OpsDomain<NoticeConfig>
  pvpRate: OpsDomain<PvpRateConfig>
  home: OpsDomain<HomeConfig>
  research: OpsDomain<ResearchConfig>
  storyEvent: OpsDomain<StoryEventConfig>
  loginBonus: OpsDomain<LoginBonusConfig>
  /** 显示顺序 */
  list: OpsDomain[]
  /** 卡池编辑用：卡的稀有度与目录核对 */
  lookup: CatalogLookup
  rarityOf: (resourceType: number, id: string) => number | undefined
  cardStatus: (resourceType: number, id: string) => 'ok' | 'missing' | 'pending' | 'error'
  /** 读取全部（每份各读一次服务端） */
  loadAll: () => Promise<void>
}

const KEY: InjectionKey<OpsDomains> = Symbol('ops-domains')

const MAX_BLOCKERS = 6

export function provideOpsDomains(): OpsDomains {
  const { t, te } = useI18n()
  const ctx = useGmContext()
  const lookup = useCatalogLookup(ctx)
  const labels = useGashaLabels()
  const p = (scope: string) => (key: string, params?: Record<string, string | number>) => t(`gmConfig.${scope}.problems.${key}`, params || {})
  const limit = (out: string[]) => (out.length > MAX_BLOCKERS ? [...out.slice(0, MAX_BLOCKERS), t('gmConfig.common.andMore', { n: out.length - MAX_BLOCKERS })] : out)
  const unnamed = () => t('game.unnamed')

  // ---- 卡池 ----
  const gasha = useConfigDomain<GashaConfig>('gasha')
  const rarityOf = (resourceType: number, id: string) => {
    const c = gasha.working.value
    return (c && rarityInPools(c, resourceType, id)) ?? lookup.lookup(catalogKindOf(resourceType), id)?.rarity
  }
  const cardStatus = (resourceType: number, id: string) => lookup.status(catalogKindOf(resourceType), id)
  const gashaDomain: OpsDomain<GashaConfig> = {
    key: 'gasha',
    state: gasha,
    blockers: computed(() => {
      const c = gasha.working.value
      if (!c) return []
      const out: string[] = []
      for (const g of c.gashas) for (const x of gashaChecks(c, g, { rarityOf })) if (x.level === 'bad') out.push(`${labels.gasha(g)}：${labels.problem(x, c, g)}`)
      return limit(out)
    }),
    groupLabel: (path) => {
      const [section, id] = path
      const c = gasha.working.value
      const pub = gasha.published.value
      const title = section && te(`gmConfig.gasha.sections.${section}`) ? t(`gmConfig.gasha.sections.${section}`) : section || ''
      if (section === 'gashas' && id) return `${title} · ${labels.gasha(c?.gashas.find((x) => x.id === id) || pub?.gashas.find((x) => x.id === id))}`
      if (section === 'points' && id) return `${title} · ${(c?.points.find((v) => v.id === id) || pub?.points.find((v) => v.id === id))?.name || unnamed()}`
      if (section === 'lotteries' && id) {
        const src = c?.lotteries.some((l) => l.id === id) ? c : pub
        return src ? `${title} · ${labels.lottery(src, id)}` : title
      }
      if (section === 'pools' && id) {
        const src = c?.pools.some((l) => l.id === id) ? c : pub
        const pool = src?.pools.find((x) => x.id === id)
        return src && pool ? `${title} · ${labels.pool(src, pool)}` : title
      }
      return title
    },
    refLabel: (collection, id) => labels.ref([gasha.working.value, gasha.published.value], collection, id),
  }

  // ---- 公告 ----
  const notice = useConfigDomain<NoticeConfig>('notice')
  const noticeTitle = (id: string) => {
    const n = notice.working.value?.notices.find((x) => x.id === id) || notice.published.value?.notices.find((x) => x.id === id)
    return n ? n.title || t('gmConfig.notice.untitled') : ''
  }
  const noticeDomain: OpsDomain<NoticeConfig> = {
    key: 'notice',
    state: notice,
    blockers: computed(() => {
      const c = notice.working.value
      if (!c) return []
      const out = configProblems(c).map((x) => p('notice')(x.key))
      for (const n of c.notices) for (const x of noticeProblems(c, n)) if (x.level === 'bad') out.push(`${n.title || t('gmConfig.notice.untitled')}：${p('notice')(x.key, x.params)}`)
      return limit(out)
    }),
    groupLabel: ([section, id]) => (section === 'notices' && id ? `${t('gmConfig.notice.title')} · ${noticeTitle(id)}` : section === 'pageSize' ? t('gmConfig.notice.pageSize') : ''),
    refLabel: (collection, id) => (collection === 'notices' ? noticeTitle(id) : ''),
  }

  // ---- 竞赛 ----
  const pvpRate = useConfigDomain<PvpRateConfig>('pvpRate')
  const pvpDomain: OpsDomain<PvpRateConfig> = {
    key: 'pvpRate',
    state: pvpRate,
    blockers: computed(() => {
      const c = pvpRate.working.value
      if (!c) return []
      const out = ruleProblems(c).map((x) => `${t('gmConfig.pvp.rules')}：${p('pvp')(x.key)}`)
      for (const s of c.seasons) for (const x of seasonProblems(c, s)) out.push(`${seasonLabel(t, s, ctx.name)}：${p('pvp')(x.key, x.params)}`)
      for (const g of c.grades) for (const x of gradeProblems(g)) out.push(`${t('gmConfig.pvp.gradeN', { n: g.grade })}：${p('pvp')(x.key)}`)
      return limit(out)
    }),
    groupLabel: ([section, id]) => {
      if (section === 'seasons' && id) {
        const s = pvpRate.working.value?.seasons.find((x) => x.pvpRateConfigId === id) || pvpRate.published.value?.seasons.find((x) => x.pvpRateConfigId === id)
        return `${t('gmConfig.pvp.seasons')} · ${s ? seasonLabel(t, s, ctx.name) : id}`
      }
      if (section === 'grades' && id?.startsWith('#')) return `${t('gmConfig.pvp.grades')} · ${t('gmConfig.pvp.gradeN', { n: Number(id.slice(1)) })}`
      if (section === 'winRates' || section === 'loseRates') return t('gmConfig.pvp.rates')
      if (section === 'maxDailyPlayCount' || section === 'historyLimit') return t('gmConfig.pvp.rules')
      return ''
    },
  }

  // ---- 活动日程、研究活动、剧情活动 ----
  const home = useConfigDomain<HomeConfig>('home')
  const eventName = (id: string) =>
    (home.working.value?.events.find((x) => x.eventId === id) || home.published.value?.events.find((x) => x.eventId === id))?.eventName || id
  const homeDomain: OpsDomain<HomeConfig> = {
    key: 'home',
    state: home,
    blockers: computed(() => {
      const c = home.working.value
      if (!c) return []
      const out: string[] = []
      for (const e of c.events) for (const x of homeEventProblems(c, e)) if (x.level === 'bad') out.push(`${e.eventName || unnamed()}：${p('events')(x.key, x.params)}`)
      return limit(out)
    }),
    groupLabel: ([section, id]) => (section === 'events' && id ? `${t('gmConfig.events.title')} · ${eventName(id)}` : ''),
  }

  const research = useConfigDomain<ResearchConfig>('research')
  const researchDomain: OpsDomain<ResearchConfig> = {
    key: 'research',
    state: research,
    blockers: computed(() => {
      const c = research.working.value
      if (!c) return []
      const out: string[] = []
      for (const r of c.researches) for (const x of researchProblems(c, r)) if (x.level === 'bad') out.push(`${t('gmConfig.events.researchName')}：${p('events')(x.key, x.params)}`)
      return limit(out)
    }),
    groupLabel: ([section]) => (section === 'researches' ? t('gmConfig.events.researchName') : ''),
  }

  const storyEvent = useConfigDomain<StoryEventConfig>('storyEvent')
  const storyDomain: OpsDomain<StoryEventConfig> = {
    key: 'storyEvent',
    state: storyEvent,
    blockers: computed(() => {
      const c = storyEvent.working.value
      if (!c) return []
      const out: string[] = []
      for (const s of c.storyEvents) {
        for (const x of storyProblems(c, s, home.working.value?.events)) if (x.level === 'bad') out.push(`${eventName(s.storyEventId)}：${p('events')(x.key, x.params)}`)
      }
      return limit(out)
    }),
    groupLabel: ([section, id]) => (section === 'storyEvents' && id ? `${t('gmConfig.events.title')} · ${eventName(id)}` : ''),
  }

  // ---- 登录奖励 ----
  const loginBonus = useConfigDomain<LoginBonusConfig>('loginBonus')
  const loginDomain: OpsDomain<LoginBonusConfig> = {
    key: 'loginBonus',
    state: loginBonus,
    blockers: computed(() => {
      const c = loginBonus.working.value
      if (!c) return []
      const out = [
        ...normalProblems(c).map((x) => `${t('gmConfig.login.normal')}：${p('login')(x.key, x.params)}`),
        ...totalProblems(c).map((x) => `${t('gmConfig.login.total')}：${p('login')(x.key, x.params)}`),
      ]
      for (const e of c.events) for (const x of eventProblems(c, e)) out.push(`${e.name || unnamed()}：${p('login')(x.key, x.params)}`)
      for (const s of c.selects) for (const x of selectProblems(c, s)) out.push(`${s.name || unnamed()}：${p('login')(x.key, x.params)}`)
      return limit(out)
    }),
    groupLabel: ([section, id]) => {
      if (section === 'normal') return t('gmConfig.login.normal')
      if (section === 'total') return t('gmConfig.login.total')
      if ((section === 'events' || section === 'selects') && id) {
        const all = [...(loginBonus.working.value?.[section] || []), ...(loginBonus.published.value?.[section] || [])]
        return `${t('gmConfig.login.title')} · ${all.find((v) => v.id === id)?.name || id}`
      }
      return ''
    },
  }

  const list = [gashaDomain, homeDomain, researchDomain, storyDomain, pvpDomain, loginDomain, noticeDomain] as OpsDomain[]
  const domains: OpsDomains = {
    gasha: gashaDomain,
    notice: noticeDomain,
    pvpRate: pvpDomain,
    home: homeDomain,
    research: researchDomain,
    storyEvent: storyDomain,
    loginBonus: loginDomain,
    list,
    lookup,
    rarityOf,
    cardStatus,
    loadAll: async () => {
      await Promise.allSettled(list.map((d) => d.state.load()))
    },
  }
  provide(KEY, domains)
  return domains
}

export function useOpsDomains(): OpsDomains {
  const d = inject(KEY)
  if (!d) throw new Error('useOpsDomains() must be used inside 运营管理')
  return d
}
