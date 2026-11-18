// 运营管理里每一项配置的表单（GmSchemaForm 渲染）：活动日程、研究活动、剧情活动、登录奖励、竞赛赛季与规则。
// 每个字段都在这里，都能改：引用游戏数据的用名称选择（x-picker 为目录种类），图片从图里选，时间按服务器时区，千分率按百分比；
// 标题与说明在 gmConfig.schema.*。图片来源（imageSources 的键）由打开的窗口按配置里用过的图给出。
import type { HomeEvent } from '@/games/gakumas/types'
import type { JSONSchema } from '@/games/mgp'
import { EVENT_CONTENT_FIELDS, EVENT_TYPES } from './events/events'

type T = (key: string, params?: Record<string, unknown>) => string
type TE = (key: string) => boolean

/** 字段：标题、说明来自 gmConfig.schema.<path>.title / .desc */
function fieldOf(t: T, te: TE) {
  return (path: string, schema: JSONSchema, group?: string): JSONSchema => {
    const base = `gmConfig.schema.${path}`
    return {
      ...schema,
      title: t(`${base}.title`),
      description: te(`${base}.desc`) ? t(`${base}.desc`) : undefined,
      ...(group ? { 'x-group': t(`gmConfig.schema.groups.${group}`) } : {}),
    }
  }
}

const obj = (properties: Record<string, JSONSchema>, extra: Partial<JSONSchema> = {}): JSONSchema => ({ type: 'object', properties, ...extra })
const str = (extra: Partial<JSONSchema> = {}): JSONSchema => ({ type: 'string', ...extra })
const int = (extra: Partial<JSONSchema> = {}): JSONSchema => ({ type: 'integer', ...extra })
const bool: JSONSchema = { type: 'boolean' }
const time: JSONSchema = { type: 'integer', format: 'unix-ms' }
const permil: JSONSchema = { type: 'integer', 'x-unit': 'permil' }
const reward: JSONSchema = { type: 'object', 'x-picker': 'reward' }
const rewards: JSONSchema = { type: 'array', 'x-picker': 'reward' }
const ref = (kind: string): JSONSchema => ({ type: 'string', 'x-picker': kind })
const refs = (kind: string, numeric = false): JSONSchema => ({ type: 'array', items: { type: numeric ? 'integer' : 'string', 'x-picker': kind } })
const image = (source: string): JSONSchema => ({ type: 'string', 'x-picker': 'image', 'x-images': source })
const list = (items: JSONSchema, itemTitle: string): JSONSchema => ({ type: 'array', items, 'x-item-title': itemTitle })
const enumOf = (values: number[], labels: Record<string, string>, extra: Partial<JSONSchema> = {}): JSONSchema => ({ type: 'integer', enum: values, 'x-enum-labels': labels, ...extra })
const labelsOf = (t: T, path: string, values: number[]) => Object.fromEntries(values.map((v) => [String(v), t(`gmConfig.schema.enums.${path}.${v}`)]))

const RARITIES = [1, 2, 3]
const POTENTIAL_RANKS = [0, 1, 2, 3, 4]
const LEVEL_LIMIT_RANKS = [0, 1, 2, 3, 4]
const SKILL_RARITIES = [0, 1, 2, 3, 4]
const MOTION_TYPES = [1, 2, 3]
const RESULT_GRADES = Array.from({ length: 19 }, (_, i) => i + 1)
const RIVALS = [1, 2, 3]
const LOGIN_EVENT_TYPES = [3, 2]

