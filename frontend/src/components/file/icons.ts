// 文件管理统一图标集：细线 Heroicons(outline) + 两个自绘 SVG（软盘保存 / 四角全屏）。
// 全部文件相关特性只从这里取图标，避免再混入 Element-Plus 实心图标导致风格不一致。
import { defineComponent, h } from 'vue'

export {
  ArrowPathIcon as IconRefresh,
  ArrowDownTrayIcon as IconDownload,
  ArrowUpTrayIcon as IconUpload,
  PencilSquareIcon as IconRename,
  TrashIcon as IconDelete,
  FolderPlusIcon as IconNewFolder,
  DocumentPlusIcon as IconNewFile,
  FolderIcon as IconFolder,
  FolderOpenIcon as IconFolderOpen,
  DocumentIcon as IconFile,
  DocumentTextIcon as IconDocumentText,
  ChevronRightIcon as IconChevronRight,
  EllipsisHorizontalIcon as IconMore,
  EllipsisVerticalIcon as IconMoreVertical,
  MagnifyingGlassIcon as IconSearch,
  FunnelIcon as IconFilter,
  ArrowLeftIcon as IconBack,
  ArrowRightIcon as IconForward,
  ArrowUpIcon as IconUp,
  HomeIcon as IconHome,
  MinusIcon as IconMinimize,
  XMarkIcon as IconClose,
  ArrowUturnLeftIcon as IconReset,
  PlusIcon as IconPlus,
  LinkIcon as IconLink,
  ShareIcon as IconShare,
  ArchiveBoxIcon as IconCompress,
  ArchiveBoxArrowDownIcon as IconUnzip,
  ClipboardIcon as IconCopy,
  ScissorsIcon as IconCut,
  ClipboardDocumentIcon as IconPaste,
  SunIcon as IconLight,
  MoonIcon as IconDark,
  ComputerDesktopIcon as IconAuto,
  ChevronDownIcon as IconChevronDown,
  ChevronLeftIcon as IconChevronLeft,
  ChevronUpIcon as IconChevronUp,
  PhotoIcon as IconImage,
  FilmIcon as IconVideo,
  MusicalNoteIcon as IconAudio,
  ExclamationTriangleIcon as IconWarning,
  MagnifyingGlassPlusIcon as IconZoomIn,
  MagnifyingGlassMinusIcon as IconZoomOut,
} from '@heroicons/vue/24/outline'

const strokeProps = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  'stroke-width': 1.5,
  'stroke-linecap': 'round',
  'stroke-linejoin': 'round',
  'aria-hidden': 'true',
}

// 软盘「保存」图标（Heroicons 没有软盘，自绘以贴合参考图）。
export const IconSave = defineComponent({
  name: 'IconSave',
  render() {
    return h('svg', strokeProps, [
      h('path', {
        d: 'M4.5 5.25A1.75 1.75 0 0 1 6.25 3.5h9.19c.46 0 .91.19 1.24.51l2.81 2.82c.33.33.51.78.51 1.24v10.68a1.75 1.75 0 0 1-1.75 1.75H6.25A1.75 1.75 0 0 1 4.5 18.75z',
      }),
      h('path', { d: 'M8 3.5v4.25c0 .41.34.75.75.75h5a.75.75 0 0 0 .75-.75V3.5' }),
      h('rect', { x: 7.5, y: 12, width: 9, height: 6, rx: 0.75 }),
    ])
  },
})

// 四角括号「全屏」图标。
export const IconFullscreen = defineComponent({
  name: 'IconFullscreen',
  render() {
    return h('svg', strokeProps, [
      h('path', { d: 'M4 9V5.5A1.5 1.5 0 0 1 5.5 4H9' }),
      h('path', { d: 'M20 9V5.5A1.5 1.5 0 0 0 18.5 4H15' }),
      h('path', { d: 'M4 15v3.5A1.5 1.5 0 0 0 5.5 20H9' }),
      h('path', { d: 'M20 15v3.5A1.5 1.5 0 0 1 18.5 20H15' }),
    ])
  },
})

// 内向四角括号「退出全屏」图标。
export const IconExitFullscreen = defineComponent({
  name: 'IconExitFullscreen',
  render() {
    return h('svg', strokeProps, [
      h('path', { d: 'M9 4.5V8a1 1 0 0 1-1 1H4.5' }),
      h('path', { d: 'M15 4.5V8a1 1 0 0 0 1 1h3.5' }),
      h('path', { d: 'M9 19.5V16a1 1 0 0 0-1-1H4.5' }),
      h('path', { d: 'M15 19.5V16a1 1 0 0 1 1-1h3.5' }),
    ])
  },
})
