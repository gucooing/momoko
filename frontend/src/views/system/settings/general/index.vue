<!-- 系统设置（第二期）：标题(作用范围 · 上次修改) → 子导航(系统设置 / OIDC) → 左侧本页目录（手机为横向 chips）+
     右侧分组卡片：常规 / 登录 / 邮件（含测试与模板列表）/ 版本更新 → 底部吸附保存条（按分组记录修改，一次保存）。
     模板编辑、模板测试发送、预览用 FormDialog。权限：PERM.SYSTEM_CONFIG_EDIT 控制编辑，system:update 控制检查更新。 -->
<template>
  <div class="st-page">
    <PageHeader :title="t('system.settings.pageTitle')">
      <template #description>
        {{ t('system.settings.scope') }}
        <template v-if="lastModifiedText"> · {{ lastModifiedText }}</template>
      </template>
    </PageHeader>

    <SettingsSubNav />

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

          <div class="st-row st-row--switch">
            <div class="st-row__l">
              <span class="st-row__t">{{ t('system.settings.autoCheck') }}</span>
              <span class="st-row__d">{{ t('system.settings.autoCheckDesc') }}</span>
            </div>
            <AppSwitch v-model="siteForm.autoCheckUpdate" :disabled="!canEdit" :aria-label="t('system.settings.autoCheck')" />
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
import SettingsSubNav from '@/views/system/components/SettingsSubNav.vue'
import { getLoginConfig, updateLoginConfig } from '@/api/login'
import {
  getEmailConfig,
  getSiteConfig,
  listEmailTemplates,
  testEmailConfig,
  updateEmailConfig,
  updateEmailTemplate,
  updateSiteConfig,
} from '@/api/system'
import { EmailEncryption, EmailTemplateType, UpdateChannel } from '@/types/v1/system'
import type { EmailConfig, EmailTemplate, EmailTestResult, SiteConfig } from '@/types/v1/system'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { checkForUpdate } from '@/utils/updateCheck'
import { getRequestErrorMessage, showRequestError } from '@/utils/request'

defineOptions({ name: 'SystemSettingsView' })

const { t } = useI18n()
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
type SiteForm = Pick<SiteConfig, 'siteName' | 'externalUrl' | 'defaultLanguage' | 'autoCheckUpdate' | 'updateChannel'>
const siteForm = reactive<SiteForm>({
  siteName: '',
  externalUrl: '',
  defaultLanguage: 'auto',
  autoCheckUpdate: true,
  updateChannel: UpdateChannel.UpdateChannel_Stable,
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

/* ───────── 分组修改记录 ───────── */
type SecKey = 'general' | 'login' | 'mail' | 'update'
const pickGeneral = () => ({ siteName: siteForm.siteName, externalUrl: siteForm.externalUrl, defaultLanguage: siteForm.defaultLanguage })
const pickUpdate = () => ({ autoCheckUpdate: siteForm.autoCheckUpdate, updateChannel: siteForm.updateChannel })
const snapshot = reactive<Record<SecKey, string>>({ general: '', login: '', mail: '', update: '' })
const loaded = reactive<Record<SecKey, boolean>>({ general: false, login: false, mail: false, update: false })
const current = computed<Record<SecKey, string>>(() => ({
  general: JSON.stringify(pickGeneral()),
  login: JSON.stringify({ ...loginForm }),
  mail: JSON.stringify({ ...emailForm }),
  update: JSON.stringify(pickUpdate()),
}))
const dirty = computed<Record<SecKey, boolean>>(() => {
  const out = {} as Record<SecKey, boolean>
  for (const k of ['general', 'login', 'mail', 'update'] as SecKey[]) out[k] = loaded[k] && current.value[k] !== snapshot[k]
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
  { key: 'update' as SecKey, label: t('system.settings.sec.update'), short: t('system.settings.sec.updateShort'), icon: 'i-lucide-download' },
])
const activeSec = ref<SecKey>('general')
const secGeneral = ref<HTMLElement>()
const secLogin = ref<HTMLElement>()
const secMail = ref<HTMLElement>()
const secUpdate = ref<HTMLElement>()
const secEls = () => ({ general: secGeneral.value, login: secLogin.value, mail: secMail.value, update: secUpdate.value })
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
      const order: SecKey[] = ['general', 'login', 'mail', 'update']
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
const handleCheckUpdate = async () => {
  if (!canCheckUpdate.value || checking.value) return
  checking.value = true
  try {
    await checkForUpdate()
  } catch (e) {
    showRequestError(e)
  } finally {
    checking.value = false
    void refreshSiteMeta()
  }
}

onMounted(async () => {
  testRecipient.value = userStore.userInfo?.email || ''
  setupSpy()
  await Promise.all([loadSiteConfig(), loadLoginConfig(), loadEmailConfig(), loadTemplates()])
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
  grid-template-columns: 228px minmax(0, 1fr);
  gap: 24px;
  align-items: start;
}

/* ── 本页目录 ── */
.st-toc {
  position: sticky;
  top: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.st-toc__label {
  padding: 0 14px 6px;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--el-text-color-secondary);
}
.st-toc__list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.st-toc__item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 46px;
  padding: 0 14px;
  border: none;
  border-left: 3px solid transparent;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
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
  font-weight: 800;
  box-shadow: 0 1px 3px color-mix(in srgb, var(--m-ink) 10%, transparent);
}
.st-toc__ico {
  width: 18px;
  height: 18px;
  flex: none;
}
.st-toc__text {
  flex: 1;
  min-width: 0;
}
.st-toc__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--m-a);
  flex: none;
}
.st-toc__tip {
  margin-top: 14px;
  padding: 12px 14px;
  border-radius: 14px;
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
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 600;
}