/** 活动加成（研究活动的 eventBonus、剧情活动的 storyEventBonus 同一种结构）。 */
function bonusSchema(t: T, f: ReturnType<typeof fieldOf>): JSONSchema {
  const rank = obj({
    potentialRank: f('bonus.potentialRank', enumOf(POTENTIAL_RANKS, labelsOf(t, 'potentialRank', POTENTIAL_RANKS))),
    permil: f('bonus.permil', permil),
  })
  return obj({
    characterBonuses: f(
      'bonus.characterBonuses',
      list(
        obj({
          characterIds: f('bonus.characterIds', refs('character')),
          idolCardRarities: f('bonus.idolCardRarities', { type: 'array', items: enumOf(RARITIES, labelsOf(t, 'rarity', RARITIES)) }),
          permil: f('bonus.characterPermil', permil),
          potentialRankBonusPermils: f('bonus.potentialRankBonusPermils', list(rank, t('gmConfig.schema.items.potential'))),
        }),
        t('gmConfig.schema.items.characterBonus'),
      ),
    ),
    idolCardBonuses: f(
      'bonus.idolCardBonuses',
      list(
        obj({
          idolCardIds: f('bonus.idolCardIds', refs('idolCard')),
          potentialRankBonusPermils: f('bonus.potentialRankBonusPermils', list(rank, t('gmConfig.schema.items.potential'))),
        }),
        t('gmConfig.schema.items.idolCardBonus'),
      ),
    ),
    supportCardBonuses: f(
      'bonus.supportCardBonuses',
      list(
        obj({
          supportCardIds: f('bonus.supportCardIds', refs('supportCard')),
          levelLimitRankBonusPermils: f(
            'bonus.levelLimitRankBonusPermils',
            list(
              obj({
                levelLimitRank: f('bonus.levelLimitRank', enumOf(LEVEL_LIMIT_RANKS, labelsOf(t, 'levelLimitRank', LEVEL_LIMIT_RANKS))),
                permil: f('bonus.permil', permil),
              }),
              t('gmConfig.schema.items.levelLimit'),
            ),
          ),
        }),
        t('gmConfig.schema.items.supportCardBonus'),
      ),
    ),
  })
}

/** 积分奖励（研究活动、剧情活动；剧情活动多一个「置顶显示」）。 */
function pointRewardsSchema(t: T, f: ReturnType<typeof fieldOf>, withDisplayTop: boolean): JSONSchema {
  return list(
    obj({
      point: f('points.point', int({ minimum: 0 })),
      reward: f('points.reward', reward),
      feature: f('points.feature', bool),
      ...(withDisplayTop ? { displayTop: f('points.displayTop', bool) } : {}),
      repeat: f('points.repeat', bool),
      repeatPoint: f('points.repeatPoint', int({ minimum: 0 })),
    }),
    t('gmConfig.schema.items.point'),
  )
}

/**
 * 活动日程里的一个活动（Home.json 的 events，common.Event）。「活动内容」只列出这种活动类型要设置的任务 / 剧情 / 道具
 * （EVENT_CONTENT_FIELDS）；别的类型的字段只有已经填了值时才列出（可以清掉）。
 */
export function homeEventSchema(t: T, te: TE, event: HomeEvent): JSONSchema {
  const f = fieldOf(t, te)
  const content: Record<string, JSONSchema> = {
    missionDailyReleaseGroupId: f('home.missionDailyReleaseGroupId', ref('missionDailyReleaseGroup')),
    missionPanelSheetGroupId: f('home.missionPanelSheetGroupId', ref('missionPanelSheetGroup')),
    dearnessMissionPanelSheetGroupIds: f('home.dearnessMissionPanelSheetGroupIds', refs('missionPanelSheetGroup')),
    missionGroupId: f('home.missionGroupId', ref('missionGroup')),
    dailyMissionGroupId: f('home.dailyMissionGroupId', ref('missionGroup')),
    storyGroupId: f('home.storyGroupId', ref('storyGroup')),
    characterIds: f('home.characterIds', refs('character')),
    itemId: f('home.itemId', ref('item')),
    playItemId: f('home.playItemId', ref('item')),
  }
  const used = EVENT_CONTENT_FIELDS[event.eventType] || []
  const filled = (v: unknown) => (Array.isArray(v) ? v.length > 0 : !!v)
  const shown = Object.entries(content).filter(([key]) => used.includes(key) || filled(event[key]))
  if (shown[0]) shown[0][1] = { ...shown[0][1], 'x-group': t('gmConfig.schema.groups.content') }
  return obj({
    eventName: f('home.eventName', str(), 'basic'),
    eventType: f('home.eventType', enumOf(EVENT_TYPES, Object.fromEntries(EVENT_TYPES.map((v) => [String(v), t(`gmConfig.calendar.eventTypes.${v}`)])))),
    bannerAssetId: f('home.bannerAssetId', image('eventBanner')),
    homeIconAssetId: f('home.homeIconAssetId', image('homeIcon')),
    priority: f('home.priority', int()),
    noti: f('home.noti', bool),
    startTime: f('home.startTime', time, 'time'),
    endTime: f('home.endTime', time),
    fixRankTime: f('home.fixRankTime', time),
    closeTime: f('home.closeTime', time),
    ...Object.fromEntries(shown),
    eventId: f('home.eventId', str(), 'other'),
  })
}

