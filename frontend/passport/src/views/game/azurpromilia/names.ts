// 蓝色星原游戏空间按通行证当前语言显示的奖励与条目名称。
import { azurNames } from '@/games/azurpromilia/names'
import { i18n } from '@passport/locales'

export const { kindName, rewardTypeName, itemName, rewardsText, searchItems } = azurNames(() => i18n.global.locale.value)
