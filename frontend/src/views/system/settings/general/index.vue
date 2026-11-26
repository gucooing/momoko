<!-- 系统设置（第二期）：标题(作用范围 · 上次修改) → 左侧本页目录（手机为横向 chips）+
     右侧分组卡片：常规 / 登录 / 邮件（含测试与模板列表）/ 储存（头像与应用图片放到哪个对象存储来源、游戏素材地址）/ 版本更新 → 底部吸附保存条（按分组记录修改，一次保存）。
     模板编辑、模板测试发送、预览用 FormDialog。权限：PERM.SYSTEM_CONFIG_EDIT 控制编辑，system:update 控制检查更新。 -->
<template>
  <div class="st-page">
    <PageHeader :title="t('system.settings.pageTitle')">
      <template #description>
        {{ t('system.settings.scope') }}
        <template v-if="lastModifiedText"> · {{ lastModifiedText }}</template>
      </template>
    </PageHeader>

    <div class="st-layout">
      <!-- 本页目录 -->
      <nav class="st-toc" :aria-label="t('system.settings.toc')">
        <div class="st-toc__label">{{ t('system.settings.toc') }}</div>
        <div class="st-toc__list">
          <button
            v-for="s in sections"
            :key="s.key"
            type="button"
            class="st-toc__item"
            :class="{ 'is-on': activeSec === s.key }"
            :aria-current="activeSec === s.key ? 'true' : undefined"
            @click="scrollToSec(s.key)"
          >
            <UIcon :name="s.icon" class="st-toc__ico" />
            <span class="st-toc__text">{{ menuStore.isMobile && s.short ? s.short : s.label }}</span>
            <span v-if="dirty[s.key]" class="st-toc__dot" :aria-label="t('system.settings.dirtyMark')" />
          </button>
        </div>
        <p class="st-toc__tip">{{ t('system.settings.tocTip') }}</p>
      </nav>

      <div class="st-main">
        <div v-if="!canEdit" class="st-readonly">
          <UIcon name="i-lucide-lock" />
          {{ t('system.settings.noEditPermission') }}
        </div>

        <!-- 常规 -->
        <section id="st-sec-general" ref="secGeneral" class="st-card" aria-labelledby="st-h-general">
          <header class="st-card__head">
            <span class="st-card__sticker m-sticker m-sticker--p" style="--rot: -4deg"><UIcon name="i-lucide-settings" /></span>
            <div class="st-card__title">
              <h2 id="st-h-general">{{ t('system.settings.sec.general') }}</h2>
              <span>{{ t('system.settings.sec.generalDesc') }}</span>
            </div>
          </header>

          <div class="st-row">
            <div class="st-row__l">
              <label class="st-row__t" for="st-site-name">{{ t('system.settings.siteName') }}</label>
              <span class="st-row__d">{{ t('system.settings.siteNameDesc', { var: tokenOf('site') }) }}</span>
            </div>
            <div class="st-row__c">
              <input id="st-site-name" v-model.trim="siteForm.siteName" class="app-input st-input" maxlength="64" placeholder="Momoko" :disabled="!canEdit" />
            </div>
          </div>
          <div class="st-row">
            <div class="st-row__l">
              <label class="st-row__t" for="st-site-url">{{ t('system.settings.externalUrl') }}</label>
              <span class="st-row__d">{{ t('system.settings.externalUrlDesc', { var: tokenOf('url') }) }}</span>
            </div>
            <div class="st-row__c st-row__c--col">
              <input
                id="st-site-url"
                v-model.trim="siteForm.externalUrl"
                type="url"
                class="app-input st-input m-mono"
                placeholder="https://panel.example.com"
                :disabled="!canEdit"
              />
              <span v-if="!siteForm.externalUrl" class="st-hint">{{ t('system.settings.externalUrlHint', { var: tokenOf('url') }) }}</span>
            </div>
          </div>
          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.defaultLanguage') }}</span>
              <span class="st-row__d">{{ t('system.settings.defaultLanguageDesc') }}</span>
            </div>
            <div class="st-row__c">
              <div class="st-seg st-seg--wrap" role="group" :aria-label="t('system.settings.defaultLanguage')">
                <button
                  v-for="l in languageOptions"
                  :key="l.value"
                  type="button"
                  :class="{ 'is-on': siteForm.defaultLanguage === l.value }"
                  :aria-pressed="siteForm.defaultLanguage === l.value"
                  :disabled="!canEdit"
                  @click="siteForm.defaultLanguage = l.value"
                >
                  {{ l.label }}
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- 登录 -->
        <section id="st-sec-login" ref="secLogin" class="st-card" aria-labelledby="st-h-login">
          <header class="st-card__head">
            <span class="st-card__sticker m-sticker m-sticker--a2" style="--rot: 3deg"><UIcon name="i-lucide-log-in" /></span>
            <div class="st-card__title">
              <h2 id="st-h-login">{{ t('system.settings.sec.login') }}</h2>
              <span>{{ t('system.settings.sec.loginDesc') }}</span>
            </div>
          </header>

          <div v-for="sw in loginSwitches" :key="sw.key" class="st-row st-row--switch">
            <div class="st-row__l">
              <span class="st-row__t">{{ t(sw.label) }}</span>
              <span class="st-row__d">{{ t(sw.desc) }}</span>
            </div>
            <AppSwitch v-model="loginForm[sw.key]" :disabled="!canEdit" :aria-label="t(sw.label)" />
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.tokenShort') }}</span>
              <span class="st-row__d">{{ t('system.settings.tokenAccessDesc') }}；{{ t('system.settings.tokenRefreshDesc') }}</span>
            </div>
            <div class="st-row__c st-pair">
              <label class="st-pair__item">
                <span class="st-pair__k">{{ t('system.settings.tokenAccessShort') }}</span>
                <span class="st-unit">
                  <input
                    v-model.number="loginForm.accessTokenTtlMinutes"
                    type="number"
                    min="5"
                    max="1440"
                    inputmode="numeric"
                    :aria-label="t('system.settings.tokenAccess')"
                    :disabled="!canEdit"
                  />
                  <span class="st-unit__u">{{ t('system.settings.unit.minutes') }}</span>
                </span>
              </label>
              <label class="st-pair__item">
                <span class="st-pair__k">{{ t('system.settings.tokenRefreshShort') }}</span>
                <span class="st-unit">
                  <input
                    v-model.number="loginForm.refreshTokenTtlDays"
                    type="number"
                    min="1"
                    max="365"
                    inputmode="numeric"
                    :aria-label="t('system.settings.tokenRefresh')"
                    :disabled="!canEdit"
                  />
                  <span class="st-unit__u">{{ t('system.settings.unit.days') }}</span>
                </span>
              </label>
            </div>
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">
                {{ t('system.settings.lockout') }}
                <span v-if="!loginForm.lockoutThreshold" class="st-flag">{{ t('system.settings.lockoutOff') }}</span>
              </span>
              <span class="st-row__d">{{ t(menuStore.isMobile ? 'system.settings.lockoutShortDesc' : 'system.settings.lockoutDesc') }}</span>
            </div>
            <div class="st-row__c st-pair">
              <label class="st-pair__item">
                <span class="st-pair__k">{{ t('system.settings.lockoutFails') }}</span>
                <span class="st-unit">
                  <input v-model.number="loginForm.lockoutThreshold" type="number" min="0" max="100" inputmode="numeric" :disabled="!canEdit" />
                  <span class="st-unit__u">{{ t('system.settings.unit.times') }}</span>
                </span>
              </label>
              <label class="st-pair__item">
                <span class="st-pair__k">{{ t('system.settings.lockoutLock') }}</span>
                <span class="st-unit">
                  <input
                    v-model.number="loginForm.lockoutMinutes"
                    type="number"
                    min="1"
                    max="1440"
                    inputmode="numeric"
                    :disabled="!canEdit || !loginForm.lockoutThreshold"
                  />
                  <span class="st-unit__u">{{ t('system.settings.unit.minutes') }}</span>
                </span>
              </label>
            </div>
          </div>
        </section>

        <!-- 邮件 -->
        <section id="st-sec-mail" ref="secMail" class="st-card" aria-labelledby="st-h-mail">
          <header class="st-card__head">
            <span class="st-card__sticker m-sticker m-sticker--a" style="--rot: -2deg"><UIcon name="i-lucide-mail" /></span>
            <div class="st-card__title">
              <h2 id="st-h-mail">{{ t('system.settings.sec.mail') }}</h2>
              <span>{{ t('system.settings.sec.mailDesc') }}</span>
            </div>
            <StatusPill class="st-card__pill" :variant="mailStatus.variant" :label="mailStatus.label" />
          </header>

          <div class="st-row st-row--switch">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.enableEmailService') }}</span>
              <span class="st-row__d">{{ t('system.settings.mailEnableDesc') }}</span>
            </div>
            <AppSwitch v-model="emailForm.enabled" :disabled="!canEdit" :aria-label="t('system.settings.enableEmailService')" />
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <label class="st-row__t" for="st-smtp-host">{{ t('system.settings.smtpServer') }}</label>
              <span class="st-row__d">{{ t('system.settings.smtpServerDesc') }}</span>
            </div>
            <div class="st-row__c st-host">
              <input id="st-smtp-host" v-model.trim="emailForm.host" class="app-input m-mono" placeholder="smtp.example.com" :disabled="!canEdit" />
              <label class="st-unit st-unit--pre">
                <span class="st-unit__u">{{ t('system.settings.port') }}</span>
                <input v-model.number="emailForm.port" type="number" min="1" max="65535" inputmode="numeric" :aria-label="t('system.settings.smtpPort')" :disabled="!canEdit" />
              </label>
            </div>
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.encryption') }}</span>
              <span class="st-row__d">{{ encHint }}</span>
            </div>
            <div class="st-row__c">
              <div class="st-seg" role="group" :aria-label="t('system.settings.encryption')">
                <button
                  v-for="e in encOptions"
                  :key="e.value"
                  type="button"
                  :class="{ 'is-on': emailForm.encryption === e.value }"
                  :aria-pressed="emailForm.encryption === e.value"
                  :disabled="!canEdit"
                  @click="pickEncryption(e.value)"
                >
                  {{ e.label }}
                </button>
              </div>
            </div>
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.sender') }}</span>
              <span class="st-row__d">{{ t('system.settings.senderDesc') }}</span>
            </div>
            <div class="st-row__c st-sender">
              <input v-model.trim="emailForm.fromName" class="app-input" :aria-label="t('system.settings.fromName')" placeholder="Momoko" :disabled="!canEdit" />
              <input
                v-model.trim="emailForm.from"
                type="email"
                class="app-input m-mono"
                :aria-label="t('system.settings.fromEmail')"
                placeholder="noreply@example.com"
                :disabled="!canEdit"
              />
            </div>
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <label class="st-row__t" for="st-smtp-user">{{ t('system.settings.account') }}</label>
              <span class="st-row__d">{{ t('system.settings.accountDesc') }}</span>
            </div>
            <div class="st-row__c">
              <input id="st-smtp-user" v-model.trim="emailForm.username" class="app-input st-input m-mono" placeholder="user@example.com" autocomplete="off" :disabled="!canEdit" />
            </div>
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <label class="st-row__t" for="st-smtp-pwd">{{ t('system.settings.password') }}</label>
              <span class="st-row__d">{{ t('system.settings.passwordDesc') }}</span>
            </div>
            <div class="st-row__c">
              <span class="st-pwd">
                <input
                  id="st-smtp-pwd"
                  v-model="emailForm.password"
                  :type="showPwd ? 'text' : 'password'"
                  class="m-mono"
                  autocomplete="new-password"
                  :placeholder="t('system.settings.smtpPasswordPlaceholder')"
                  :disabled="!canEdit"
                />
                <AppIconButton :label="showPwd ? t('system.settings.hidePwd') : t('system.settings.showPwd')" :box="40" @click="showPwd = !showPwd">
                  <UIcon :name="showPwd ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="st-pwd__ico" />
                </AppIconButton>
              </span>
            </div>
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.sendParams') }}</span>
              <span class="st-row__d">{{ t('system.settings.sendParamsDesc') }}</span>
            </div>
            <div class="st-row__c st-pair">
              <label class="st-pair__item">
                <span class="st-pair__k">{{ t('system.settings.timeout') }}</span>
                <span class="st-unit">
                  <input v-model.number="emailForm.timeoutSeconds" type="number" min="1" max="60" inputmode="numeric" :disabled="!canEdit" />
                  <span class="st-unit__u">{{ t('system.settings.unit.seconds') }}</span>
                </span>
              </label>
              <label class="st-pair__item">
                <span class="st-pair__k">{{ t('system.settings.concurrency') }}</span>
                <span class="st-unit">
                  <input v-model.number="emailForm.ccsN" type="number" min="1" max="50" inputmode="numeric" :disabled="!canEdit" />
                  <span class="st-unit__u">{{ t('system.settings.unit.count') }}</span>
                </span>
              </label>
            </div>
          </div>

          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.test') }}</span>
              <span class="st-row__d">{{ t('system.settings.testDesc') }}</span>
            </div>
            <div class="st-row__c st-row__c--col">
              <div class="st-test">
                <input
                  v-model.trim="testRecipient"
                  type="email"
                  class="app-input"
                  :aria-label="t('system.settings.recipientEmail')"
                  :placeholder="t('system.settings.recipientEmailPlaceholder')"
                />
                <UButton color="neutral" variant="outline" icon="i-lucide-send" :loading="emailTesting" :disabled="!canEdit || !testRecipient" @click="handleEmailTest">
                  {{ t('system.settings.sendTest') }}
                </UButton>
              </div>
              <span class="st-test__res" :class="`is-${lastTestState}`">
                <span v-if="lastTest" class="st-test__pill">{{ t(`system.settings.lastTestLabel.${lastTest.success ? 'ok' : 'failed'}`) }}</span>
                <span class="st-test__txt">{{ lastTestText }}</span>
              </span>
            </div>
          </div>

          <!-- 邮件模板 -->
          <div class="st-sub">
            <h3>{{ t('system.settings.templates') }}</h3>
            <span>{{ t('system.settings.templatesHint') }}</span>
          </div>
          <div v-for="(tp, idx) in templateRows" :key="tp.type" class="st-tpl">
            <span class="st-tpl__sticker m-sticker" :class="tp.sticker" :style="{ '--rot': ['-4deg', '3deg', '-2deg'][idx % 3] }">
              <UIcon :name="tp.icon" />
            </span>
            <span class="st-tpl__main">
              <span class="st-tpl__name">
                {{ tp.name }}
                <span v-if="tp.customized" class="st-flag st-flag--a2">{{ t('system.settings.tplCustomized') }}</span>
                <span v-else-if="!tp.subject" class="st-flag st-flag--warn">{{ t('system.settings.tplNotSet') }}</span>
              </span>
              <span class="st-tpl__subj">{{ tp.subject ? t('system.settings.tplSubject', { subject: tp.subject }) : t('system.settings.tplNotSetDesc') }}</span>
              <span v-if="tp.vars.length" class="st-tpl__mvars">
                <span v-for="v in tp.vars.slice(0, 3)" :key="v" class="st-var m-mono">{{ tokenOf(v) }}</span>
              </span>
            </span>
            <span class="st-tpl__vars">
              <span v-for="v in tp.vars.slice(0, 3)" :key="v" class="st-var m-mono">{{ tokenOf(v) }}</span>
            </span>
            <span class="st-tpl__date m-num">{{ tp.updated }}</span>
            <span class="st-tpl__acts">
              <AppIconButton :label="t('system.settings.preview')" :box="40" :disabled="!tp.subject && !tp.template" @click="openPreview(tp)">
                <UIcon name="i-lucide-eye" class="st-tpl__ico" />
              </AppIconButton>
              <UButton
                class="st-tpl__edit"
                color="neutral"
                variant="soft"
                size="sm"
                icon="i-lucide-pencil"
                :aria-label="t('system.settings.tplEditTitle', { name: tp.name })"
                :disabled="!canEdit"
                @click="openTemplateEditor(tp)"
              >
                <span class="st-tpl__edit-t">{{ t('system.settings.tplEdit') }}</span>
              </UButton>
            </span>
          </div>
        </section>

        <!-- 储存 -->
        <section id="st-sec-storage" ref="secStorage" class="st-card" aria-labelledby="st-h-storage">
          <header class="st-card__head">
            <span class="st-card__sticker m-sticker m-sticker--p" style="--rot: 3deg"><UIcon name="i-lucide-hard-drive" /></span>
            <div class="st-card__title">
              <h2 id="st-h-storage">{{ t('system.settings.sec.storage') }}</h2>
              <span>{{ t('system.settings.sec.storageDesc') }}</span>
            </div>
            <StatusPill
              class="st-card__pill"
              :variant="storageForm.sourceId ? 'success' : 'neutral'"
              :label="storageForm.sourceId ? t('system.settings.storage.on') : t('system.settings.storage.off')"
            />
          </header>

          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.storage.source') }}</span>
              <span class="st-row__d">{{ t('system.settings.storage.sourceDesc') }}</span>
            </div>
            <div class="st-row__c st-row__c--col">
              <div class="st-seg st-seg--wrap" role="group" :aria-label="t('system.settings.storage.source')">
                <button
                  v-for="o in storageOptions"
                  :key="o.id"
                  type="button"
                  :class="{ 'is-on': storageForm.sourceId === o.id }"
                  :aria-pressed="storageForm.sourceId === o.id"
                  :disabled="!canEdit"
                  @click="storageForm.sourceId = o.id"
                >
                  {{ o.name }}
                </button>
              </div>
              <span class="st-hint">
                {{ storageSources.length ? t('system.settings.storage.hint') : t('system.settings.storage.empty') }}
                <RouterLink :to="pageTabTo('file', 'source')">{{ t('system.settings.storage.manage') }}</RouterLink>
              </span>
            </div>
          </div>
          <div class="st-row">
            <div class="st-row__l">
              <label class="st-row__t" for="st-game-assets">{{ t('system.settings.storage.gameAssets') }}</label>
              <span class="st-row__d">{{ t('system.settings.storage.gameAssetsDesc') }}</span>
            </div>
            <div class="st-row__c st-row__c--col">
              <div class="st-test">
                <input
                  id="st-game-assets"
                  v-model.trim="storageForm.gameAssetUrl"
                  type="url"
                  class="app-input m-mono"
                  placeholder="https://cdn.example.com/momoko-games"
                  :disabled="!canEdit"
                />
                <UButton
                  color="neutral"
                  variant="outline"
                  icon="i-lucide-image"
                  :loading="assetProbing"
                  :disabled="!storageForm.gameAssetUrl"
                  @click="handleAssetProbe"
                >
                  {{ t('system.settings.storage.gameAssetsTest') }}
                </UButton>
              </div>
              <span v-if="assetProbe" class="st-test__res" :class="`is-${assetProbe.ok ? 'ok' : 'bad'}`">
                <span class="st-test__pill">{{ assetProbe.ok ? t('system.settings.storage.gameAssetsOk') : t('system.settings.storage.gameAssetsBad') }}</span>
                <span class="st-test__txt">{{ assetProbe.text }}</span>
              </span>
              <span class="st-hint">
                {{ storageForm.gameAssetUrl ? t('system.settings.storage.gameAssetsHint', { example: assetExample }) : t('system.settings.storage.gameAssetsEmpty') }}
              </span>
            </div>
          </div>
        </section>

        <!-- 版本更新 -->
        <section id="st-sec-update" ref="secUpdate" class="st-card" aria-labelledby="st-h-update">
          <header class="st-card__head">
            <span class="st-card__sticker m-sticker m-sticker--ok" style="--rot: 4deg"><UIcon name="i-lucide-download" /></span>
            <div class="st-card__title">
              <h2 id="st-h-update">{{ t('system.settings.sec.update') }}</h2>
              <span>{{ t('system.settings.sec.updateDesc') }}</span>
            </div>
          </header>

          <div class="st-ver">
            <span class="st-ver__blob" aria-hidden="true" />
            <span class="st-ver__ring" aria-hidden="true" />
            <div class="st-ver__body">
              <div class="st-ver__text">
                <span class="st-ver__k">{{ t('system.settings.updateCurrent') }}</span>
                <span class="st-ver__v m-num">{{ currentVersion }}</span>
                <span class="st-ver__s">
                  {{ checkText }}
                  <a v-if="lastCheck?.hasUpdate && lastCheck.releaseUrl" :href="lastCheck.releaseUrl" target="_blank" rel="noopener" class="st-ver__link">
                    {{ t('system.settings.viewRelease') }}
                  </a>
                </span>
              </div>
              <div class="st-ver__btns">
                <button
                  v-if="canOneClickUpdate"
                  type="button"
                  class="st-ver__btn st-ver__btn--go"
                  :disabled="updBusy"
                  @click="confirmStartUpdate"
                >
                  <UIcon name="i-lucide-download" />
                  {{ t('system.settings.updater.updateTo', { version: lastCheck?.latestVersion }) }}
                </button>
                <button
                  type="button"
                  class="st-ver__btn"
                  :disabled="!canCheckUpdate || checking"
                  :title="canCheckUpdate ? undefined : t('system.settings.noUpdatePermission')"
                  @click="handleCheckUpdate"
                >
                  <UIcon name="i-lucide-refresh-cw" :class="{ 'is-spin': checking }" />
                  {{ t('system.settings.checkUpdate') }}
                </button>
              </div>
            </div>
          </div>

          <!-- 在线更新：不能更新的原因 / 进度 / 上传的包等确认 / 失败 -->
          <p v-if="upd && !upd.supported" class="st-upd-note is-warn">
            <UIcon name="i-lucide-triangle-alert" />
            {{ t('system.settings.updater.unsupported', { reason: upd.unsupportedReason }) }}
          </p>
          <p v-else-if="lastCheck?.hasUpdate && checkedNoAsset" class="st-upd-note">
            <UIcon name="i-lucide-info" />
            {{ t('system.settings.updater.noAsset', { platform: upd?.platform || '' }) }}
          </p>

          <div v-if="updPanel" class="st-upd" :class="`is-${updPanel}`" role="status" aria-live="polite">
            <div class="st-upd__head">
              <span class="st-upd__sticker m-sticker" :class="`m-sticker--${updTone}`">
                <UIcon :name="updIcon" :class="{ 'is-spin': updPanel === 'restarting' || updPanel === 'installing' }" />
              </span>
              <span class="st-upd__text">
                <span class="st-upd__title">{{ updTitle }}</span>
                <span v-if="updSub" class="st-upd__sub m-num">{{ updSub }}</span>
              </span>
              <span class="st-upd__acts">
                <template v-if="updPanel === 'downloading' && upd?.download">
                  <AppIconButton
                    v-if="dlPaused"
                    icon="HOutline:PlayIcon"
                    :label="t('downloads.resume')"
                    :box="34"
                    :size="17"
                    :disabled="updBusy"
                    @click="toggleUpdatePause"
                  />
                  <AppIconButton
                    v-else
                    icon="HOutline:PauseIcon"
                    :label="t('downloads.pause')"
                    :box="34"
                    :size="17"
                    :disabled="updBusy"
                    @click="toggleUpdatePause"
                  />
                </template>
                <AppIconButton
                  v-if="updPanel === 'downloading' && !uploading"
                  icon="HOutline:XMarkIcon"
                  :label="t('system.settings.updater.cancel')"
                  :box="34"
                  :size="17"
                  :disabled="updBusy"
                  @click="handleCancelUpdate"
                />
              </span>
            </div>

            <div v-if="updPanel === 'downloading' || updPanel === 'uploading'" class="st-upd__bar">
              <span class="st-upd__fill" :class="{ 'is-paused': dlPaused }" :style="{ width: `${updPercent}%` }" />
            </div>

            <dl v-if="upd?.package && (updPanel === 'ready' || updPanel === 'installing' || updPanel === 'restarting')" class="st-upd__pkg">
              <dt>{{ t('system.settings.updater.pkgVersion') }}</dt>
              <dd class="m-num">{{ upd.package.version || t('system.settings.updater.pkgUnknown') }}</dd>
              <dt>{{ t('system.settings.updater.pkgPlatform') }}</dt>
              <dd class="m-num">{{ upd.package.goos }}/{{ upd.package.goarch }}</dd>
              <template v-if="upd.package.revision">
                <dt>{{ t('system.settings.updater.pkgCommit') }}</dt>
                <dd class="m-num">{{ upd.package.revision.slice(0, 10) }}<template v-if="fmtDate(upd.package.commitTime)"> · {{ fmtDate(upd.package.commitTime) }}</template></dd>
              </template>
              <dt>{{ t('system.settings.updater.pkgSize') }}</dt>
              <dd class="m-num">{{ formatFileSize(upd.package.size) }}</dd>
              <dt>SHA256</dt>
              <dd class="m-num st-upd__hash" :title="upd.package.sha256">{{ upd.package.sha256 }}</dd>
              <template v-if="upd.package.fileName">
                <dt>{{ t('system.settings.updater.pkgFile') }}</dt>
                <dd class="st-upd__hash">{{ upd.package.fileName }}</dd>
              </template>
            </dl>
            <p v-if="updPanel === 'ready' && upd?.package?.downgrade" class="st-upd-note is-warn">
              <UIcon name="i-lucide-triangle-alert" />
              {{ t('system.settings.updater.downgrade', { current: upd.currentVersion }) }}
            </p>
            <p v-else-if="updPanel === 'ready' && upd?.package?.sameVersion" class="st-upd-note">
              <UIcon name="i-lucide-info" />
              {{ t('system.settings.updater.sameVersion') }}
            </p>
            <p v-if="updPanel === 'failed'" class="st-upd__err">{{ upd?.error }}</p>

            <div v-if="updPanel === 'ready' || updPanel === 'failed'" class="st-upd__foot">
              <template v-if="updPanel === 'ready'">
                <UButton color="neutral" variant="soft" :disabled="updBusy" @click="handleCancelUpdate">{{ t('system.settings.updater.cancel') }}</UButton>
                <UButton color="primary" icon="i-lucide-rocket" :loading="updBusy" @click="confirmApplyUpdate">{{ t('system.settings.updater.install') }}</UButton>
              </template>
              <template v-else>
                <UButton color="neutral" variant="soft" :disabled="updBusy" @click="handleCancelUpdate">{{ t('system.settings.updater.dismiss') }}</UButton>
                <UButton
                  v-if="upd?.source === UpdateSource.UpdateSource_Github && canCheckUpdate"
                  color="primary"
                  icon="i-lucide-rotate-ccw"
                  :loading="updBusy"
                  @click="retryUpdate"
                >
                  {{ t('system.settings.updater.retry') }}
                </UButton>
              </template>
            </div>
          </div>

          <p v-if="upd?.backup && !updPanel" class="st-upd-note">
            <UIcon name="i-lucide-archive" />
            {{ t('system.settings.updater.backup', { path: upd.backup }) }}
          </p>

          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.updater.upload') }}</span>
              <span class="st-row__d">{{ t('system.settings.updater.uploadDesc') }}</span>
            </div>
            <div class="st-row__c">
              <input ref="uploadInput" type="file" class="st-upd__file" tabindex="-1" aria-hidden="true" @change="onPickUpdateFile" />
              <UButton
                color="neutral"
                variant="soft"
                icon="i-lucide-upload"
                :loading="uploading"
                :disabled="!canCheckUpdate || !upd?.supported || updRunning"
                @click="uploadInput?.click()"
              >
                {{ uploading ? (uploadPercent >= 100 ? t('system.settings.updater.verifying') : t('system.settings.updater.uploading', { percent: uploadPercent })) : t('system.settings.updater.upload') }}
              </UButton>
            </div>
          </div>

          <div class="st-row st-row--switch">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.autoCheck') }}</span>
              <span class="st-row__d">{{ t('system.settings.autoCheckDesc') }}</span>
            </div>
            <AppSwitch v-model="siteForm.autoCheckUpdate" :disabled="!canEdit" :aria-label="t('system.settings.autoCheck')" />
          </div>
          <div class="st-row st-row--switch">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.updater.autoInstall') }}</span>
              <span class="st-row__d">{{ t('system.settings.updater.autoInstallDesc') }}</span>
            </div>
            <AppSwitch
              v-model="siteForm.autoInstallUpdate"
              :disabled="!canEdit || !siteForm.autoCheckUpdate"
              :aria-label="t('system.settings.updater.autoInstall')"
            />
          </div>
          <div class="st-row">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.channel') }}</span>
              <span class="st-row__d">{{ t('system.settings.channelDesc') }}</span>
            </div>
            <div class="st-row__c">
              <div class="st-seg" role="group" :aria-label="t('system.settings.channel')">
                <button
                  v-for="c in channelOptions"
                  :key="c.value"
                  type="button"
                  :class="{ 'is-on': siteForm.updateChannel === c.value }"
                  :aria-pressed="siteForm.updateChannel === c.value"
                  :disabled="!canEdit"
                  @click="siteForm.updateChannel = c.value"
                >
                  {{ c.label }}
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- 保存条 -->
        <Transition name="savebar">
          <div v-if="canEdit && (dirtyCount > 0 || saving || justSaved)" class="st-bar" :class="{ 'is-dirty': dirtyCount > 0 }" role="status">
            <span class="st-bar__dot" />
            <span class="st-bar__text">
              <template v-if="dirtyCount > 0">
                {{ menuStore.isMobile ? t('system.settings.bar.dirtyShort') : t('system.settings.bar.dirty', { n: dirtyCount }) }}
              </template>
              <template v-else>{{ justSaved ? t('system.settings.bar.saved') : t('system.settings.bar.clean') }}</template>
            </span>
            <button type="button" class="st-bar__btn st-bar__btn--ghost" :disabled="!dirtyCount || saving" @click="discardChanges">
              {{ menuStore.isMobile ? t('system.settings.bar.discardShort') : t('system.settings.bar.discard') }}
            </button>
            <button type="button" class="st-bar__btn st-bar__btn--primary" :disabled="!dirtyCount || saving || !canEdit" @click="saveAll">
              <UIcon :name="saving ? 'i-lucide-loader-circle' : 'i-lucide-check'" :class="{ 'is-spin': saving }" />
              {{ menuStore.isMobile ? t('system.settings.bar.saveShort') : t('system.settings.bar.save') }}
            </button>
          </div>
        </Transition>
      </div>
    </div>

    <!-- 模板编辑 -->
    <FormDialog v-model="editorVisible" :title="t('system.settings.tplEditTitle', { name: editorName })" :width="760">
      <div class="st-form">
        <div class="app-field">
          <label class="app-label" for="st-tpl-subject">{{ t('system.settings.emailSubject') }}</label>
          <input
            id="st-tpl-subject"
            v-model="templateForm.subject"
            class="app-input"
            :placeholder="t('system.settings.subjectPlaceholder')"
            @focus="onInputFocus"
          />
        </div>
        <div class="app-field">
          <label class="app-label" for="st-tpl-body">{{ t('system.settings.emailContent') }}</label>
          <textarea
            id="st-tpl-body"
            v-model="templateForm.template"
            class="app-textarea st-code"
            rows="12"
            placeholder="<html><body>...</body></html>"
            @focus="onInputFocus"
          />
        </div>
        <div class="st-vars">
          <span class="st-vars__label">{{ t('system.settings.tplVars') }} · {{ t('system.settings.tplVarsHint') }}</span>
          <button v-for="v in editorVars" :key="v" type="button" class="st-var st-var--btn m-mono" @click="insertPlaceholder(tokenOf(v))">
            {{ tokenOf(v) }}
          </button>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton class="st-foot-left" color="neutral" variant="ghost" icon="i-lucide-eye" @click="openPreview({ subject: templateForm.subject, template: templateForm.template })">
          {{ t('system.settings.preview') }}
        </UButton>
        <UButton color="neutral" variant="soft" icon="i-lucide-send" :disabled="!templateForm.subject && !templateForm.template" @click="openTemplateTestDialog">
          {{ t('system.settings.testSend') }}
        </UButton>
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.cancel') }}</UButton>
        <UButton color="primary" :loading="templateSaving" @click="handleTemplateSave">{{ t('system.settings.saveTemplate') }}</UButton>
      </template>
    </FormDialog>

    <!-- 模板测试发送 -->
    <FormDialog v-model="templateTestDialogVisible" :title="t('system.settings.testTemplateEmailTitle')" :width="440">
      <div class="st-form">
        <div class="app-field">
          <label class="app-label">{{ t('system.settings.recipientEmail') }}</label>
          <input v-model.trim="templateTestRecipient" class="app-input" :placeholder="t('system.settings.recipientEmailPlaceholder')" />
        </div>
        <div v-for="field in templateTestFields" :key="field.name" class="app-field">
          <label class="app-label m-mono">{{ tokenOf(field.name) }}</label>
          <input v-model="field.value" class="app-input" :placeholder="t('system.settings.templateFieldPlaceholder', { placeholder: tokenOf(field.name) })" />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.cancel') }}</UButton>
        <UButton color="primary" :loading="templateTesting" :disabled="!templateTestRecipient" @click="handleTemplateTest">
          {{ t('system.settings.send') }}
        </UButton>
      </template>
    </FormDialog>

    <!-- 模板预览 -->
    <FormDialog v-model="previewDialogVisible" :title="t('system.settings.templatePreviewTitle')" :width="760">
      <div class="st-preview">
        <div class="st-preview__subject">
          <span class="st-preview__label">{{ t('system.settings.subjectLabel') }}</span>
          <span>{{ renderedPreviewSubject }}</span>
        </div>
        <iframe :srcdoc="renderedPreviewBody" class="st-preview__iframe" sandbox="allow-same-origin" />
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.close') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getLoginConfig, updateLoginConfig } from '@/api/login'
import {
  applyUpdate,
  cancelUpdate,
  getEmailConfig,
  getSiteConfig,
  getUpdateStatus,
  pauseDownload,
  resumeDownload,
  startUpdate,
  uploadUpdate,
  getStorageConfig,
  listEmailTemplates,
  testEmailConfig,
  updateEmailConfig,
  updateEmailTemplate,
  updateSiteConfig,
  updateStorageConfig,
} from '@/api/system'
import { DownloadState, EmailEncryption, EmailTemplateType, UpdateChannel, UpdateSource, UpdateState } from '@/types/v1/system'
import type { EmailConfig, EmailTemplate, EmailTestResult, SiteConfig, StorageConfig, StorageSource, UpdateStatus } from '@/types/v1/system'
import { progressText, percentOf, refreshDownloads } from '@/layouts/app/downloads/useDownloads'
import { formatFileSize } from '@/utils/file'
import { Dialog } from '@/utils/dialog'
import { pageTabTo } from '@/config/pageGroups'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { checkForUpdate } from '@/utils/updateCheck'
import { getRequestErrorMessage, showRequestError } from '@/utils/request'
import { isRemoteNode } from '@/utils/nodeContext'
import { probeGameAssets, setGameAssetBase } from '@/games/core'