/** 研究活动（Research.json 的 researches）。 */
export function researchSchema(t: T, te: TE): JSONSchema {
  const f = fieldOf(t, te)
  return obj({
    titleAssetId: f('research.titleAssetId', image('researchTitle'), 'basic'),
    bgmAssetId: f('research.bgmAssetId', ref('bgm')),
    startTime: f('research.startTime', time, 'time'),
    endTime: f('research.endTime', time),
    closeTime: f('research.closeTime', time),
    produceGroupIds: f('research.produceGroupIds', refs('produceGroup'), 'produce'),
    playableExamEffectTypes: f('research.playableExamEffectTypes', refs('examEffectType', true)),
    externalProduceCardIds: f('research.externalProduceCardIds', refs('produceCard')),
    memoryPickupProduceCardIds: f('research.memoryPickupProduceCardIds', refs('produceCard')),
    produceItemIds: f('research.produceItemIds', refs('produceItem')),
    rerollItemId: f('research.rerollItemId', ref('item')),
    ignoreMemoryAbilityRarity: f('research.ignoreMemoryAbilityRarity', enumOf(SKILL_RARITIES, labelsOf(t, 'skillRarity', SKILL_RARITIES))),
    excludeRerollMemoryAbilityProduceGroupIds: f('research.excludeRerollMemoryAbilityProduceGroupIds', refs('produceGroup')),
    missionGroupId: f('research.missionGroupId', ref('missionGroup'), 'content'),
    exchangeId: f('research.exchangeId', ref('exchange')),
    exchangeConsumptionItemId: f('research.exchangeConsumptionItemId', ref('item')),
    eventBonus: f('research.eventBonus', bonusSchema(t, f), 'bonus'),
    pointRewards: f('research.pointRewards', pointRewardsSchema(t, f, false), 'points'),
    id: f('research.id', str(), 'other'),
  })
}

