<!-- 创建 / 重建容器（容器标签内的子视图：/docker?tab=container&create=1 或 &recreate=<id>）。
     PC：返回 + 面包屑 + 标题 +「从 Compose 导入」→ 左列吸顶步骤条 + 6 个分区（基本 / 端口 / 存储 / 环境变量 / 网络 / 资源与重启策略）
         + 底部吸底就绪条；右列吸顶摘要（名称镜像 · 6 项摘要 · 等效 docker run 命令可复制）。
     手机：分步向导（第 N 步 · 进度条 · 步骤 chips）+ 底部 上一步 / 下一步（最后一步「创建并启动」）。
     本地没有镜像时后端先拉取再创建，进度显示在后台任务里。 -->
<template>
  <div class="ce" :class="{ 'is-mobile': isMobile }">
    <!-- 头部 -->
    <header class="ce-head">
      <AppIconButton class="ce-back" icon="HOutline:ChevronLeftIcon" :label="t('docker.wizard.backToList')" :box="44" :size="22" @click="emit('back')" />
      <div class="ce-head__text">
        <nav v-if="!isMobile" class="ce-crumb" :aria-label="t('docker.wizard.breadcrumb')">
          <button type="button" @click="emit('back')">{{ t('pageGroup.docker.label') }}</button> /
          <button type="button" @click="emit('back')">{{ t('pageGroup.docker.container') }}</button> /
          {{ isRecreate ? t('docker.wizard.crumbRecreate') : t('docker.wizard.crumbCreate') }}
        </nav>
        <h1 class="ce-title">{{ isRecreate ? t('docker.wizard.recreateTitle', { name: recreateName || '' }) : t('docker.container.createContainer') }}</h1>
        <span v-if="isMobile" class="ce-stepline m-num">{{ t('docker.wizard.stepOf', { n: stepIndex + 1, total: STEPS.length, name: stepLabel(STEPS[stepIndex]) }) }}</span>
      </div>
      <UButton v-if="!isRecreate && !isMobile" color="neutral" variant="outline" icon="i-lucide-file-code-2" class="ce-pill-btn" @click="composeOpen = true">
        {{ t('docker.wizard.importCompose') }}
      </UButton>
      <AppIconButton v-else-if="!isRecreate" icon="HOutline:DocumentArrowUpIcon" :label="t('docker.wizard.importCompose')" :box="44" @click="composeOpen = true" />
    </header>

    <template v-if="isMobile">
      <div class="ce-progress" aria-hidden="true"><span :style="{ width: `${Math.round(((stepIndex + 1) / STEPS.length) * 100)}%` }" /></div>
      <nav class="ce-mchips" :aria-label="t('docker.wizard.steps')">
        <button
          v-for="(s, i) in STEPS"
          :key="s"
          type="button"
          class="ce-mchip"
          :class="{ 'is-on': i === stepIndex, 'is-done': i !== stepIndex && doneOf(s) }"
          :aria-current="i === stepIndex ? 'step' : undefined"
          @click="stepIndex = i"
        >
          <span class="ce-sn m-num">
            <UIcon v-if="i !== stepIndex && doneOf(s)" name="i-lucide-check" />
            <template v-else>{{ i + 1 }}</template>
          </span>
          {{ stepLabel(s) }}
        </button>
      </nav>
    </template>

    <div v-if="loadingSource" class="ce-loading">{{ t('system.common.loading') }}</div>

    <div v-else class="ce-cols">
      <div class="ce-main">
        <!-- PC 吸顶步骤条 -->
        <nav v-if="!isMobile" class="ce-steps" :aria-label="t('docker.wizard.steps')">
          <button
            v-for="(s, i) in STEPS"
            :key="s"
            type="button"
            class="ce-stp"
            :class="{ 'is-on': s === activeStep, 'is-done': s !== activeStep && doneOf(s) }"
            :aria-current="s === activeStep ? 'step' : undefined"
            @click="scrollTo(s)"
          >
            <span class="ce-sn m-num">
              <UIcon v-if="s !== activeStep && doneOf(s)" name="i-lucide-check" />
              <template v-else>{{ i + 1 }}</template>
            </span>
            {{ stepLabel(s) }}
          </button>
        </nav>

        <!-- 1 基本 -->
        <section v-show="showStep('basic')" :ref="(el) => setSecRef('basic', el)" class="ce-sec" :class="{ 'is-on': secOn('basic') }" data-step="basic" aria-labelledby="ce-h-basic">
          <div class="ce-sec__head">
            <span v-if="!isMobile" class="ce-badge m-sticker m-sticker--p m-num" style="--rot: -4deg">1</span>
            <div class="ce-sec__title">
              <h2 id="ce-h-basic">{{ isMobile ? t('docker.wizard.pickImage') : t('docker.wizard.secBasic') }}</h2>
              <p>{{ isMobile ? t('docker.wizard.pickImageDesc') : t('docker.wizard.secBasicDesc') }}</p>
            </div>
          </div>

          <div class="ce-fl">
            <label for="ce-img">{{ t('docker.common.image') }}</label>
            <span class="ce-fld" :class="{ 'is-focus': sugOpen }">
              <UIcon name="i-lucide-search" />
              <input
                id="ce-img"
                ref="imgInput"
                v-model="query"
                class="m-mono"
                role="combobox"
                autocomplete="off"
                :aria-expanded="sugOpen"
                aria-controls="ce-sug"
                :placeholder="t('docker.wizard.imagePlaceholder')"
                @focus="sugOpen = true"
                @input="sugOpen = true"
                @keydown.enter.prevent="pickTyped"
                @keydown.esc="sugOpen = false"
              />
              <span v-if="!isMobile" class="ce-kbd">{{ t('docker.wizard.imageSources') }}</span>
            </span>
            <div v-if="sugOpen" id="ce-sug" class="ce-sug" role="listbox" :aria-label="t('docker.wizard.suggestions')">
              <button
                v-for="g in suggestions"
                :key="g.ref"
                type="button"
                role="option"
                class="ce-sug__item"
                :class="{ 'is-on': g.ref === form.image }"
                :aria-selected="g.ref === form.image"
                @mousedown.prevent
                @click="pickImage(g.ref)"
              >
                <span class="ce-sug__av m-sticker" :class="g.local ? 'm-sticker--ok' : 'm-sticker--a2'">{{ g.letter }}</span>
                <span class="ce-sug__text">
                  <span class="ce-sug__ref m-mono">{{ g.ref }}</span>
                  <span class="ce-sug__desc">{{ g.desc }}</span>
                </span>
                <span v-if="g.badge && g.ref !== form.image" class="ce-bdg" :class="`is-${g.badgeTone}`">{{ g.badge }}</span>
                <UIcon v-if="g.ref === form.image" name="i-lucide-check" class="ce-sug__check" />
              </button>
              <div v-if="searching && !suggestions.length" class="ce-sug__empty">{{ t('system.common.loading') }}</div>
              <div v-else-if="!suggestions.length" class="ce-sug__empty">{{ t('docker.wizard.noSuggestion', { q: query.trim() || '…' }) }}</div>
            </div>
            <div v-else-if="form.image" class="ce-picked">
              <UIcon name="i-lucide-check" class="ce-picked__ok" />
              <span class="ce-picked__ref m-mono">{{ form.image }}</span>
              <span class="ce-fh ce-picked__info">{{ imageInfoText }}</span>
              <UButton size="sm" color="neutral" variant="soft" class="ce-pill-btn" @click="reopenSug">{{ t('docker.wizard.change') }}</UButton>
            </div>
          </div>

          <div class="ce-grid2">
            <div class="ce-fl">
              <label for="ce-name">{{ isMobile ? t('docker.wizard.containerName') : t('docker.common.name') }}</label>
              <span class="ce-fld" :class="{ 'is-bad': nameError }">
                <input id="ce-name" v-model.trim="form.name" class="m-mono" :placeholder="t('docker.wizard.namePlaceholder')" />
              </span>
              <span class="ce-fh" :class="{ 'is-bad': nameError }">{{ nameError || t('docker.wizard.nameHint') }}</span>
            </div>
            <div class="ce-fl">
              <label for="ce-cmd">{{ t('docker.container.startCommand') }} <span class="ce-fh">{{ t('docker.common.optional') }}</span></label>
              <span class="ce-fld"><input id="ce-cmd" v-model="form.cmd" class="m-mono" :placeholder="t('docker.wizard.cmdPlaceholder')" /></span>
              <span v-if="!isMobile" class="ce-fh">{{ t('docker.wizard.cmdHint') }}</span>
            </div>
          </div>
        </section>

        <!-- 2 端口 -->
        <section v-show="showStep('ports')" :ref="(el) => setSecRef('ports', el)" class="ce-sec" :class="{ 'is-on': secOn('ports') }" data-step="ports" aria-labelledby="ce-h-ports">
          <div class="ce-sec__head">
            <span v-if="!isMobile" class="ce-badge m-sticker m-sticker--a2 m-num" style="--rot: 3deg">2</span>
            <div class="ce-sec__title">
              <h2 id="ce-h-ports">{{ t('docker.container.ports') }}</h2>
              <p>
                {{ t('docker.wizard.declaredPorts') }}
                <template v-if="imageMeta?.exposedPorts?.length">
                  <button v-for="p in imageMeta.exposedPorts" :key="p" type="button" class="ce-chipbtn m-mono" :title="t('docker.wizard.useDeclared')" @click="addDeclaredPort(p)">{{ p }}</button>
                </template>
                <span v-else class="m-mono">{{ t('docker.wizard.unknown') }}</span>
              </p>
            </div>
            <UButton v-if="!isMobile" size="sm" color="neutral" variant="outline" icon="i-lucide-plus" class="ce-pill-btn" @click="portEd?.add()">{{ t('docker.wizard.addPort') }}</UButton>
          </div>
          <div v-if="hostNet" class="ce-warn">{{ t('docker.wizard.hostNetPorts') }}</div>
          <ContainerPortEditor ref="portEd" v-model="ports" :taken="hostNet ? {} : takenPorts" :empty-text="t('docker.wizard.noPorts')" />
        </section>

        <!-- 3 存储 -->
        <section v-show="showStep('storage')" :ref="(el) => setSecRef('storage', el)" class="ce-sec" :class="{ 'is-on': secOn('storage') }" data-step="storage" aria-labelledby="ce-h-storage">
          <div class="ce-sec__head">
            <span v-if="!isMobile" class="ce-badge m-sticker m-sticker--a m-num" style="--rot: -3deg">3</span>
            <div class="ce-sec__title">
              <h2 id="ce-h-storage">{{ isMobile ? t('docker.wizard.secStorageM') : t('docker.wizard.secStorage') }}</h2>
              <p>
                {{ t('docker.wizard.secStorageDesc') }}
                <template v-if="imageMeta?.volumes?.length">
                  · {{ t('docker.wizard.declaredVolumes') }}
                  <button v-for="v in imageMeta.volumes" :key="v" type="button" class="ce-chipbtn m-mono" :title="t('docker.wizard.useDeclared')" @click="addDeclaredVolume(v)">{{ v }}</button>
                </template>
              </p>
            </div>
            <UButton v-if="!isMobile" size="sm" color="neutral" variant="outline" icon="i-lucide-plus" class="ce-pill-btn" @click="mountEd?.add()">{{ t('docker.wizard.addMount') }}</UButton>
          </div>
          <ContainerMountEditor ref="mountEd" v-model="mounts" :browsable="browsable" :empty-text="t('docker.wizard.noMounts')" />
        </section>

        <!-- 4 环境变量 -->
        <section v-show="showStep('env')" :ref="(el) => setSecRef('env', el)" class="ce-sec" :class="{ 'is-on': secOn('env') }" data-step="env" aria-labelledby="ce-h-env">
          <div class="ce-sec__head">
            <span v-if="!isMobile" class="ce-badge m-sticker m-sticker--ok m-num" style="--rot: 4deg">4</span>
            <div class="ce-sec__title">
              <h2 id="ce-h-env">{{ t('docker.container.env') }}</h2>
              <p>{{ t('docker.wizard.secEnvDesc') }}</p>
            </div>
            <template v-if="!isMobile">
              <UButton size="sm" color="neutral" variant="soft" icon="i-lucide-clipboard-paste" class="ce-pill-btn" @click="openPaste">{{ t('docker.wizard.pasteEnv') }}</UButton>
              <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-plus" class="ce-pill-btn" @click="envEd?.add()">{{ t('docker.common.add') }}</UButton>
            </template>
          </div>
          <ContainerEnvEditor ref="envEd" v-model="envs" :empty-text="t('docker.wizard.noEnv')" @paste="openPaste" />
        </section>

        <!-- 5 网络 -->
        <section v-show="showStep('network')" :ref="(el) => setSecRef('network', el)" class="ce-sec" :class="{ 'is-on': secOn('network') }" data-step="network" aria-labelledby="ce-h-net">
          <div class="ce-sec__head">
            <span v-if="!isMobile" class="ce-badge m-sticker m-sticker--p m-num" style="--rot: -2deg">5</span>
            <div class="ce-sec__title">
              <h2 id="ce-h-net">{{ t('docker.common.network') }}</h2>
              <p>{{ t('docker.wizard.secNetDesc') }}</p>
            </div>
            <UButton v-if="!isMobile" size="sm" color="neutral" variant="soft" class="ce-pill-btn" @click="router.push(pageTabTo('docker', 'network'))">{{ t('docker.wizard.manageNetworks') }}</UButton>
          </div>
          <div class="ce-opts" role="radiogroup" :aria-label="t('docker.common.network')">
            <button
              v-for="n in networkOptions"
              :key="n.name"
              type="button"
              role="radio"
              class="ce-opt"
              :class="{ 'is-on': form.network === n.name }"
              :aria-checked="form.network === n.name"
              @click="form.network = n.name"
            >
              <span class="ce-rd" />
              <span class="ce-opt__text">
                <span class="ce-opt__t m-mono">{{ n.name }}</span>
                <span class="ce-opt__d">{{ n.desc }}</span>
                <span class="ce-opt__s m-mono">{{ n.sub }}</span>
              </span>
            </button>
          </div>
        </section>

        <!-- 6 资源与重启策略 -->
        <section v-show="showStep('res')" :ref="(el) => setSecRef('res', el)" class="ce-sec" :class="{ 'is-on': secOn('res') }" data-step="res" aria-labelledby="ce-h-res">
          <div class="ce-sec__head">
            <span v-if="!isMobile" class="ce-badge m-sticker m-sticker--a2 m-num" style="--rot: 3deg">6</span>
            <div class="ce-sec__title">
              <h2 id="ce-h-res">{{ t('docker.wizard.secRes') }}</h2>
              <p>{{ isMobile ? t('docker.wizard.secResDescM') : t('docker.wizard.secResDesc') }}</p>
            </div>
          </div>
          <div class="ce-fl">
            <span>{{ t('docker.container.restartPolicy') }}</span>
            <div v-if="!isMobile" class="ce-seg" role="radiogroup" :aria-label="t('docker.container.restartPolicy')">
              <button
                v-for="r in RESTARTS"
                :key="r"
                type="button"
                role="radio"
                class="m-mono"
                :class="{ 'is-on': form.restart === r }"
                :aria-checked="form.restart === r"
                @click="form.restart = r"
              >{{ r }}</button>
            </div>
            <div v-else class="ce-opts ce-opts--grid2" role="radiogroup" :aria-label="t('docker.container.restartPolicy')">
              <button
                v-for="r in RESTARTS"
                :key="r"
                type="button"
                role="radio"
                class="ce-opt ce-opt--col"
                :class="{ 'is-on': form.restart === r }"
                :aria-checked="form.restart === r"
                @click="form.restart = r"
              >
                <span class="ce-opt__t m-mono">{{ r }}</span>
                <span class="ce-opt__d is-sm">{{ t(`docker.wizard.restartShort.${restartKey(r)}`) }}</span>
              </button>
            </div>
            <span v-if="!isMobile" class="ce-fh">{{ t(`docker.wizard.restartDesc.${restartKey(form.restart)}`) }}</span>
          </div>
          <div v-if="form.restart === 'on-failure'" class="ce-fl ce-retry">
            <label for="ce-retry">{{ t('docker.wizard.maxRetries') }}</label>
            <span class="ce-fld"><input id="ce-retry" v-model.number="form.retries" class="m-num" type="number" min="0" inputmode="numeric" /><span class="ce-sfx">{{ t('docker.wizard.times') }}</span></span>
          </div>
          <div class="ce-grid2">
            <div class="ce-fl">
              <label for="ce-cpu">{{ t('docker.wizard.cpuLimit') }}</label>
              <span class="ce-fld" :class="{ 'is-bad': cpuError }">
                <UIcon name="i-lucide-cpu" />
                <input id="ce-cpu" v-model.trim="form.cpu" class="m-num" :placeholder="t('docker.wizard.noLimit')" inputmode="decimal" />
                <span class="ce-sfx">{{ t('docker.common.cpuUnit') }}</span>
              </span>
              <span class="ce-fh">{{ hostCpus ? t('docker.wizard.cpuHint', { n: hostCpus }) : t('docker.wizard.blankNoLimit') }}</span>
            </div>
            <div class="ce-fl">
              <label for="ce-mem">{{ t('docker.wizard.memLimit') }}</label>
              <span class="ce-fld" :class="{ 'is-bad': memError }">
                <UIcon name="i-lucide-memory-stick" />
                <input id="ce-mem" v-model.trim="form.mem" class="m-num" :placeholder="t('docker.wizard.noLimit')" inputmode="numeric" />
                <span class="ce-sfx">MB</span>
              </span>
              <span class="ce-fh">{{ hostMemory ? t('docker.wizard.memHint', { size: formatBytes(hostMemory) }) : t('docker.wizard.blankNoLimit') }}</span>
            </div>
          </div>
          <label v-if="isRecreate" class="ce-switch">
            <span><strong>{{ t('docker.container.forceDeleteOld') }}</strong><small>{{ t('docker.wizard.forceDesc') }}</small></span>
            <AppSwitch v-model="form.force" />
          </label>

          <!-- 手机：最后一步展示 docker run 命令 -->
          <div v-if="isMobile" class="ce-fl">
            <div class="ce-cmdhead">
              <span class="ce-sm">{{ t('docker.wizard.runCommand') }}</span>
              <UButton size="sm" color="primary" variant="ghost" :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'" @click="copyCmd">{{ copied ? t('docker.wizard.copied') : t('docker.wizard.copy') }}</UButton>
            </div>
            <pre class="ce-term m-mono" tabindex="0" :aria-label="t('docker.wizard.runCommand')"><template v-for="(l, i) in cmdLines" :key="i"><span class="ce-term__dim">{{ l.pre }}</span><span class="ce-term__flag">{{ l.flag }}</span><span :class="{ 'is-strong': l.strong }">{{ l.value }}</span><span class="ce-term__dim">{{ l.tail }}</span>