defineOptions({ name: 'SystemSettingsView' })

const { t } = useI18n()
const route = useRoute()
const menuStore = useMenuStore()
const userStore = useUserStore()

const canEdit = useButtonPermission([PERM.SYSTEM_CONFIG_EDIT], [])
const canCheckUpdate = computed(() => menuStore.hasButtonPermission('system:update'))
const currentVersion = computed(() => menuStore.currentVersion || 'dev')

// 模板变量字面量（避免在模板插值里写 {{ 触发 Vue 解析）
const tokenOf = (name: string) => `{{.${name}}}`

/* ───────── 时间显示 ───────── */
const pad = (n: number) => String(n).padStart(2, '0')
const toDate = (v: unknown): Date | null => {
  if (!v) return null
  const d = v instanceof Date ? v : new Date(String(v))
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1971) return null
  return d
}
const fmtWhen = (v: unknown) => {
  const d = toDate(v)
  if (!d) return ''
  const now = new Date()
  const hm = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  if (d.toDateString() === now.toDateString()) return hm
  if (d.getFullYear() === now.getFullYear()) return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${hm}`
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}
const fmtDate = (v: unknown) => {
  const d = toDate(v)
  return d ? `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` : ''
}

/* ───────── 表单 ───────── */
type SiteForm = Pick<SiteConfig, 'siteName' | 'externalUrl' | 'defaultLanguage' | 'autoCheckUpdate' | 'updateChannel' | 'autoInstallUpdate'>
const siteForm = reactive<SiteForm>({
  siteName: '',
  externalUrl: '',
  defaultLanguage: 'auto',
  autoCheckUpdate: true,
  updateChannel: UpdateChannel.UpdateChannel_Stable,
  autoInstallUpdate: false,
})
const siteMeta = ref<SiteConfig | null>(null)

const loginForm = reactive({
  registerEnabled: false,
  usernameLoginEnabled: true,
  emailLoginEnabled: false,
  registerEmailVerificationRequired: false,
  accessTokenTtlMinutes: 60,
  refreshTokenTtlDays: 14,
  lockoutThreshold: 5,
  lockoutMinutes: 15,
})
type LoginSwitchKey = 'registerEnabled' | 'usernameLoginEnabled' | 'emailLoginEnabled' | 'registerEmailVerificationRequired'
const loginSwitches: { key: LoginSwitchKey; label: string; desc: string }[] = [
  { key: 'registerEnabled', label: 'system.settings.registerFeature', desc: 'system.settings.registerFeatureDesc' },
  { key: 'registerEmailVerificationRequired', label: 'system.settings.registerEmailVerification', desc: 'system.settings.registerEmailVerificationDesc' },
  { key: 'usernameLoginEnabled', label: 'system.settings.usernameLogin', desc: 'system.settings.usernameLoginDesc' },
  { key: 'emailLoginEnabled', label: 'system.settings.emailLogin', desc: 'system.settings.emailLoginDesc' },
]

type EmailForm = Omit<EmailConfig, 'lastTest'>
const emailForm = reactive<EmailForm>({
  enabled: false,
  host: '',
  port: 465,
  username: '',
  password: '',
  from: '',
  fromName: '',
  encryption: EmailEncryption.EmailEncryption_SSL,
  timeoutSeconds: 10,
  ccsN: 5,
})
const lastTest = ref<EmailTestResult | null>(null)
const showPwd = ref(false)

// 储存：头像、应用图标与背景存放到哪个对象存储来源（来源在文件管理 · 文件源里配置）
const storageForm = reactive({ sourceId: '', gameAssetUrl: '' })
const storageSources = ref<StorageSource[]>([])
const storageOptions = computed(() => [{ id: '', name: t('system.settings.storage.none') }, ...storageSources.value])

/* ───────── 分组修改记录 ───────── */
type SecKey = 'general' | 'login' | 'mail' | 'storage' | 'update'
const pickGeneral = () => ({ siteName: siteForm.siteName, externalUrl: siteForm.externalUrl, defaultLanguage: siteForm.defaultLanguage })
const pickUpdate = () => ({
  autoCheckUpdate: siteForm.autoCheckUpdate,
  updateChannel: siteForm.updateChannel,
  autoInstallUpdate: siteForm.autoInstallUpdate,
})
const snapshot = reactive<Record<SecKey, string>>({ general: '', login: '', mail: '', storage: '', update: '' })
const loaded = reactive<Record<SecKey, boolean>>({ general: false, login: false, mail: false, storage: false, update: false })
const current = computed<Record<SecKey, string>>(() => ({
  general: JSON.stringify(pickGeneral()),
  login: JSON.stringify({ ...loginForm }),
  mail: JSON.stringify({ ...emailForm }),
  storage: JSON.stringify({ ...storageForm }),
  update: JSON.stringify(pickUpdate()),
}))
const dirty = computed<Record<SecKey, boolean>>(() => {
  const out = {} as Record<SecKey, boolean>
  for (const k of ['general', 'login', 'mail', 'storage', 'update'] as SecKey[]) out[k] = loaded[k] && current.value[k] !== snapshot[k]
  return out
})
const dirtyCount = computed(() => Object.values(dirty.value).filter(Boolean).length)
const markClean = (k: SecKey) => {
  snapshot[k] = current.value[k]
  loaded[k] = true
}
const justSaved = ref(false)
// 保存成功的提示只停留片刻，之后保存条随之收起
let justSavedTimer: ReturnType<typeof setTimeout> | undefined
watch(justSaved, (v) => {
  clearTimeout(justSavedTimer)
  if (v) justSavedTimer = setTimeout(() => (justSaved.value = false), 2400)
})
onBeforeUnmount(() => clearTimeout(justSavedTimer))
watch(dirtyCount, (n) => {
  if (n > 0) justSaved.value = false
})

/* ───────── 目录 ───────── */
const sections = computed(() => [
  { key: 'general' as SecKey, label: t('system.settings.sec.general'), short: '', icon: 'i-lucide-settings' },
  { key: 'login' as SecKey, label: t('system.settings.sec.login'), short: '', icon: 'i-lucide-log-in' },
  { key: 'mail' as SecKey, label: t('system.settings.sec.mail'), short: '', icon: 'i-lucide-mail' },
  { key: 'storage' as SecKey, label: t('system.settings.sec.storage'), short: '', icon: 'i-lucide-hard-drive' },
  { key: 'update' as SecKey, label: t('system.settings.sec.update'), short: t('system.settings.sec.updateShort'), icon: 'i-lucide-download' },
])
const activeSec = ref<SecKey>('general')
const secGeneral = ref<HTMLElement>()
const secLogin = ref<HTMLElement>()
const secMail = ref<HTMLElement>()
const secStorage = ref<HTMLElement>()
const secUpdate = ref<HTMLElement>()
const secEls = () => ({
  general: secGeneral.value,
  login: secLogin.value,
  mail: secMail.value,
  storage: secStorage.value,
  update: secUpdate.value,
})
let lockSpyUntil = 0
const scrollToSec = (k: SecKey) => {
  activeSec.value = k
  lockSpyUntil = Date.now() + 800
  secEls()[k]?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
let observer: IntersectionObserver | null = null
const visibleSecs = new Map<SecKey, number>()
const setupSpy = () => {
  if (typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        const key = (e.target as HTMLElement).dataset.sec as SecKey
        if (e.isIntersecting) visibleSecs.set(key, e.boundingClientRect.top)
        else visibleSecs.delete(key)
      }
      if (Date.now() < lockSpyUntil || !visibleSecs.size) return
      const order: SecKey[] = ['general', 'login', 'mail', 'storage', 'update']
      const first = order.find((k) => visibleSecs.has(k))
      if (first) activeSec.value = first
    },
    { rootMargin: '-80px 0px -55% 0px', threshold: 0 },
  )
  const els = secEls()
  for (const [k, el] of Object.entries(els)) {
    if (!el) continue
    el.dataset.sec = k
    observer.observe(el)
  }
}

/* ───────── 选项 ───────── */
const languageOptions = computed(() => [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-HK', label: '繁體中文' },
  { value: 'en-US', label: 'English' },
  { value: 'auto', label: t('system.settings.langAuto') },
])
const encOptions = computed(() => [
  { value: EmailEncryption.EmailEncryption_SSL, label: 'SSL', port: 465, hint: 'ssl' },
  { value: EmailEncryption.EmailEncryption_STARTTLS, label: 'STARTTLS', port: 587, hint: 'starttls' },
  { value: EmailEncryption.EmailEncryption_NONE, label: t('system.settings.encNone'), port: 25, hint: 'none' },
])
const encHint = computed(() => {
  const opt = encOptions.value.find((e) => e.value === emailForm.encryption) || encOptions.value[0]
  return t(`system.settings.encHint.${opt?.hint || 'ssl'}`)
})
const DEFAULT_PORTS = [465, 587, 25]
const pickEncryption = (value: EmailEncryption) => {
  emailForm.encryption = value
  // 端口仍是某个默认端口时，跟着加密方式切换到对应的常用端口
  const opt = encOptions.value.find((e) => e.value === value)
  if (opt && (!emailForm.port || DEFAULT_PORTS.includes(Number(emailForm.port)))) emailForm.port = opt.port
}
const channelOptions = computed(() => [
  { value: UpdateChannel.UpdateChannel_Stable, label: t('system.settings.channelStable') },
  { value: UpdateChannel.UpdateChannel_Beta, label: t('system.settings.channelBeta') },
])

/* ───────── 状态文字 ───────── */
const lastModifiedText = computed(() => {
  const m = siteMeta.value?.lastModified
  const time = fmtWhen(m?.time)
  if (!time) return ''
  return m?.userName ? t('system.settings.lastModified', { time, user: m.userName }) : t('system.settings.lastModifiedNoUser', { time })
})

const mailStatus = computed<{ variant: 'success' | 'error' | 'neutral' | 'warning'; label: string }>(() => {
  if (!emailForm.enabled) return { variant: 'neutral', label: t('system.settings.mailStatus.disabled') }
  if (!lastTest.value) return { variant: 'warning', label: t('system.settings.mailStatus.untested') }
  return lastTest.value.success
    ? { variant: 'success', label: t('system.settings.mailStatus.ok') }
    : { variant: 'error', label: t('system.settings.mailStatus.failed') }
})
const lastTestState = computed(() => (!lastTest.value ? 'none' : lastTest.value.success ? 'ok' : 'bad'))
const lastTestText = computed(() => {
  const r = lastTest.value
  if (!r) return t('system.settings.lastTestNone')
  const time = fmtWhen(r.time)
  return r.success
    ? t('system.settings.lastTestOk', { time, email: r.recipient, ms: r.durationMs })
    : t('system.settings.lastTestFail', { time, email: r.recipient, error: r.error })
})

const lastCheck = computed(() => siteMeta.value?.lastUpdateCheck)
const checkText = computed(() => {
  const c = lastCheck.value
  const time = fmtWhen(c?.time)
  if (!c || !time) return t('system.settings.neverChecked')
  const result = c.error ? t('system.settings.checkFailed') : c.hasUpdate ? t('system.settings.hasUpdate', { version: c.latestVersion }) : t('system.settings.upToDate')
  const text = t('system.settings.lastCheck', { time, result })
  return c.auto ? `${text} · ${t('system.settings.autoTag')}` : text
})

/* ───────── 加载 ───────── */
const applySite = (cfg?: SiteConfig) => {
  if (!cfg) return
  siteMeta.value = cfg
  siteForm.siteName = cfg.siteName
  siteForm.externalUrl = cfg.externalUrl
  siteForm.defaultLanguage = cfg.defaultLanguage || 'auto'
  siteForm.autoCheckUpdate = cfg.autoCheckUpdate
  siteForm.updateChannel = cfg.updateChannel || UpdateChannel.UpdateChannel_Stable
  siteForm.autoInstallUpdate = cfg.autoInstallUpdate
  markClean('general')
  markClean('update')
}
const loadSiteConfig = async () => {
  try {
    const { data } = await getSiteConfig()
    applySite(data?.config)
  } catch (e) {
    showRequestError(e)
  }
}
/** 只刷新只读信息（上次修改 / 上次检查），不覆盖正在编辑的表单 */
const refreshSiteMeta = async () => {
  try {
    const { data } = await getSiteConfig()
    if (data?.config && siteMeta.value) {
      siteMeta.value = { ...siteMeta.value, lastModified: data.config.lastModified, lastUpdateCheck: data.config.lastUpdateCheck }
    }
  } catch {
    /* 只是展示信息，失败不打扰 */
  }
}

const loadLoginConfig = async () => {
  try {
    const { data } = await getLoginConfig()
    if (data?.config) {
      const c = data.config
      loginForm.registerEnabled = c.registerEnabled
      loginForm.usernameLoginEnabled = c.usernameLoginEnabled
      loginForm.emailLoginEnabled = c.emailLoginEnabled
      loginForm.registerEmailVerificationRequired = c.registerEmailVerificationRequired
      loginForm.accessTokenTtlMinutes = c.accessTokenTtlMinutes
      loginForm.refreshTokenTtlDays = c.refreshTokenTtlDays
      loginForm.lockoutThreshold = c.lockoutThreshold
      loginForm.lockoutMinutes = c.lockoutMinutes
      markClean('login')
    }
  } catch {
    feedback.error(t('system.settings.getLoginConfigFailed'))
  }
}

const applyEmail = (c?: EmailConfig) => {
  if (!c) return
  emailForm.enabled = c.enabled
  emailForm.host = c.host
  emailForm.port = c.port
  emailForm.username = c.username
  emailForm.password = c.password
  emailForm.from = c.from
  emailForm.fromName = c.fromName
  emailForm.encryption = c.encryption || EmailEncryption.EmailEncryption_SSL
  emailForm.timeoutSeconds = c.timeoutSeconds
  emailForm.ccsN = c.ccsN
  lastTest.value = c.lastTest || null
  markClean('mail')
}
const loadEmailConfig = async () => {
  try {
    const { data } = await getEmailConfig()
    applyEmail(data?.config)
  } catch {
    feedback.error(t('system.settings.getEmailConfigFailed'))
  }
}

const applyStorage = (c?: StorageConfig) => {
  if (!c) return
  storageSources.value = c.sources || []
  storageForm.sourceId = c.sourceId || ''
  storageForm.gameAssetUrl = c.gameAssetUrl || ''
  markClean('storage')
}

const assetExample = computed(() => `${(storageForm.gameAssetUrl || 'https://cdn.example.com/momoko-games').replace(/\/+$/, '')}/gakumas/…`)
const assetProbing = ref(false)
const assetProbe = ref<{ ok: boolean; text: string } | null>(null)
watch(
  () => storageForm.gameAssetUrl,
  () => (assetProbe.value = null),
)
const handleAssetProbe = async () => {
  const url = storageForm.gameAssetUrl
  if (!url || assetProbing.value) return
  if (!/^https?:\/\/[^/?#]+[^?#]*$/i.test(url)) {
    assetProbe.value = { ok: false, text: t('system.settings.storage.gameAssetsInvalid') }
    return
  }
  assetProbing.value = true
  try {
    const list = await probeGameAssets(url)
    if (url !== storageForm.gameAssetUrl) return
    const bad = list.find((p) => !p.ok)
    if (!list.length) assetProbe.value = { ok: false, text: t('system.settings.storage.gameAssetsNoGames') }
    else if (bad) assetProbe.value = { ok: false, text: t('system.settings.storage.gameAssetsBadText', { url: bad.url }) }
    else assetProbe.value = { ok: true, text: t('system.settings.storage.gameAssetsOkText', { games: list.map((p) => p.game).join(', ') }) }
  } finally {
    assetProbing.value = false
  }
}
const loadStorageConfig = async () => {
  try {
    const { data } = await getStorageConfig()
    applyStorage(data?.config)
  } catch {
    feedback.error(t('system.settings.storage.getFailed'))
  }
}

/* ───────── 保存 / 放弃 ───────── */
const saving = ref(false)
const saveAll = async () => {
  if (!dirtyCount.value || saving.value) return
  saving.value = true
  let ok = true
  try {
    if (dirty.value.general || dirty.value.update) {
      try {
        const { data } = await updateSiteConfig({ ...siteForm })
        applySite(data?.config)
      } catch (e) {
        ok = false
        showRequestError(e)
      }
    }
    if (dirty.value.login) {
      try {
        await updateLoginConfig({ ...loginForm })
        markClean('login')
      } catch (e) {
        ok = false
        showRequestError(e)
      }
    }
    if (dirty.value.mail) {
      try {
        const { data } = await updateEmailConfig({ ...emailForm })
        if (data?.config) applyEmail(data.config)
        else markClean('mail')
      } catch (e) {
        ok = false
        showRequestError(e)
      }
    }
    if (dirty.value.storage) {
      try {
        const { data } = await updateStorageConfig({ sourceId: storageForm.sourceId, gameAssetUrl: storageForm.gameAssetUrl })
        if (data?.config) {
          applyStorage(data.config)
          if (!isRemoteNode()) setGameAssetBase(data.config.gameAssetUrl)
        } else markClean('storage')
      } catch (e) {
        ok = false
        showRequestError(e)
      }
    }
    await refreshSiteMeta()
    if (ok) {
      justSaved.value = true
      feedback.success(t('system.common.saveSuccess'))
    }
  } finally {
    saving.value = false
  }
}
const discardChanges = () => {
  if (dirty.value.general) Object.assign(siteForm, JSON.parse(snapshot.general))
  if (dirty.value.update) Object.assign(siteForm, JSON.parse(snapshot.update))
  if (dirty.value.login) Object.assign(loginForm, JSON.parse(snapshot.login))
  if (dirty.value.mail) Object.assign(emailForm, JSON.parse(snapshot.mail))
  if (dirty.value.storage) Object.assign(storageForm, JSON.parse(snapshot.storage))
}

/* ───────── 测试邮件 ───────── */
const testRecipient = ref('')
const emailTesting = ref(false)
const refreshLastTest = async () => {
  try {
    const { data } = await getEmailConfig()
    lastTest.value = data?.config?.lastTest || lastTest.value
  } catch {
    /* 展示信息 */
  }
}
const handleEmailTest = async () => {
  if (!testRecipient.value) return
  emailTesting.value = true
  try {
    await testEmailConfig({
      recipient: testRecipient.value,
      config: { ...emailForm, lastTest: undefined },
      Data: {}, // 用当前填写的配置（不必先保存）发默认测试邮件
    })
    feedback.success(t('system.settings.emailTestSuccess'))
  } catch (e) {
    feedback.error(getRequestErrorMessage(e, t('system.settings.emailTestFailed')))
  } finally {
    emailTesting.value = false
    void refreshLastTest()
  }
}

/* ───────── 模板 ───────── */
const templates = ref<EmailTemplate[]>([])
const TEMPLATE_META: Record<string, { name: string; icon: string; sticker: string; vars: string[] }> = {
  [EmailTemplateType.EmailTemplateType_Register]: {
    name: 'system.settings.tplName.register',
    icon: 'i-lucide-user-plus',
    sticker: 'm-sticker--p',
    vars: ['code', 'email', 'minutes', 'site', 'url'],
  },
  [EmailTemplateType.EmailTemplateType_Login]: {
    name: 'system.settings.tplName.login',
    icon: 'i-lucide-key-round',
    sticker: 'm-sticker--a2',
    vars: ['code', 'name', 'email', 'minutes', 'site', 'url'],
  },
  [EmailTemplateType.EmailTemplateType_ResetPassword]: {
    name: 'system.settings.tplName.reset',
    icon: 'i-lucide-shield-check',
    sticker: 'm-sticker--a',
    vars: ['code', 'name', 'email', 'minutes', 'site', 'url'],
  },
}
const templateRows = computed(() => {
  const byType = new Map(templates.value.map((tp) => [tp.type, tp]))
  return Object.entries(TEMPLATE_META).map(([type, meta]) => {
    const tp = byType.get(type as EmailTemplateType)
    return {
      type: type as EmailTemplateType,
      name: t(meta.name),
      icon: meta.icon,
      sticker: meta.sticker,
      vars: meta.vars,
      subject: tp?.subject || '',
      template: tp?.template || '',
      customized: !!tp?.customized,
      updated: fmtDate(tp?.updateTime),
    }
  })
})
type TemplateRow = (typeof templateRows.value)[number]

const loadTemplates = async () => {
  try {
    const { data } = await listEmailTemplates()
    templates.value = data?.templates || []
  } catch (e) {
    showRequestError(e)
  }
}

const editorVisible = ref(false)
const editorType = ref<EmailTemplateType>(EmailTemplateType.EmailTemplateType_Register)
const editorName = computed(() => t(TEMPLATE_META[editorType.value]?.name || 'system.settings.templateConfig'))
const editorVars = computed(() => TEMPLATE_META[editorType.value]?.vars || [])
const templateForm = reactive({ subject: '', template: '' })
const templateSaving = ref(false)

const openTemplateEditor = (tp: TemplateRow) => {
  editorType.value = tp.type
  templateForm.subject = tp.subject
  templateForm.template = tp.template
  lastFocusedEl.value = null
  editorVisible.value = true
}
const handleTemplateSave = async () => {
  templateSaving.value = true
  try {
    await updateEmailTemplate({ type: editorType.value, subject: templateForm.subject, template: templateForm.template })
    feedback.success(t('system.common.saveSuccess'))
    editorVisible.value = false
    await Promise.all([loadTemplates(), refreshSiteMeta()])
  } catch (e) {
    showRequestError(e)
  } finally {
    templateSaving.value = false
  }
}

// 占位符插入到最近聚焦的输入框光标处
const lastFocusedEl = ref<HTMLInputElement | HTMLTextAreaElement | null>(null)
const onInputFocus = (e: FocusEvent) => {
  lastFocusedEl.value = e.target as HTMLInputElement | HTMLTextAreaElement
}
const insertPlaceholder = (text: string) => {
  const el = lastFocusedEl.value
  if (!el) {
    templateForm.template += text
    return
  }
  const start = el.selectionStart ?? el.value.length
  const end = el.selectionEnd ?? el.value.length
  el.value = el.value.slice(0, start) + text + el.value.slice(end)
  el.setSelectionRange(start + text.length, start + text.length)
  el.focus()
  el.dispatchEvent(new Event('input', { bubbles: true }))
}

const extractPlaceholders = (content: string): string[] => {
  const regex = /\{\{\.(\w+)\}\}/g
  const names = new Set<string>()
  let match: RegExpExecArray | null
  while ((match = regex.exec(content)) !== null) {
    if (match[1]) names.add(match[1])
  }
  return Array.from(names)
}
const renderTemplate = (source: string, data: Record<string, string>) =>
  source.replace(/\{\{\.(\w+)\}\}/g, (_, name: string) => data[name] || `{{.${name}}}`)
const highlightPlaceholders = (source: string) =>
  source.replace(
    /\{\{\.(\w+)\}\}/g,
    (_, name: string) => `<span style="background:#ffe58f;padding:0 3px;border-radius:2px;font-family:monospace;">${name}</span>`,
  )

interface TemplateField {
  name: string
  value: string
}
const templateTestFields = ref<TemplateField[]>([])
const templateTestDialogVisible = ref(false)
const templateTestRecipient = ref('')
const templateTesting = ref(false)

const openTemplateTestDialog = () => {
  const names = extractPlaceholders(`${templateForm.subject} ${templateForm.template}`)
  const prev = new Map(templateTestFields.value.map((f) => [f.name, f.value]))
  templateTestFields.value = names.map((name) => ({ name, value: prev.get(name) || '' }))
  templateTestRecipient.value = templateTestRecipient.value || testRecipient.value
  templateTestDialogVisible.value = true
}
const handleTemplateTest = async () => {
  const data: Record<string, string> = {}
  for (const field of templateTestFields.value) {
    if (field.value) data[field.name] = field.value
  }
  templateTesting.value = true
  try {
    await testEmailConfig({
      recipient: templateTestRecipient.value,
      config: undefined, // 使用已保存的邮件配置
      messages: {
        subject: templateForm.subject,
        template: templateForm.template,
        type: editorType.value,
        customized: false,
        updateTime: undefined,
      },
      Data: data,
    })
    feedback.success(t('system.settings.emailTestSuccess'))
    templateTestDialogVisible.value = false
  } catch (e) {
    feedback.error(getRequestErrorMessage(e, t('system.settings.emailTestFailed')))
  } finally {
    templateTesting.value = false
    void refreshLastTest()
  }
}

const previewDialogVisible = ref(false)
const renderedPreviewSubject = ref('')
const renderedPreviewBody = ref('')
const openPreview = (tp: { subject: string; template: string }) => {
  const data: Record<string, string> = {}
  for (const field of templateTestFields.value) data[field.name] = field.value
  renderedPreviewSubject.value = renderTemplate(tp.subject, data)
  renderedPreviewBody.value = highlightPlaceholders(tp.template)
  previewDialogVisible.value = true
}

/* ───────── 检查更新 ───────── */
const checking = ref(false)
// 最近一次手动检查时，最新发行版里有没有本机系统能用的包（没有就不给一键更新）
const checkedNoAsset = ref(false)
const handleCheckUpdate = async () => {
  if (!canCheckUpdate.value || checking.value) return
  checking.value = true
  try {
    // 弹窗里点「立即更新」已经确认过了，直接开始，进度显示在下面
    const data = await checkForUpdate({ onStartUpdate: () => void doStartUpdate() })
    checkedNoAsset.value = !!data?.hasUpdate && !data.canAutoUpdate
  } catch (e) {
    showRequestError(e)
  } finally {
    checking.value = false
    void refreshSiteMeta()
  }
}

/* ───────── 在线更新 ───────── */
const upd = ref<UpdateStatus | null>(null)
const updBusy = ref(false)
const uploading = ref(false)
const uploadPercent = ref(0)
const uploadInput = ref<HTMLInputElement>()
// 重启：记下当时的进程标识，换了就说明新程序起来了
const restartFrom = ref('')
const restartSince = ref(0)
const restartSlow = ref(false)
let pollTimer: ReturnType<typeof setTimeout> | undefined

type UpdPanel = 'uploading' | 'downloading' | 'ready' | 'installing' | 'restarting' | 'failed'
const updPanel = computed<UpdPanel | null>(() => {
  if (uploading.value) return 'uploading'
  switch (upd.value?.state) {
    case UpdateState.UpdateState_Downloading:
      return 'downloading'
    case UpdateState.UpdateState_Ready:
      return 'ready'
    case UpdateState.UpdateState_Installing:
      return 'installing'
    case UpdateState.UpdateState_Restarting:
      return 'restarting'
    case UpdateState.UpdateState_Failed:
      return 'failed'
    default:
      return null
  }
})
const updRunning = computed(() => ['uploading', 'downloading', 'installing', 'restarting'].includes(updPanel.value || ''))
const dlState = computed(() => upd.value?.download?.state)
const dlPaused = computed(() => dlState.value === DownloadState.DownloadState_Paused)
const canOneClickUpdate = computed(
  () => canCheckUpdate.value && !!lastCheck.value?.hasUpdate && !checkedNoAsset.value && !!upd.value?.supported && !updPanel.value,
)

const updPercent = computed(() => {
  if (uploading.value) return uploadPercent.value
  const d = upd.value?.download
  return percentOf(d?.downloaded ?? upd.value?.downloaded, d?.total ?? upd.value?.total) ?? 0
})
const updTone = computed(() => {
  switch (updPanel.value) {
    case 'failed':
      return 'bad'
    case 'ready':
      return 'ok'
    case 'restarting':
    case 'installing':
      return 'a'
    default:
      return dlPaused.value ? 'a2' : 'p'
  }
})
const updIcon = computed(() => {
  switch (updPanel.value) {
    case 'failed':
      return 'i-lucide-triangle-alert'
    case 'ready':
      return 'i-lucide-shield-check'
    case 'restarting':
    case 'installing':
      return 'i-lucide-loader-circle'
    case 'uploading':
      return 'i-lucide-upload'
    default:
      return dlPaused.value ? 'i-lucide-pause' : 'i-lucide-download'
  }
})
const updTitle = computed(() => {
  const s = upd.value
  switch (updPanel.value) {
    case 'uploading':
      // 传完后服务器还要解包、读构建信息、算 SHA256，几百兆要几秒
      return uploadPercent.value >= 100
        ? t('system.settings.updater.verifying')
        : t('system.settings.updater.uploading', { percent: uploadPercent.value })
    case 'downloading':
      if (s?.source === UpdateSource.UpdateSource_Upload) return t('system.settings.updater.receiving')
      if (dlPaused.value) return t('system.settings.updater.paused')
      if (dlState.value === DownloadState.DownloadState_Retrying) return t('system.settings.updater.retrying')
      return t('system.settings.updater.downloading', { version: s?.targetVersion })
    case 'ready':
      return t('system.settings.updater.ready')
    case 'installing':
      return t('system.settings.updater.installing')
    case 'restarting':
      return restartSlow.value ? t('system.settings.updater.restartSlow') : t('system.settings.updater.restarting')
    case 'failed':
      return t('system.settings.updater.failed')
    default:
      return ''
  }
})
const updSub = computed(() => {
  const s = upd.value
  if (!s) return ''
  if (updPanel.value === 'downloading' && s.download) {
    const d = s.download
    if (dlState.value === DownloadState.DownloadState_Retrying && d.error) return d.error
    if (dlPaused.value && !d.resumable) return t('downloads.noResume')
    return progressText(d)
  }
  const parts: string[] = []
  if (s.targetVersion && updPanel.value !== 'ready') parts.push(`${s.currentVersion} → ${s.targetVersion}`)
  const when = fmtWhen(s.startedAt)
  if (when) parts.push(when)
  if (s.operator) parts.push(t('downloads.by', { name: s.operator }))
  return parts.join(' · ')
})

const schedulePoll = (ms: number) => {
  clearTimeout(pollTimer)
  pollTimer = setTimeout(pollUpdate, ms)
}
const pollUpdate = async () => {
  const restarting = upd.value?.state === UpdateState.UpdateState_Restarting
  try {
    const { data } = await getUpdateStatus(restarting)
    const next = data?.status || null
    if (restarting && next && next.bootId !== restartFrom.value) {
      // 新程序起来了：刷新页面，用上新版本的前端
      feedback.success(t('system.settings.updater.done', { version: next.currentVersion }))
      setTimeout(() => window.location.reload(), 800)
      return
    }
    if (!restarting) upd.value = next
  } catch (e) {
    // 重启期间连不上是正常的，接着等；回答 404 说明服务已经起来了，只是新程序没有这个接口（比如换回了老版本）
    if (restarting && (e as { status?: number })?.status === 404) {
      setTimeout(() => window.location.reload(), 800)
      return
    }
  }
  watchUpdate()
}
/** 按当前状态决定要不要继续轮询 */
const watchUpdate = () => {
  const state = upd.value?.state
  if (state === UpdateState.UpdateState_Restarting) {
    if (!restartFrom.value) {
      restartFrom.value = upd.value?.bootId || ''
      restartSince.value = Date.now()
    }
    restartSlow.value = Date.now() - restartSince.value > 90_000
    schedulePoll(1500)
  } else if (state === UpdateState.UpdateState_Downloading || state === UpdateState.UpdateState_Installing) {
    schedulePoll(1000)
  } else {
    clearTimeout(pollTimer)
  }
}
const loadUpdateStatus = async () => {
  if (!canCheckUpdate.value) return
  try {
    const { data } = await getUpdateStatus(true)
    upd.value = data?.status || null
    watchUpdate()
  } catch {
    /* 没权限或连不上：不显示在线更新 */
  }
}
const setUpd = (status?: UpdateStatus) => {
  if (!status) return
  upd.value = status
  watchUpdate()
}

const doStartUpdate = async () => {
  updBusy.value = true
  try {
    const { data } = await startUpdate()
    setUpd(data?.status)
    feedback.success(t('system.settings.updater.started'))
    void refreshDownloads()
    scrollToSec('update')
  } catch (e) {
    showRequestError(e)
  } finally {
    updBusy.value = false
  }
}
const confirmStartUpdate = () => {
  if (!canCheckUpdate.value) return
  Dialog.confirm({
    title: t('system.settings.updater.confirmTitle', { version: lastCheck.value?.latestVersion || '' }),
    content: t('system.settings.updater.confirmContent', { platform: upd.value?.platform || '' }),
    confirmText: t('system.settings.updater.updateNow'),
    cancelText: t('system.settings.updater.cancel'),
    onConfirm: doStartUpdate,
  })
}
const retryUpdate = async () => {
  updBusy.value = true
  try {
    await cancelUpdate()
    const { data } = await startUpdate()
    setUpd(data?.status)
    void refreshDownloads()
  } catch (e) {
    showRequestError(e)
    void loadUpdateStatus()
  } finally {
    updBusy.value = false
  }
}
const toggleUpdatePause = async () => {
  const id = upd.value?.download?.id
  if (!id) return
  updBusy.value = true
  try {
    if (dlPaused.value) await resumeDownload(id)
    else await pauseDownload(id)
    await loadUpdateStatus()
    void refreshDownloads()
  } catch (e) {
    showRequestError(e)
  } finally {
    updBusy.value = false
  }
}
const handleCancelUpdate = async () => {
  updBusy.value = true
  try {
    const { data } = await cancelUpdate()
    setUpd(data?.status)
    void refreshDownloads()
  } catch (e) {
    showRequestError(e)
  } finally {
    updBusy.value = false
  }
}
const onPickUpdateFile = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || uploading.value) return
  uploading.value = true
  uploadPercent.value = 0
  try {
    const { data } = await uploadUpdate(file, (p) => (uploadPercent.value = p))
    setUpd(data?.status)
  } catch (err) {
    showRequestError(err)
    void loadUpdateStatus()
  } finally {
    uploading.value = false
  }
}
const confirmApplyUpdate = () => {
  Dialog.confirm({
    title: t('system.settings.updater.confirmInstallTitle'),
    content: t('system.settings.updater.confirmInstallContent'),
    confirmText: t('system.settings.updater.install'),
    cancelText: t('system.settings.updater.cancel'),
    onConfirm: async () => {
      updBusy.value = true
      try {
        const { data } = await applyUpdate()
        setUpd(data?.status)
      } catch (e) {
        showRequestError(e)
        void loadUpdateStatus()
      } finally {
        updBusy.value = false
      }
    },
  })
}
onBeforeUnmount(() => clearTimeout(pollTimer))

onMounted(async () => {
  testRecipient.value = userStore.userInfo?.email || ''
  setupSpy()
  await Promise.all([loadSiteConfig(), loadLoginConfig(), loadEmailConfig(), loadStorageConfig(), loadTemplates(), loadUpdateStatus()])
  // 从「发现新版本」弹窗等处跳过来：?sec=update 直接滚到版本更新
  if (route.query.sec === 'update') scrollToSec('update')
})
onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
})
</script>

<style scoped lang="scss">
.st-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.st-layout {
  display: grid;
  grid-template-columns: 208px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

/* ── 本页目录 ── */
.st-toc {
  position: sticky;
  top: 16px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.st-toc__label {
  padding: 0 12px 6px;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.st-toc__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.st-toc__item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 30px;
  padding: 0 12px;
  border: none;
  border-left: 3px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.st-toc__item:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.st-toc__item.is-on {
  background: var(--el-bg-color);
  border-left-color: var(--m-p);
  color: var(--el-text-color-primary);
  font-weight: 600;
  box-shadow: 0 1px 3px color-mix(in srgb, var(--m-ink) 10%, transparent);
}
.st-toc__ico {
  width: 16px;
  height: 16px;
  flex: none;
}
.st-toc__text {
  flex: 1;
  min-width: 0;
}
.st-toc__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--m-a);
  flex: none;
}
.st-toc__tip {
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--m-aSoft);
  color: var(--el-text-color-regular);
  font-size: 0.75rem;
  line-height: 1.55;
}

/* ── 主列 ── */
.st-main {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.st-readonly {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 500;
}

.st-card {
  padding: calc(var(--cmp-card-unit) * 22) calc(var(--cmp-card-unit) * 26) calc(var(--cmp-card-unit) * 8);
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  scroll-margin-top: 16px;
}
.st-card__head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 12px;
}
.st-card__sticker {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  transform: rotate(var(--rot, 0deg));
}
.st-card__sticker :deep(svg),
.st-card__sticker > span {
  width: 16px;
  height: 16px;
}
.st-card__title {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}
.st-card__title h2 {
  font-family: var(--app-font-display);
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.st-card__title span {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.st-card__pill {
  flex: none;
}

/* 设置行 */
.st-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 16px;
  padding: 12px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.st-row--switch {
  flex-wrap: nowrap;
  justify-content: space-between;
}
.st-row__l {
  flex: 1 1 260px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.st-row__t {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.st-row__d {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.st-row__c {
  flex: 1 1 360px;
  max-width: 480px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}
.st-row__c--col {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}
.st-row :deep(.app-switch) {
  flex: none;
}
.st-input {
  width: 100%;
}
.st-row .app-input {
  height: 32px;
  border-radius: 10px;
}
.st-hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.st-hint a {
  color: var(--m-p);
  font-weight: 500;
  text-decoration: none;
}
.st-hint a:hover {
  text-decoration: underline;
}

.st-flag {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 11.5px;
  font-weight: 500;
}
.st-flag--a2 {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.st-flag--warn {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}

/* 分段 */
.st-seg {
  display: inline-flex;
  max-width: 100%;
  padding: 3px;
  gap: 2px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.st-seg button {
  min-height: 26px;
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 12.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.st-seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font-weight: 600;
  box-shadow: 0 1px 3px color-mix(in srgb, var(--m-ink) 14%, transparent);
}
.st-seg button:disabled {
  cursor: not-allowed;
}
.st-seg button:disabled:not(.is-on) {
  opacity: 0.6;
}

/* 带单位的数字框 */
.st-unit {
  display: inline-flex;
  align-items: stretch;
  min-height: 32px;
  border-radius: 10px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow: hidden;
  transition: border-color 0.15s;
}
.st-unit:focus-within {
  border-color: var(--m-p);
}
.st-unit input {
  width: 72px;
  min-width: 0;
  flex: 1;
  padding: 0 10px;
  border: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
  outline: none;
}
.st-unit input:disabled {
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
}
.st-unit__u {
  display: flex;
  align-items: center;
  padding: 0 10px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  white-space: nowrap;
}
.st-unit--pre {
  flex: none;
}
.st-unit--pre input {
  width: 76px;
}

.st-pair {
  flex-wrap: wrap;
  gap: 12px;
}
.st-pair__item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.st-pair__k {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.st-host .app-input {
  flex: 1;
  min-width: 0;
}
.st-sender {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 3fr);
}

.st-pwd {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 32px;
  padding-right: 1px;
  border-radius: 10px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-fill-color-lighter);
  transition: border-color 0.15s;
}
.st-pwd:focus-within {
  border-color: var(--m-p);
  background: var(--el-bg-color);
}
.st-pwd input {
  flex: 1;
  min-width: 0;
  height: 29px;
  padding: 0 10px;
  border: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  outline: none;
}
.st-pwd__ico {
  width: 16px;
  height: 16px;
}

.st-test {
  display: flex;
  gap: 10px;
}
.st-test .app-input {
  flex: 1;
  min-width: 0;
}
.st-test :deep(button) {
  min-height: 32px;
  border-radius: 999px;
  padding-inline: 12px;
  font-weight: 500;
  flex: none;
}
.st-test__res {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.st-test__pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 20px;
  padding: 0 7px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 500;
}
.st-test__pill::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
.st-test__res.is-ok .st-test__pill {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.st-test__res.is-bad .st-test__pill {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.st-test__txt {
  min-width: 0;
  word-break: break-word;
}

/* 模板 */
.st-sub {
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 16px 0 4px;
  border-top: 1px solid var(--el-border-color-light);
}
.st-sub h3 {
  font-family: var(--app-font-display);
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.st-sub span {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.st-tpl {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 7px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.st-sub + .st-tpl {
  border-top: none;
}
.st-tpl__sticker {
  width: 30px;
  height: 30px;
  border-radius: 9px;
  transform: rotate(var(--rot, 0deg));
}
.st-tpl__sticker :deep(svg),
.st-tpl__sticker > span {
  width: 15px;
  height: 15px;
}
.st-tpl__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.st-tpl__name {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.st-tpl__subj {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.st-tpl__vars {
  display: flex;
  gap: 4px;
  flex: none;
}
.st-var {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 6px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 11.5px;
  font-weight: 500;
  white-space: nowrap;
}
.st-var--btn {
  height: 26px;
  border: none;
  cursor: pointer;
  transition: filter 0.15s;
}
.st-var--btn:hover {
  filter: brightness(0.95);
}
.st-tpl__date {
  flex: none;
  width: 84px;
  text-align: right;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.st-tpl__acts {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: none;
}
.st-tpl__ico {
  width: 16px;
  height: 16px;
}
.st-tpl__edit {
  min-height: 28px;
  border-radius: 999px;
  font-weight: 500;
}

/* 版本卡（主色实底 + 黄色圆 + 紫色圆环） */
.st-ver {
  position: relative;
  overflow: hidden;
  margin-bottom: 8px;
  padding: 16px 14px;
  border-radius: var(--app-radius-lg);
  background: var(--m-p);
  color: var(--m-pOn);
}
.st-ver__blob {
  position: absolute;
  right: -40px;
  bottom: -90px;
  width: 190px;
  height: 190px;
  border-radius: 50%;
  background: var(--m-a);
}
.st-ver__ring {
  position: absolute;
  right: 170px;
  top: -40px;
  width: 92px;
  height: 92px;
  border-radius: 50%;
  border: 14px solid var(--m-a2);
}
.st-ver__body {
  position: relative;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}
.st-ver__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.st-ver__k {
  font-size: 0.75rem;
  font-weight: 500;
  opacity: 0.9;
}
.st-ver__v {
  font-family: var(--app-font-display);
  font-size: 22px;
  line-height: 1.2;
  font-weight: 600;
  word-break: break-all;
}
.st-ver__s {
  font-size: 0.75rem;
  opacity: 0.95;
}
.st-ver__link {
  margin-left: 6px;
  color: inherit;
  font-weight: 500;
  text-decoration: underline;
}
.st-ver__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  box-shadow: 0 2px 8px color-mix(in srgb, var(--m-ink) 18%, transparent);
}
.st-ver__btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
.st-ver__btns {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
/* 一键更新：和版本卡的主色区分开，最显眼 */
.st-ver__btn--go {
  background: var(--m-a);
  color: var(--m-aOn);
}

/* 在线更新：进度 / 等待安装 / 失败 */
.st-upd-note {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 8px 0;
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.st-upd-note > :first-child {
  flex: none;
  width: 16px;
  height: 16px;
  margin-top: 2px;
}
.st-upd-note.is-warn {
  color: var(--m-warn);
}
.st-upd {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 8px 0 12px;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--m-s2);
}
.st-upd.is-failed {
  border-color: color-mix(in srgb, var(--m-bad) 35%, transparent);
}
.st-upd__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.st-upd__sticker {
  flex: none;
  width: 30px;
  height: 30px;
  border-radius: 9px;
}
.st-upd__sticker :deep(svg),
.st-upd__sticker :deep(.iconify) {
  width: 15px;
  height: 15px;
}
.st-upd__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.st-upd__title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.st-upd__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.st-upd__acts {
  display: flex;
  flex: none;
  gap: 2px;
}
.st-upd__bar {
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--m-line);
}
.st-upd__fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-p);
  transition: width 0.4s ease;
}
.st-upd__fill.is-paused {
  background: var(--el-text-color-placeholder);
}
.st-upd__pkg {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 4px 16px;
  margin: 0;
  font-size: 0.75rem;
}
.st-upd__pkg dt {
  color: var(--el-text-color-secondary);
}
.st-upd__pkg dd {
  margin: 0;
  min-width: 0;
  color: var(--el-text-color-primary);
}
.st-upd__hash {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.st-upd__err {
  margin: 0;
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--m-bad);
  word-break: break-word;
}
.st-upd__foot {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}
.st-upd__file {
  display: none;
}

.is-spin {
  animation: st-spin 0.9s linear infinite;
}
@keyframes st-spin {
  to {
    transform: rotate(360deg);
  }
}

/* 保存条 */
.savebar-enter-active,
.savebar-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.savebar-enter-from,
.savebar-leave-to {
  opacity: 0;
  transform: translateY(16px);
}
.st-bar {
  position: sticky;
  bottom: 16px;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 6px 6px 16px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: 0 10px 30px color-mix(in srgb, var(--m-ink) 28%, transparent);
}
.st-bar__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--m-ok);
  flex: none;
}
.st-bar.is-dirty .st-bar__dot {
  background: var(--m-a);
}
.st-bar__text {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.st-bar__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: opacity 0.15s, background 0.15s;
}
.st-bar__btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.st-bar__btn--ghost {
  background: transparent;
  color: var(--m-bg);
}
.st-bar__btn--ghost:hover:not(:disabled) {
  background: color-mix(in srgb, var(--m-bg) 12%, transparent);
}
.st-bar__btn--primary {
  background: var(--m-p);
  color: var(--m-pOn);
}

.st-tpl__mvars {
  display: none;
  flex-wrap: wrap;
  gap: 4px;
}

/* 弹窗 */
.st-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.st-code {
  width: 100%;
  height: auto;
  padding: 10px 12px;
  font-family: var(--app-font-mono);
  font-size: 12.5px;
  line-height: 1.5;
}
.st-vars {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.st-vars__label {
  width: 100%;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.st-foot-left {
  margin-right: auto;
}
.st-preview__subject {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.st-preview__label {
  font-weight: 600;
  flex-shrink: 0;
}
.st-preview__iframe {
  width: 100%;
  height: 420px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
  background: var(--el-bg-color);
}

/* ── 平板：目录收成横向 chips ── */
@media (width <= 1100px) {
  .st-layout {
    grid-template-columns: minmax(0, 1fr);
    gap: 14px;
  }
  .st-toc {
    top: 0;
    z-index: 15;
    margin: 0 -4px;
    padding: 6px 4px;
    background: var(--el-bg-color-page, var(--m-bg));
  }
  .st-toc__label,
  .st-toc__tip {
    display: none;
  }
  .st-toc__list {
    flex-direction: row;
    gap: 8px;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .st-toc__list::-webkit-scrollbar {
    display: none;
  }
  .st-toc__item {
    flex: none;
    min-height: 40px;
    padding: 0 16px;
    gap: 8px;
    border-left: none;
    border-radius: 999px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
  }
  .st-toc__item.is-on {
    background: var(--m-ink);
    border-color: var(--m-ink);
    color: var(--m-bg);
    box-shadow: none;
  }
  .st-card {
    scroll-margin-top: 64px;
  }
}

/* ── 手机 ── */
@media (width <= 768px) {
  .st-page {
    gap: 12px;
  }
  .st-toc__item {
    min-height: 40px;
  }
  .st-toc__ico {
    display: none;
  }
  .st-card {
    padding: 14px 16px 4px;
    border-radius: var(--app-radius-lg);
  }
  .st-card__head {
    padding-bottom: 12px;
  }
  .st-card__title span {
    display: none;
  }
  .st-card__sticker {
    width: 30px;
    height: 30px;
    border-radius: 12px;
  }
  .st-row {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    padding: 14px 0;
  }
  .st-row--switch {
    flex-direction: row;
    align-items: center;
    gap: 14px;
  }
  .st-row__l {
    flex: none;
  }
  .st-row--switch .st-row__l {
    flex: 1 1 auto;
  }
  .st-row__c {
    flex: none;
    max-width: none;
  }
  .st-seg {
    display: flex;
    width: 100%;
  }
  .st-seg button {
    flex: 1;
    min-height: 40px;
    padding: 0 10px;
  }
  .st-seg--wrap {
    display: grid;
    grid-template-columns: 1fr 1fr;
    border-radius: 14px;
  }
  .st-seg--wrap button {
    border-radius: 14px;
  }
  .st-pair {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .st-pair__item {
    flex-direction: column;
    align-items: stretch;
    gap: 6px;
  }
  .st-pair__item .st-unit input {
    width: 0;
  }
  .st-sender {
    grid-template-columns: minmax(0, 1fr);
  }
  .st-test {
    flex-direction: column;
  }
  .st-tpl {
    gap: 12px;
  }
  .st-tpl__vars,
  .st-tpl__date,
  .st-tpl__edit-t,
  .st-tpl__subj {
    display: none;
  }
  .st-tpl__mvars {
    display: flex;
  }
  .st-tpl__edit {
    width: 40px;
    min-height: 40px;
    justify-content: center;
    padding: 0;
  }
  .st-ver {
    padding: 14px;
  }
  .st-ver__ring {
    right: 90px;
  }
  .st-ver__v {
    font-size: 1.25rem;
  }
  .st-ver__btns {
    width: 100%;
    flex-direction: column;
  }
  .st-ver__btn {
    width: 100%;
    justify-content: center;
  }
  .st-bar {
    bottom: calc(12px + env(safe-area-inset-bottom));
    padding: 8px 8px 8px 16px;
    gap: 6px;
  }
  .st-bar__btn {
    min-height: 40px;
    padding: 0 14px;
  }
  .st-preview__iframe {
    height: 280px;
  }
}
</style>