/** 剧情活动的内容（StoryEvent.json 的 storyEvents；期间在活动日程里同 ID 的活动上）。 */
export function storyEventSchema(t: T, te: TE): JSONSchema {
  const f = fieldOf(t, te)
  return obj({
    backgroundAssetId: f('story.backgroundAssetId', image('storyBackground'), 'look'),
    appealImageAssetId: f('story.appealImageAssetId', image('storyAppeal')),
    pointRewardBannerAssetId: f('story.pointRewardBannerAssetId', image('storyRewardBanner')),
    gradationColor1: f('story.gradationColor1', str({ format: 'color' })),
    gradationColor2: f('story.gradationColor2', str({ format: 'color' })),
    bgmAssetId: f('story.bgmAssetId', ref('bgm')),
    sceneLayoutId: f('story.sceneLayoutId', str()),
    sceneLayoutCollectionId: f('story.sceneLayoutCollectionId', str()),
    storySynopsis: f('story.storySynopsis', str({ format: 'textarea' })),
    maxIdolCardPotentialRank: f('story.maxIdolCardPotentialRank', enumOf(POTENTIAL_RANKS, labelsOf(t, 'potentialRank', POTENTIAL_RANKS)), 'rules'),
    producePoint: f('story.producePoint', bool),
    produceCoins: f(
      'story.produceCoins',
      list(obj({ grade: f('story.coinGrade', enumOf(RESULT_GRADES, labelsOf(t, 'resultGrade', RESULT_GRADES))), quantity: f('story.coinQuantity', int({ minimum: 0 })) }), t('gmConfig.schema.items.coin')),
    ),
    storyEventBonus: f('story.storyEventBonus', bonusSchema(t, f), 'bonus'),
    pointRewards: f('story.pointRewards', pointRewardsSchema(t, f, true), 'points'),
    motions: f(
      'story.motions',
      list(
        obj({
          characterId: f('motion.characterId', ref('character')),
          costumeId: f('motion.costumeId', ref('costume')),
          costumeHeadId: f('motion.costumeHeadId', ref('costumeHead')),
          motionType: f('motion.motionType', enumOf(MOTION_TYPES, labelsOf(t, 'motionType', MOTION_TYPES))),
          conditionSetId: f('motion.conditionSetId', ref('conditionSet')),
          bodyAssetId: f('motion.bodyAssetId', str()),
          facialAssetId: f('motion.facialAssetId', str()),
          voiceAssetId: f('motion.voiceAssetId', str()),
        }),
        t('gmConfig.schema.items.motion'),
      ),
      'motions',
    ),
    guildMissions: f(
      'story.guildMissions',
      list(
        obj({
          number: f('guild.number', int({ minimum: 1 })),
          produceConditionSetDescription1: f('guild.description1', str()),
          produceConditionSetDescription2: f('guild.description2', str()),
          produceConditionSetDescription3: f('guild.description3', str()),
          produceConditionSetCount: f('guild.conditionCount', int({ minimum: 0 })),
          threshold: f('guild.threshold', int({ minimum: 0 })),
          phaseThresholds: f('guild.phaseThresholds', { type: 'array', items: { type: 'integer' } }),
          reward: f('guild.reward', reward),
          icon: f(
            'guild.icon',
            obj({
              type: f('guild.iconType', int()),
              resourceId1: f('guild.iconResource1', str()),
              resourceId2: f('guild.iconResource2', str()),
              grade: f('guild.iconGrade', enumOf([0, ...RESULT_GRADES], { 0: t('gmConfig.schema.enums.none'), ...labelsOf(t, 'resultGrade', RESULT_GRADES) })),
            }),
          ),
          id: f('guild.id', str()),
        }),
        t('gmConfig.schema.items.guild'),
      ),
      'guild',
    ),
    storyEventId: f('story.storyEventId', str({ readOnly: true }), 'other'),
  })
}

/** 登录奖励：normal 每日 / total 累计 / event 限时 / select 自选。 */
export function loginSchema(t: T, te: TE, kind: 'normal' | 'total' | 'event' | 'select'): JSONSchema {
  const f = fieldOf(t, te)
  const motions = {
    startLoginBonusMotionId: f('login.startMotion', ref('loginBonusMotion')),
    endLoginBonusMotionId: f('login.endMotion', ref('loginBonusMotion')),
  }
  const looks = {
    titleAssetId: f('login.titleAssetId', image('loginTitle')),
    backgroundAssetId: f('login.backgroundAssetId', image('loginBackground')),
  }
  const dayRewards = list(reward, t('gmConfig.schema.items.day'))
  switch (kind) {
    case 'normal':
      return obj({
        name: f('login.name', str(), 'basic'),
        description: f('login.description', str({ format: 'textarea' })),
        ...looks,
        ...motions,
        rewards: f('login.normalRewards', dayRewards, 'rewards'),
      })
    case 'total':
      return obj({
        name: f('login.name', str(), 'basic'),
        ...motions,
        rewards: f('login.totalRewards', list(obj({ days: f('login.days', int({ minimum: 1 })), reward: f('login.reward', reward) }), t('gmConfig.schema.items.tier')), 'rewards'),
        repeatFromDays: f('login.repeatFromDays', int({ minimum: 1 })),
        repeatIntervalDays: f('login.repeatIntervalDays', int({ minimum: 1 })),
        repeatReward: f('login.repeatReward', reward),
      })
    case 'event':
      return obj({
        name: f('login.name', str(), 'basic'),
        description: f('login.description', str({ format: 'textarea' })),
        type: f('login.eventType', enumOf(LOGIN_EVENT_TYPES, labelsOf(t, 'loginEventType', LOGIN_EVENT_TYPES))),
        ...looks,
        ...motions,
        openConditionSetId: f('login.openConditionSetId', ref('conditionSet')),
        startTime: f('login.startTime', time, 'time'),
        endTime: f('login.endTime', time),
        rewards: f('login.eventRewards', dayRewards, 'rewards'),
        id: f('login.id', str(), 'other'),
      })
    case 'select':
      return obj({
        name: f('login.name', str(), 'basic'),
        ...looks,
        startTime: f('login.startTime', time, 'time'),
        endTime: f('login.endTime', time),
        days: f(
          'login.selectDays',
          list(
            obj({
              slots: f(
                'login.slots',
                list(obj({ number: f('login.slotNumber', int({ minimum: 1 })), rewards: f('login.slotRewards', rewards) }), t('gmConfig.schema.items.slot')),
              ),
            }),
            t('gmConfig.schema.items.day'),
          ),
          'rewards',
        ),
        id: f('login.id', str(), 'other'),
      })
  }
}