</template></pre>
          </div>
        </section>

        <!-- PC 吸底就绪条 -->
        <div v-if="!isMobile" class="ce-bar">
          <StatusPill :variant="ready ? 'success' : 'error'" :label="readyTitle" />
          <span class="ce-bar__sub">{{ readySub }}</span>
          <UButton color="neutral" variant="soft" class="ce-pill-btn ce-pill-btn--lg" @click="emit('back')">{{ t('docker.common.cancel') }}</UButton>
          <UButton color="primary" icon="i-lucide-play" class="ce-pill-btn ce-pill-btn--lg" :loading="submitting" :disabled="!ready" @click="submit">
            {{ submitting ? t('docker.wizard.creating') : isRecreate ? t('docker.wizard.recreateStart') : t('docker.wizard.createStart') }}
          </UButton>
        </div>
      </div>

      <!-- PC 摘要 -->
      <aside v-if="!isMobile" class="ce-side" :aria-label="t('docker.wizard.summary')">
        <div class="ce-side__head">
          <h2>{{ t('docker.wizard.summary') }}</h2>
          <StatusPill :variant="ready ? 'success' : 'error'" :label="readyTitle" />
        </div>
        <div class="ce-side__id">
          <span class="ce-side__sticker m-sticker m-sticker--a2"><UIcon name="i-lucide-box" /></span>
          <span class="ce-side__idtext">
            <span class="m-mono ce-side__name">{{ form.name || t('docker.wizard.randomName') }}</span>
            <span class="m-mono ce-side__img">{{ form.image || t('docker.wizard.noImage') }}</span>
          </span>
        </div>
        <dl class="ce-sum">
          <div v-for="s in summary" :key="s.k">
            <dt>{{ s.k }}</dt>
            <dd class="m-mono" :title="s.v">{{ s.v }}</dd>
          </div>
        </dl>
        <div class="ce-cmdhead">
          <span class="ce-cmdhead__t">{{ t('docker.wizard.runCommand') }}</span>
          <UButton size="sm" color="neutral" variant="soft" :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'" class="ce-pill-btn" @click="copyCmd">{{ copied ? t('docker.wizard.copied') : t('docker.wizard.copy') }}</UButton>
        </div>
        <pre class="ce-term m-mono" tabindex="0" :aria-label="t('docker.wizard.runCommand')"><template v-for="(l, i) in cmdLines" :key="i"><span class="ce-term__dim">{{ l.pre }}</span><span class="ce-term__flag">{{ l.flag }}</span><span :class="{ 'is-strong': l.strong }">{{ l.value }}</span><span class="ce-term__dim">{{ l.tail }}</span>
