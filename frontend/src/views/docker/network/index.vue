<!-- Docker 网络（第二期）：标题(引擎状态) + 统计说明 + 清理未使用/创建 → 子导航 → ListToolbar(全部/自定义/内置 chips + 搜索)
     → 桌面可展开行（展开看已连接容器的 IP/MAC/别名、断开/连接，右侧网络元信息）/ 手机卡片（同样可展开）→ 分页 → 内置网络说明。
     行内常用：连接容器 / 重建(编辑) / 删除，其余（详情、编辑标签）收进「⋯」。内置网络不能删除或重建。 -->
<template>
  <div class="dk-page">
    <DockerHeader @tasks="openTasks">
      <!-- 引擎不可用时没有统计，不再显示一句功能介绍，与容器 / 连接配置标签一致 -->
      <template v-if="stats" #description>
        {{ t('docker.network.statsLine', { total: stats.total, custom: stats.custom }) }}
        · {{ t('docker.network.statsIps') }} <strong class="dk-em m-num">{{ stats.allocatedIps }}</strong>
      </template>
      <template #actions>
        <UButton v-if="canManage && ready" color="neutral" variant="outline" icon="i-lucide-brush-cleaning" @click="handlePrune">
          {{ t('docker.network.prune') }}
        </UButton>
        <UButton v-if="canManage" color="primary" icon="i-lucide-plus" :disabled="!ready" @click="openCreate">
          {{ t('docker.network.createNetwork') }}
        </UButton>
      </template>
      <template #mobile-actions>
        <AppIconButton v-if="canManage && ready" class="dk-hdr__ib" icon="HOutline:SparklesIcon" :label="t('docker.network.prune')" :box="44" :size="22" @click="handlePrune" />
      </template>
    </DockerHeader>

    <DockerSubNav />

    <DockerUnavailable v-if="engineLoaded && !ready" @retry="init(true)" />

    <template v-else>
      <ListToolbar
        v-model:status="queryForm.kind"
        v-model:keyword="queryForm.keyword"
        :chips="kindChips"
        :chips-label="t('docker.network.kind')"
        :placeholder="t('docker.network.searchPlaceholder')"
        @search="search"
      />

      <div class="dk-page__body">
        <!-- 桌面：可展开行 -->
        <div v-if="!menuStore.isMobile" class="nt-table" role="table" :aria-label="t('docker.network.title')">
          <div class="nt-row nt-row--head" role="row">
            <span role="columnheader"><span class="sr-only">{{ t('docker.network.expand') }}</span></span>
            <span role="columnheader">{{ t('docker.common.name') }}</span>
            <span role="columnheader">{{ t('docker.common.driver') }}</span>
            <span role="columnheader">{{ t('docker.network.subnetGateway') }}</span>
            <span role="columnheader">{{ t('docker.common.scope') }}</span>
            <span role="columnheader">{{ t('docker.network.connectedContainers') }}</span>
            <span role="columnheader">{{ t('docker.network.attrs') }}</span>
            <span role="columnheader" class="is-right">{{ t('docker.common.operation') }}</span>
          </div>

          <template v-if="loading && !list.length">
            <div v-for="i in 4" :key="`sk-${i}`" class="nt-skel" />
          </template>
          <EmptyState
            v-else-if="!list.length"
            icon="HOutline:GlobeAltIcon"
            :title="t('docker.network.noNetworks')"
            :description="t('docker.network.emptyDesc')"
          />

          <div v-for="row in list" v-else :key="row.id" role="rowgroup" class="nt-group" :class="{ 'is-open': isOpen(row.id) }">
            <div class="nt-row" role="row">
              <span role="cell">
                <AppIconButton
                  :icon="isOpen(row.id) ? 'HOutline:ChevronDownIcon' : 'HOutline:ChevronRightIcon'"
                  :label="isOpen(row.id) ? t('docker.network.collapse') : t('docker.network.expand')"
                  :aria-expanded="isOpen(row.id)"
                  :box="36"
                  :size="18"
                  @click="toggle(row.id)"
                />
              </span>
              <span role="cell" class="nt-name">
                <span class="nt-sticker m-sticker" :class="stickerOf(row)" :style="{ transform: `rotate(${rotOf(row.name)}deg)` }">
                  <component :is="menuStore.iconComponents[iconOf(row)]" />
                </span>
                <span class="nt-name__text">
                  <span class="nt-name__main m-mono">{{ row.name }}</span>
                  <span class="nt-name__id m-mono">{{ shortId(row.id) }}</span>
                </span>
              </span>
              <span role="cell"><span class="nt-drv m-mono">{{ row.driver || '—' }}</span></span>
              <span role="cell" class="nt-subnet m-mono">
                <span>{{ subnetsOf(row).join(', ') || '—' }}</span>
                <small>{{ gatewaysOf(row).length ? `${t('docker.network.gateway')} ${gatewaysOf(row).join(', ')}` : '—' }}</small>
              </span>
              <span role="cell" class="m-mono dk-sub">{{ row.scope || '—' }}</span>
              <span role="cell" class="nt-ctrs">
                <template v-if="containersOf(row).length">
                  <span class="nt-avs">
                    <span
                      v-for="c in containersOf(row).slice(0, 4)"
                      :key="c.id"
                      class="nt-av m-sticker"
                      :class="stickerFor(c.name)"
                      :title="c.name"
                    >{{ initialOf(c.name) }}</span>
                  </span>
                  <strong class="m-num">{{ t('docker.network.ctrCount', { n: containersOf(row).length }) }}</strong>
                </template>
                <span v-else class="dk-sub">{{ t('docker.network.noCtr') }}</span>
              </span>
              <span role="cell" class="nt-badges">
                <span v-for="b in badgesOf(row)" :key="b.key" class="nt-bdg" :class="`is-${b.tone}`" :title="b.title">
                  <component :is="menuStore.iconComponents[b.icon]" />{{ b.label }}
                </span>
              </span>
              <span role="cell" class="dk-acts">
                <template v-if="canManage">
                  <AppIconButton
                    icon="HOutline:LinkIcon"
                    :label="t('docker.network.connectTo', { name: row.name })"
                    :box="36"
                    :size="18"
                    :disabled="!canConnect(row)"
                    @click="openConnect(row)"
                  />
                  <AppIconButton
                    icon="HOutline:LinkSlashIcon"
                    :label="t('docker.netx.disconnectFrom', { name: row.name })"
                    :box="36"
                    :size="18"
                    :disabled="!containersOf(row).length"
                    @click="openDisconnect(row)"
                  />
                  <AppIconButton
                    icon="HOutline:ArrowPathIcon"
                    :label="t('docker.network.recreate')"
                    :box="36"
                    :size="18"
                    :disabled="row.builtin"
                    @click="openEdit(row, true)"
                  />
                  <AppIconButton
                    class="is-bad"
                    icon="HOutline:TrashIcon"
                    :label="deleteLabel(row)"
                    :box="36"
                    :size="18"
                    :disabled="!canDelete(row)"
                    @click="handleDelete(row)"
                  />
                </template>
                <ActionMenu :items="menuActionsFor(row)" @select="(key) => onRowAction(key, row)" />
              </span>
            </div>

            <!-- 展开：已连接容器 + 元信息 -->
            <div v-if="isOpen(row.id)" class="nt-expand">
              <div class="nt-cbox">
                <div class="nt-crow nt-crow--head">
                  <span>{{ t('docker.network.container') }}</span><span>IPv4 / MAC</span><span>{{ t('docker.network.aliases') }}</span><span />
                </div>
                <div v-for="c in containersOf(row)" :key="c.id" class="nt-crow">
                  <span class="nt-cname">
                    <span class="nt-av m-sticker" :class="stickerFor(c.name)">{{ initialOf(c.name) }}</span>
                    <span class="nt-cname__text">
                      <strong>{{ c.name || shortId(c.id) }}</strong>
                      <span class="nt-state" :class="c.state === 'running' ? 'is-ok' : 'is-bad'">{{ stateLabel(c.state) }}</span>
                    </span>
                  </span>
                  <span class="nt-ip m-mono">
                    <strong>{{ c.ipv4Address || '—' }}</strong>
                    <small>{{ c.macAddress || '—' }}</small>
                  </span>
                  <span class="nt-aliases">
                    <span v-for="al in c.aliases" :key="al" class="dk-tag dk-tag--plain">{{ al }}</span>
                    <span v-if="!c.aliases.length" class="dk-dim">—</span>
                  </span>
                  <span class="is-right">
                    <UButton
                      v-if="canManage"
                      size="sm"
                      color="error"
                      variant="outline"
                      icon="i-lucide-unplug"
                      class="nt-disc"
                      @click="handleDisconnect(row, c.id, c.name)"
                    >
                      {{ t('docker.network.disconnect') }}
                    </UButton>
                  </span>
                </div>
                <div v-if="!containersOf(row).length" class="nt-cempty">{{ t('docker.network.noCtrLong') }}</div>
                <button v-if="canManage && canConnect(row)" type="button" class="nt-connect" @click="openConnect(row)">
                  <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />{{ t('docker.network.connectOther', { name: row.name }) }}
                </button>
              </div>
              <dl class="nt-meta">
                <div v-for="m in metaOf(row)" :key="m.k">
                  <dt>{{ m.k }}</dt>
                  <dd class="m-mono">{{ m.v }}</dd>
                </div>
              </dl>
            </div>
          </div>
        </div>

        <!-- 手机：卡片 -->
        <template v-else>
          <div v-if="loading && !list.length" class="dk-cards">
            <div v-for="i in 3" :key="i" class="dk-skeleton" />
          </div>
          <EmptyState
            v-else-if="!list.length"
            icon="HOutline:GlobeAltIcon"
            :title="t('docker.network.noNetworks')"
            :description="t('docker.network.emptyDesc')"
          />
          <div v-else class="dk-cards">
            <article v-for="row in list" :key="row.id" class="dk-card" :class="{ 'is-open': isOpen(row.id) }">
              <header class="dk-card__head">
                <span class="nt-sticker nt-sticker--lg m-sticker" :class="stickerOf(row)" :style="{ transform: `rotate(${rotOf(row.name)}deg)` }">
                  <component :is="menuStore.iconComponents[iconOf(row)]" />
                </span>
                <span class="nt-name__text">
                  <span class="nt-name__main m-mono">{{ row.name }}</span>
                  <span class="nt-name__id m-mono">{{ shortId(row.id) }}</span>
                </span>
                <span class="nt-drv m-mono">{{ row.driver || '—' }}</span>
              </header>
              <div v-if="badgesOf(row).length" class="nt-badges nt-badges--card">
                <span v-for="b in badgesOf(row)" :key="b.key" class="nt-bdg" :class="`is-${b.tone}`">
                  <component :is="menuStore.iconComponents[b.icon]" />{{ b.label }}
                </span>
              </div>
              <dl class="dk-card__info">
                <div><dt>{{ t('docker.network.subnet') }}</dt><dd class="m-mono">{{ subnetsOf(row).join(', ') || '—' }}</dd></div>
                <div><dt>{{ t('docker.network.gateway') }}</dt><dd class="m-mono">{{ gatewaysOf(row).join(', ') || '—' }}</dd></div>
                <div><dt>{{ t('docker.common.scope') }}</dt><dd class="m-mono">{{ row.scope || '—' }}</dd></div>
                <div><dt>{{ t('docker.network.connected') }}</dt><dd class="m-num">{{ t('docker.network.ctrCount', { n: containersOf(row).length }) }}</dd></div>
              </dl>
              <div v-if="isOpen(row.id)" class="nt-mlist">
                <div v-for="c in containersOf(row)" :key="c.id" class="nt-mrow">
                  <span class="nt-av m-sticker" :class="stickerFor(c.name)">{{ initialOf(c.name) }}</span>
                  <span class="nt-mrow__text">
                    <span class="nt-mrow__name">
                      <strong>{{ c.name || shortId(c.id) }}</strong>
                      <span class="nt-dot" :class="c.state === 'running' ? 'is-ok' : 'is-bad'" :title="stateLabel(c.state)" />
                    </span>
                    <span class="m-mono dk-sub">{{ c.ipv4Address || '—' }}<template v-if="c.aliases.length"> · {{ c.aliases.join(', ') }}</template></span>
                  </span>
                  <AppIconButton
                    v-if="canManage"
                    class="is-bad"
                    icon="HOutline:LinkSlashIcon"
                    :label="t('docker.network.disconnect')"
                    :box="44"
                    @click="handleDisconnect(row, c.id, c.name)"
                  />
                </div>
                <div v-if="!containersOf(row).length" class="nt-cempty">{{ t('docker.network.noCtrLong') }}</div>
                <button v-if="canManage && canConnect(row)" type="button" class="nt-connect nt-connect--flat" @click="openConnect(row)">
                  <component :is="menuStore.iconComponents['HOutline:PlusIcon']" />{{ t('docker.network.connectShort') }}
                </button>
              </div>
              <footer class="dk-card__foot">
                <div class="dk-card__lead nt-foot-lead">
                  <span v-if="containersOf(row).length" class="nt-avs">
                    <span v-for="c in containersOf(row).slice(0, 4)" :key="c.id" class="nt-av m-sticker" :class="stickerFor(c.name)">{{ initialOf(c.name) }}</span>
                  </span>
                  <span v-else class="dk-sub">{{ t('docker.network.noCtr') }}</span>
                </div>
                <template v-if="canManage">
                  <AppIconButton
                    icon="HOutline:LinkIcon"
                    :label="t('docker.network.connectTo', { name: row.name })"
                    :box="44"
                    :disabled="!canConnect(row)"
                    @click="openConnect(row)"
                  />
                  <AppIconButton
                    icon="HOutline:ArrowPathIcon"
                    :label="`${t('docker.network.recreate')} ${row.name}`"
                    :box="44"
                    :disabled="row.builtin"
                    @click="openEdit(row, true)"
                  />
                  <AppIconButton
                    class="is-bad"
                    icon="HOutline:TrashIcon"
                    :label="deleteLabel(row)"
                    :box="44"
                    :disabled="!canDelete(row)"
                    @click="handleDelete(row)"
                  />
                </template>
                <AppIconButton
                  :icon="isOpen(row.id) ? 'HOutline:ChevronUpIcon' : 'HOutline:ChevronDownIcon'"
                  :label="isOpen(row.id) ? t('docker.network.collapse') : t('docker.network.expand')"
                  :aria-expanded="isOpen(row.id)"
                  :box="44"
                  @click="toggle(row.id)"
                />
                <ActionMenu :items="menuActionsFor(row, true)" :label="t('docker.netx.moreOf', { name: row.name })" @select="(key) => onRowAction(key, row)" />
              </footer>
            </article>
          </div>
        </template>

        <Pagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          @change="getList"
        />

        <div class="nt-tip">
          <component :is="menuStore.iconComponents['HOutline:LightBulbIcon']" />
          <span>{{ t('docker.network.builtinTip') }}</span>
        </div>
      </div>
    </template>

    <!-- 手机：悬浮创建 -->
    <button v-if="menuStore.isMobile && canManage && ready" type="button" class="dk-fab" :aria-label="t('docker.network.createNetwork')" @click="openCreate">
      <UIcon name="i-lucide-plus" />
    </button>

    <!-- 从网络断开容器 -->
    <FormDialog v-model="disconnectVisible" :title="t('docker.netx.disconnectFrom', { name: disconnectNetwork?.name || '' })" :width="480" @confirm="submitDisconnect">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('docker.network.container') }}</label>
          <AppSelect v-model="disconnectId" :options="disconnectOptions" :placeholder="t('docker.network.pickContainer')" />
        </div>
        <label class="nt-switch nt-switch--box">
          <span><strong>{{ t('docker.netx.forceDisconnect') }}</strong><small>{{ t('docker.netx.forceDisconnectDesc') }}</small></span>
          <AppSwitch v-model="disconnectForce" />
        </label>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="error" :disabled="!disconnectId" @click="submitDisconnect">{{ t('docker.network.disconnect') }}</UButton>
      </template>
    </FormDialog>

    <!-- 创建网络 -->
    <FormDialog v-model="createVisible" :title="t('docker.network.createNetwork')" :width="560" :loading="createSubmitting" @confirm="submitCreate">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('docker.common.name') }}</label>
          <input v-model="createForm.name" class="app-input m-mono" :placeholder="t('docker.network.namePlaceholder')" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.driver') }}</label>
          <div class="nt-seg" role="radiogroup" :aria-label="t('docker.common.driver')">
            <button
              v-for="d in DRIVERS"
              :key="d"
              type="button"
              role="radio"
              class="nt-seg__btn m-mono"
              :class="{ 'is-on': createForm.driver === d }"
              :aria-checked="createForm.driver === d"
              @click="createForm.driver = d"
            >{{ d }}</button>
          </div>
        </div>
        <div class="dk-grid">
          <div class="app-field">
            <label class="app-label">{{ t('docker.network.subnet') }}</label>
            <input v-model="createForm.subnet" class="app-input m-mono" :placeholder="t('docker.network.subnetPlaceholder')" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('docker.network.gateway') }}</label>
            <input v-model="createForm.gateway" class="app-input m-mono" :placeholder="t('docker.network.gatewayPlaceholder')" />
          </div>
        </div>
        <div class="nt-switches">
          <label class="nt-switch">
            <span><strong>{{ t('docker.network.internal') }}</strong><small>{{ t('docker.network.internalDesc') }}</small></span>
            <AppSwitch v-model="createForm.internal" />
          </label>
          <label class="nt-switch">
            <span><strong>{{ t('docker.network.attachable') }}</strong><small>{{ t('docker.network.attachableDesc') }}</small></span>
            <AppSwitch v-model="createForm.attachable" />
          </label>
          <label class="nt-switch">
            <span><strong>IPv6</strong><small>{{ t('docker.network.ipv6Desc') }}</small></span>
            <AppSwitch v-model="createForm.enableIpv6" />
          </label>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="createSubmitting" @click="submitCreate">{{ t('docker.common.create') }}</UButton>
      </template>
    </FormDialog>

    <!-- 连接容器 -->
    <FormDialog v-model="connectVisible" :title="t('docker.network.connectTo', { name: connectNetwork?.name || '' })" :width="520" :loading="connectSubmitting" @confirm="submitConnect">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('docker.network.container') }}</label>
          <AppSelect
            v-model="connectForm.containerId"
            :options="connectOptions"
            searchable
            :placeholder="t('docker.network.pickContainer')"
            :no-match-text="t('docker.network.noCandidates')"
          />
        </div>
        <div class="dk-grid">
          <div class="app-field">
            <label class="app-label">{{ t('docker.network.aliases') }}</label>
            <input v-model="connectForm.aliases" class="app-input m-mono" :placeholder="t('docker.network.aliasesPlaceholder')" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('docker.network.fixedIp') }}</label>
            <input v-model="connectForm.ipv4" class="app-input m-mono" :placeholder="t('docker.common.optional')" />
          </div>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="connectSubmitting" @click="submitConnect">{{ t('docker.network.connect') }}</UButton>
      </template>
    </FormDialog>

    <!-- 网络详情 -->
    <FormDialog v-model="detailVisible" :title="t('docker.network.networkDetail')" :width="760" :show-footer="false">
      <div v-if="detail" class="dk-detail">
        <div class="dk-detail__hero">
          <span class="dk-detail__sticker m-sticker" :class="stickerOf(detail)"><component :is="menuStore.iconComponents[iconOf(detail)]" /></span>
          <div class="dk-detail__head">
            <div class="dk-detail__title">{{ detail.name }} <span class="nt-drv">{{ detail.driver }}</span></div>
            <div class="dk-detail__sub">{{ detail.id }}</div>
          </div>
        </div>
        <dl class="dk-kv">
          <div><dt>{{ t('docker.common.scope') }}</dt><dd>{{ detail.scope || '-' }}</dd></div>
          <div><dt>{{ t('docker.network.internal') }}</dt><dd>{{ detail.internal ? t('docker.common.yes') : t('docker.common.no') }}</dd></div>
          <div><dt>{{ t('docker.network.attachable') }}</dt><dd>{{ detail.attachable ? t('docker.common.yes') : t('docker.common.no') }}</dd></div>
          <div><dt>IPv6</dt><dd>{{ detail.enableIpv6 ? t('docker.common.enabled') : t('docker.common.notEnabled') }}</dd></div>
          <div><dt>IPAM</dt><dd>{{ detail.ipam?.driver || 'default' }}</dd></div>
          <div><dt>{{ t('docker.common.createdAt') }}</dt><dd class="m-num">{{ formatDateTime(detail.created) }}</dd></div>
        </dl>
        <div v-if="detail.ipam?.config?.length">
          <div class="dk-detail__label">{{ t('docker.network.ipamConfig') }}</div>
          <div class="dk-code">
            <div v-for="(item, index) in detail.ipam.config" :key="index">
              {{ item.subnet || '-' }} · {{ t('docker.network.gateway') }} {{ item.gateway || '-' }}<template v-if="item.ipRange"> · {{ item.ipRange }}</template>
            </div>
          </div>
        </div>
        <div v-if="Object.keys(detail.options || {}).length">
          <div class="dk-detail__label">{{ t('docker.network.driverOptions') }}</div>
          <div class="dk-tags"><span v-for="(v, k) in detail.options" :key="k" class="dk-tag dk-tag--plain">{{ k }}={{ v }}</span></div>
        </div>
        <div v-if="Object.keys(detail.labels || {}).length">
          <div class="dk-detail__label">{{ t('docker.common.labels') }}</div>
          <div class="dk-tags"><span v-for="(v, k) in detail.labels" :key="k" class="dk-tag dk-tag--plain">{{ k }}={{ v }}</span></div>
        </div>
      </div>
      <div v-else class="dk-detail__loading">{{ t('docker.common.noData') }}</div>
    </FormDialog>

    <!-- 编辑 / 重建网络 -->
    <FormDialog v-model="editVisible" :title="editRecreate ? t('docker.network.recreate') : t('docker.network.editNetwork')" :width="560" :loading="editSubmitting" @confirm="submitEdit">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.labels') }}</label>
          <textarea v-model="editLabelsText" class="app-textarea m-mono" rows="2" :placeholder="t('docker.common.lineKvPlaceholder')" />
        </div>
        <label class="nt-switch nt-switch--box">
          <span><strong>{{ t('docker.network.recreate') }}</strong><small>{{ t('docker.network.recreateTip') }}</small></span>
          <AppSwitch v-model="editRecreate" />
        </label>
        <template v-if="editRecreate">
          <div class="app-field">
            <label class="app-label">{{ t('docker.common.driver') }}</label>
            <div class="nt-seg" role="radiogroup" :aria-label="t('docker.common.driver')">
              <button
                v-for="d in DRIVERS"
                :key="d"
                type="button"
                role="radio"
                class="nt-seg__btn m-mono"
                :class="{ 'is-on': editDriver === d }"
                :aria-checked="editDriver === d"
                @click="editDriver = d"
              >{{ d }}</button>
            </div>
          </div>
          <div class="dk-grid">
            <div class="app-field">
              <label class="app-label">{{ t('docker.network.subnet') }}</label>
              <input v-model="editSubnet" class="app-input m-mono" :placeholder="t('docker.network.subnetPlaceholder')" />
            </div>
            <div class="app-field">
              <label class="app-label">{{ t('docker.network.gateway') }}</label>
              <input v-model="editGateway" class="app-input m-mono" :placeholder="t('docker.network.gatewayPlaceholder')" />
            </div>
          </div>
          <label class="nt-switch">
            <span><strong>{{ t('docker.common.forceRecreate') }}</strong><small>{{ t('docker.network.forceDesc') }}</small></span>
            <AppSwitch v-model="editForce" />
          </label>
        </template>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="editSubmitting" @click="submitEdit">{{ t('docker.common.save') }}</UButton>
      </template>
    </FormDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import DockerSubNav from '@/views/docker/components/DockerSubNav.vue'
