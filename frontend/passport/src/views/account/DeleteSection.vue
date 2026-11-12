<!-- 注销通行证：申请后 N 天删除（解绑全部游戏账号，存档留在各服务器上），期间可撤销。 -->
<template>
  <section v-if="me" class="pp-card pp-section del">
    <h1 class="pp-section__title">{{ t('account.delete.title') }}</h1>
    <template v-if="me.deletionScheduledAt">
      <p class="pp-note warn">{{ t('account.delete.scheduled', { date: fmtTime(me.deletionScheduledAt) }) }}</p>
      <PButton class="self" variant="primary" :loading="busy" @click="cancel">{{ t('account.delete.cancel') }}</PButton>
    </template>
    <template v-else>
      <p class="pp-section__desc">{{ t('account.delete.desc', { days: session.config?.deletionGraceDays ?? 7 }) }}</p>
      <form class="pp-form form" @submit.prevent="request">
        <PField :label="t('account.security.currentPassword')">
          <input v-model="password" type="password" autocomplete="current-password" required />
        </PField>
        <PButton type="submit" class="self" variant="danger" :loading="busy">{{ t('account.delete.request') }}</PButton>
      </form>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PButton from '@passport/components/PButton.vue'
import PField from '@passport/components/PField.vue'
import * as api from '@passport/api/passport'
import { useSessionStore } from '@passport/stores/session'
import { useUiStore } from '@passport/stores/ui'
import { errorText, fmtTime } from '@passport/utils/format'

const { t } = useI18n()
const session = useSessionStore()
const ui = useUiStore()

const me = computed(() => session.me)
const password = ref('')
const busy = ref(false)

const request = async () => {
  busy.value = true
  try {
    session.applyMe(await api.requestDeletion(password.value))
    password.value = ''
    ui.toast(t('account.delete.requested'), 'ok')
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    busy.value = false
  }
}

const cancel = async () => {
  busy.value = true
  try {
    session.applyMe(await api.cancelDeletion())
    ui.toast(t('account.delete.cancelled'), 'ok')
  } catch (e) {
    ui.toast(errorText(e), 'bad')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.del {
  gap: 12px;
}
.form {
  max-width: 420px;
}
.self {
  align-self: flex-start;
}
.warn {
  background: var(--m-warnSoft);
  color: var(--m-ink);
}
</style>