</template></pre>
        <span class="ce-fh">{{ t('docker.wizard.runHint') }}</span>
      </aside>
    </div>

    <!-- 手机底部：上一步 / 下一步 -->
    <div v-if="isMobile && !loadingSource" class="ce-mbar">
      <AppIconButton class="ce-mbar__prev" icon="HOutline:ChevronLeftIcon" :label="t('docker.wizard.prev')" :box="52" :size="22" :disabled="stepIndex === 0" @click="stepIndex = Math.max(0, stepIndex - 1)" />
      <UButton
        color="primary"
        class="ce-mbar__next"
        :icon="isLastStep ? 'i-lucide-play' : 'i-lucide-arrow-right'"
        :loading="submitting"
        :disabled="isLastStep && !ready"
        @click="nextStep"
      >
        {{ isLastStep ? (submitting ? t('docker.wizard.creating') : isRecreate ? t('docker.wizard.recreateStart') : t('docker.wizard.createStart')) : t('docker.wizard.nextStep', { name: stepLabel(STEPS[stepIndex + 1]) }) }}
      </UButton>
    </div>

    <!-- 从 Compose 导入 -->
    <FormDialog v-model="composeOpen" :title="t('docker.wizard.importCompose')" :width="640" :loading="composeParsing">
      <div class="ce-dlg">
        <template v-if="!composeServices.length">
          <p class="ce-fh">{{ t('docker.wizard.composeDesc') }}</p>
          <textarea v-model="composeText" class="app-textarea m-mono ce-dlg__ta" rows="12" :placeholder="COMPOSE_SAMPLE" />
          <div class="ce-dlg__row">
            <UButton size="sm" color="neutral" variant="soft" icon="i-lucide-upload" class="ce-pill-btn" @click="composeFile?.click()">{{ t('docker.wizard.composeFile') }}</UButton>
            <input ref="composeFile" type="file" class="ce-hidden" accept=".yml,.yaml,text/yaml,text/plain" @change="onComposeFile" />
          </div>
        </template>
        <template v-else>
          <p class="ce-fh">{{ t('docker.wizard.composePick') }}</p>
          <div class="ce-opts" role="radiogroup" :aria-label="t('docker.wizard.composeServices')">
            <button
              v-for="svc in composeServices"
              :key="svc.name"
              type="button"
              role="radio"
              class="ce-opt"
              :class="{ 'is-on': composePicked === svc.name }"
              :aria-checked="composePicked === svc.name"
              @click="composePicked = svc.name"
            >
              <span class="ce-rd" />
              <span class="ce-opt__text">
                <span class="ce-opt__t m-mono">{{ svc.name }}</span>
                <span class="ce-opt__d m-mono">{{ svc.options?.image || t('docker.wizard.noImage') }}</span>
                <span v-if="svc.warnings.length" class="ce-opt__s">{{ t('docker.wizard.composeIgnored', { keys: svc.warnings.join(', ') }) }}</span>
              </span>
            </button>
          </div>
        </template>
      </div>
      <template #footer="{ close }">
        <UButton v-if="composeServices.length" color="neutral" variant="ghost" @click="composeServices = []">{{ t('docker.wizard.composeBack') }}</UButton>
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton v-if="!composeServices.length" color="primary" :loading="composeParsing" :disabled="!composeText.trim()" @click="parseCompose">{{ t('docker.wizard.composeParse') }}</UButton>
        <UButton v-else color="primary" :disabled="!composePicked" @click="applyCompose">{{ t('docker.wizard.composeApply') }}</UButton>
      </template>
    </FormDialog>

    <!-- 粘贴 .env -->
    <FormDialog v-model="pasteOpen" :title="t('docker.wizard.pasteEnv')" :width="560" @confirm="applyPaste">
      <div class="ce-dlg">
        <p class="ce-fh">{{ t('docker.wizard.pasteDesc') }}</p>
        <textarea v-model="pasteText" class="app-textarea m-mono ce-dlg__ta" rows="10" placeholder="TZ=Asia/Shanghai&#10;# comment&#10;export APP_ENV=production" />
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :disabled="!pasteText.trim()" @click="applyPaste">{{ t('docker.wizard.pasteApply') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { pageTabTo } from '@/config/pageGroups'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import {
  createDockerContainer, getDockerConfig, getDockerContainer, getDockerImage, listDockerContainers,
  listDockerImages, listDockerNetworks, parseDockerCompose, searchDockerImages, updateDockerContainer,
} from '@/api/docker'
import { showRequestError } from '@/utils/request'
import { copyTextToClipboard } from '@/utils/file'
import ContainerPortEditor from '@/views/docker/components/ContainerPortEditor.vue'
import ContainerMountEditor from '@/views/docker/components/ContainerMountEditor.vue'
import ContainerEnvEditor from '@/views/docker/components/ContainerEnvEditor.vue'
import { useDockerStatus } from '@/views/docker/components/useDockerStatus'
import { formatAgo, formatBytes, joinCommand, splitCommand } from '@/views/docker/components/format'
import { isHostPath, type EnvRow, type MountRow, type PortRow } from '@/views/docker/components/specTypes'
import type {
  DockerComposeService, DockerContainerCreateOptions, DockerImageInfo, DockerImageSearchResult,
  DockerImageSummary, DockerNetworkInfo, DockerTaskInfo,
} from '@/types/v1/docker'

const props = defineProps<{
  mode: 'create' | 'recreate'
  /** 重建时的容器 ID */
  recreateId?: string
  /** 预选镜像（镜像页「用它建容器」） */
  initialImage?: string
}>()
const emit = defineEmits<{ back: []; done: [result: { id?: string; task?: DockerTaskInfo }] }>()

const { t } = useI18n()
const router = useRouter()
const menuStore = useMenuStore()
const accessible = useAccessibleRoutes()
const { status } = useDockerStatus()
const isMobile = computed(() => menuStore.isMobile)
const isRecreate = computed(() => props.mode === 'recreate')

type Step = 'basic' | 'ports' | 'storage' | 'env' | 'network' | 'res'
const STEPS: Step[] = ['basic', 'ports', 'storage', 'env', 'network', 'res']
const RESTARTS = ['no', 'always', 'unless-stopped', 'on-failure'] as const
const restartKey = (r: string) => (r === 'unless-stopped' ? 'unlessStopped' : r === 'on-failure' ? 'onFailure' : r || 'no')
const stepLabel = (s: Step | undefined) => (s ? t(`docker.wizard.step.${s}`) : '')

const form = reactive({
  image: '',
  name: '',
  cmd: '',
  network: 'bridge',
  restart: 'unless-stopped' as string,
  retries: 5,
  cpu: '',
  mem: '',
  force: false,
})
const ports = ref<PortRow[]>([])
const mounts = ref<MountRow[]>([])
const envs = ref<EnvRow[]>([])
/** 重建时沿用旧容器里表单没有展示的参数 */
const preserved = ref<Partial<DockerContainerCreateOptions>>({})
const recreateName = ref('')
const loadingSource = ref(false)

const portEd = ref<InstanceType<typeof ContainerPortEditor> | null>(null)
const mountEd = ref<InstanceType<typeof ContainerMountEditor> | null>(null)
const envEd = ref<InstanceType<typeof ContainerEnvEditor> | null>(null)

/* ───────── 主机 / 引擎信息 ───────── */
const hostCpus = computed(() => Number(status.value?.info?.cpus || 0))
const hostMemory = computed(() => Number(status.value?.info?.memoryTotal || 0))
const dockerLocal = ref(false)
const browsable = computed(() => dockerLocal.value && accessible.value.has('/file/index'))

/* ───────── 镜像建议：本地镜像 + Docker Hub ───────── */
interface Suggestion { ref: string; desc: string; local: boolean; letter: string; badge?: string; badgeTone?: string }
const localImages = ref<DockerImageSummary[]>([])
const hubResults = ref<DockerImageSearchResult[]>([])
const query = ref('')
const sugOpen = ref(false)
const searching = ref(false)
const imgInput = ref<HTMLInputElement | null>(null)

const localRefs = computed(() => {
  const out: { ref: string; size: number; created: unknown }[] = []
  for (const img of localImages.value) {
    for (const tag of img.repoTags || []) {
      if (tag && tag !== '<none>:<none>') out.push({ ref: tag, size: Number(img.size), created: img.created })
    }
  }
  return out
})
const localMap = computed(() => new Map(localRefs.value.map((r) => [r.ref, r])))
/** 没写标签的引用按 :latest 查本地镜像（与引擎解析一致） */
const localOf = (ref: string) => {
  const r = ref.trim()
  const hasTag = r.includes('@') || r.lastIndexOf(':') > r.lastIndexOf('/')
  return localMap.value.get(hasTag ? r : `${r}:latest`)
}
const letterOf = (ref: string) => (ref.split('/').pop() || ref).charAt(0).toUpperCase()
const suggestions = computed<Suggestion[]>(() => {
  const q = query.value.trim().toLowerCase()
  const local = localRefs.value
    .filter((r) => !q || r.ref.toLowerCase().includes(q))
    .slice(0, q ? 4 : 6)
    .map((r) => ({
      ref: r.ref,
      local: true,
      letter: letterOf(r.ref),
      desc: t('docker.wizard.localDesc', { size: formatBytes(r.size), ago: formatAgo(r.created, t) }),
      badge: t('docker.wizard.badgeLocal'),
      badgeTone: 'ok',
    }))
  const seen = new Set(local.map((s) => s.ref))
  const hub = hubResults.value
    .map((h) => ({ h, ref: h.official ? `${h.name}:latest` : h.name }))
    .filter(({ ref }) => !seen.has(ref) && !localMap.value.has(ref))
    .slice(0, 5)
    .map(({ h, ref }) => ({
      ref,
      local: false,
      letter: letterOf(ref),
      desc: [h.description, Number(h.stars) ? `★ ${formatStars(Number(h.stars))}` : ''].filter(Boolean).join(' · ') || t('docker.wizard.fromHub'),
      badge: h.official ? t('docker.wizard.badgeOfficial') : '',
      badgeTone: 'a2',
    }))
  return [...local, ...hub]
})
const formatStars = (n: number) => (n >= 1000 ? `${(n / 1000).toFixed(n >= 10000 ? 0 : 1)}k` : String(n))

const searchHub = useDebounceFn(async (term: string) => {
  if (term.length < 2 || term.includes(':')) {
    hubResults.value = []
    searching.value = false
    return
  }
  try {
    const { data } = await searchDockerImages({ term, limit: 8 })
    if (query.value.trim() === term) hubResults.value = data?.items || []
  } catch {
    hubResults.value = []
  } finally {
    searching.value = false
  }
}, 400)
watch(query, (q) => {
  const term = q.trim()
  if (term.length >= 2 && !term.includes(':')) searching.value = true
  void searchHub(term)
})

const pickImage = (ref: string) => {
  form.image = ref.trim()
  query.value = form.image
  sugOpen.value = false
}
const pickTyped = () => {
  const first = suggestions.value[0]
  const typed = query.value.trim()
  if (!typed && first) pickImage(first.ref)
  else if (typed) pickImage(suggestions.value.find((s) => s.ref === typed)?.ref || typed)
}
const reopenSug = () => {
  sugOpen.value = true
  void nextTick(() => imgInput.value?.focus())
}
const onDocPointer = (e: PointerEvent) => {
  const target = e.target as HTMLElement | null
  if (sugOpen.value && target && !target.closest('.ce-sug') && !target.closest('#ce-img')) {
    sugOpen.value = false
    const typed = query.value.trim()
    if (typed && !form.image) pickImage(typed)
    else query.value = form.image
  }
}

/* 选中镜像的声明端口 / 数据卷（本地镜像才能 inspect） */
const imageMeta = ref<DockerImageInfo | null>(null)
const loadImageMeta = async (ref: string) => {
  imageMeta.value = null
  if (!ref || !localOf(ref)) return
  try {
    const { data } = await getDockerImage({ id: ref })
    if (form.image === ref) imageMeta.value = data?.info || null
  } catch { /* 声明信息只是提示 */ }
}
watch(() => form.image, (ref) => void loadImageMeta(ref))
const imageLocal = computed(() => !!form.image && !!localOf(form.image))
const imageInfoText = computed(() => {
  const local = localOf(form.image)
  if (local) return t('docker.wizard.localDesc', { size: formatBytes(local.size), ago: formatAgo(local.created, t) })
  return t('docker.wizard.willPull')
})
const addDeclaredPort = (spec: string) => {
  const [port = '', proto = 'tcp'] = spec.split('/')
  if (ports.value.some((p) => p.containerPort === port && p.protocol === proto)) return
  ports.value = [...ports.value, { hostIp: '', hostPort: port, containerPort: port, protocol: proto === 'udp' ? 'udp' : 'tcp' }]
}
const addDeclaredVolume = (target: string) => {
  if (mounts.value.some((m) => m.target === target)) return
  mounts.value = [...mounts.value, { type: 'bind', source: '', target, readOnly: false }]
}

/* ───────── 主机端口占用（运行中容器发布的端口） ───────── */
const takenPorts = ref<Record<string, string>>({})
const loadTaken = async () => {
  try {
    const { data } = await listDockerContainers({ page: 1, pageSize: 1000, status: '', keyword: '', network: '', labels: {} })
    const map: Record<string, string> = {}
    for (const c of data?.items || []) {
      if (isRecreate.value && c.id === props.recreateId) continue
      const name = (c.names?.[0] || c.id).replace(/^\//, '')
      for (const p of c.ports || []) {
        if (p.publicPort) map[`${p.publicPort}/${(p.type || 'tcp').toLowerCase()}`] = name
      }
    }
    takenPorts.value = map
  } catch { /* 冲突检查只是提示 */ }
}

/* ───────── 网络 ───────── */
const networks = ref<DockerNetworkInfo[]>([])
const hostNet = computed(() => form.network === 'host')
const networkOptions = computed(() => {
  const subOf = (n: DockerNetworkInfo) => {
    const subnet = (n.ipam?.config || []).map((c) => c.subnet).filter(Boolean).join(', ')
    const count = Object.keys(n.containers || {}).length
    return [subnet, t('docker.wizard.netCount', { n: count })].filter(Boolean).join(' · ')
  }
  const bridge = networks.value.find((n) => n.name === 'bridge')
  const out: { name: string; desc: string; sub: string }[] = [
    { name: 'bridge', desc: t('docker.wizard.netBridge'), sub: bridge ? subOf(bridge) : '—' },
  ]
  networks.value
    .filter((n) => !n.builtin && n.driver !== 'null')
    .sort((a, b) => a.name.localeCompare(b.name))
    .forEach((n) => out.push({ name: n.name, desc: t('docker.wizard.netCustom', { driver: n.driver || 'bridge' }), sub: subOf(n) }))
  out.push({ name: 'host', desc: t('docker.wizard.netHost'), sub: t('docker.wizard.netHostSub') })
  out.push({ name: 'none', desc: t('docker.wizard.netNone'), sub: t('docker.wizard.netNoneSub') })
  if (form.network && !out.some((o) => o.name === form.network)) out.splice(1, 0, { name: form.network, desc: t('docker.wizard.netOther'), sub: '—' })
  return out
})

/* ───────── 校验与就绪 ───────── */
const nameError = computed(() => (form.name && !/^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/.test(form.name) ? t('docker.wizard.nameInvalid') : ''))
const cpuValue = computed(() => (form.cpu ? Number(form.cpu) : 0))
const memValue = computed(() => (form.mem ? Number(form.mem) : 0))
const cpuError = computed(() => !!form.cpu && (!Number.isFinite(cpuValue.value) || cpuValue.value <= 0))
const memError = computed(() => !!form.mem && (!Number.isInteger(memValue.value) || memValue.value <= 0))
const validPorts = computed(() => (hostNet.value ? [] : ports.value.filter((p) => p.containerPort.trim())))
const conflicts = computed(() => validPorts.value.filter((p) => p.hostPort.trim() && takenPorts.value[`${p.hostPort.trim()}/${p.protocol}`]).length)
const validMounts = computed(() => mounts.value.filter((m) => m.target.trim()))
const validEnvs = computed(() => envs.value.filter((e) => e.key.trim()))
const errorCount = computed(() => conflicts.value + (nameError.value ? 1 : 0) + (cpuError.value ? 1 : 0) + (memError.value ? 1 : 0))
const ready = computed(() => !!form.image && errorCount.value === 0)
const readyTitle = computed(() => {
  if (!form.image) return t('docker.wizard.noImageYet')
  if (conflicts.value) return t('docker.wizard.conflicts', { n: conflicts.value })
  if (errorCount.value) return t('docker.wizard.invalidFields', { n: errorCount.value })
  return t('docker.wizard.ready')
})
const readySub = computed(() => {
  if (!ready.value) return t('docker.wizard.fixRed')
  return imageLocal.value ? t('docker.wizard.readyLocal') : t('docker.wizard.readyPull')
})

/* ───────── 步骤 ───────── */
const doneOf = (s: Step) => {
  if (s === 'basic') return !!form.image && !nameError.value
  if (s === 'ports') return validPorts.value.length > 0
  if (s === 'storage') return validMounts.value.length > 0
  if (s === 'env') return validEnvs.value.length > 0
  return true
}
const stepIndex = ref(0)
const isLastStep = computed(() => stepIndex.value === STEPS.length - 1)
const showStep = (s: Step) => !isMobile.value || STEPS[stepIndex.value] === s
const activeStep = ref<Step>('basic')
const secOn = (s: Step) => !isMobile.value && activeStep.value === s
const nextStep = () => {
  if (isLastStep.value) void submit()
  else stepIndex.value++
}
watch(stepIndex, () => {
  document.querySelector('.app-content')?.scrollTo({ top: 0 })
})

const secRefs = new Map<Step, HTMLElement>()
const setSecRef = (s: Step, el: unknown) => {
  if (el instanceof HTMLElement) secRefs.set(s, el)
}
const scrollTo = (s: Step) => {
  activeStep.value = s
  secRefs.get(s)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
let observer: IntersectionObserver | null = null
const setupSpy = () => {
  observer?.disconnect()
  if (isMobile.value || typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(
    (entries) => {
      const visible = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
      const step = (visible[0]?.target as HTMLElement | undefined)?.dataset.step as Step | undefined
      if (step) activeStep.value = step
    },
    { rootMargin: '-90px 0px -55% 0px', threshold: 0 },
  )
  secRefs.forEach((el) => observer?.observe(el))
}

/* ───────── 摘要与等效命令 ───────── */
const portText = (p: PortRow) => `${p.hostIp ? `${p.hostIp}:` : ''}${p.hostPort ? `${p.hostPort}:` : ''}${p.containerPort}${p.protocol === 'udp' ? '/udp' : ''}`
const summary = computed(() => [
  {
    k: t('docker.wizard.sumPorts'),
    v: hostNet.value
      ? t('docker.wizard.sumHostNet')
      : validPorts.value.length
        ? validPorts.value.map((p) => `${p.hostPort || '*'}→${p.containerPort}${p.protocol === 'udp' ? '/udp' : ''}`).join('  ')
        : t('docker.wizard.none'),
  },
  {
    k: t('docker.wizard.sumStorage'),
    v: validMounts.value.length
      ? t('docker.wizard.sumMounts', { n: validMounts.value.length, ro: validMounts.value.filter((m) => m.readOnly).length })
      : t('docker.wizard.none'),
  },
  { k: t('docker.container.env'), v: validEnvs.value.length ? validEnvs.value.map((e) => e.key.trim()).join(', ') : t('docker.wizard.none') },
  { k: t('docker.common.network'), v: form.network },
  { k: t('docker.wizard.sumRestart'), v: form.restart === 'on-failure' ? `on-failure:${form.retries || 0}` : form.restart },
  {
    k: t('docker.wizard.sumRes'),
    v: `${form.cpu && !cpuError.value ? `${form.cpu} ${t('docker.common.cpuUnit')}` : t('docker.wizard.cpuUnlimited')} · ${form.mem && !memError.value ? `${form.mem} MB` : t('docker.wizard.memUnlimited')}`,
  },
])
const q = (v: string) => (/[\s"'$`\\]/.test(v) ? `"${v.replace(/(["\\$`])/g, '\\$1')}"` : v)
interface CmdLine { pre: string; flag: string; value: string; tail: string; strong?: boolean }
const cmdLines = computed<CmdLine[]>(() => {
  const lines: CmdLine[] = [{ pre: '$ ', flag: '', value: 'docker run -d', tail: ' \\', strong: true }]
  const add = (flag: string, value: string) => lines.push({ pre: '    ', flag, value: ` ${value}`, tail: ' \\' })
  if (form.name) add('--name', form.name)
  validPorts.value.forEach((p) => add('-p', portText(p)))
  validMounts.value.forEach((m) => add('-v', `${m.source ? `${m.source}:` : ''}${m.target}${m.readOnly ? ':ro' : ''}`))
  validEnvs.value.forEach((e) => add('-e', q(`${e.key.trim()}=${e.value}`)))
  if (form.network && form.network !== 'bridge') add('--network', form.network)
  if (form.restart !== 'no') add('--restart', form.restart === 'on-failure' ? `on-failure:${form.retries || 0}` : form.restart)
  if (form.cpu && !cpuError.value) add('--cpus', form.cpu)
  if (form.mem && !memError.value) add('--memory', `${form.mem}m`)
  const args = splitCommand(form.cmd)
  lines.push({ pre: '    ', flag: '', value: `${form.image || t('docker.wizard.imageToken')}${args.length ? ` ${joinCommand(args)}` : ''}`, tail: '', strong: true })
  return lines
})
const copied = ref(false)
const copyCmd = async () => {
  const text = cmdLines.value.map((l, i) => (i === 0 ? l.value : `  ${l.flag}${l.value}`)).join(' \\\n')
  await copyTextToClipboard(text)
  copied.value = true
  setTimeout(() => (copied.value = false), 2000)
}

/* ───────── 回填（重建 / Compose） ───────── */
const parseEnv = (items: string[] = []): EnvRow[] =>
  items.map((raw) => {
    const i = raw.indexOf('=')
    return i >= 0 ? { key: raw.slice(0, i), value: raw.slice(i + 1) } : { key: raw, value: '' }
  }).filter((e) => e.key)
const applyOptions = (opts: Partial<DockerContainerCreateOptions>) => {
  if (opts.image) pickImage(opts.image)
  form.name = opts.name || form.name
  form.cmd = opts.cmd?.length ? joinCommand(opts.cmd) : form.cmd
  form.network = opts.network || 'bridge'
  form.restart = opts.restartPolicy || 'no'
  form.retries = Number(opts.restartMaxRetries) || 5
  form.cpu = Number(opts.nanoCpus) ? String(Math.round((Number(opts.nanoCpus) / 1e9) * 100) / 100) : ''
  form.mem = Number(opts.memory) ? String(Math.round(Number(opts.memory) / 1024 / 1024)) : ''
  ports.value = (opts.ports || []).map((p) => {
    const [port = '', proto = 'tcp'] = (p.containerPort || '').split('/')
    return { hostIp: p.hostIp || '', hostPort: p.hostPort || '', containerPort: port, protocol: proto === 'udp' ? 'udp' : 'tcp' }
  })
  mounts.value = (opts.mounts || []).map((m) => ({
    type: m.type || (m.source && isHostPath(m.source) ? 'bind' : 'volume'),
    source: m.source || '',
    target: m.target || '',
    readOnly: !!m.readOnly,
  }))
  envs.value = parseEnv(opts.env)
  preserved.value = {
    hostname: opts.hostname || '', user: opts.user || '', entrypoint: opts.entrypoint || [], workingDir: opts.workingDir || '',
    labels: opts.labels || {}, tty: !!opts.tty, openStdin: !!opts.openStdin, autoRemove: !!opts.autoRemove,
    privileged: !!opts.privileged, platform: opts.platform || '',
  }
}

const loadRecreate = async (id: string) => {
  loadingSource.value = true
  try {
    const { data } = await getDockerContainer({ id })
    const info = data?.info
    if (!info) return
    recreateName.value = info.name || id.slice(0, 12)
    const host = info.hostConfig
    const cfg = info.config
    const hostMounts = host?.mounts?.length
      ? host.mounts
      : (info.mounts || []).map((m) => ({ type: m.type || 'bind', source: m.type === 'volume' ? m.name || '' : m.source || '', target: m.destination || '', readOnly: !m.rw }))
    applyOptions({
      name: info.name,
      image: cfg?.image || info.image,
      cmd: cfg?.cmd || [],
      env: cfg?.env || [],
      network: host?.networkMode && host.networkMode !== 'default' ? host.networkMode : 'bridge',
      restartPolicy: host?.restartPolicy || 'no',
      restartMaxRetries: host?.restartMaxRetries || 0,
      nanoCpus: host?.nanoCpus || (host?.cpuQuota && host?.cpuPeriod ? Math.round((host.cpuQuota / host.cpuPeriod) * 1e9) : 0),
      memory: host?.memory || 0,
      ports: host?.portBindings || [],
      mounts: hostMounts,
      hostname: '',
      user: cfg?.user || '',
      entrypoint: cfg?.entrypoint || [],
      workingDir: cfg?.workingDir || '',
      labels: cfg?.labels || {},
      tty: !!cfg?.tty,
      openStdin: !!cfg?.openStdin,
      autoRemove: !!host?.autoRemove,
      privileged: !!host?.privileged,
    })
  } catch (e) {
    showRequestError(e, t('docker.container.getDetailFailed'))
  } finally {
    loadingSource.value = false
  }
}

/* 去掉与镜像默认值相同的启动命令（重建时 inspect 出的 Cmd 含镜像默认 CMD） */
watch(imageMeta, (meta) => {
  if (!meta?.cmd?.length || !form.cmd) return
  if (joinCommand(meta.cmd) === form.cmd) form.cmd = ''
})

/* Compose 导入 */
const COMPOSE_SAMPLE = 'services:\n  web:\n    image: nginx:1.27-alpine\n    ports:\n      - "8088:80"\n    volumes:\n      - /srv/web/html:/usr/share/nginx/html\n    restart: unless-stopped'
const composeOpen = ref(false)
const composeText = ref('')
const composeParsing = ref(false)
const composeServices = ref<DockerComposeService[]>([])
const composePicked = ref('')
const composeFile = ref<HTMLInputElement | null>(null)
watch(composeOpen, (open) => {
  if (open) {
    composeServices.value = []
    composePicked.value = ''
  }
})
const onComposeFile = async (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 512 * 1024) {
    feedback.error(t('docker.wizard.composeTooLarge'))
    return
  }
  composeText.value = await file.text()
}
const parseCompose = async () => {
  composeParsing.value = true
  try {
    const { data } = await parseDockerCompose({ content: composeText.value })
    const services = data?.services || []
    if (services.length === 1) {
      composeServices.value = services
      composePicked.value = services[0]?.name || ''
      applyCompose()
      return
    }
    composeServices.value = services
    composePicked.value = services[0]?.name || ''
  } catch (e) {
    showRequestError(e, t('docker.wizard.composeFailed'))
  } finally {
    composeParsing.value = false
  }
}
const applyCompose = () => {
  const svc = composeServices.value.find((s) => s.name === composePicked.value)
  if (!svc?.options) return
  applyOptions(svc.options)
  composeOpen.value = false
  if (svc.warnings.length) feedback.warning(t('docker.wizard.composeIgnored', { keys: svc.warnings.join(', ') }))
  else feedback.success(t('docker.wizard.composeApplied', { name: svc.name }))
}

/* 粘贴 .env */
const pasteOpen = ref(false)
const pasteText = ref('')
const openPaste = () => {
  pasteText.value = ''
  pasteOpen.value = true
}
const applyPaste = () => {
  const next = [...envs.value.filter((e) => e.key.trim())]
  let n = 0
  for (const raw of pasteText.value.split(/\r?\n/)) {
    const line = raw.trim().replace(/^export\s+/, '')
    if (!line || line.startsWith('#')) continue
    const i = line.indexOf('=')
    if (i <= 0) continue
    const key = line.slice(0, i).trim()
    let value = line.slice(i + 1).trim()
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) value = value.slice(1, -1)
    const existing = next.find((e) => e.key === key)
    if (existing) existing.value = value
    else next.push({ key, value })
    n++
  }
  envs.value = next
  pasteOpen.value = false
  feedback.success(t('docker.wizard.pasted', { n }))
}

/* ───────── 提交 ───────── */
const submitting = ref(false)
const buildOptions = (): DockerContainerCreateOptions => ({
  hostname: '', user: '', entrypoint: [], workingDir: '', labels: {}, tty: false, openStdin: false,
  autoRemove: false, privileged: false, platform: '',
  ...preserved.value,
  name: form.name,
  image: form.image,
  cmd: splitCommand(form.cmd),
  env: validEnvs.value.map((e) => `${e.key.trim()}=${e.value}`),
  network: form.network === 'bridge' ? '' : form.network,
  restartPolicy: form.restart === 'no' ? '' : form.restart,
  restartMaxRetries: form.restart === 'on-failure' ? Math.max(0, Number(form.retries) || 0) : 0,
  ports: validPorts.value.map((p) => ({ hostIp: p.hostIp.trim(), hostPort: p.hostPort.trim(), containerPort: `${p.containerPort.trim()}/${p.protocol}` })),
  mounts: validMounts.value.map((m) => ({ type: m.source ? (isHostPath(m.source) ? 'bind' : 'volume') : 'volume', source: m.source, target: m.target.trim(), readOnly: m.readOnly })),
  memory: memValue.value ? memValue.value * 1024 * 1024 : 0,
  memorySwap: 0,
  cpuShares: 0,
  cpuQuota: 0,
  cpuPeriod: 0,
  nanoCpus: cpuValue.value ? Math.round(cpuValue.value * 1e9) : 0,
  start: true,
})
const submit = async () => {
  if (!ready.value || submitting.value) return
  submitting.value = true
  try {
    const options = buildOptions()
    if (isRecreate.value && props.recreateId) {
      const { data } = await updateDockerContainer({
        id: props.recreateId, name: '', restartPolicy: options.restartPolicy, memory: options.memory, memorySwap: 0,
        cpuShares: 0, cpuQuota: 0, cpuPeriod: 0, nanoCpus: options.nanoCpus, recreate: true, force: form.force,
        removeVolumes: false, options,
      })
      feedback.success(t('docker.container.recreateTaskCreated'))
      emit('done', { task: data?.task })
    } else {
      const { data } = await createDockerContainer({ options })
      if (data?.task?.id) feedback.success(t('docker.wizard.pullingThenCreate'))
      else feedback.success(t('docker.container.createSuccess'))
      emit('done', { id: data?.id || undefined, task: data?.task })
    }
  } catch (e) {
    showRequestError(e, isRecreate.value ? t('docker.container.updateFailed') : t('docker.container.createFailed'))
  } finally {
    submitting.value = false
  }
}

/* ───────── 初始化 ───────── */
onMounted(async () => {
  document.addEventListener('pointerdown', onDocPointer)
  void loadTaken()
  void listDockerImages({ all: false, page: 1, pageSize: 500, keyword: '', usage: '', sort: 'created', dangling: undefined, labels: {} })
    .then(({ data }) => { localImages.value = data?.items || [] })
    .catch(() => {})
  void listDockerNetworks({ page: 1, pageSize: 200, name: '', driver: '', scope: '', labels: {}, kind: '', keyword: '' })
    .then(({ data }) => { networks.value = data?.items || [] })
    .catch(() => {})
  void getDockerConfig()
    .then(({ data }) => {
      const host = data?.config?.host || ''
      dockerLocal.value = !host || host.startsWith('unix://') || host.startsWith('npipe://')
    })
    .catch(() => {})
  if (isRecreate.value && props.recreateId) await loadRecreate(props.recreateId)
  else if (props.initialImage) pickImage(props.initialImage)
  else sugOpen.value = !isMobile.value
  await nextTick()
  setupSpy()
})
watch(isMobile, () => nextTick(setupSpy))
// 本地镜像列表晚到时，再取一次选中镜像的声明信息
watch(localImages, () => {
  if (form.image && !imageMeta.value) void loadImageMeta(form.image)
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocPointer)
  observer?.disconnect()
})
</script>

<style scoped lang="scss">
.ce {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 0;
}
.ce-head {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}
.ce-back {
  flex: none;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.ce-head__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ce-crumb {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.ce-crumb button {
  padding: 0;
  border: none;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}
.ce-crumb button:hover {
  color: var(--m-pInk);
}
.ce-title {
  font-family: var(--app-font-display);
  font-size: 2rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
  line-height: 1.15;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ce-stepline {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.ce-pill-btn {
  border-radius: 999px;
  font-weight: 700;
}
.ce-pill-btn:not(.ce-pill-btn--lg) {
  min-height: 36px;
}
.ce-pill-btn--lg {
  min-height: 44px;
  padding-inline: 20px;
}
.ce-loading {
  padding: 80px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}

/* 两栏 */
.ce-cols {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 24px;
  min-width: 0;
}
.ce-main {
  flex: 3 1 600px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ce-steps {
  position: sticky;
  top: 0;
  z-index: 3;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px;
  border-radius: 999px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow-x: auto;
  scrollbar-width: none;
}
.ce-steps::-webkit-scrollbar {
  display: none;
}
.ce-stp {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 14px 0 6px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.ce-stp:hover {
  background: var(--el-fill-color-light);
}
.ce-stp.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.ce-sn {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 800;
}
.ce-sn :deep(svg),
.ce-sn :deep(.iconify) {
  width: 15px;
  height: 15px;
}
.is-done .ce-sn {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.is-on .ce-sn {
  background: var(--m-p);
  color: var(--m-pOn);
}

/* 分区卡 */
.ce-sec {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 22px 24px 24px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  scroll-margin-top: 80px;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.ce-sec.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.ce-sec__head {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.ce-badge {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 13px;
  font-size: 1.0625rem;
  font-weight: 800;
  transform: rotate(var(--rot, 0deg));
}
.ce-sec__title {
  flex: 1;
  min-width: 0;
}
.ce-sec__title h2 {
  font-family: var(--app-font-display);
  font-size: 1.1875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.ce-sec__title p {
  margin-top: 2px;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.ce-chipbtn {
  display: inline-flex;
  align-items: center;
  height: 22px;
  margin: 0 2px;
  padding: 0 8px;
  border: none;
  border-radius: 8px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
}
.ce-chipbtn:hover {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}

/* 字段 */
.ce-fl {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.ce-fh {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.ce-fh.is-bad {
  color: var(--m-bad);
  font-weight: 700;
}
.ce-sm {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ce-grid2 {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 14px;
}
.ce-fld {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
  min-height: 44px;
  padding: 0 14px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  border: 1.5px solid transparent;
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s, background 0.15s;
}
.ce-fld:focus-within,
.ce-fld.is-focus {
  border-color: var(--m-p);
  background: var(--el-bg-color);
}
.ce-fld.is-bad {
  border-color: var(--m-bad);
}
.ce-fld > :deep(svg),
.ce-fld > :deep(.iconify) {
  flex: none;
  width: 16px;
  height: 16px;
}
.ce-fld input {
  flex: 1;
  min-width: 0;
  height: 40px;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  font-weight: 600;
}
.ce-fld input::placeholder {
  color: var(--el-text-color-placeholder);
  font-weight: 500;
}
.ce-sfx {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ce-kbd {
  flex: none;
  padding: 3px 8px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.ce-retry {
  max-width: 240px;
}
.ce-warn {
  padding: 8px 12px;
  border-radius: 14px;
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.8125rem;
  font-weight: 700;
  line-height: 1.5;
}

/* 镜像建议 */
.ce-sug {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px;
  border-radius: 18px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
}
.ce-sug__item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 56px;
  padding: 8px 12px;
  border: none;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
}
.ce-sug__item:hover {
  background: var(--el-fill-color-light);
}
.ce-sug__item.is-on {
  background: var(--m-pSoft);
}
.ce-sug__av {
  flex: none;
  width: 38px;
  height: 38px;
  border-radius: 12px;
  font-size: 0.9375rem;
  font-weight: 800;
}
.ce-sug__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ce-sug__ref {
  font-size: 0.8125rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ce-sug__desc {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ce-sug__check {
  flex: none;
  width: 20px;
  height: 20px;
  color: var(--m-pInk);
}
.ce-sug__empty {
  padding: 14px 12px;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.ce-bdg {
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 7px;
  font-size: 0.6875rem;
  font-weight: 700;
  white-space: nowrap;
}
.ce-bdg.is-ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.ce-bdg.is-a2 {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
.ce-picked {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.ce-picked__ok {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--m-ok);
}
.ce-picked__ref {
  font-size: 0.8125rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 50%;
}
.ce-picked__info {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 单选卡（网络 / 重启策略） */
.ce-opts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.ce-opt {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
  padding: 14px;
  border-radius: 18px;
  border: 2px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.ce-opt:hover {
  border-color: var(--m-p);
}
.ce-opt.is-on {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.ce-opt--col {
  flex-direction: column;
  gap: 4px;
  padding: 12px;
}
.ce-rd {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  margin-top: 1px;
  border-radius: 50%;
  border: 2px solid var(--el-border-color-light);
}
.is-on > .ce-rd {
  border-color: var(--m-p);
}
.is-on > .ce-rd::after {
  content: '';
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--m-p);
}
.ce-opt__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.ce-opt__t {
  font-size: 0.875rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ce-opt__d {
  font-size: 0.8125rem;
  font-weight: 600;
}
.ce-opt__d.is-sm {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}
.ce-opt__s {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ce-seg {
  align-self: flex-start;
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  padding: 3px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
}
.ce-seg button {
  min-height: 38px;
  padding: 0 14px;
  border: none;
  border-radius: 11px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.78125rem;
  font-weight: 700;
  cursor: pointer;
}
.ce-seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.ce-switch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  cursor: pointer;
}
.ce-switch > span {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.ce-switch strong {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.ce-switch small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

/* 就绪条 */
.ce-bar {
  position: sticky;
  bottom: 20px;
  z-index: 3;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 10px 10px 20px;
  border-radius: 999px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
}
.ce-bar__sub {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 摘要 */
.ce-side {
  flex: 2 1 380px;
  min-width: 0;
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 22px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.ce-side__head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.ce-side__head h2 {
  flex: 1;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.ce-side__id {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border-radius: 18px;
  background: var(--el-fill-color-light);
  min-width: 0;
}
.ce-side__sticker {
  flex: none;
  width: 48px;
  height: 48px;
  border-radius: 16px;
  transform: rotate(-4deg);
}
.ce-side__sticker :deep(svg),
.ce-side__sticker :deep(.iconify) {
  width: 24px;
  height: 24px;
}
.ce-side__idtext {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ce-side__name {
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ce-side__img {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ce-sum {
  margin: 0;
  display: flex;
  flex-direction: column;
}
.ce-sum > div {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding: 9px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.ce-sum dt {
  flex: none;
  width: 72px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ce-sum dd {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 0.78125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ce-cmdhead {
  display: flex;
  align-items: center;
  gap: 10px;
}
.ce-cmdhead__t {
  flex: 1;
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.ce-cmdhead .ce-sm {
  flex: 1;
}
/* 终端固定色（不随主题） */
.ce-term {
  margin: 0;
  padding: 16px 18px;
  border-radius: 18px;
  background: #0f0b14;
  color: #e8e3ee;
  font-size: 0.78125rem;
  line-height: 1.75;
  white-space: pre;
  overflow-x: auto;
  scrollbar-width: none;
}
.ce-term::-webkit-scrollbar {
  display: none;
}
.ce-term__dim {
  color: #9c93a8;
}
.ce-term__flag {
  color: var(--m-a);
}
.ce-term .is-strong {
  font-weight: 800;
}

/* 弹窗 */
.ce-dlg {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ce-dlg__ta {
  min-height: 220px;
  font-size: 0.78125rem;
}
.ce-dlg__row {
  display: flex;
  gap: 8px;
}
.ce-hidden {
  display: none;
}

/* —— 手机 —— */
.ce.is-mobile {
  gap: 12px;
}
.is-mobile .ce-head {
  gap: 4px;
  margin: -4px -8px 0;
}
.is-mobile .ce-back {
  border: none;
  background: transparent;
}
.is-mobile .ce-title {
  font-size: 1.25rem;
}
.ce-progress {
  height: 4px;
  margin: 0 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.ce-progress span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--m-p);
  transition: width 0.2s;
}
.ce-mchips {
  display: flex;
  gap: 8px;
  margin: 0 -16px;
  padding: 0 16px;
  overflow-x: auto;
  scrollbar-width: none;
}
.ce-mchips::-webkit-scrollbar {
  display: none;
}
.ce-mchip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 14px 0 6px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.ce-mchip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.is-mobile .ce-sec {
  padding: 4px 0 0;
  border: none;
  background: transparent;
  gap: 16px;
}
.is-mobile .ce-sec__title h2 {
  font-size: 1.25rem;
}
.is-mobile .ce-fl {
  font-size: 0.875rem;
}
.is-mobile .ce-fld {
  min-height: 48px;
}
.is-mobile .ce-fld input {
  height: 44px;
  font-size: 0.9375rem;
}
.is-mobile .ce-grid2:not(:has(#ce-cpu)) {
  grid-template-columns: minmax(0, 1fr);
}
.is-mobile .ce-grid2 {
  gap: 10px;
}
.is-mobile .ce-opts {
  grid-template-columns: minmax(0, 1fr);
  gap: 10px;
}
.is-mobile .ce-opts--grid2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.is-mobile .ce-sug {
  box-shadow: none;
  border-radius: 20px;
}
.is-mobile .ce-sug__item {
  min-height: 60px;
  padding: 8px 10px;
}
.is-mobile .ce-picked__ref {
  max-width: 60%;
}
.is-mobile .ce-picked__info {
  display: none;
}
.is-mobile .ce-term {
  padding: 14px 16px;
  font-size: 0.75rem;
}
.ce-mbar {
  position: sticky;
  bottom: calc(78px + env(safe-area-inset-bottom));
  z-index: 5;
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 4px -4px 0;
  padding: 10px;
  border-radius: 24px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
}
.ce-mbar__prev {
  flex: none;
  border-radius: 18px;
  background: var(--el-fill-color-light);
}
.ce-mbar__next {
  flex: 1;
  min-width: 0;
  min-height: 52px;
  justify-content: center;
  border-radius: 999px;
  font-size: 1rem;
  font-weight: 800;
}
@media (width < 1100px) {
  .ce-side {
    position: static;
  }
}
</style>