import DockerHeader from '@/views/docker/components/DockerHeader.vue'
import DockerUnavailable from '@/views/docker/components/DockerUnavailable.vue'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import { useDockerStatus } from '@/views/docker/components/useDockerStatus'
import { formatDateTime, shortId, stickerFor } from '@/views/docker/components/format'
import {
  connectDockerNetwork, createDockerNetwork, deleteDockerNetwork, disconnectDockerNetwork,
  getDockerNetwork, listDockerContainers, listDockerNetworks, pruneDockerNetworks, updateDockerNetwork,
} from '@/api/docker'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DockerContainerSummary, DockerNetworkInfo, DockerNetworkListStats, DockerTaskInfo } from '@/types/v1/docker'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'DockerNetworkView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const canManage = useButtonPermission([PERM.DOCKER_NETWORK_MANAGE], [])
const { ready, loaded: engineLoaded, refresh: refreshEngine } = useDockerStatus()

type Kind = '' | 'custom' | 'builtin'
const list = ref<DockerNetworkInfo[]>([])
const stats = ref<DockerNetworkListStats | null>(null)
const loading = ref(false)
const queryForm = reactive({ kind: '' as Kind, keyword: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })
const openIds = ref(new Set<string>())

const DRIVERS = ['bridge', 'overlay', 'macvlan', 'ipvlan']

