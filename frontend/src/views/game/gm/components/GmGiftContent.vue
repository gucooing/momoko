<!-- 礼物内容：奖励（来自目录，GmRewardPicker）与礼物箱文案。发送到礼物箱与兑换码批次共用。 -->
<template>
  <div class="ggc">
    <div class="ggc__field">
      <span class="ggc__label">{{ t('gmData.gift.rewards') }}</span>
      <GmRewardPicker v-model="rewards" :max="maxRewards" :disabled="disabled || rewardsLocked" />
    </div>
    <label class="ggc__field">
      <span class="ggc__label">
        {{ t('gmData.gift.message') }}
        <span class="ggc__hint">{{ t('gmData.gift.messageNote', { max: maxMessage }) }}</span>
      </span>
      <input v-model="message" class="ggc__input" :maxlength="maxMessage" :placeholder="t('gmData.gift.messageHint')" :disabled="disabled" />
    </label>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import GmRewardPicker from './GmRewardPicker.vue'
import type { GmReward } from './types'

withDefaults(
  defineProps<{
    /** 奖励项数上限（操作 gift.send 的参数 Schema） */
    maxRewards?: number
    /** 文案长度上限 */
    maxMessage?: number
    disabled?: boolean
    /** 奖励已定、不能再改（编辑兑换码批次） */
    rewardsLocked?: boolean
  }>(),
  { maxRewards: 20, maxMessage: 64, disabled: false, rewardsLocked: false },
)
const rewards = defineModel<GmReward[]>('rewards', { required: true })
const message = defineModel<string>('message', { required: true })

const { t } = useI18n()
</script>

<style scoped>
.ggc {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.ggc__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.ggc__label {
  font-size: 13px;
  font-weight: 700;
}
.ggc__hint {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.ggc__input {
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
  outline: none;
}
.ggc__input:focus {
  border-color: var(--m-p);
}
</style>
