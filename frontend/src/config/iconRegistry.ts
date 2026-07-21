// 图标注册表：仅 Heroicons outline/solid。Element Plus 图标包已卸载；
// 历史 Element: 前缀统一映射到 Heroicons 等价物，避免菜单/旧数据断裂。
import {
  AdjustmentsHorizontalIcon,
  ArchiveBoxIcon,
  ArrowDownTrayIcon,
  ArrowPathIcon,
  ArrowPathRoundedSquareIcon,
  ArrowRightOnRectangleIcon,
  ArrowTrendingUpIcon,
  ArrowUpTrayIcon,
  ArrowsPointingInIcon,
  ArrowsPointingOutIcon,
  Bars3BottomLeftIcon,
  Bars3BottomRightIcon,
  Bars3CenterLeftIcon,
  BellAlertIcon,
  BoltIcon,
  CalendarIcon,
  ChartBarIcon,
  CheckBadgeIcon,
  CheckCircleIcon,
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ClipboardDocumentIcon,
  ClipboardDocumentListIcon,
  ClockIcon,
  Cog6ToothIcon,
  CommandLineIcon,
  ComputerDesktopIcon,
  CubeIcon,
  DevicePhoneMobileIcon,
  DocumentTextIcon,
  EllipsisHorizontalIcon,
  ExclamationTriangleIcon,
  EyeIcon,
  FolderIcon,
  GlobeAltIcon,
  HandRaisedIcon,
  HomeIcon,
  IdentificationIcon,
  KeyIcon,
  LinkIcon,
  MagnifyingGlassIcon,
  MapPinIcon,
  MinusCircleIcon,
  NoSymbolIcon,
  PlayIcon,
  PlusCircleIcon,
  PlusIcon,
  PrinterIcon,
  QuestionMarkCircleIcon,
  ServerStackIcon,
  ShieldCheckIcon,
  SparklesIcon,
  Squares2X2Icon,
  SquaresPlusIcon,
  StopIcon,
  TrashIcon,
  UserCircleIcon,
  UserGroupIcon,
  UserIcon,
  WindowIcon,
  WrenchScrewdriverIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import {
  BanknotesIcon,
  CameraIcon,
  CheckCircleIcon as CheckCircleSolidIcon,
  ExclamationTriangleIcon as ExclamationTriangleSolidIcon,
  InformationCircleIcon,
  PhotoIcon,
  PlayCircleIcon,
  QuestionMarkCircleIcon as QuestionMarkCircleSolidIcon,
  ShieldCheckIcon as ShieldCheckSolidIcon,
  ShoppingCartIcon,
  SparklesIcon as SparklesSolidIcon,
  StopCircleIcon,
  TrophyIcon,
  UserPlusIcon,
  XCircleIcon,
  XMarkIcon as XMarkSolidIcon,
} from '@heroicons/vue/24/solid'
import { defineAsyncComponent, markRaw, type Component } from 'vue'

export type IconPrefix = 'Element:' | 'HOutline:' | 'HSolid:'

const ELEMENT_PREFIX: IconPrefix = 'Element:'
const HERO_OUTLINE_PREFIX: IconPrefix = 'HOutline:'
const HERO_SOLID_PREFIX: IconPrefix = 'HSolid:'

type IconModule = Record<string, Component>

const fallbackIcon = markRaw(QuestionMarkCircleIcon)

// 历史 Element: 名称 → Heroicons 组件（覆盖菜单种子与旧库数据）
const ELEMENT_TO_HERO: Record<string, Component> = {
  ArrowLeft: ChevronLeftIcon,
  ArrowRight: ChevronRightIcon,
  Calendar: CalendarIcon,
  Check: CheckCircleIcon,
  Delete: TrashIcon,
  Edit: DocumentTextIcon,
  House: HomeIcon,
  Monitor: ComputerDesktopIcon,
  Orange: SparklesIcon,
  Plus: PlusIcon,
  Refresh: ArrowPathIcon,
  Search: MagnifyingGlassIcon,
  Top: ArrowTrendingUpIcon,
  MessageBox: WindowIcon,
  Folder: FolderIcon,
  Box: CubeIcon,
  Archive: ArchiveBoxIcon,
}

const iconRegistry = markRaw<Record<string, Component>>({
  Search: MagnifyingGlassIcon,
  Refresh: ArrowPathIcon,
  Plus: PlusIcon,
  Delete: TrashIcon,
  Edit: DocumentTextIcon,
  Check: CheckCircleIcon,
  'Element:ArrowLeft': ChevronLeftIcon,
  'Element:ArrowRight': ChevronRightIcon,
  'Element:Calendar': CalendarIcon,
  'Element:Check': CheckCircleIcon,
  'Element:Delete': TrashIcon,
  'Element:Edit': DocumentTextIcon,
  'Element:House': HomeIcon,
  'Element:Monitor': ComputerDesktopIcon,
  'Element:Orange': SparklesIcon,
  'Element:Plus': PlusIcon,
  'Element:Refresh': ArrowPathIcon,
  'Element:Search': MagnifyingGlassIcon,
  'Element:Top': ArrowTrendingUpIcon,
  'Element:MessageBox': WindowIcon,
  'Element:Folder': FolderIcon,
  'Element:Box': CubeIcon,
  'Element:Archive': ArchiveBoxIcon,
  'HOutline:AdjustmentsHorizontalIcon': AdjustmentsHorizontalIcon,
  'HOutline:ArchiveBoxIcon': ArchiveBoxIcon,
  'HOutline:ArrowDownTrayIcon': ArrowDownTrayIcon,
  'HOutline:ArrowPathIcon': ArrowPathIcon,
  'HOutline:ArrowPathRoundedSquareIcon': ArrowPathRoundedSquareIcon,
  'HOutline:ArrowRightOnRectangleIcon': ArrowRightOnRectangleIcon,
  'HOutline:ArrowTrendingUpIcon': ArrowTrendingUpIcon,
  'HOutline:ArrowUpTrayIcon': ArrowUpTrayIcon,
  'HOutline:ArrowsPointingInIcon': ArrowsPointingInIcon,
  'HOutline:ArrowsPointingOutIcon': ArrowsPointingOutIcon,
  'HOutline:Bars3BottomLeftIcon': Bars3BottomLeftIcon,
  'HOutline:Bars3BottomRightIcon': Bars3BottomRightIcon,
  'HOutline:Bars3CenterLeftIcon': Bars3CenterLeftIcon,
  'HOutline:BellAlertIcon': BellAlertIcon,
  'HOutline:BoltIcon': BoltIcon,
  'HOutline:CalendarIcon': CalendarIcon,
  'HOutline:ChartBarIcon': ChartBarIcon,
  'HOutline:CheckBadgeIcon': CheckBadgeIcon,
  'HOutline:CheckCircleIcon': CheckCircleIcon,
  'HOutline:ChevronDoubleLeftIcon': ChevronDoubleLeftIcon,
  'HOutline:ChevronDoubleRightIcon': ChevronDoubleRightIcon,
  'HOutline:ChevronLeftIcon': ChevronLeftIcon,
  'HOutline:ChevronRightIcon': ChevronRightIcon,
  'HOutline:ClipboardDocumentIcon': ClipboardDocumentIcon,
  'HOutline:ClipboardDocumentListIcon': ClipboardDocumentListIcon,
  'HOutline:ClockIcon': ClockIcon,
  'HOutline:Cog6ToothIcon': Cog6ToothIcon,
  'HOutline:CommandLineIcon': CommandLineIcon,
  'HOutline:ComputerDesktopIcon': ComputerDesktopIcon,
  'HOutline:CubeIcon': CubeIcon,
  'HOutline:DevicePhoneMobileIcon': DevicePhoneMobileIcon,
  'HOutline:DocumentTextIcon': DocumentTextIcon,
  'HOutline:EllipsisHorizontalIcon': EllipsisHorizontalIcon,
  'HOutline:ExclamationTriangleIcon': ExclamationTriangleIcon,
  'HOutline:EyeIcon': EyeIcon,
  'HOutline:FolderIcon': FolderIcon,
  'HOutline:GlobeAltIcon': GlobeAltIcon,
  'HOutline:HandRaisedIcon': HandRaisedIcon,
  'HOutline:HomeIcon': HomeIcon,
  'HOutline:IdentificationIcon': IdentificationIcon,
  'HOutline:KeyIcon': KeyIcon,
  'HOutline:LinkIcon': LinkIcon,
  'HOutline:MagnifyingGlassIcon': MagnifyingGlassIcon,
  'HOutline:MapPinIcon': MapPinIcon,
  'HOutline:MinusCircleIcon': MinusCircleIcon,
  'HOutline:NoSymbolIcon': NoSymbolIcon,
  'HOutline:PlayIcon': PlayIcon,
  'HOutline:PlusCircleIcon': PlusCircleIcon,
  'HOutline:PlusIcon': PlusIcon,
  'HOutline:PrinterIcon': PrinterIcon,
  'HOutline:QuestionMarkCircleIcon': QuestionMarkCircleIcon,
  'HOutline:ServerStackIcon': ServerStackIcon,
  'HOutline:ShieldCheckIcon': ShieldCheckIcon,
  'HOutline:SparklesIcon': SparklesIcon,
  'HOutline:Squares2X2Icon': Squares2X2Icon,
  'HOutline:SquaresPlusIcon': SquaresPlusIcon,
  'HOutline:StopIcon': StopIcon,
  'HOutline:TrashIcon': TrashIcon,
  'HOutline:UserCircleIcon': UserCircleIcon,
  'HOutline:UserGroupIcon': UserGroupIcon,
  'HOutline:UserIcon': UserIcon,
  'HOutline:WindowIcon': WindowIcon,
  'HOutline:WrenchScrewdriverIcon': WrenchScrewdriverIcon,
  'HOutline:XMarkIcon': XMarkIcon,
  'HSolid:BanknotesIcon': BanknotesIcon,
  'HSolid:CameraIcon': CameraIcon,
  'HSolid:CheckCircleIcon': CheckCircleSolidIcon,
  'HSolid:ExclamationTriangleIcon': ExclamationTriangleSolidIcon,
  'HSolid:InformationCircleIcon': InformationCircleIcon,
  'HSolid:PhotoIcon': PhotoIcon,
  'HSolid:PlayCircleIcon': PlayCircleIcon,
  'HSolid:QuestionMarkCircleIcon': QuestionMarkCircleSolidIcon,
  'HSolid:ShieldCheckIcon': ShieldCheckSolidIcon,
  'HSolid:ShoppingCartIcon': ShoppingCartIcon,
  'HSolid:SparklesIcon': SparklesSolidIcon,
  'HSolid:StopCircleIcon': StopCircleIcon,
  'HSolid:TrophyIcon': TrophyIcon,
  'HSolid:UserPlusIcon': UserPlusIcon,
  'HSolid:XCircleIcon': XCircleIcon,
  'HSolid:XMarkIcon': XMarkSolidIcon,
})

const iconPackCache: Partial<Record<IconPrefix, Promise<IconModule>>> = {}

const loadIconPack = (prefix: IconPrefix) => {
  if (iconPackCache[prefix]) return iconPackCache[prefix]

  iconPackCache[prefix] = (() => {
    if (prefix === ELEMENT_PREFIX) {
      // 不再加载 @element-plus/icons-vue；Element: 走 ELEMENT_TO_HERO / 预注册表
      return Promise.resolve(ELEMENT_TO_HERO as IconModule)
    }
    if (prefix === HERO_OUTLINE_PREFIX) {
      return import('@heroicons/vue/24/outline').then((module) => module as unknown as IconModule)
    }
    return import('@heroicons/vue/24/solid').then((module) => module as unknown as IconModule)
  })()

  return iconPackCache[prefix]
}

const parseIconName = (key: string, prefix: IconPrefix) => {
  if (prefix === ELEMENT_PREFIX) {
    return key.startsWith(ELEMENT_PREFIX) ? key.slice(ELEMENT_PREFIX.length) : key
  }
  return key.slice(prefix.length)
}

const resolveIconPrefix = (key: string): IconPrefix | undefined => {
  if (key.startsWith(ELEMENT_PREFIX)) return ELEMENT_PREFIX
  if (key.startsWith(HERO_OUTLINE_PREFIX)) return HERO_OUTLINE_PREFIX
  if (key.startsWith(HERO_SOLID_PREFIX)) return HERO_SOLID_PREFIX
  if (key in iconRegistry) return undefined
  // 未前缀名：优先当 Hero outline 动态名，不再默认 Element
  return HERO_OUTLINE_PREFIX
}

const cacheLoadedIcon = (prefix: IconPrefix, iconName: string, component: Component) => {
  const rawComponent = markRaw(component)
  const prefixedName = `${prefix}${iconName}`
  iconRegistry[prefixedName] = rawComponent
  if (prefix === ELEMENT_PREFIX) {
    iconRegistry[iconName] = rawComponent
  }
  return rawComponent
}

const createAsyncIcon = (loader: () => Promise<Component | undefined>) =>
  markRaw(
    defineAsyncComponent({
      loader: async () => (await loader()) || fallbackIcon,
      delay: 0,
      suspensible: false,
    }),
  )

const ensureDynamicIcon = (key: string) => {
  const prefix = resolveIconPrefix(key)
  if (!prefix) return iconRegistry[key]

  const asyncIcon = createAsyncIcon(async () => {
    const iconName = parseIconName(key, prefix)
    const iconModule = await loadIconPack(prefix)
    const resolvedIcon = iconModule[iconName]
    if (!resolvedIcon) return fallbackIcon
    return cacheLoadedIcon(prefix, iconName, resolvedIcon)
  })

  iconRegistry[key] = asyncIcon
  return asyncIcon
}

export const iconComponents = new Proxy(iconRegistry, {
  get(target, prop) {
    if (typeof prop !== 'string') return Reflect.get(target, prop)
    if (prop in target) return target[prop]
    return ensureDynamicIcon(prop)
  },
})

/** 图标选择器目录：Element 目录返回兼容映射名；Hero 目录动态列包内导出。 */
export const loadIconCatalog = async (prefix: IconPrefix): Promise<string[]> => {
  if (prefix === ELEMENT_PREFIX) {
    return Object.keys(ELEMENT_TO_HERO)
      .sort()
      .map((name) => `${ELEMENT_PREFIX}${name}`)
  }

  const iconModule = await loadIconPack(prefix)
  return Object.keys(iconModule)
    .filter((name) => /^[A-Z]/.test(name) && name.endsWith('Icon'))
    .sort()
    .map((name) => `${prefix}${name}`)
}