const kindChips = computed(() => [
  { label: t('docker.common.all'), value: '' as Kind, count: stats.value?.total ?? null },
  { label: t('docker.network.kindCustom'), value: 'custom' as Kind, count: stats.value?.custom ?? null },
  { label: t('docker.network.kindBuiltin'), value: 'builtin' as Kind, count: stats.value?.builtin ?? null },
])

const isOpen = (id: string) => openIds.value.has(id)
const toggle = (id: string) => {
  const next = new Set(openIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  openIds.value = next
}

const subnetsOf = (row: DockerNetworkInfo) => (row.ipam?.config || []).map((i) => i.subnet).filter(Boolean)
const gatewaysOf = (row: DockerNetworkInfo) => (row.ipam?.config || []).map((i) => i.gateway).filter(Boolean)
const containersOf = (row: DockerNetworkInfo) =>
  Object.entries(row.containers || {})
    .map(([id, c]) => ({ id, name: c.name, ipv4Address: c.ipv4Address, macAddress: c.macAddress, state: c.state || '', aliases: (c.aliases || []).filter((a) => a && !id.startsWith(a)) }))
    .sort((a, b) => a.name.localeCompare(b.name))
const initialOf = (name: string) => (name || '?').replace(/^\//, '').charAt(0).toUpperCase()
const rotOf = (name: string) => [-4, 3, -2, 4, -5][(name || '').length % 5]
const iconOf = (row: DockerNetworkInfo) => {
  if (row.name === 'host') return 'HOutline:ComputerDesktopIcon'
  if (row.name === 'none' || row.driver === 'null') return 'HOutline:NoSymbolIcon'
  if (row.internal) return 'HOutline:ShieldCheckIcon'
  return 'HOutline:GlobeAltIcon'
}
const stickerOf = (row: DockerNetworkInfo) => (row.builtin ? 'm-sticker--a2' : stickerFor(row.name))
const badgesOf = (row: DockerNetworkInfo) => {
  const out: { key: string; label: string; title: string; icon: string; tone: string }[] = []
  if (row.builtin) out.push({ key: 'b', label: t('docker.network.badgeBuiltin'), title: 'built-in', icon: 'HOutline:LockClosedIcon', tone: 'muted' })
  if (row.internal) out.push({ key: 'i', label: t('docker.network.badgeInternal'), title: t('docker.network.internalDesc'), icon: 'HOutline:ShieldCheckIcon', tone: 'warn' })
  if (row.attachable) out.push({ key: 'a', label: t('docker.network.attachable'), title: t('docker.network.attachableDesc'), icon: 'HOutline:LinkIcon', tone: 'a2' })
  if (row.enableIpv6) out.push({ key: '6', label: 'IPv6', title: 'IPv6', icon: 'HOutline:GlobeAltIcon', tone: 'muted' })
  return out
}
const canConnect = (row: DockerNetworkInfo) => row.name !== 'host' && row.name !== 'none' && row.driver !== 'null'
const canDelete = (row: DockerNetworkInfo) => !row.builtin && !containersOf(row).length
const deleteLabel = (row: DockerNetworkInfo) =>
  row.builtin ? t('docker.network.builtinNoDelete') : containersOf(row).length ? t('docker.network.inUseNoDelete') : t('docker.common.delete')
const STATE_KEYS: Record<string, string> = {
  running: 'docker.states.running', exited: 'docker.states.exited', paused: 'docker.states.paused',
  restarting: 'docker.states.restarting', created: 'docker.states.created', dead: 'docker.states.dead',
}
const stateLabel = (s: string) => (STATE_KEYS[s] ? t(STATE_KEYS[s] as string) : s || '—')
const metaOf = (row: DockerNetworkInfo) => {
  const opts = row.options || {}
  const items = [
    { k: t('docker.network.networkId'), v: row.id.slice(0, 16) + (row.id.length > 16 ? '…' : '') },
    { k: t('docker.common.createdAt'), v: formatDateTime(row.created) },
    { k: 'IPAM', v: `${row.ipam?.driver || 'default'} · ${subnetsOf(row).join(', ') || (row.enableIpv6 ? 'IPv4 / IPv6' : 'IPv4')}` },
  ]
  const mtu = opts['com.docker.network.driver.mtu']
  const bridge = opts['com.docker.network.bridge.name']
  if (mtu) items.push({ k: 'MTU', v: mtu })
  if (bridge) items.push({ k: t('docker.network.bridgeName'), v: bridge })
  else if (row.driver === 'bridge') items.push({ k: t('docker.network.bridgeName'), v: `br-${row.id.slice(0, 12)}` })
  const project = row.labels?.['com.docker.compose.project']
  if (project) items.push({ k: 'Compose', v: project })
  if (row.name === 'host') items.push({ k: t('docker.network.note'), v: t('docker.network.hostNote') })
  if (row.name === 'none') items.push({ k: t('docker.network.note'), v: t('docker.network.noneNote') })
  return items
}

const menuActionsFor = (row: DockerNetworkInfo, mobile = false): ActionMenuItem[] => {
  const manage = canManage.value
  return [
    { key: 'detail', label: t('docker.common.detail'), icon: 'HOutline:InformationCircleIcon' },
    { key: 'disconnect', label: t('docker.netx.disconnectShort'), icon: 'HOutline:LinkSlashIcon', hidden: !manage || !mobile || !containersOf(row).length },
    { key: 'edit', label: t('docker.network.editLabels'), icon: 'HOutline:PencilSquareIcon', hidden: !manage || row.builtin },
  ]
}
const onRowAction = (key: string, row: DockerNetworkInfo) => {
  if (key === 'detail') void openDetail(row)
  else if (key === 'edit') void openEdit(row, false)
  else if (key === 'recreate') void openEdit(row, true)
  else if (key === 'connect') void openConnect(row)
  else if (key === 'disconnect') openDisconnect(row)
}

const getList = async () => {
  if (!ready.value) return
  loading.value = true
  try {
    const { data } = await listDockerNetworks({
      page: pagination.value.page, pageSize: pagination.value.pageSize,
      name: '', driver: '', scope: '', labels: {}, kind: queryForm.kind, keyword: queryForm.keyword.trim(),
    })
    list.value = data?.items || []
    stats.value = data?.stats || null
    pagination.value.total = Number(data?.total || 0)
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally { loading.value = false }
}
const search = () => { pagination.value.page = 1; void getList() }
const init = async (force = false) => {
  await refreshEngine(force)
  if (ready.value) await getList()
}

// —— 任务 ——
const taskDialogsVisible = ref(false)
const activeTask = ref<DockerTaskInfo | null>(null)
const openTasks = () => { activeTask.value = null; taskDialogsVisible.value = true }
const openTask = (task: DockerTaskInfo | undefined) => {
  if (!task?.id) return
  activeTask.value = task
  taskDialogsVisible.value = true
}
const handleTaskFinished = async () => { await getList() }

// —— 清理 ——
const handlePrune = () => {
  Dialog.confirm({
    title: t('docker.network.pruneTitle'),
    content: t('docker.network.pruneContent'),
    confirmText: t('docker.network.pruneConfirm'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      try {
        const { data } = await pruneDockerNetworks({})
        const n = data?.deleted?.length || 0
        feedback.success(n ? t('docker.network.pruneSuccess', { n, names: (data?.deleted || []).join('、') }) : t('docker.network.pruneNone'))
        await getList()
        void refreshEngine(true)
      } catch (e) { showRequestError(e, t('docker.network.pruneFailed')) }
    },
  })
}

// —— 创建 ——
const createVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive({ name: '', driver: 'bridge', internal: false, enableIpv6: false, attachable: false, subnet: '', gateway: '' })
const openCreate = () => {
  Object.assign(createForm, { name: '', driver: 'bridge', internal: false, enableIpv6: false, attachable: false, subnet: '', gateway: '' })
  createVisible.value = true
}
const submitCreate = async () => {
  const name = createForm.name.trim()
  if (!name) { feedback.error(t('docker.network.enterName')); return }
  createSubmitting.value = true
  try {
    const ipamConfig = createForm.subnet.trim() ? [{ subnet: createForm.subnet.trim(), gateway: createForm.gateway.trim(), ipRange: '', auxAddress: {} }] : []
    await createDockerNetwork({
      options: {
        name, driver: createForm.driver, internal: createForm.internal,
        enableIpv6: createForm.enableIpv6, attachable: createForm.attachable,
        ingress: false, scope: '', enableIpv4: undefined,
        ipam: ipamConfig.length ? { driver: 'default', options: {}, config: ipamConfig } : undefined,
        options: {}, labels: {},
      },
    })
    feedback.success(t('docker.network.createSuccess'))
    createVisible.value = false
    await getList()
    void refreshEngine(true)
  } catch (e) { showRequestError(e, t('docker.network.createFailed')) }
  finally { createSubmitting.value = false }
}

// —— 删除 ——
const handleDelete = (row: DockerNetworkInfo) => {
  if (!canDelete(row)) return
  Dialog.confirm({
    title: t('docker.network.confirmDeleteTitle'),
    content: t('docker.network.confirmDeleteContent', { name: row.name }),
    confirmText: t('docker.network.confirmDeleteText'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      try {
        await deleteDockerNetwork({ id: row.id })
        feedback.success(t('docker.common.deletedName', { name: row.name }))
        await getList()
        void refreshEngine(true)
      } catch (e) { showRequestError(e, t('docker.network.deleteFailed')) }
    },
  })
}

// —— 连接 / 断开 ——
const connectVisible = ref(false)
const connectSubmitting = ref(false)
const connectNetwork = ref<DockerNetworkInfo | null>(null)
const connectCandidates = ref<DockerContainerSummary[]>([])
const connectForm = reactive({ containerId: '', aliases: '', ipv4: '' })
const connectOptions = computed(() => {
  const joined = new Set(Object.keys(connectNetwork.value?.containers || {}))
  return connectCandidates.value
    .filter((c) => !joined.has(c.id))
    .map((c) => ({ label: `${(c.names[0] || c.id).replace(/^\//, '')} · ${c.image}`, value: c.id }))
})
const openConnect = async (row: DockerNetworkInfo) => {
  if (!canConnect(row)) return
  connectNetwork.value = row
  Object.assign(connectForm, { containerId: '', aliases: '', ipv4: '' })
  connectVisible.value = true
  try {
    const { data } = await listDockerContainers({ page: 1, pageSize: 500, status: '', keyword: '', network: '', labels: {} })
    connectCandidates.value = data?.items || []
  } catch { connectCandidates.value = [] }
}
const submitConnect = async () => {
  if (!connectNetwork.value) return
  if (!connectForm.containerId) { feedback.error(t('docker.network.pickContainer')); return }
  connectSubmitting.value = true
  try {
    await connectDockerNetwork({
      networkId: connectNetwork.value.id,
      containerId: connectForm.containerId,
      aliases: connectForm.aliases.split(/[\s,]+/).map((s) => s.trim()).filter(Boolean),
      ipv4Address: connectForm.ipv4.trim(),
      ipv6Address: '',
    })
    feedback.success(t('docker.network.connectSuccess'))
    connectVisible.value = false
    openIds.value = new Set([...openIds.value, connectNetwork.value.id])
    await getList()
  } catch (e) { showRequestError(e, t('docker.network.connectFailed')) }
  finally { connectSubmitting.value = false }
}
const handleDisconnect = (network: DockerNetworkInfo, containerId: string, name: string) => {
  Dialog.confirm({
    title: t('docker.network.disconnectTitle'),
    content: t('docker.network.disconnectNamed', { name: name || containerId.slice(0, 12), network: network.name }),
    confirmText: t('docker.network.disconnectConfirm'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      try {
        await disconnectDockerNetwork({ networkId: network.id, containerId, force: false })
        feedback.success(t('docker.network.disconnectSuccess'))
        await getList()
      } catch (e) { showRequestError(e, t('docker.network.disconnectFailed')) }
    },
  })
}

// 行内「断开」：只连了一个容器时直接确认，多个时先选容器
const disconnectVisible = ref(false)
const disconnectNetwork = ref<DockerNetworkInfo | null>(null)
const disconnectId = ref('')
const disconnectForce = ref(false)
const disconnectOptions = computed(() =>
  disconnectNetwork.value ? containersOf(disconnectNetwork.value).map((c) => ({ label: `${c.name || shortId(c.id)}${c.ipv4Address ? ` · ${c.ipv4Address}` : ''}`, value: c.id })) : [],
)
const openDisconnect = (row: DockerNetworkInfo) => {
  const ctrs = containersOf(row)
  if (!ctrs.length) return
  if (ctrs.length === 1 && ctrs[0]) { handleDisconnect(row, ctrs[0].id, ctrs[0].name); return }
  disconnectNetwork.value = row
  disconnectId.value = ''
  disconnectForce.value = false
  disconnectVisible.value = true
}
const submitDisconnect = async () => {
  const network = disconnectNetwork.value
  if (!network || !disconnectId.value) return
  try {
    await disconnectDockerNetwork({ networkId: network.id, containerId: disconnectId.value, force: disconnectForce.value })
    feedback.success(t('docker.network.disconnectSuccess'))
    disconnectVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.network.disconnectFailed')) }
}

// —— 详情 ——
const detailVisible = ref(false)
const detail = ref<DockerNetworkInfo | null>(null)
const openDetail = async (row: DockerNetworkInfo) => {
  detailVisible.value = true
  detail.value = null
  try { const { data } = await getDockerNetwork({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.network.getDetailFailed')) }
}

// —— 编辑 / 重建 ——
const editVisible = ref(false)
const editSubmitting = ref(false)
const editId = ref('')
const editLabelsText = ref('')
const editRecreate = ref(false)
const editForce = ref(false)
const editDriver = ref('bridge')
const editSubnet = ref('')
const editGateway = ref('')
const openEdit = async (row: DockerNetworkInfo, recreate: boolean) => {
  if (row.builtin) return
  editId.value = row.id
  editRecreate.value = recreate
  editForce.value = false
  editDriver.value = row.driver || 'bridge'
  editSubnet.value = subnetsOf(row)[0] || ''
  editGateway.value = gatewaysOf(row)[0] || ''
  editLabelsText.value = Object.entries(row.labels || {}).map(([k, v]) => `${k}=${v}`).join('\n')
  editVisible.value = true
  try {
    const { data } = await getDockerNetwork({ id: row.id })
    const info = data?.info
    editLabelsText.value = info?.labels ? Object.entries(info.labels).map(([k, v]) => `${k}=${v}`).join('\n') : ''
    editDriver.value = info?.driver || editDriver.value
  } catch { /* 用列表里的值 */ }
}
const parseLabels = (text: string) => {
  const obj: Record<string, string> = {}
  text.trim().split('\n').filter(Boolean).forEach((line) => {
    const idx = line.indexOf('=')
    if (idx > 0) obj[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  })
  return obj
}
const submitEdit = async () => {
  editSubmitting.value = true
  try {
    if (editRecreate.value) {
      const ipamConfig = editSubnet.value.trim() ? [{ subnet: editSubnet.value.trim(), gateway: editGateway.value.trim(), ipRange: '', auxAddress: {} }] : []
      const { data } = await updateDockerNetwork({
        id: editId.value, labels: parseLabels(editLabelsText.value), force: editForce.value,
        options: {
          name: '', driver: editDriver.value, scope: '', enableIpv4: undefined, enableIpv6: undefined,
          internal: false, attachable: false, ingress: false,
          ipam: ipamConfig.length ? { driver: 'default', options: {}, config: ipamConfig } : undefined,
          options: {}, labels: parseLabels(editLabelsText.value),
        },
      })
      openTask(data?.task)
    } else {
      await updateDockerNetwork({ id: editId.value, labels: parseLabels(editLabelsText.value), force: false, options: undefined })
    }
    feedback.success(editRecreate.value ? t('docker.network.recreateTaskCreated') : t('docker.network.updateSuccess'))
    editVisible.value = false
    if (!editRecreate.value) await getList()
  } catch (e) { showRequestError(e, t('docker.network.updateFailed')) }
  finally { editSubmitting.value = false }
}

onMounted(() => { void init() })
</script>

<style scoped lang="scss">
@use '../components/dk-page.scss';

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
}
.dk-em {
  color: var(--el-text-color-primary);
  font-weight: 800;
}
.dk-sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.is-right {
  text-align: right;
  justify-self: end;
}

/* 桌面可展开表 */
.nt-table {
  padding: 4px 8px 8px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow-x: auto;
}
.nt-row {
  display: grid;
  grid-template-columns: 44px minmax(190px, 1.4fr) 92px minmax(150px, 1fr) 70px minmax(140px, 1fr) minmax(110px, 0.9fr) 160px;
  align-items: center;
  gap: 12px;
  min-width: 1040px;
  min-height: 68px;
  padding: 6px 8px;
}
.nt-row--head {
  min-height: 44px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.nt-group + .nt-group .nt-row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.nt-group.is-open > .nt-row {
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius) var(--app-radius) 0 0;
}
.nt-skel {
  height: 60px;
  margin: 8px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.nt-name {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.nt-sticker {
  width: 38px;
  height: 38px;
  border-radius: 13px;
}
.nt-sticker :deep(svg) {
  width: 19px;
  height: 19px;
}
.nt-sticker--lg {
  width: 44px;
  height: 44px;
  border-radius: 15px;
}
.nt-name__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.nt-name__main {
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nt-name__id {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.nt-drv {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-family: var(--app-font-mono);
  font-size: 0.75rem;
  font-weight: 700;
}
.nt-subnet {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.nt-subnet small {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.nt-ctrs {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.8125rem;
  min-width: 0;
}
.nt-avs {
  display: flex;
  padding-left: 6px;
}
.nt-av {
  width: 26px;
  height: 26px;
  margin-left: -6px;
  border-radius: 50%;
  border: 2px solid var(--el-bg-color);
  font-size: 0.6875rem;
}
.nt-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.nt-badges--card {
  margin-top: -4px;
}
.nt-bdg {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 0.71875rem;
  font-weight: 700;
  white-space: nowrap;
}
.nt-bdg :deep(svg) {
  width: 13px;
  height: 13px;
}
.nt-bdg.is-muted {
  background: var(--m-s2);
  color: var(--m-ink2);
}
.nt-bdg.is-warn {
  background: var(--m-warnSoft);
  color: var(--m-warn);
}
.nt-bdg.is-a2 {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}

/* 展开区 */
.nt-expand {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  min-width: 1040px;
  padding: 4px 12px 16px 60px;
  background: var(--el-fill-color-light);
  border-radius: 0 0 var(--app-radius) var(--app-radius);
}
.nt-cbox {
  flex: 2 1 0;
  min-width: 0;
  padding: 4px 4px 8px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.nt-crow {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr) minmax(0, 1fr) 96px;
  align-items: center;
  gap: 12px;
  min-height: 56px;
  padding: 6px 10px;
}
.nt-crow + .nt-crow {
  border-top: 1px solid var(--el-border-color-lighter);
}
.nt-crow--head {
  min-height: 38px;
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--el-text-color-secondary);
}
.nt-cname {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.nt-cname .nt-av,
.nt-mrow .nt-av {
  margin-left: 0;
  flex: none;
  width: 30px;
  height: 30px;
  border: none;
}
.nt-cname__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  font-size: 0.8125rem;
}
.nt-cname__text strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--el-text-color-primary);
}
.nt-state {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 0.75rem;
  font-weight: 600;
}
.nt-state::before,
.nt-dot {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
.nt-state.is-ok,
.nt-dot.is-ok {
  color: var(--m-ok);
}
.nt-state.is-bad,
.nt-dot.is-bad {
  color: var(--m-bad);
}
.nt-dot {
  display: inline-block;
  flex: none;
  width: 7px;
  height: 7px;
}
.nt-ip {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  font-size: 0.8125rem;
}
.nt-ip small {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
}
.nt-aliases {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}
.nt-disc {
  border-radius: 999px;
}
.nt-cempty {
  padding: 16px 12px;
  text-align: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.nt-connect {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: calc(100% - 16px);
  min-height: 44px;
  margin: 8px 8px 0;
  border: 1.5px dashed var(--el-border-color-light);
  border-radius: var(--app-radius);
  background: transparent;
  color: var(--m-pInk);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.nt-connect:hover {
  border-color: var(--m-p);
  background: var(--m-pSoft);
}
.nt-connect :deep(svg) {
  width: 16px;
  height: 16px;
}
.nt-meta {
  flex: 1 1 0;
  min-width: 240px;
  margin: 0;
  padding: calc(var(--cmp-card-unit) * 14) calc(var(--cmp-card-unit) * 18);
  display: flex;
  flex-direction: column;
  gap: 12px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.nt-meta dt {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.nt-meta dd {
  margin: 2px 0 0;
  font-size: 0.8125rem;
  font-weight: 600;
  word-break: break-all;
  color: var(--el-text-color-primary);
}

/* 手机展开 */
.dk-card.is-open {
  border-color: var(--m-p);
}
.nt-mlist {
  display: flex;
  flex-direction: column;
  margin-right: 6px;
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-light);
  overflow: hidden;
}
.nt-mrow {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 56px;
  padding: 4px 0 4px 12px;
}
.nt-mrow + .nt-mrow {
  border-top: 1px solid var(--el-border-color-lighter);
}
.nt-mrow__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.nt-mrow__text > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nt-mrow__name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.875rem;
  min-width: 0;
}
.nt-mrow__name strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nt-connect--flat {
  width: 100%;
  margin: 0;
  border: none;
  border-top: 1px dashed var(--el-border-color-light);
  border-radius: 0;
  min-height: 48px;
}
.nt-foot-lead {
  align-items: center;
  padding-left: 2px;
}

/* 说明条 */
.nt-tip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 18px;
  border-radius: var(--app-radius-lg);
  background: var(--m-aSoft);
  color: var(--m-ink);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.5;
}
.nt-tip :deep(svg) {
  flex: none;
  width: 20px;
  height: 20px;
}

/* 表单 */
.nt-seg {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  padding: 3px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.nt-seg__btn {
  flex: 1;
  min-width: 72px;
  min-height: 36px;
  border: none;
  border-radius: 9px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.nt-seg__btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.nt-switches {
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-lighter);
}
.nt-switch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 56px;
  padding: 8px 14px;
  cursor: pointer;
}
.nt-switches .nt-switch + .nt-switch {
  border-top: 1px solid var(--el-border-color-lighter);
}
.nt-switch--box {
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.nt-switch > span {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.nt-switch strong {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.nt-switch small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
