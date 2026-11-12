<!-- 文件管理：页头(标题 + 当前目录汇总 + 新建/上传) + 页面子导航插槽 + 来源胶囊(带容量) +
     路径条(返回/前进/上级 · 面包屑 · 复制路径/刷新) + 搜索/排序/视图 +
     三栏(左：可收起的目录树 + 上传队列 · 中：列表/网格 · 右：文件预览) + 剪贴板条 + 底部悬浮深色批量条。
     文件编辑为弹窗（FileEditor）。手机：单栏列表 + 上传进度卡，行「⋯」打开底部操作面板，右下角悬浮上传按钮。
     逻辑全部复用 fileClient（scope 无关，系统级/实例级同构）。 -->
<template>
  <div class="fb" :class="{ 'is-mobile': isMobile, 'fb--inset': inset }">
    <div class="fb__inner">
      <!-- 页头（手机端由 PageHeader 自带深浅色切换；实例文件页内嵌时不带返回/切换） -->
      <PageHeader
        class="fb-head"
        :class="{ 'fb-head--embedded': !isSystemScope }"
        :title="t('instance.fileManagerTitle')"
        :back="isSystemScope ? 'auto' : false"
        :theme-toggle="isSystemScope"
      >
        <template #description>
          <span class="m-num fb-head__desc">
            <span class="m-mono fb-head__path">{{ displayPath }}</span>
            <span> · {{ t('fileManager.summaryShort', { folders: folderCount, files: fileCount }) }}</span>
            <span v-if="dirSize"> · {{ formatFileSize(dirSize) }}</span>
          </span>
        </template>
        <template #actions>
          <template v-if="!isMobile">
            <UButton color="neutral" variant="soft" icon="i-lucide-file-plus" @click="openCreateFile">
              {{ t('fileManager.createFile') }}
            </UButton>
            <UButton color="neutral" variant="outline" icon="i-lucide-folder-plus" @click="openCreateFolder">
              {{ t('fileManager.createFolder') }}
            </UButton>
            <UButton color="primary" icon="i-lucide-upload" @click="pickUpload">
              {{ t('fileManager.upload') }}
            </UButton>
          </template>
          <AppDropdown v-else align="end" :width="200">
            <template #trigger>
              <button type="button" class="fb-ico fb-ico--lg" :aria-label="t('fileManager.newItem')" :title="t('fileManager.newItem')">
                <UIcon name="i-lucide-plus" />
              </button>
            </template>
            <template #default="{ close }">
              <div class="fb-menu">
                <button type="button" class="fb-menu__item" @click="() => { openCreateFolder(); close() }">
                  <UIcon name="i-lucide-folder-plus" class="fb-menu__ico" />{{ t('fileManager.createFolder') }}
                </button>
                <button type="button" class="fb-menu__item" @click="() => { openCreateFile(); close() }">
                  <UIcon name="i-lucide-file-plus" class="fb-menu__ico" />{{ t('fileManager.createFile') }}
                </button>
              </div>
            </template>
          </AppDropdown>
        </template>
      </PageHeader>

      <slot name="nav" />

      <!-- 来源切换 + 磁盘用量 -->
      <div v-if="isSystemScope || usage" class="fb-sources">
        <div v-if="isSystemScope" class="fb-sources__chips" role="tablist" :aria-label="t('fileSource.switchSource')">
          <button
            v-for="(chip, i) in sourceChips"
            :key="chip.id || 'local'"
            type="button"
            role="tab"
            class="fb-chip"
            :class="{ 'is-on': chip.id === currentSourceId }"
            :aria-selected="chip.id === currentSourceId"
            @click="chip.id !== currentSourceId && onSourceChange(chip.id)"
          >
            <span class="fb-chip__sticker m-sticker" :class="`m-sticker--${chip.tone}`" :style="{ transform: `rotate(${ROTS[i % ROTS.length]})` }">
              <UIcon :name="chip.icon" />
            </span>
            <span class="fb-chip__name">{{ chip.name }}</span>
            <span class="fb-chip__type">{{ chip.type }}</span>
          </button>
          <RouterLink v-if="canManageSources && !isMobile" :to="pageTabTo('file', 'source')" class="fb-chip fb-chip--dashed">
            <UIcon name="i-lucide-database" class="fb-chip__ico" />
            {{ t('fileManager.manageSources') }}
          </RouterLink>
        </div>
        <span class="fb-sources__spacer" />
        <div v-if="usage" class="fb-disk" :title="usage.title">
          <span class="fb-disk__text m-num" :class="{ 'is-warn': usage.warn }">{{ usage.text }}</span>
          <span class="fb-disk__bar"><span :style="{ width: `${usage.pct}%` }" :class="{ 'is-warn': usage.warn }" /></span>
        </div>
      </div>

      <!-- 路径条 + 搜索/排序/视图 -->
      <div class="fb-bar">
        <nav class="fb-crumb" :aria-label="t('fileManager.filePath')">
          <button type="button" class="fb-ico" :disabled="!canGoBack" :aria-label="t('fileManager.back')" :title="t('fileManager.back')" @click="goBack">
            <UIcon name="i-lucide-arrow-left" />
          </button>
          <button v-if="!isMobile" type="button" class="fb-ico" :disabled="!canGoForward" :aria-label="t('fileManager.forward')" :title="t('fileManager.forward')" @click="goForward">
            <UIcon name="i-lucide-arrow-right" />
          </button>
          <button type="button" class="fb-ico" :disabled="!canGoUp" :aria-label="t('fileManager.up')" :title="t('fileManager.up')" @click="goUp">
            <UIcon name="i-lucide-corner-left-up" />
          </button>
          <div class="fb-crumb__path" :title="t('fileManager.editPath')" @click="startEditPath">
            <input
              v-if="editingPath"
              ref="pathInputRef"
              v-model="pathDraft"
              class="fb-crumb__input m-mono"
              :aria-label="t('fileManager.filePath')"
              @keyup.enter="commitPath"
              @keyup.esc="editingPath = false"
              @blur="editingPath = false"
              @click.stop
            />
            <template v-else>
              <button type="button" class="fb-crumb__seg" :class="{ 'is-current': !pathSegments.length }" @click.stop="navigateTo('')">
                <UIcon :name="currentChip.icon" class="fb-crumb__root" />
                <span>{{ currentChip.name }}</span>
              </button>
              <template v-for="(segment, i) in pathSegments" :key="segment.path">
                <UIcon name="i-lucide-chevron-right" class="fb-crumb__sep" />
                <button
                  type="button"
                  class="fb-crumb__seg"
                  :class="{ 'is-current': i === pathSegments.length - 1 }"
                  @click.stop="navigateTo(segment.path)"
                >
                  {{ segment.name }}
                </button>
              </template>
            </template>
          </div>
          <button v-if="!isMobile" type="button" class="fb-ico" :aria-label="t('fileManager.copyPath')" :title="t('fileManager.copyPath')" @click="copyCurrentPath">
            <UIcon name="i-lucide-copy" />
          </button>
          <button type="button" class="fb-ico" :class="{ 'is-spin': loading }" :aria-label="t('fileManager.refresh')" :title="t('fileManager.refresh')" @click="refresh">
            <UIcon name="i-lucide-refresh-cw" />
          </button>
        </nav>

        <div class="fb-bar__tools">
          <label class="fb-search">
            <UIcon name="i-lucide-search" class="fb-search__ico" />
            <input
              v-model="keywords"
              type="search"
              class="fb-search__input"
              :placeholder="t('fileManager.searchPlaceholder')"
              :aria-label="t('fileManager.searchPlaceholder')"
            />
            <button
              type="button"
              class="fb-search__sub"
              :class="{ 'is-on': includeSubDir }"
              :aria-pressed="includeSubDir"
              :title="t('fileManager.subdirectories')"
              @click="toggleSubDir"
            >
              <UIcon name="i-lucide-folder-tree" />
              <span v-if="!isMobile">{{ t('fileManager.subdirectories') }}</span>
            </button>
          </label>

          <AppDropdown align="end" :width="200">
            <template #trigger>
              <button type="button" class="fb-pill" :class="{ 'fb-pill--round': isMobile }" :aria-label="sortText" :title="sortText">
                <UIcon name="i-lucide-arrow-down-wide-narrow" />
                <span v-if="!isMobile">{{ sortText }}</span>
              </button>
            </template>
            <template #default="{ close }">
              <div class="fb-menu">
                <button
                  v-for="opt in sortOptions"
                  :key="opt.key"
                  type="button"
                  class="fb-menu__item"
                  :class="{ 'is-on': opt.key === sortKey }"
                  @click="() => { applySort(opt.key); close() }"
                >
                  <UIcon :name="opt.icon" class="fb-menu__ico" />
                  <span>{{ opt.label }}</span>
                  <UIcon v-if="opt.key === sortKey" name="i-lucide-check" class="fb-menu__check" />
                </button>
              </div>
            </template>
          </AppDropdown>

          <div v-if="!isMobile" class="fb-seg" role="group" :aria-label="t('fileManager.viewList')">
            <button type="button" class="fb-seg__btn" :class="{ 'is-active': viewMode === 'list' }" :aria-label="t('fileManager.viewList')" :title="t('fileManager.viewList')" @click="viewMode = 'list'">
              <UIcon name="i-lucide-list" />
            </button>
            <button type="button" class="fb-seg__btn" :class="{ 'is-active': viewMode === 'grid' }" :aria-label="t('fileManager.viewGrid')" :title="t('fileManager.viewGrid')" @click="viewMode = 'grid'">
              <UIcon name="i-lucide-layout-grid" />
            </button>
          </div>
        </div>
      </div>

      <!-- 剪贴板条：复制/剪切后出现，可在任意目录粘贴 -->
      <div v-if="clipboard" class="fb-clip">
        <UIcon :name="clipboard.mode === 'copy' ? 'i-lucide-copy' : 'i-lucide-scissors'" class="fb-clip__ico" />
        <span class="fb-clip__text">
          {{ clipboard.mode === 'copy' ? t('fileManager.clipboardCopy', { count: clipboard.paths.length }) : t('fileManager.clipboardCut', { count: clipboard.paths.length }) }}
        </span>
        <span class="fb-clip__spacer" />
        <UButton color="primary" size="sm" icon="i-lucide-clipboard-paste" @click="paste">{{ t('fileManager.pasteHere') }}</UButton>
        <button type="button" class="fb-ico" :aria-label="t('fileManager.clearClipboard')" :title="t('fileManager.clearClipboard')" @click="clipboard = null">
          <UIcon name="i-lucide-x" />
        </button>
      </div>

      <!-- 手机：上传进度卡 -->
      <FileUploadQueue v-if="isMobile && upload.current.value" :upload="upload" compact />

      <!-- 主体：左（目录树 + 上传队列，可收起）+ 中（列表/网格）+ 右（预览） -->
      <div class="fb-body">
        <div v-if="!isMobile" class="fb-side" :class="{ 'is-closed': !treeOpen }">
          <!-- 收起：窄栏只留展开按钮 + 上传进度 -->
          <section v-if="!treeOpen" class="fb-rail" :aria-label="t('fileManager.treeCollapsed')">
            <AppIconButton
              icon="HOutline:ChevronDoubleRightIcon"
              :label="t('fileManager.expandTree')"
              :box="44"
              :size="18"
              @click="treeOpen = true"
            />
            <template v-if="upload.items.value.length">
              <span class="fb-rail__sep" />
              <button
                type="button"
                class="fb-rail__up"
                :aria-label="t('fileManager.uploadQueueExpand', { pct: upload.overallProgress.value })"
                :title="t('fileManager.uploadQueueExpand', { pct: upload.overallProgress.value })"
                @click="treeOpen = true"
              >
                <UIcon name="i-lucide-upload" />
                <span class="m-num">{{ upload.overallProgress.value }}%</span>
              </button>
            </template>
          </section>
          <template v-else>
            <aside class="fb-aside" :aria-label="t('fileManager.treeTitle')">
              <div class="fb-aside__head">
                <span class="fb-aside__label">{{ t('fileManager.treeTitle') }}</span>
                <AppIconButton
                  icon="HOutline:ChevronDoubleLeftIcon"
                  :label="t('fileManager.collapseTree')"
                  :box="36"
                  :size="16"
                  @click="treeOpen = false"
                />
              </div>
              <FileTree
                ref="treeRef"
                :client="client"
                root-path=""
                :active-path="currentPath"
                selectable="all"
                dirs-only
                @select="onTreeSelect"
              />
            </aside>
            <FileUploadQueue v-if="upload.items.value.length" :upload="upload" />
          </template>
        </div>

        <div
          class="fb-main"
          :class="{ 'is-drop': dropActive }"
          @dragenter.prevent="onDragEnter"
          @dragover.prevent
          @dragleave="onDragLeave"
          @drop.prevent="onDrop"
        >
          <!-- 加载骨架 -->
          <div v-if="loading && !items.length" class="fb-card fb-skeleton">
            <div v-for="i in 7" :key="i" class="fb-skeleton__row" />
          </div>

          <!-- 读取失败（来源连不上等） -->
          <section v-else-if="loadError && !items.length" class="fb-card fb-error">
            <span class="fb-error__sticker m-sticker m-sticker--bad"><UIcon name="i-lucide-unplug" /></span>
            <h2 class="fb-error__title">{{ t('fileManager.loadErrorTitle', { name: currentChip.name }) }}</h2>
            <p class="fb-error__msg m-mono">{{ loadError }}</p>
            <p v-if="currentSourceId" class="fb-error__hint">{{ t('fileManager.loadErrorHint') }}</p>
            <div class="fb-error__actions">
              <UButton color="primary" icon="i-lucide-refresh-cw" @click="refresh">{{ t('fileManager.retry') }}</UButton>
              <UButton v-if="canManageSources && currentSourceId" color="neutral" variant="outline" :to="pageTabTo('file', 'source')">
                {{ t('fileManager.checkSource') }}
              </UButton>
            </div>
          </section>

          <!-- 空态 -->
          <div v-else-if="!items.length" class="fb-card">
            <EmptyState
              icon="HOutline:FolderOpenIcon"
              :title="keywords ? t('fileManager.noMatch') : t('fileManager.directoryEmpty')"
              :description="keywords ? t('system.common.noData') : t('fileManager.emptyHint')"
            >
              <template v-if="!keywords" #action>
                <UButton color="primary" icon="i-lucide-upload" @click="pickUpload">{{ t('fileManager.upload') }}</UButton>
              </template>
            </EmptyState>
          </div>

          <!-- 手机：计数行 + 行列表 -->
          <div v-else-if="isMobile" class="fb-mhead m-num">
            <span>{{ mobileCount }}</span>
            <span>{{ sortText }}</span>
          </div>
          <div v-if="isMobile && items.length" class="fb-card fb-mlist">
            <div
              v-for="(row, i) in items"
              :key="row.path"
              class="fb-mrow"
              :class="{ 'is-selected': selectedPaths.has(row.path) }"
              @click="onMobileRowClick(row)"
            >
              <span
                v-if="hasSelection"
                class="fb-mrow__check"
                :class="{ 'is-on': selectedPaths.has(row.path) }"
                aria-hidden="true"
              >
                <UIcon v-if="selectedPaths.has(row.path)" name="i-lucide-check" />
              </span>
              <span class="fb-sticker m-sticker" :class="`m-sticker--${rowVisual(row).tone}`" :style="{ transform: `rotate(${ROTS[i % ROTS.length]})` }">
                <UIcon :name="rowVisual(row).icon" />
              </span>
              <span class="fb-mrow__text">
                <span class="fb-mrow__name">{{ row.name }}</span>
                <span class="fb-mrow__meta m-num">{{ sizeText(row) }} · {{ shortTime(row.updateTime) }}</span>
              </span>
              <button type="button" class="fb-ico fb-ico--lg" :aria-label="t('fileManager.moreFor', { name: row.name })" @click.stop="openSheet(row)">
                <UIcon name="i-lucide-ellipsis" />
              </button>
            </div>
          </div>

          <!-- 列表视图 -->
          <div v-else-if="items.length && !isMobile && viewMode === 'list'" class="fb-card fb-table">
            <div class="fb-thead">
              <div class="fb-col fb-col--check">
                <input
                  type="checkbox"
                  class="fb-cb"
                  :checked="allSelected"
                  :indeterminate.prop="someSelected"
                  :aria-label="t('fileManager.selectAll')"
                  @change="toggleSelectAll"
                />
              </div>
              <button type="button" class="fb-col fb-col--name fb-sort" @click="sortBy('name')">
                {{ t('fileManager.tableName') }}
                <UIcon
                  v-if="sortField === FileSortField.FILE_SORT_FIELD_NAME"
                  :name="isDesc ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'"
                  class="fb-sort__ico"
                />
              </button>
              <button type="button" class="fb-col fb-col--size fb-sort" @click="sortBy('size')">
                {{ t('fileManager.size') }}
                <UIcon
                  v-if="sortField === FileSortField.FILE_SORT_FIELD_SIZE"
                  :name="isDesc ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'"
                  class="fb-sort__ico"
                />
              </button>
              <button type="button" class="fb-col fb-col--time fb-sort" @click="sortBy('time')">
                {{ t('fileManager.updatedAt') }}
                <UIcon
                  v-if="sortField === FileSortField.FILE_SORT_FIELD_UPDATE_TIME"
                  :name="isDesc ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'"
                  class="fb-sort__ico"
                />
              </button>
              <div class="fb-col fb-col--ops">{{ t('fileManager.operation') }}</div>
            </div>

            <div
              v-for="(row, i) in items"
              :key="row.path"
              class="fb-row"
              :class="{ 'is-selected': selectedPaths.has(row.path), 'is-preview': preview?.path === row.path }"
              @click="onRowClick(row, $event)"
              @dblclick="!row.isDir && onRowOpen(row)"
            >
              <div class="fb-col fb-col--check" @click.stop>
                <input
                  type="checkbox"
                  class="fb-cb"
                  :checked="selectedPaths.has(row.path)"
                  :aria-label="t('fileManager.selectItem', { name: row.name })"
                  @change="toggleSelect(row)"
                />
              </div>
              <div class="fb-col fb-col--name">
                <span class="fb-sticker m-sticker" :class="`m-sticker--${rowVisual(row).tone}`" :style="{ transform: `rotate(${ROTS[i % ROTS.length]})` }">
                  <UIcon :name="rowVisual(row).icon" />
                </span>
                <span class="fb-row__text">
                  <span class="fb-row__name" :title="row.name">{{ row.name }}</span>
                  <span class="fb-row__kind">{{ kindText(row) }}</span>
                </span>
              </div>
              <div class="fb-col fb-col--size m-num" :class="{ 'is-dim': row.isDir }">{{ sizeText(row) }}</div>
              <div class="fb-col fb-col--time m-num" :title="formatDateTime(row.updateTime)">{{ shortTime(row.updateTime) }}</div>
              <div class="fb-col fb-col--ops" @click.stop>
                <AppIconButton
                  v-if="!row.isDir"
                  icon="HOutline:ArrowDownTrayIcon"
                  :label="t('fileManager.download')"
                  :box="36"
                  :size="18"
                  @click="downloadOne(row)"
                />
                <AppIconButton
                  icon="HOutline:PencilSquareIcon"
                  :label="t('fileManager.rename')"
                  :box="36"
                  :size="18"
                  @click="openRename(row)"
                />
                <ActionMenu
                  :items="rowActions(row).filter((a) => a.key !== 'rename' && a.key !== 'download')"
                  :label="t('fileManager.moreFor', { name: row.name })"
                  @select="(key) => onRowAction(key, row)"
                />
              </div>
            </div>
          </div>

          <!-- 网格视图 -->
          <div v-else-if="items.length && !isMobile" class="fb-grid">
            <article
              v-for="(row, i) in items"
              :key="row.path"
              class="fb-tile"
              :class="{ 'is-selected': selectedPaths.has(row.path) || preview?.path === row.path }"
              @click="onRowClick(row, $event)"
              @dblclick="!row.isDir && onRowOpen(row)"
            >
              <span class="fb-tile__cover m-sticker" :class="`m-sticker--${rowVisual(row).tone}`">
                <UIcon :name="rowVisual(row).icon" :style="{ transform: `rotate(${ROTS[i % ROTS.length]})` }" />
              </span>
              <input
                type="checkbox"
                class="fb-cb fb-tile__check"
                :checked="selectedPaths.has(row.path)"
                :aria-label="t('fileManager.selectItem', { name: row.name })"
                @click.stop
                @change="toggleSelect(row)"
              />
              <div class="fb-tile__foot">
                <span class="fb-tile__text">
                  <span class="fb-tile__name" :title="row.name">{{ row.name }}</span>
                  <span class="fb-tile__meta m-num">{{ sizeText(row) }} · {{ shortDay(row.updateTime) }}</span>
                </span>
                <span @click.stop>
                  <ActionMenu
                    :items="rowActions(row)"
                    :label="t('fileManager.moreFor', { name: row.name })"
                    @select="(key) => onRowAction(key, row)"
                  />
                </span>
              </div>
            </article>
          </div>

          <div v-if="dropActive" class="fb-drop">
            <UIcon name="i-lucide-upload" />
            <span>{{ t('fileManager.dropRelease') }}</span>
          </div>

          <!-- 底部：提示 + 分页 -->
          <div v-if="items.length || total" class="fb-foot">
            <span v-if="!isMobile" class="fb-foot__hint">
              <UIcon name="i-lucide-upload" />
              {{ t('fileManager.dropHint') }}
            </span>
            <span v-else class="fb-foot__hint m-num">{{ t('system.common.total', { total }) }}</span>
            <Pagination
              v-model:page="page"
              v-model:page-size="pageSize"
              :total="total"
              @change="loadList"
            />
          </div>
        </div>

        <FilePreviewPanel
          v-if="!isMobile && preview"
          class="fb-preview"
          :client="client"
          :row="preview"
          :icon="rowVisual(preview).icon"
          :tone="rowVisual(preview).tone"
          :kind-label="typeLabel(preview)"
          :can-share="canShare"
          @close="preview = null"
          @edit="openEditor"
          @download="downloadOne"
          @share="(row) => openShare([row])"
          @media="(row) => openMedia(row.path, row.name)"
        />
      </div>

      <!-- 批量操作条（悬浮深色） -->
      <div v-if="hasSelection" class="fb-bulk" role="toolbar" :aria-label="t('fileManager.selectedCount', { count: selectedPaths.size })">
        <span class="fb-bulk__count m-num">
          {{ t('fileManager.selectedCount', { count: selectedPaths.size }) }}<template v-if="selectedSize && !isMobile"> · {{ formatFileSize(selectedSize) }}</template>
        </span>
        <button
          v-for="act in bulkActions"
          :key="act.key"
          type="button"
          class="fb-bulk__btn"
          :class="{ 'is-danger': act.danger }"
          :aria-label="act.label"
          :title="act.label"
          @click="onBulkAction(act.key)"
        >
          <UIcon :name="act.icon" />
          <span v-if="!isMobile">{{ act.label }}</span>
        </button>
        <button type="button" class="fb-bulk__btn fb-bulk__btn--x" :aria-label="t('fileManager.cancelSelect')" :title="t('fileManager.cancelSelect')" @click="clearSelection">
          <UIcon name="i-lucide-x" />
        </button>
      </div>
    </div>

    <!-- 上传选择器（上传按钮 / 悬浮按钮共用；选好即加入队列开始上传） -->
    <input ref="uploadInputRef" type="file" multiple class="fb-upload-input" @change="onUploadPick" />

    <!-- 手机：悬浮上传 -->
    <button v-if="isMobile && !hasSelection" type="button" class="fb-fab" @click="pickUpload">
      <UIcon name="i-lucide-upload" />
      {{ t('fileManager.upload') }}
    </button>

    <!-- 手机：行操作底部面板 -->
    <FileActionSheet
      v-if="isMobile"
      v-model="sheet.open"
      :title="sheet.row?.name || ''"
      :meta="sheetMeta"
      :path="sheet.row?.path"
      :icon="sheet.row ? rowVisual(sheet.row).icon : undefined"
      :tone="sheet.row ? rowVisual(sheet.row).tone : undefined"
      :quick="sheetQuick"
      :items="sheetItems"
      @select="onSheetSelect"
    />

    <!-- 子弹窗 -->
    <FilePromptDialog
      v-model="prompt.open"
      :title="prompt.title"
      :label="prompt.label"
      :description="prompt.description"
      :placeholder="prompt.placeholder"
      :initial-value="prompt.initialValue"
      :confirming="prompt.confirming"
      @confirm="onPromptConfirm"
    />

    <FileMediaDialog v-model="media.open" :name="media.name" :kind="media.kind" :url="media.url" />

    <FileMoveDialog
      v-model="transfer.open"
      :client="client"
      :mode="transfer.mode"
      :count="transfer.rows.length"
      :initial-path="currentPath"
      :root-label="currentChip.name"
      :confirming="transfer.pending"
      @confirm="confirmTransfer"
    />

    <FileEditor
      v-model="editor.open"
      :client="client"
      :path="editor.path"
      root-path=""
      @saved="loadList"
      @renamed="afterMutation"
      @deleted="afterMutation"
    />

    <ShareFormDialog v-model="shareOpen" :items="shareItems" />

    <!-- 删除确认 -->
    <FormDialog v-model="del.open" :title="t('fileManager.confirmDelete')" :width="420" :loading="del.pending" @confirm="confirmDelete">
      <p class="fb-confirm">{{ del.message }}</p>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.cancel') }}</UButton>
        <UButton color="error" :loading="del.pending" @click="confirmDelete">{{ t('system.common.confirm') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import { isRemoteNode } from '@/utils/nodeContext'
import { pageTabTo } from '@/config/pageGroups'
import {
  FileSortField,
  type FileEntryInfo,
  type FileDirectoryInfo,
  type FileSourceInfo,
  type FileSourceCaps,
} from '@/types/v1/file'
import { listFileSourcesRequest } from '@/api/fileSource'
import {
  formatDateTime,
  formatFileSize,
  getFileExtension,
  getParentPath,
  isMediaFile,
  joinPath,
  resolveFilePreviewKind,
  resolvePreSignedFileUrl,
  splitPathSegments,
  downloadFileFromUrl,
  copyTextToClipboard,
  isFileTooLargeForEditor,
  type FilePreviewKind,
} from '@/utils/file'
import { createFileClient, displayDirPath } from './fileClient'
import type { FileClient, FileClipboard, FileScope, PickedFile } from './types'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import FileTree from './FileTree.vue'
import FileActionSheet from './FileActionSheet.vue'
import FilePromptDialog from './FilePromptDialog.vue'
import FileMediaDialog from './FileMediaDialog.vue'
import FileUploadQueue from './FileUploadQueue.vue'
import FilePreviewPanel from './FilePreviewPanel.vue'
import FileMoveDialog from './FileMoveDialog.vue'
import FileEditor from './FileEditor.vue'
import { useFileUpload } from './useFileUpload'
import ShareFormDialog from '@/components/share/ShareFormDialog.vue'

const props = defineProps<{
  scope: FileScope
  // 自定义文件客户端（同构接口的其他后端，如托管游戏服务器的服务端目录）；缺省按 scope 创建
  client?: FileClient
  initialPath?: string
  // 系统级：初始打开的文件来源 id（如从文件源页「浏览」进入）；空=本地磁盘
  initialSourceId?: string
  // 嵌在已有内边距、自己会滚动的页面里（如游戏服务器的「文件」标签）：不再加整页内边距、背景与内部滚动
  inset?: boolean
}>()

const { t } = useI18n()
const menuStore = useMenuStore()
const fb = useFeedback()
const accessible = useAccessibleRoutes()
const isMobile = computed(() => menuStore.isMobile)
// 贴纸图标的轻微旋转角度（按序号轮换）。
const ROTS = ['-4deg', '3deg', '-2deg', '4deg']

// 请求错误反馈：走统一 toast（getRequestErrorMessage 解析后端消息）。
const notifyError = (error: unknown, fallback: string) => fb.error(getRequestErrorMessage(error, fallback))

// ---- 视图模式（列表/网格，持久化）----
const viewMode = ref<'list' | 'grid'>(
  (localStorage.getItem('fm-view') as 'list' | 'grid') === 'grid' ? 'grid' : 'list',
)
watch(viewMode, (v) => localStorage.setItem('fm-view', v))

// ---- 目录树收起（持久化）：收起后左栏只剩窄栏（展开按钮 + 上传进度），把空间让给列表 ----
const readTreeOpen = () => {
  try {
    const v = localStorage.getItem('fm-tree')
    if (v === '0') return false
    if (v === '1') return true
  } catch {
    /* ignore */
  }
  return typeof window === 'undefined' || window.innerWidth >= 1100
}
const treeOpen = ref(readTreeOpen())
watch(treeOpen, (v) => {
  try {
    localStorage.setItem('fm-tree', v ? '1' : '0')
  } catch {
    /* ignore */
  }
})

// ---- 文件来源（仅系统级显示切换）----
const isSystemScope = props.scope.kind === 'system'
const currentSourceId = ref(props.scope.kind === 'system' ? props.initialSourceId || '' : '')
const sources = ref<FileSourceInfo[]>([])
const localCaps: FileSourceCaps = {
  presign: false,
  copy: true,
  move: true,
  compress: true,
  resumableUpload: true,
}
const currentCaps = computed<FileSourceCaps>(() => {
  if (!currentSourceId.value) return localCaps
  return sources.value.find((s) => s.id === currentSourceId.value)?.caps ?? localCaps
})
// 来源胶囊：本地磁盘 + 已启用的远端来源（贴纸图标 + 类型短名）。
type SourceChip = { id: string; name: string; type: string; icon: string; tone: string }
const SOURCE_VISUAL: Record<string, { icon: string; tone: string }> = {
  local: { icon: 'i-lucide-hard-drive', tone: 'a' },
  oss: { icon: 'i-lucide-cloud', tone: 'p' },
  ftp: { icon: 'i-lucide-arrow-left-right', tone: 'ok' },
  webdav: { icon: 'i-lucide-globe', tone: 'a2' },
}
// 实例级：根目录就是应用的工作目录，面包屑根显示「工作目录」而不是「本地磁盘」
const localChip = computed<SourceChip>(() => ({
  id: '',
  name: isSystemScope ? t('fileSource.localDisk') : t('instance.ui.workdir'),
  type: t('fileManager.localTag'),
  icon: isSystemScope ? 'i-lucide-hard-drive' : 'i-lucide-folder',
  tone: 'a',
}))
const sourceChips = computed<SourceChip[]>(() => [
  localChip.value,
  ...sources.value.map((s) => ({
    id: s.id,
    name: s.name,
    type: t(`fileSource.typeShort.${s.type}`),
    icon: SOURCE_VISUAL[s.type]?.icon ?? 'i-lucide-database',
    tone: SOURCE_VISUAL[s.type]?.tone ?? 'a2',
  })),
])
const currentChip = computed<SourceChip>(
  () => sourceChips.value.find((c) => c.id === currentSourceId.value) ?? localChip.value,
)
// 管理文件源入口：仅系统级、且当前用户菜单里有文件源页时显示。
const canManageSources = computed(() => isSystemScope && accessible.value.has('/file/source'))

// scope 在组件生命周期内稳定；source_id 动态读取，切换来源无需重建 client。
const client: FileClient = props.client ?? createFileClient(props.scope, () => currentSourceId.value)

// 左侧目录树引用：结构性变更（新建/删除/重命名/粘贴/切换来源）后重载。
const treeRef = ref<InstanceType<typeof FileTree> | null>(null)
const reloadTree = () => treeRef.value?.reload()
// 变更后统一刷新列表 + 目录树
const afterMutation = () => {
  loadList()
  reloadTree()
}

const loadSources = async () => {
  if (!isSystemScope) return
  try {
    const { data } = await listFileSourcesRequest({ enabledOnly: true })
    sources.value = data.items ?? []
  } catch {
    sources.value = []
  }
}

const onSourceChange = (id: string) => {
  // 切换来源：清空历史/选择，回到该来源根目录。
  currentSourceId.value = id
  history.value = []
  historyIndex.value = -1
  selectedPaths.value = new Set()
  clipboard.value = null
  keywords.value = ''
  page.value = 1
  currentPath.value = ''
  items.value = []
  directory.value = undefined
  preview.value = null
  afterMutation()
}

// ---- 列表状态 ----
const currentPath = ref(props.initialPath || '')
const directory = ref<FileDirectoryInfo>()
const items = ref<FileEntryInfo[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keywords = ref('')
const includeSubDir = ref(false)
const sortField = ref<FileSortField>(FileSortField.FILE_SORT_FIELD_NAME)
const isDesc = ref(false)
const selectedPaths = ref<Set<string>>(new Set())

// ---- 导航历史 ----
const history = ref<string[]>([])
const historyIndex = ref(-1)
const canGoBack = computed(() => historyIndex.value > 0)
const canGoForward = computed(() => historyIndex.value < history.value.length - 1)
const canGoUp = computed(
  () => !!currentPath.value && getParentPath(currentPath.value) !== currentPath.value,
)

// ---- 面包屑 ----
// POSIX 根「/」与实例根「.」由面包屑首个「来源」按钮代表，不再重复显示。
const pathSegments = computed(() =>
  splitPathSegments(currentPath.value).filter((seg) => seg.name !== '/' && seg.name !== '.'),
)
const editingPath = ref(false)
const pathDraft = ref('')
const pathInputRef = ref<HTMLInputElement | null>(null)

// ---- 剪贴板 ----
const clipboard = ref<FileClipboard | null>(null)

// ---- 弹窗状态 ----
type PromptMode = 'createFolder' | 'createFile' | 'rename' | 'compress' | 'unzip'
const prompt = reactive({
  open: false,
  mode: 'createFolder' as PromptMode,
  title: '',
  label: '' as string | undefined,
  description: '' as string | undefined,
  placeholder: '' as string | undefined,
  initialValue: '' as string | undefined,
  confirming: false,
  targets: [] as FileEntryInfo[],
})

const media = reactive({
  open: false,
  name: '',
  kind: 'image' as FilePreviewKind,
  url: '',
})

const editor = reactive({ open: false, path: '' })
const shareOpen = ref(false)
const shareItems = ref<PickedFile[]>([])

// ---- 选择 ----
const selectedRows = computed(() =>
  items.value.filter((item) => selectedPaths.value.has(item.path)),
)
const hasSelection = computed(() => selectedPaths.value.size > 0)
const allSelected = computed(
  () => items.value.length > 0 && items.value.every((item) => selectedPaths.value.has(item.path)),
)
const someSelected = computed(() => hasSelection.value && !allSelected.value)

const toggleSelect = (row: FileEntryInfo) => {
  const next = new Set(selectedPaths.value)
  if (next.has(row.path)) next.delete(row.path)
  else next.add(row.path)
  selectedPaths.value = next
}
const toggleSelectAll = () => {
  selectedPaths.value = allSelected.value
    ? new Set()
    : new Set(items.value.map((item) => item.path))
}
const clearSelection = () => {
  selectedPaths.value = new Set()
}

// ---- 汇总 ----
const folderCount = computed(() => Number(directory.value?.dirCount ?? 0))
const fileCount = computed(() => Number(directory.value?.fileCount ?? 0))
const dirSize = computed(() => Number(directory.value?.totalSize ?? 0))
const displayPath = computed(() => displayDirPath(currentPath.value))
// 磁盘用量（仅本地磁盘/实例目录有值；远端来源为 0 不显示）
const diskTotal = computed(() => Number(directory.value?.diskTotal ?? 0))
const diskFree = computed(() => Number(directory.value?.diskFree ?? 0))
// 来源容量条：本地类来源取目录所在磁盘；远端来源取最近一次测试测得的容量（WebDAV 配额等）；连不上时标红。
const usage = computed(() => {
  if (loadError.value && currentSourceId.value) {
    return { text: t('fileManager.connectFailed'), pct: 0, warn: true, title: loadError.value }
  }
  let total = diskTotal.value
  let used = total ? total - diskFree.value : 0
  if (!total && currentSourceId.value) {
    const last = sources.value.find((s) => s.id === currentSourceId.value)?.lastTest
    total = Number(last?.totalBytes) || 0
    used = Number(last?.usedBytes) || 0
  }
  if (!total && !used) return null
  const pct = total ? Math.min(100, Math.round((used / total) * 100)) : 0
  return {
    text: total
      ? t('fileManager.diskUsed', { used: formatFileSize(used), total: formatFileSize(total) })
      : t('fileManager.usedOnly', { used: formatFileSize(used) }),
    pct,
    warn: pct >= 90,
    title: total ? t('fileManager.diskFree', { free: formatFileSize(total - used) }) : '',
  }
})
const selectedSize = computed(() =>
  selectedRows.value.reduce((sum, row) => sum + (row.isDir ? 0 : Number(row.size) || 0), 0),
)
const loadError = ref('')

// ---- 加载 ----
// 返回是否成功：导航失败时调用方必须回滚 currentPath / 历史，避免「报错了却已经切进去」。
const loadList = async (): Promise<boolean> => {
  loading.value = true
  try {
    const res = await client.list({
      path: currentPath.value,
      page: page.value,
      pageSize: pageSize.value,
      keywords: keywords.value.trim() || undefined,
      includeSubDir: includeSubDir.value || undefined,
      sortField: sortField.value,
      isDesc: isDesc.value,
    })
    directory.value = res.directory
    items.value = res.items
    total.value = Number(res.total) || 0
    selectedPaths.value = new Set()
    if (res.directory?.path) currentPath.value = res.directory.path
    // 首次加载后用规范化路径播种历史
    if (historyIndex.value === -1) {
      history.value = [currentPath.value]
      historyIndex.value = 0
    }
    loadError.value = ''
    return true
  } catch (error) {
    loadError.value = getRequestErrorMessage(error, t('fileManager.refreshFailed'))
    notifyError(error, t('fileManager.refreshFailed'))
    return false
  } finally {
    loading.value = false
  }
}

const refresh = () => afterMutation()

// ---- 导航（失败回滚路径，禁止“报错却切进去”）----
const navigateTo = async (path: string, pushHistory = true) => {
  const prevPath = currentPath.value
  const prevPage = page.value
  const prevKeywords = keywords.value
  const prevHistory = history.value.slice()
  const prevHistoryIndex = historyIndex.value

  if (pushHistory) {
    history.value = history.value.slice(0, historyIndex.value + 1)
    history.value.push(path)
    historyIndex.value = history.value.length - 1
  }
  currentPath.value = path
  page.value = 1
  keywords.value = ''
  preview.value = null
  const ok = await loadList()
  if (!ok) {
    currentPath.value = prevPath
    page.value = prevPage
    keywords.value = prevKeywords
    history.value = prevHistory
    historyIndex.value = prevHistoryIndex
  }
}
const goBack = async () => {
  if (!canGoBack.value) return
  const prevIndex = historyIndex.value
  const prevPath = currentPath.value
  const prevPage = page.value
  historyIndex.value -= 1
  currentPath.value = history.value[historyIndex.value] ?? currentPath.value
  page.value = 1
  const ok = await loadList()
  if (!ok) {
    historyIndex.value = prevIndex
    currentPath.value = prevPath
    page.value = prevPage
  }
}
const goForward = async () => {
  if (!canGoForward.value) return
  const prevIndex = historyIndex.value
  const prevPath = currentPath.value
  const prevPage = page.value
  historyIndex.value += 1
  currentPath.value = history.value[historyIndex.value] ?? currentPath.value
  page.value = 1
  const ok = await loadList()
  if (!ok) {
    historyIndex.value = prevIndex
    currentPath.value = prevPath
    page.value = prevPage
  }
}
const goUp = () => {
  if (!canGoUp.value) return
  void navigateTo(getParentPath(currentPath.value))
}

const startEditPath = () => {
  if (editingPath.value) return
  pathDraft.value = currentPath.value
  editingPath.value = true
  nextTick(() => pathInputRef.value?.focus())
}
const commitPath = () => {
  editingPath.value = false
  const target = pathDraft.value.trim()
  if (target && target !== currentPath.value) navigateTo(target)
}

// ---- 目录树点击：目录→进入，文件→打开 ----
const onTreeSelect = (path: string, name: string, isDir: boolean) => {
  if (isDir) {
    if (path !== currentPath.value) navigateTo(path)
    return
  }
  openFileByName(path, name, 0)
}

// ---- 搜索/筛选 ----
const debouncedReload = useDebounceFn(() => {
  page.value = 1
  loadList()
}, 350)
watch(keywords, () => debouncedReload())

const toggleSubDir = () => {
  includeSubDir.value = !includeSubDir.value
  page.value = 1
  loadList()
}

const copyCurrentPath = async () => {
  try {
    await copyTextToClipboard(displayPath.value)
    fb.success(t('fileManager.pathCopied'))
  } catch {
    fb.error(t('fileManager.copyLinkFailed'))
  }
}

// ---- 排序 ----
type SortName = 'name' | 'time' | 'size'
const SORT_FIELD: Record<SortName, FileSortField> = {
  name: FileSortField.FILE_SORT_FIELD_NAME,
  time: FileSortField.FILE_SORT_FIELD_UPDATE_TIME,
  size: FileSortField.FILE_SORT_FIELD_SIZE,
}
const sortBy = (field: SortName) => {
  const target = SORT_FIELD[field]
  if (sortField.value === target) {
    isDesc.value = !isDesc.value
  } else {
    sortField.value = target
    // 时间、大小默认从新到旧 / 从大到小
    isDesc.value = field !== 'name'
  }
  page.value = 1
  loadList()
}

// 排序下拉（名称 / 修改时间 / 大小 × 升降序；目录恒在前）
type SortKey = `${SortName}-${'asc' | 'desc'}`
const sortKey = computed<SortKey>(() => {
  const name = (Object.keys(SORT_FIELD) as SortName[]).find((k) => SORT_FIELD[k] === sortField.value) ?? 'name'
  return `${name}-${isDesc.value ? 'desc' : 'asc'}`
})
const sortFieldLabel = (name: SortName) =>
  name === 'time' ? t('fileManager.updatedAt') : name === 'size' ? t('fileManager.size') : t('fileManager.tableName')
const sortOptions = computed(() => [
  { key: 'name-asc' as SortKey, icon: 'i-lucide-arrow-down-a-z', label: `${t('fileManager.tableName')} · ${t('fileManager.sortAsc')}` },
  { key: 'name-desc' as SortKey, icon: 'i-lucide-arrow-down-z-a', label: `${t('fileManager.tableName')} · ${t('fileManager.sortDesc')}` },
  { key: 'time-desc' as SortKey, icon: 'i-lucide-clock-arrow-down', label: `${t('fileManager.updatedAt')} · ${t('fileManager.sortNewest')}` },
  { key: 'time-asc' as SortKey, icon: 'i-lucide-clock-arrow-up', label: `${t('fileManager.updatedAt')} · ${t('fileManager.sortOldest')}` },
  { key: 'size-desc' as SortKey, icon: 'i-lucide-arrow-down-wide-narrow', label: `${t('fileManager.size')} · ${t('fileManager.sortLargest')}` },
  { key: 'size-asc' as SortKey, icon: 'i-lucide-arrow-up-narrow-wide', label: `${t('fileManager.size')} · ${t('fileManager.sortSmallest')}` },
])
const sortText = computed(() => t('fileManager.sortLabel', { field: sortFieldLabel(sortKey.value.split('-')[0] as SortName) }))
const applySort = (key: SortKey) => {
  const [name, dir] = key.split('-') as [SortName, 'asc' | 'desc']
  sortField.value = SORT_FIELD[name]
  isDesc.value = dir === 'desc'
  page.value = 1
  loadList()
}

// ---- 行展示（Lucide iconify name + 贴纸色调）----
const ARCHIVE_RE = /\.(zip|tar|gz|tgz|rar|7z|bz2|xz)$/i
const CODE_RE = /\.(js|mjs|cjs|ts|tsx|jsx|vue|go|py|rs|java|kt|c|cc|cpp|h|cs|php|rb|sh|bash|ps1|bat|json|ya?ml|toml|xml|html?|css|scss|sql|lua)$/i
const TEXT_RE = /\.(txt|md|ini|conf|cfg|env|properties|csv)$|^\.env/i
const rowVisual = (row: FileEntryInfo): { icon: string; tone: string } => {
  if (row.isDir) return { icon: 'i-lucide-folder', tone: 'a' }
  const kind = resolveFilePreviewKind(row.name)
  if (kind === 'image') return { icon: 'i-lucide-image', tone: 'ok' }
  if (kind === 'video') return { icon: 'i-lucide-film', tone: 'p' }
  if (kind === 'audio') return { icon: 'i-lucide-music', tone: 'a2' }
  if (ARCHIVE_RE.test(row.name)) return { icon: 'i-lucide-file-archive', tone: 'warn' }
  if (/\.log$/i.test(row.name)) return { icon: 'i-lucide-scroll-text', tone: 'neutral' }
  if (CODE_RE.test(row.name)) return { icon: 'i-lucide-file-code', tone: 'a2' }
  if (TEXT_RE.test(row.name)) return { icon: 'i-lucide-file-text', tone: 'a2' }
  return { icon: 'i-lucide-file', tone: 'neutral' }
}
// 目录的「大小」列展示直接子项数（来源可统计时），文件展示大小。
const childCount = (row: FileEntryInfo) => (row.childCount === undefined ? null : Number(row.childCount))
const sizeText = (row: FileEntryInfo) => {
  if (!row.isDir) return formatFileSize(row.size)
  const n = childCount(row)
  return n === null ? t('fileManager.typeFolder') : t('fileManager.itemsCount', { n })
}
const kindText = (row: FileEntryInfo) => {
  const n = row.isDir ? childCount(row) : null
  return n === null ? typeLabel(row) : `${typeLabel(row)} · ${t('fileManager.itemsCount', { n })}`
}
const mobileCount = computed(() => {
  const parts = [t('fileManager.itemsCount', { n: total.value })]
  if (dirSize.value) parts.push(formatFileSize(dirSize.value))
  return parts.join(' · ')
})
const pad2 = (n: number) => String(n).padStart(2, '0')
const toDate = (value: unknown): Date | null => {
  if (!value) return null
  const d = new Date(value as string)
  return Number.isNaN(d.getTime()) ? null : d
}
// 手机/网格用短时间：今年省略年份。
const shortTime = (value: unknown) => {
  const d = toDate(value)
  if (!d) return '—'
  const md = `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`
  return d.getFullYear() === new Date().getFullYear() ? md : `${d.getFullYear()}-${md}`
}
const shortDay = (value: unknown) => {
  const d = toDate(value)
  if (!d) return '—'
  const md = `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
  return d.getFullYear() === new Date().getFullYear() ? md : `${d.getFullYear()}-${md}`
}
const typeLabel = (row: FileEntryInfo) => {
  if (row.isDir) return t('fileManager.typeFolder')
  const ext = getFileExtension(row.name)
  const extUpper = ext ? ext.toUpperCase() : ''
  const kind = resolveFilePreviewKind(row.name)
  if (kind === 'image') return t('fileManager.typeImage', { ext: extUpper })
  if (kind === 'video') return t('fileManager.typeVideo', { ext: extUpper })
  if (kind === 'audio') return t('fileManager.typeAudio', { ext: extUpper })
  if (ext) return t('fileManager.typeNamed', { ext: extUpper })
  return t('fileManager.typeFile')
}

// ---- 行交互 ----
const onRowClick = (row: FileEntryInfo, event: MouseEvent) => {
  // Ctrl/⌘ 单击 = 多选；普通单击：目录进入、文件打开（单击即打开）
  if (event.ctrlKey || event.metaKey) {
    toggleSelect(row)
    return
  }
  // PC：目录进入；文件在右侧预览（双击 / 预览栏「编辑」打开编辑器）。
  if (!row.isDir && !isMobile.value) {
    preview.value = preview.value?.path === row.path ? null : row
    return
  }
  onRowOpen(row)
}
// ---- 预览侧栏（切换目录 / 来源时关闭）----
const preview = ref<FileEntryInfo | null>(null)
watch(currentPath, () => {
  preview.value = null
})
const openEditor = (row: FileEntryInfo) => {
  if (isFileTooLargeForEditor(row.path, row.size)) {
    fb.warning(t('fileManager.fileTooLargeForEditor'))
    return
  }
  editor.path = row.path
  editor.open = true
}
const onRowOpen = (row: FileEntryInfo) => {
  if (row.isDir) {
    navigateTo(row.path)
    return
  }
  if (isMediaFile(row.name)) {
    openMedia(row.path, row.name)
    return
  }
  if (isFileTooLargeForEditor(row.path, row.size)) {
    fb.warning(t('fileManager.fileTooLargeForEditor'))
    return
  }
  editor.path = row.path
  editor.open = true
}
// 目录树点击文件：无 size 信息，媒体走预览，其它进编辑器（大小校验在编辑器内兜底）。
const openFileByName = (path: string, name: string, size: number) => {
  if (isMediaFile(name)) {
    openMedia(path, name)
    return
  }
  if (size && isFileTooLargeForEditor(path, size)) {
    fb.warning(t('fileManager.fileTooLargeForEditor'))
    return
  }
  editor.path = path
  editor.open = true
}

// ---- 媒体预览 ----
const openMedia = async (path: string, name: string) => {
  try {
    const downloadPath = await client.preSignDownload(path, true)
    media.url = resolvePreSignedFileUrl(downloadPath)
    media.name = name
    media.kind = (resolveFilePreviewKind(name) || 'image') as FilePreviewKind
    media.open = true
  } catch (error) {
    notifyError(error, t('fileManager.openFailed'))
  }
}

// ---- 创建 / 重命名 / 压缩 / 解压（共用 prompt） ----
const openCreateFolder = () => {
  Object.assign(prompt, {
    mode: 'createFolder',
    title: t('fileManager.createFolder'),
    label: '',
    description: t('fileManager.currentDirectory', { path: currentPath.value }),
    placeholder: t('fileManager.createFolderPlaceholder'),
    initialValue: '',
    targets: [],
    open: true,
  })
}
const openCreateFile = () => {
  Object.assign(prompt, {
    mode: 'createFile',
    title: t('fileManager.createFile'),
    label: '',
    description: t('fileManager.currentDirectory', { path: currentPath.value }),
    placeholder: t('fileManager.createFilePlaceholder'),
    initialValue: '',
    targets: [],
    open: true,
  })
}
const openRename = (row: FileEntryInfo) => {
  Object.assign(prompt, {
    mode: 'rename',
    title: t('fileManager.renameTitle'),
    label: '',
    description: t('fileManager.renameTarget', { name: row.name }),
    placeholder: t('fileManager.renamePlaceholder'),
    initialValue: row.name,
    targets: [row],
    open: true,
  })
}
const openCompress = (rows: FileEntryInfo[]) => {
  const first = rows[0]
  if (!first) {
    fb.warning(t('fileManager.selectCompressTarget'))
    return
  }
  const description =
    rows.length === 1
      ? t('fileManager.compressOne', { name: first.name })
      : t('fileManager.compressMany', { count: rows.length })
  Object.assign(prompt, {
    mode: 'compress',
    title: t('fileManager.compressTitle'),
    label: '',
    description,
    placeholder: t('fileManager.compressPlaceholder'),
    initialValue: `${first.name}.zip`,
    targets: rows,
    open: true,
  })
}
const openUnzip = (row: FileEntryInfo) => {
  Object.assign(prompt, {
    mode: 'unzip',
    title: t('fileManager.unzipTitle'),
    label: '',
    description: t('fileManager.unzipTo', { name: row.name }),
    placeholder: t('fileManager.unzipPlaceholder'),
    initialValue: row.name.replace(/\.[^.]+$/, ''),
    targets: [row],
    open: true,
  })
}

const onPromptConfirm = async (value: string) => {
  prompt.confirming = true
  try {
    if (prompt.mode === 'createFolder') {
      await client.create({ path: joinPath(currentPath.value, value), isDir: true })
      fb.success(t('fileManager.folderCreateSuccess'))
    } else if (prompt.mode === 'createFile') {
      const newPath = joinPath(currentPath.value, value)
      await client.create({ path: newPath, isDir: false, content: '' })
      fb.success(t('fileManager.fileCreateSuccess'))
      // 新建文件后直接进入编辑器（创建即编辑，符合直觉）
      editor.path = newPath
      editor.open = true
    } else if (prompt.mode === 'rename') {
      const target = prompt.targets[0]
      if (target) await client.rename(target.path, value)
      if (target && preview.value?.path === target.path) preview.value = null
      fb.success(t('fileManager.renameSuccess'))
    } else if (prompt.mode === 'compress') {
      const name = /\.(zip|tar|gz|tgz)$/i.test(value) ? value : `${value}.zip`
      const output = await client.compress(
        prompt.targets.map((item) => item.path),
        joinPath(currentPath.value, name),
      )
      fb.success(
        t('fileManager.compressDone', { path: t('fileManager.outputPath', { path: output }) }),
      )
    } else if (prompt.mode === 'unzip') {
      const target = prompt.targets[0]
      if (!target) return
      const output = await client.unzip(target.path, joinPath(currentPath.value, value))
      fb.success(
        t('fileManager.unzipDone', { path: t('fileManager.outputPath', { path: output }) }),
      )
    }
    prompt.open = false
    afterMutation()
  } catch (error) {
    notifyError(error, t('fileManager.refreshFailed'))
  } finally {
    prompt.confirming = false
  }
}

// ---- 删除 ----
// 删除确认走令牌 FormDialog（内联状态），不用 EP 命令式弹窗。
const del = reactive({ open: false, message: '', pending: false, rows: [] as FileEntryInfo[] })
const deleteRows = (rows: FileEntryInfo[]) => {
  if (!rows.length) {
    fb.warning(t('fileManager.selectDeleteTarget'))
    return
  }
  del.rows = rows
  del.message =
    rows.length === 1
      ? t('fileManager.deleteOneConfirm', { name: rows[0]?.name ?? '' })
      : t('fileManager.deleteManyConfirm', { count: rows.length })
  del.open = true
}
const confirmDelete = async () => {
  del.pending = true
  try {
    await client.remove(del.rows.map((item) => item.path))
    fb.success(t('fileManager.deleteSuccess'))
    if (preview.value && del.rows.some((r) => r.path === preview.value?.path)) preview.value = null
    del.open = false
    afterMutation()
  } catch (error) {
    notifyError(error, t('fileManager.deleteFailed'))
  } finally {
    del.pending = false
  }
}
const deleteSelected = () => deleteRows(selectedRows.value)

// ---- 下载 / 复制链接 ----
const downloadOne = async (row: FileEntryInfo) => {
  if (row.isDir) return
  try {
    const downloadPath = await client.preSignDownload(row.path)
    downloadFileFromUrl(resolvePreSignedFileUrl(downloadPath), row.name)
    fb.success(t('fileManager.downloadStarted', { name: row.name }))
  } catch (error) {
    notifyError(error, t('fileManager.downloadFailed'))
  }
}
const downloadSelected = async () => {
  const files = selectedRows.value.filter((row) => !row.isDir)
  if (!files.length) {
    fb.warning(t('fileManager.selectDownloadFile'))
    return
  }
  for (const file of files) {
    await downloadOne(file)
  }
}
const copyLink = async (row: FileEntryInfo) => {
  if (row.isDir) {
    fb.warning(t('fileManager.selectCopyLinkFile'))
    return
  }
  try {
    const downloadPath = await client.preSignDownload(row.path)
    await copyTextToClipboard(resolvePreSignedFileUrl(downloadPath))
    fb.success(t('fileManager.copiedTemporaryLink', { name: row.name }))
  } catch (error) {
    notifyError(error, t('fileManager.copyLinkFailed'))
  }
}

// ---- 复制 / 剪切 / 粘贴 ----
const setClipboard = (mode: 'copy' | 'cut', rows: FileEntryInfo[]) => {
  if (!rows.length) {
    fb.warning(
      mode === 'copy' ? t('fileManager.selectCopyTarget') : t('fileManager.selectCutTarget'),
    )
    return
  }
  clipboard.value = { mode, paths: rows.map((item) => item.path) }
  fb.success(
    mode === 'copy'
      ? t('fileManager.copiedItems', { count: rows.length })
      : t('fileManager.cutItems', { count: rows.length }),
  )
}
const paste = async () => {
  if (!clipboard.value || !clipboard.value.paths.length) {
    fb.warning(t('fileManager.copyOrCutFirst'))
    return
  }
  const { mode, paths } = clipboard.value
  const pending = fb.add({ title: t('fileManager.pasteProcessing'), color: 'info', duration: 0 })
  try {
    if (mode === 'copy') await client.copy(paths, currentPath.value)
    else await client.move(paths, currentPath.value)
    fb.success(t('fileManager.pasteDone'))
    if (mode === 'cut') clipboard.value = null
    afterMutation()
  } catch (error) {
    notifyError(error, t('fileManager.pasteFailed'))
  } finally {
    fb.remove(pending.id)
  }
}

// ---- 移动到… / 复制到…（选目标目录）----
const transfer = reactive({ open: false, mode: 'move' as 'move' | 'copy', rows: [] as FileEntryInfo[], pending: false })
const openTransfer = (mode: 'move' | 'copy', rows: FileEntryInfo[]) => {
  if (!rows.length) {
    fb.warning(mode === 'copy' ? t('fileManager.selectCopyTarget') : t('fileManager.selectCutTarget'))
    return
  }
  Object.assign(transfer, { mode, rows, open: true })
}
const confirmTransfer = async (target: string) => {
  transfer.pending = true
  const pending = fb.add({ title: t('fileManager.pasteProcessing'), color: 'info', duration: 0 })
  try {
    const paths = transfer.rows.map((row) => row.path)
    if (transfer.mode === 'copy') await client.copy(paths, target)
    else await client.move(paths, target)
    fb.success(transfer.mode === 'copy' ? t('fileManager.copyDone', { count: paths.length }) : t('fileManager.moveDone', { count: paths.length }))
    transfer.open = false
    if (transfer.mode === 'move' && preview.value && paths.includes(preview.value.path)) preview.value = null
    afterMutation()
  } catch (error) {
    notifyError(error, t('fileManager.pasteFailed'))
  } finally {
    fb.remove(pending.id)
    transfer.pending = false
  }
}

// ---- 分享 ----
const openShare = (rows: FileEntryInfo[]) => {
  // 选中项按当前来源打包为分享条目（分享支持跨来源，此处均来自当前浏览来源）。
  shareItems.value = rows.map((row) => ({ sourceId: currentSourceId.value, path: row.path }))
  shareOpen.value = true
}

// ---- 分享是否可用：仅系统级、且非远程节点上下文（分享链接是主控域名下的公开地址）----
const canShare = computed(() => isSystemScope && !isRemoteNode())

// ---- 批量操作条 ----
type BulkItem = { key: string; label: string; icon: string; danger?: boolean }
const bulkActions = computed<BulkItem[]>(() => {
  const caps = currentCaps.value
  const list: BulkItem[] = [{ key: 'download', label: t('fileManager.download'), icon: 'i-lucide-download' }]
  if (caps.copy) list.push({ key: 'copyTo', label: t('fileManager.copy'), icon: 'i-lucide-copy' })
  if (caps.move) list.push({ key: 'moveTo', label: t('fileManager.move'), icon: 'i-lucide-folder-input' })
  if (caps.compress) list.push({ key: 'compress', label: t('fileManager.compress'), icon: 'i-lucide-archive' })
  if (canShare.value) list.push({ key: 'share', label: t('fileManager.share'), icon: 'i-lucide-share-2' })
  list.push({ key: 'delete', label: t('fileManager.delete'), icon: 'i-lucide-trash-2', danger: true })
  return list
})
const onBulkAction = (key: string) => {
  const rows = selectedRows.value
  if (key === 'download') downloadSelected()
  else if (key === 'copyTo') openTransfer('copy', rows)
  else if (key === 'moveTo') openTransfer('move', rows)
  else if (key === 'compress') openCompress(rows)
  else if (key === 'share') openShare(rows)
  else if (key === 'delete') deleteSelected()
}

// ---- 行"更多"（列表 ⋯ 与 网格瓦片 ⋯ 共用；下载/删除并入）----
const rowActions = (row: FileEntryInfo): ActionMenuItem[] => {
  const caps = currentCaps.value
  const actions: ActionMenuItem[] = [
    { key: 'rename', label: t('fileManager.rename'), icon: 'i-lucide-pencil' },
  ]
  if (!row.isDir) actions.push({ key: 'download', label: t('fileManager.download'), icon: 'i-lucide-download' })
  if (!row.isDir) actions.push({ key: 'edit', label: t('fileManager.edit'), icon: 'i-lucide-file-pen', hidden: isMediaFile(row.name) })
  if (!row.isDir) actions.push({ key: 'copyLink', label: t('fileManager.copyLink'), icon: 'i-lucide-link' })
  if (caps.move) actions.push({ key: 'moveTo', label: t('fileManager.moveTo'), icon: 'i-lucide-folder-input' })
  if (caps.copy) actions.push({ key: 'copyTo', label: t('fileManager.copyTo'), icon: 'i-lucide-folder-plus' })
  if (caps.copy) actions.push({ key: 'copy', label: t('fileManager.copyToClipboard'), icon: 'i-lucide-copy' })
  if (caps.move) actions.push({ key: 'cut', label: t('fileManager.cut'), icon: 'i-lucide-scissors' })
  if (caps.compress) actions.push({ key: 'compress', label: t('fileManager.compressZip'), icon: 'i-lucide-archive' })
  if (caps.compress && !row.isDir && /\.(zip|tar|gz|tgz|rar|7z)$/i.test(row.name)) {
    actions.push({ key: 'unzip', label: t('fileManager.unzip'), icon: 'i-lucide-folder-archive' })
  }
  // 分享支持任意系统来源（本地/OSS/FTP/WebDAV）；实例作用域不提供分享。
  // 分享链接是主控域名下的公开地址，远程节点上下文不提供分享。
  if (canShare.value) {
    actions.push({ key: 'share', label: t('fileManager.share'), icon: 'i-lucide-share-2' })
  }
  actions.push({ key: 'delete', label: t('fileManager.delete'), icon: 'i-lucide-trash-2', danger: true })
  return actions
}
const onRowAction = (key: string, row: FileEntryInfo) => {
  if (key === 'rename') openRename(row)
  else if (key === 'download') downloadOne(row)
  else if (key === 'edit') openEditor(row)
  else if (key === 'moveTo') openTransfer('move', [row])
  else if (key === 'copyTo') openTransfer('copy', [row])
  else if (key === 'copyLink') copyLink(row)
  else if (key === 'copy') setClipboard('copy', [row])
  else if (key === 'cut') setClipboard('cut', [row])
  else if (key === 'compress') openCompress([row])
  else if (key === 'unzip') openUnzip(row)
  else if (key === 'share') openShare([row])
  else if (key === 'delete') deleteRows([row])
}

// ---- 手机：行点击 / 底部操作面板 ----
const onMobileRowClick = (row: FileEntryInfo) => {
  // 选择模式下点击即勾选；否则打开（目录进入、文件预览/编辑）。
  if (hasSelection.value) toggleSelect(row)
  else onRowOpen(row)
}
const sheet = reactive({ open: false, row: null as FileEntryInfo | null })
const openSheet = (row: FileEntryInfo) => {
  sheet.row = row
  sheet.open = true
}
const SHEET_QUICK = ['download', 'share', 'rename']
// 列表区前置的操作（其余按行菜单顺序）
const SHEET_FIRST = ['moveTo', 'compress']
const sheetQuick = computed<ActionMenuItem[]>(() => {
  const row = sheet.row
  if (!row) return []
  const all = rowActions(row)
  const quick: ActionMenuItem[] = [
    { key: 'open', label: row.isDir ? t('fileManager.open') : t('fileManager.preview'), icon: row.isDir ? 'i-lucide-folder-open' : 'i-lucide-eye' },
  ]
  for (const key of SHEET_QUICK) {
    const hit = all.find((a) => a.key === key)
    if (hit) quick.push(hit)
  }
  return quick.slice(0, 4)
})
const sheetItems = computed<ActionMenuItem[]>(() => {
  const row = sheet.row
  if (!row) return []
  const used = new Set(sheetQuick.value.map((a) => a.key))
  const rest = rowActions(row).filter((a) => !used.has(a.key) && !a.hidden)
  const first = SHEET_FIRST.map((k) => rest.find((a) => a.key === k)).filter((a): a is ActionMenuItem => !!a)
  return [
    ...first,
    ...rest.filter((a) => !SHEET_FIRST.includes(a.key) && !a.danger),
    { key: 'select', label: t('fileManager.select'), icon: 'i-lucide-square-check' },
    ...rest.filter((a) => a.danger),
  ]
})
const sheetMeta = computed(() => {
  const row = sheet.row
  if (!row) return ''
  const parts = [typeLabel(row)]
  if (!row.isDir) parts.push(formatFileSize(row.size))
  parts.push(shortTime(row.updateTime))
  return parts.join(' · ')
})
const onSheetSelect = (key: string) => {
  const row = sheet.row
  if (!row) return
  if (key === 'open') onRowOpen(row)
  else if (key === 'select') toggleSelect(row)
  else onRowAction(key, row)
}

// ---- 上传：选择 / 拖放即加入队列开始上传（目标目录与来源在加入时冻结）；当前目录有文件传完即刷新列表 ----
const upload = useFileUpload(
  () => client,
  () => currentPath.value,
  {
    getSourceId: () => currentSourceId.value,
    onUploaded: (item) => {
      if (item.targetPath === currentPath.value && item.sourceId === currentSourceId.value) loadList()
    },
  },
)
const uploadInputRef = ref<HTMLInputElement | null>(null)
const pickUpload = () => uploadInputRef.value?.click()
const onUploadPick = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.length) {
    upload.addFiles(input.files)
    if (!isMobile.value) treeOpen.value = true
  }
  input.value = ''
}

// ---- 拖放上传：把文件拖到列表区即加入上传队列 ----
const dropActive = ref(false)
let dragDepth = 0
const hasFiles = (event: DragEvent) => Array.from(event.dataTransfer?.types ?? []).includes('Files')
const onDragEnter = (event: DragEvent) => {
  if (!hasFiles(event)) return
  dragDepth += 1
  dropActive.value = true
}
const onDragLeave = () => {
  dragDepth = Math.max(0, dragDepth - 1)
  if (!dragDepth) dropActive.value = false
}
const onDrop = (event: DragEvent) => {
  dragDepth = 0
  dropActive.value = false
  const files = event.dataTransfer?.files
  if (!files?.length) return
  upload.addFiles(Array.from(files))
  if (!isMobile.value) treeOpen.value = true
}

onMounted(() => {
  loadSources()
  loadList()
})

// 切到指定来源（文件页被缓存时，从文件源页「浏览」带着 ?source= 回来用）。
const switchSource = (id: string) => {
  if (isSystemScope && id !== currentSourceId.value) onSourceChange(id)
}

defineExpose({ refresh, navigateTo, switchSource })
</script>

<style scoped lang="scss">
/* 页面自身滚动（系统页与实例页都以全出血方式挂载）：外层负责内边距 + 纵向滚动，内层限宽居中。 */
.fb {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 28px 32px 32px;
  background: var(--el-bg-color-page, var(--m-bg));
  color: var(--el-text-color-primary);
}
.fb__inner {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 1600px;
  margin: 0 auto;
  min-height: 100%;
}

/* 嵌入模式：外层页面已有内边距与滚动，这里与页面里其他内容对齐 */
.fb.fb--inset {
  flex: none;
  min-height: auto;
  overflow: visible;
  padding: 0;
  background: transparent;
}
.fb.fb--inset .fb__inner {
  max-width: none;
}

/* ===== 页头（PageHeader） ===== */
/* 内嵌在应用详情「文件」标签里：外层已有应用名与目录条，不再重复大标题 */
.fb-head--embedded :deep(.page-header__title) {
  display: none;
}
.fb-head--embedded :deep(.page-header__desc) {
  margin-top: 0;
}
.fb-head__desc {
  display: inline;
}
.fb-head__path {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}

/* ===== 通用小按钮 ===== */
.fb-ico {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 12px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fb-ico:hover:not(:disabled) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.fb-ico:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.fb-ico :deep(svg),
.fb-ico :deep(.iconify) {
  width: 18px;
  height: 18px;
}
.fb-ico--lg {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
.fb-head :deep(.page-header__actions .fb-ico--lg) {
  width: 44px;
  height: 44px;
  min-height: 44px;
  padding: 0;
  background: var(--m-pSoft);
  color: var(--m-pInk);
  border-radius: 999px;
}
.fb-ico--lg :deep(svg),
.fb-ico--lg :deep(.iconify) {
  width: 22px;
  height: 22px;
}
.fb-ico.is-spin :deep(svg) {
  animation: fb-spin 0.8s linear infinite;
}
@keyframes fb-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ===== 来源胶囊 + 磁盘用量 ===== */
.fb-sources {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.fb-sources__chips {
  display: flex;
  gap: 8px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
}
.fb-sources__chips::-webkit-scrollbar {
  display: none;
}
.fb-sources__spacer {
  flex: 1;
}
.fb-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: none;
  min-height: 44px;
  padding: 0 16px 0 6px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  text-decoration: none;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.fb-chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.fb-chip.is-on,
.fb-chip.is-on:hover {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
.fb-chip__sticker {
  width: 30px;
  height: 30px;
  border-radius: 10px;
}
.fb-chip__sticker :deep(svg),
.fb-chip__sticker :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.fb-chip__name {
  color: inherit;
}
.fb-chip__type {
  font-size: 0.75rem;
  opacity: 0.7;
}
.fb-chip--dashed {
  padding-left: 14px;
  border-style: dashed;
}
.fb-chip__ico {
  width: 16px;
  height: 16px;
}
.fb-disk {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 200px;
}
.fb-disk__text {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  text-align: right;
}
.fb-disk__text.is-warn {
  color: var(--m-bad);
}
.fb-disk__bar {
  display: block;
  height: 6px;
  border-radius: 999px;
  background: var(--m-s2);
  overflow: hidden;
}
.fb-disk__bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--m-p);
}
.fb-disk__bar span.is-warn {
  background: var(--m-bad);
}

/* ===== 路径条 + 工具 ===== */
.fb-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.fb-crumb {
  flex: 1 1 360px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
}
.fb-crumb__path {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 2px;
  min-height: 36px;
  overflow-x: auto;
  white-space: nowrap;
  cursor: text;
  scrollbar-width: none;
}
.fb-crumb__path::-webkit-scrollbar {
  display: none;
}
.fb-crumb__input {
  flex: 1;
  min-width: 0;
  height: 34px;
  padding: 0 10px;
  border: 1.5px solid var(--m-p);
  border-radius: 10px;
  outline: none;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
}
.fb-crumb__seg {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 10px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
}
.fb-crumb__seg:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.fb-crumb__seg.is-current {
  color: var(--el-text-color-primary);
  font-weight: 800;
}
.fb-crumb__root {
  width: 16px;
  height: 16px;
}
.fb-crumb__sep {
  flex: none;
  width: 14px;
  height: 14px;
  color: var(--el-text-color-secondary);
  opacity: 0.6;
}
.fb-bar__tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 auto;
  min-width: 0;
}
.fb-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 300px;
  min-width: 0;
  min-height: 44px;
  padding: 0 4px 0 14px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s;
}
.fb-search:focus-within {
  border-color: var(--m-p);
}
.fb-search__ico {
  flex: none;
  width: 16px;
  height: 16px;
}
.fb-search__input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.875rem;
}
.fb-search__sub {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 34px;
  padding: 0 10px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
}
.fb-search__sub :deep(svg),
.fb-search__sub :deep(.iconify) {
  width: 15px;
  height: 15px;
}
.fb-search__sub:hover {
  background: var(--el-fill-color-light);
}
.fb-search__sub.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.fb-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 16px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
.fb-pill:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.fb-pill :deep(svg),
.fb-pill :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.fb-pill--round {
  width: 48px;
  min-height: 48px;
  padding: 0;
  justify-content: center;
}
.fb-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
}
.fb-seg__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 40px;
  height: 36px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.fb-seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.fb-seg__btn :deep(svg),
.fb-seg__btn :deep(.iconify) {
  width: 16px;
  height: 16px;
}

/* 下拉面板 */
.fb-menu {
  display: flex;
  flex-direction: column;
  padding: 6px;
}
.fb-menu__item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 0 10px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.875rem;
  text-align: left;
  cursor: pointer;
}
.fb-menu__item:hover {
  background: var(--el-fill-color-light);
}
.fb-menu__item.is-on {
  color: var(--m-pInk);
  font-weight: 700;
}
.fb-menu__ico {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.fb-menu__check {
  margin-left: auto;
  width: 16px;
  height: 16px;
}

/* ===== 剪贴板条 ===== */
.fb-clip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 8px 8px 16px;
  border: 1.5px dashed var(--m-p);
  border-radius: var(--app-radius);
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.fb-clip__ico {
  width: 18px;
  height: 18px;
}
.fb-clip__text {
  font-size: 0.875rem;
  font-weight: 700;
}
.fb-clip__spacer {
  flex: 1;
}

/* ===== 主体 ===== */
.fb-body {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 16px;
  flex: 1;
}
/* 左栏：目录树 + 上传队列（吸顶）；收起后只剩 64px 窄栏 */
.fb-side {
  position: sticky;
  top: 0;
  flex: 0 0 236px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: calc(100vh - 140px);
}
.fb-side.is-closed {
  flex-basis: 64px;
}
.fb-rail {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 4px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.fb-rail__sep {
  width: 28px;
  height: 1px;
  background: var(--el-border-color-lighter);
}
.fb-rail__up {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  width: 44px;
  height: 56px;
  border: none;
  border-radius: 14px;
  background: transparent;
  color: var(--m-pInk);
  font-size: 0.6875rem;
  font-weight: 800;
  cursor: pointer;
}
.fb-rail__up:hover {
  background: var(--m-s2);
}
.fb-rail__up :deep(svg),
.fb-rail__up :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.fb-aside {
  flex: 0 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: calc(var(--cmp-card-unit) * 12) calc(var(--cmp-card-unit) * 8);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  scrollbar-width: thin;
  --ft-active-bg: var(--m-pSoft);
  --ft-active-fg: var(--m-pInk);
  --ft-hover: var(--m-s2);
  --ft-fg: var(--el-text-color-secondary);
}
.fb-aside__head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 0 4px 10px;
}
.fb-aside__label {
  flex: 1;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.fb-aside :deep(.ftn-row) {
  height: 34px;
  border-radius: 10px;
  font-weight: 600;
}
.fb-aside :deep(.ftn-row.is-active) {
  font-weight: 800;
}
.fb-aside :deep(.ftn-icon.is-folder) {
  color: var(--m-a);
}
.fb-aside :deep(.ftn-row.is-active .ftn-icon.is-folder) {
  color: var(--m-pInk);
}
.fb-main {
  position: relative;
  flex: 1 1 480px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.fb-main.is-drop .fb-card,
.fb-main.is-drop .fb-grid {
  opacity: 0.45;
}
.fb-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
}
.fb-drop {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border: 2px dashed var(--m-p);
  border-radius: var(--app-radius-lg);
  background: color-mix(in srgb, var(--m-pSoft) 80%, transparent);
  color: var(--m-pInk);
  font-weight: 800;
  pointer-events: none;
}
.fb-drop :deep(svg),
.fb-drop :deep(.iconify) {
  width: 32px;
  height: 32px;
}

/* 骨架 */
.fb-skeleton {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
}
.fb-skeleton__row {
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(100deg, var(--m-s2) 30%, var(--el-fill-color-light) 50%, var(--m-s2) 70%);
  background-size: 200% 100%;
  animation: fb-shimmer 1.4s ease-in-out infinite;
}
@keyframes fb-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}

/* 错误态 */
.fb-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 24px;
  text-align: center;
}
.fb-error__sticker {
  width: 64px;
  height: 64px;
  border-radius: 22px;
  transform: rotate(-6deg);
}
.fb-error__sticker :deep(svg),
.fb-error__sticker :deep(.iconify) {
  width: 30px;
  height: 30px;
}
.fb-error__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
}
.fb-error__msg {
  margin: 0;
  max-width: 520px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.fb-error__hint {
  margin: 0;
  max-width: 420px;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.fb-error__actions {
  display: flex;
  gap: 10px;
  margin-top: 6px;
}

/* 贴纸 */
.fb-sticker {
  width: 36px;
  height: 36px;
  border-radius: 12px;
}
.fb-sticker :deep(svg),
.fb-sticker :deep(.iconify) {
  width: 18px;
  height: 18px;
}
.m-sticker--neutral {
  background: var(--m-s2);
  color: var(--m-ink2);
}

/* 复选框 */
.fb-cb {
  width: 18px;
  height: 18px;
  margin: 0;
  accent-color: var(--m-p);
  cursor: pointer;
}

/* ===== 列表 ===== */
.fb-table {
  container-type: inline-size;
  padding: 4px 8px;
  overflow-x: auto;
  scrollbar-width: none;
}
.fb-table::-webkit-scrollbar {
  display: none;
}
.fb-thead,
.fb-row {
  display: grid;
  grid-template-columns: 40px minmax(200px, 1fr) 96px 112px 124px;
  align-items: center;
  min-width: 560px;
}
/* 列表变窄（打开右侧预览、或窄屏）时去掉修改时间列，操作列始终可见，不靠隐藏的横向滚动 */
@container (width < 600px) {
  .fb-thead,
  .fb-row {
    grid-template-columns: 40px minmax(0, 1fr) 84px 124px;
    min-width: 0;
  }
  .fb-thead .fb-col--time,
  .fb-row .fb-col--time {
    display: none;
  }
}
.fb-thead {
  min-height: 44px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.fb-thead .fb-col {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--el-text-color-secondary);
}
.fb-col {
  min-width: 0;
  padding: 0 10px;
}
.fb-col--check {
  display: flex;
  justify-content: center;
  padding: 0;
}
.fb-col--ops {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
}
.fb-thead .fb-col--ops {
  justify-content: flex-end;
}
.fb-sort {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 100%;
  border: none;
  background: transparent;
  text-align: left;
  cursor: pointer;
}
.fb-sort:hover {
  color: var(--el-text-color-primary);
}
.fb-sort__ico {
  width: 14px;
  height: 14px;
}
.fb-row {
  min-height: 60px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  cursor: pointer;
  transition: background 0.12s;
}
.fb-row:last-child {
  border-bottom: none;
}
.fb-row:hover {
  background: var(--m-s2);
}
.fb-row.is-selected {
  background: color-mix(in srgb, var(--m-pSoft) 70%, var(--el-bg-color));
}
.fb-row.is-preview {
  background: var(--m-s2);
  box-shadow: inset 3px 0 0 var(--m-p);
}
.fb-col--name {
  display: flex;
  align-items: center;
  gap: 12px;
}
.fb-row__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.fb-row__name {
  font-size: 0.875rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fb-row:hover .fb-row__name {
  color: var(--m-pInk);
}
.fb-row__kind {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.fb-col--size {
  font-size: 0.8125rem;
  font-weight: 700;
}
.fb-col--size.is-dim {
  color: var(--el-text-color-secondary);
  font-weight: 500;
}
.fb-col--time {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
/* 右栏：文件预览 */
.fb-preview {
  position: sticky;
  top: 0;
  flex: 0 1 320px;
}
.fb-upload-input {
  display: none;
}

/* ===== 网格 ===== */
.fb-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}
.fb-tile {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: calc(var(--cmp-card-unit) * 12);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.fb-tile:hover {
  border-color: var(--m-p);
}
.fb-tile.is-selected {
  border-color: var(--m-p);
  box-shadow: 0 0 0 3px var(--m-pSoft);
}
.fb-tile__cover {
  width: 100%;
  height: 88px;
  border-radius: 18px;
}
.fb-tile__cover :deep(svg),
.fb-tile__cover :deep(.iconify) {
  width: 34px;
  height: 34px;
}
.fb-tile__check {
  position: absolute;
  top: 20px;
  right: 20px;
  opacity: 0;
  transition: opacity 0.12s;
}
.fb-tile:hover .fb-tile__check,
.fb-tile.is-selected .fb-tile__check {
  opacity: 1;
}
.fb-tile__foot {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.fb-tile__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.fb-tile__name {
  font-size: 0.875rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fb-tile__meta {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ===== 底部 ===== */
.fb-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 0 6px;
}
.fb-foot__hint {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.fb-foot__hint :deep(svg),
.fb-foot__hint :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.fb-foot :deep(.pagination) {
  flex: 1;
}

/* ===== 批量条 ===== */
.fb-bulk {
  position: sticky;
  bottom: 16px;
  z-index: 5;
  align-self: center;
  display: flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  padding: 6px 6px 6px 20px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-lg);
}
.fb-bulk__count {
  margin-right: 10px;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
}
.fb-bulk__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 40px;
  min-height: 40px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--m-bg);
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
.fb-bulk__btn:hover {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.fb-bulk__btn.is-danger {
  color: var(--m-a);
}
.fb-bulk__btn--x {
  padding: 0;
}
.fb-bulk__btn :deep(svg),
.fb-bulk__btn :deep(.iconify) {
  width: 16px;
  height: 16px;
}

.fb-confirm {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}

/* 窄屏 PC：三栏放不下时预览栏换行到列表下方，占满一行、不再吸顶 */
@media (width < 1360px) {
  .fb-preview {
    position: static;
    flex: 1 1 100%;
  }
}

/* ===== 手机 ===== */
.fb.is-mobile {
  padding: 16px 16px calc(96px + env(safe-area-inset-bottom));
}
.fb.fb--inset.is-mobile {
  padding: 0 0 calc(96px + env(safe-area-inset-bottom));
}
.fb.is-mobile .fb__inner {
  gap: 12px;
}
.fb.is-mobile .fb-sources {
  flex-wrap: nowrap;
  flex-direction: column;
  align-items: stretch;
  margin: 0 -16px;
}
.fb.is-mobile .fb-sources__chips {
  padding: 0 16px;
}
.fb.is-mobile .fb-sources__spacer {
  display: none;
}
.fb.is-mobile .fb-disk {
  min-width: 0;
  padding: 0 16px;
}
.fb.is-mobile .fb-disk__text {
  text-align: left;
}
.fb.is-mobile .fb-bar {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}
.fb.is-mobile .fb-crumb {
  flex: none;
}
.fb.is-mobile .fb-bar__tools {
  width: 100%;
}
.fb.is-mobile .fb-search {
  flex: 1;
  min-height: 48px;
}
.fb.is-mobile .fb-ico {
  width: 44px;
  height: 44px;
}
.fb.is-mobile .fb-crumb__seg {
  min-height: 44px;
}
.fb.is-mobile .fb-ico--lg {
  width: 44px;
  height: 44px;
}
.fb.is-mobile .fb-clip {
  flex-wrap: wrap;
}
.fb-mhead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 6px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.fb-mlist {
  padding: 4px 0;
}
.fb-mrow {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 64px;
  padding: 6px 6px 6px 14px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
}
.fb-mrow:last-child {
  border-bottom: none;
}
.fb-mrow:active {
  background: var(--m-s2);
}
.fb-mrow.is-selected {
  background: color-mix(in srgb, var(--m-pSoft) 70%, var(--el-bg-color));
}
.fb-mrow .fb-sticker {
  width: 42px;
  height: 42px;
  border-radius: 14px;
}
.fb-mrow__check {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: 2px solid var(--el-border-color);
  border-radius: 7px;
  color: var(--m-pOn);
}
.fb-mrow__check.is-on {
  border-color: var(--m-p);
  background: var(--m-p);
}
.fb-mrow__check :deep(svg),
.fb-mrow__check :deep(.iconify) {
  width: 14px;
  height: 14px;
}
.fb-mrow__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.fb-mrow__name {
  font-size: 0.9375rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fb-mrow__meta {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.fb.is-mobile .fb-bulk {
  position: fixed;
  left: 12px;
  right: 12px;
  bottom: calc(78px + env(safe-area-inset-bottom));
  justify-content: space-between;
  padding-left: 16px;
}
.fb.is-mobile .fb-bulk__count {
  margin-right: auto;
}
.fb.is-mobile .fb-bulk__btn {
  padding: 0;
  min-width: 40px;
}
.fb-fab {
  position: fixed;
  right: 16px;
  bottom: calc(82px + env(safe-area-inset-bottom));
  z-index: 6;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 56px;
  padding: 0 22px;
  border: none;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 0.9375rem;
  font-weight: 700;
  box-shadow: var(--app-shadow-brand);
  cursor: pointer;
}
.fb-fab :deep(svg),
.fb-fab :deep(.iconify) {
  width: 20px;
  height: 20px;
}
</style>
