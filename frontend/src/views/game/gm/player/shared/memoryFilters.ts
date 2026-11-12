// 记忆与培育记录的稀有度、评价筛选（由服务端筛选分页）：胶囊上的条数来自列表的 summary
// （按稀有度的条数带上评价筛选，按评价的条数带上稀有度筛选）；条数为 0 且未选中的胶囊不显示。
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import { GRADE_GROUPS, RARITY_VALUES, countTotal, gradeGroupCount, gradeGroupGrades, rarityLabel } from '@/games/gakumas/display'
import type { MemoryCounts } from '@/types/v1/gakumas/types'

export function useMemoryFilters(summary: () => MemoryCounts | undefined) {
  const { t } = useI18n()
  /** 偶像卡稀有度（0 为全部） */
  const rarity = ref(0)
  /** 评价分类（GRADE_GROUPS 的 key，空串为全部） */
  const grade = ref('')
  /** 交给服务端的评价（ResultGrade） */
  const grades = computed(() => gradeGroupGrades(grade.value))

  const rarityChips = computed<ToolbarChip<number>[]>(() => {
    const counts = summary()?.rarities
    return [
      { label: t('gmData.common.all'), value: 0, count: counts ? countTotal(counts) : null },
      ...RARITY_VALUES.filter((r) => counts?.[r] || rarity.value === r).map((r) => ({
        label: rarityLabel(r),
        value: r,
        count: counts?.[r] ?? 0,
      })),
    ]
  })

  const gradeChips = computed<ToolbarChip<string>[]>(() => {
    const counts = summary()?.grades
    return [
      { label: t('gmData.memories.allGrades'), value: '', count: counts ? countTotal(counts) : null },
      ...GRADE_GROUPS.filter((g) => gradeGroupCount(counts, g) || grade.value === g.key).map((g) => ({
        label: g.label,
        value: g.key,
        count: gradeGroupCount(counts, g),
      })),
    ]
  })

  /** 选了稀有度或评价 */
  const active = computed(() => !!rarity.value || !!grade.value)

  return { rarity, grade, grades, rarityChips, gradeChips, active }
}