.st-card {
  padding: 22px 26px 8px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  scroll-margin-top: 16px;
}
.st-card__head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 16px;
}
.st-card__sticker {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  transform: rotate(var(--rot, 0deg));
}
.st-card__sticker :deep(svg),
.st-card__sticker > span {
  width: 22px;
  height: 22px;
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
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.st-card__title span {
  font-size: 0.8125rem;
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
  gap: 12px 32px;
  padding: 18px 0;
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
  gap: 3px;
}
.st-row__t {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.st-row__d {
  font-size: 0.8125rem;
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
  height: 44px;
  border-radius: 14px;
}
.st-hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

.st-flag {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.6875rem;
  font-weight: 700;
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
  padding: 4px;
  gap: 2px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.st-seg button {
  min-height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.st-seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
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
  min-height: 44px;
  border-radius: 14px;
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
  padding: 0 12px;
  border: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
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
  padding: 0 12px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
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
  font-size: 0.8125rem;
  font-weight: 600;
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
  min-height: 44px;
  padding-right: 2px;
  border-radius: 14px;
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
  height: 41px;
  padding: 0 12px;
  border: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.875rem;
  outline: none;
}
.st-pwd__ico {
  width: 18px;
  height: 18px;
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
  min-height: 44px;
  border-radius: 999px;
  padding-inline: 16px;
  font-weight: 700;
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
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  font-weight: 700;
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
  padding: 22px 0 6px;
  border-top: 1px solid var(--el-border-color-light);
}
.st-sub h3 {
  font-family: var(--app-font-display);
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.st-sub span {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.st-tpl {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.st-sub + .st-tpl {
  border-top: none;
}
.st-tpl__sticker {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  transform: rotate(var(--rot, 0deg));
}
.st-tpl__sticker :deep(svg),
.st-tpl__sticker > span {
  width: 19px;
  height: 19px;
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
  font-size: 0.875rem;
  font-weight: 700;
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
  height: 22px;
  padding: 0 7px;
  border-radius: 7px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 0.6875rem;
  font-weight: 600;
  white-space: nowrap;
}
.st-var--btn {
  height: 28px;
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
  width: 18px;
  height: 18px;
}
.st-tpl__edit {
  min-height: 36px;
  border-radius: 999px;
  font-weight: 700;
}

/* 版本卡（主色实底 + 黄色圆 + 紫色圆环） */
.st-ver {
  position: relative;
  overflow: hidden;
  margin-bottom: 8px;
  padding: 22px 24px;
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
  gap: 16px;
}
.st-ver__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.st-ver__k {
  font-size: 0.8125rem;
  font-weight: 600;
  opacity: 0.9;
}
.st-ver__v {
  font-family: var(--app-font-display);
  font-size: 2rem;
  line-height: 1.1;
  font-weight: 800;
  word-break: break-all;
}
.st-ver__s {
  font-size: 0.8125rem;
  opacity: 0.95;
}
.st-ver__link {
  margin-left: 6px;
  color: inherit;
  font-weight: 700;
  text-decoration: underline;
}
.st-ver__btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 18px;
  border: none;
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
  box-shadow: 0 2px 8px color-mix(in srgb, var(--m-ink) 18%, transparent);
}
.st-ver__btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
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
  bottom: 18px;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 10px 10px 20px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: 0 10px 30px color-mix(in srgb, var(--m-ink) 28%, transparent);
}
.st-bar__dot {
  width: 9px;
  height: 9px;
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
  font-size: 0.875rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.st-bar__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 42px;
  padding: 0 18px;
  border: none;
  border-radius: 999px;
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
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
  gap: 14px;
}
.st-code {
  width: 100%;
  height: auto;
  padding: 10px 12px;
  font-family: var(--app-font-mono);
  font-size: 0.8125rem;
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
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
}
.st-preview__label {
  font-weight: 700;
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
    min-height: 44px;
  }
  .st-toc__ico {
    display: none;
  }
  .st-card {
    padding: 18px 16px 4px;
    border-radius: var(--app-radius-lg);
  }
  .st-card__head {
    padding-bottom: 12px;
  }
  .st-card__title span {
    display: none;
  }
  .st-card__sticker {
    width: 38px;
    height: 38px;
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
    min-height: 44px;
    padding: 0 10px;
  }
  .st-seg--wrap {
    display: grid;
    grid-template-columns: 1fr 1fr;
    border-radius: 18px;
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
    padding: 18px;
  }
  .st-ver__ring {
    right: 90px;
  }
  .st-ver__v {
    font-size: 1.625rem;
  }
  .st-ver__btn {
    width: 100%;
    justify-content: center;
  }
  .st-bar {
    bottom: calc(78px + env(safe-area-inset-bottom));
    padding: 8px 8px 8px 16px;
    gap: 6px;
  }
  .st-bar__btn {
    min-height: 44px;
    padding: 0 14px;
  }
  .st-preview__iframe {
    height: 280px;
  }
}
</style>