/** 竞赛赛季（PvpRate.json 的 seasons）。 */
export function seasonSchema(t: T, te: TE): JSONSchema {
  const f = fieldOf(t, te)
  return obj({
    pvpRateConfigId: f('pvp.pvpRateConfigId', ref('pvpRateConfig')),
    prepareStartTime: f('pvp.prepareStartTime', time),
    startTime: f('pvp.startTime', time),
    endTime: f('pvp.endTime', time),
  })
}

/** 竞赛规则与奖励（PvpRate.json 里赛季以外的部分，所有赛季共用）。 */
export function pvpRulesSchema(t: T, te: TE): JSONSchema {
  const f = fieldOf(t, te)
  return obj({
    maxDailyPlayCount: f('pvp.maxDailyPlayCount', int({ minimum: 1 }), 'rules'),
    historyLimit: f('pvp.historyLimit', int({ minimum: 1 })),
    winRates: f(
      'pvp.winRates',
      list(
        obj({
          rivalType: f('pvp.rivalType', enumOf(RIVALS, labelsOf(t, 'rival', RIVALS))),
          minRate: f('pvp.minRate', int({ minimum: 0 })),
          rate: f('pvp.winRate', int({ minimum: 1 })),
          maxRate: f('pvp.maxRate', int({ minimum: 1 })),
        }),
        t('gmConfig.schema.items.row'),
      ),
      'scoring',
    ),
    loseRates: f('pvp.loseRates', list(obj({ minRate: f('pvp.minRate', int({ minimum: 0 })), rate: f('pvp.loseRate', int({ minimum: 0 })) }), t('gmConfig.schema.items.row'))),
    grades: f(
      'pvp.grades',
      list(
        obj({
          grade: f('pvp.grade', int({ readOnly: true })),
          gradeUpRate: f('pvp.gradeUpRate', int({ minimum: 0 })),
          gradeKeepRate: f('pvp.gradeKeepRate', int({ minimum: 0 })),
          winFanCount: f('pvp.winFanCount', int({ minimum: 0 })),
          loseFanCount: f('pvp.loseFanCount', int({ minimum: 0 })),
          playReward: f('pvp.playReward', reward),
          rateRewards: f('pvp.rateRewards', list(obj({ rate: f('pvp.rewardRate', int({ minimum: 1 })), reward: f('pvp.reward', reward) }), t('gmConfig.schema.items.tier'))),
          rankRewards: f(
            'pvp.rankRewards',
            list(
              obj({
                upperRank: f('pvp.upperRank', int({ minimum: 1 })),
                lowerRank: f('pvp.lowerRank', int({ minimum: 0 })),
                rewards: f('pvp.rankRewardItems', rewards),
              }),
              t('gmConfig.schema.items.tier'),
            ),
          ),
        }),
        t('gmConfig.schema.items.grade'),
      ),
      'grades',
    ),
  })
}
