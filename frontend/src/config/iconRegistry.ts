import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  Calendar,
  Check,
  Delete,
  Edit,
  House,
  Monitor,
  Orange,
  Plus,
  Refresh,
  Search,
  Top,
} from '@element-plus/icons-vue'
import {
  AdjustmentsHorizontalIcon,
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
  DevicePhoneMobileIcon,
  DocumentTextIcon,
  EllipsisHorizontalIcon,
  ExclamationTriangleIcon,
  EyeIcon,
  FolderIcon,
  GlobeAltIcon,
  HandRaisedIcon,
  IdentificationIcon,
  KeyIcon,
  LinkIcon,
  MagnifyingGlassIcon,
  MapPinIcon,
  MinusCircleIcon,
  NoSymbolIcon,
  PlayIcon,
  PlusCircleIcon,
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

const iconRegistry = markRaw<Record<string, Component>>({
  Search: Search,
  Refresh: Refresh,
  Plus: Plus,
  Delete: Delete,
  Edit: Edit,
  Check: Check,
  'Element:ArrowDown': ArrowDown,
  'Element:ArrowLeft': ArrowLeft,
  'Element:ArrowRight': ArrowRight,
  'Element:Calendar': Calendar,
  'Element:Check': Check,
  'Element:Delete': Delete,
  'Element:Edit': Edit,
  'Element:House': House,
  'Element:Monitor': Monitor,
  'Element:Orange': Orange,
  'Element:Plus': Plus,
  'Element:Refresh': Refresh,
  'Element:Search': Search,
  'Element:Top': Top,
  'HOutline:AdjustmentsHorizontalIcon': AdjustmentsHorizontalIcon,
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
  'HOutline:DevicePhoneMobileIcon': DevicePhoneMobileIcon,
  'HOutline:DocumentTextIcon': DocumentTextIcon,
  'HOutline:EllipsisHorizontalIcon': EllipsisHorizontalIcon,
  'HOutline:ExclamationTriangleIcon': ExclamationTriangleIcon,
  'HOutline:EyeIcon': EyeIcon,
  'HOutline:FolderIcon': FolderIcon,
  'HOutline:GlobeAltIcon': GlobeAltIcon,
  'HOutline:HandRaisedIcon': HandRaisedIcon,
  'HOutline:IdentificationIcon': IdentificationIcon,
  'HOutline:KeyIcon': KeyIcon,
  'HOutline:LinkIcon': LinkIcon,
  'HOutline:MagnifyingGlassIcon': MagnifyingGlassIcon,
  'HOutline:MapPinIcon': MapPinIcon,
  'HOutline:MinusCircleIcon': MinusCircleIcon,
  'HOutline:NoSymbolIcon': NoSymbolIcon,
  'HOutline:PlayIcon': PlayIcon,
  'HOutline:PlusCircleIcon': PlusCircleIcon,
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
      return import('@element-plus/icons-vue').then((module) => module as unknown as IconModule)
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
  return ELEMENT_PREFIX
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
  get(target, key) {
    if (typeof key !== 'string') return Reflect.get(target, key)
    return target[key] || ensureDynamicIcon(key)
  },
  ownKeys(target) {
    return Reflect.ownKeys(target)
  },
  getOwnPropertyDescriptor() {
    return {
      enumerable: true,
      configurable: true,
    }
  },
}) as Record<string, Component>

export const loadIconCatalog = async (prefix: IconPrefix) => {
  const iconModule = await loadIconPack(prefix)

  return Object.keys(iconModule)
    .filter((name) => name !== 'default')
    .sort()
    .map((name) => {
      const iconComponent = iconModule[name]
      if (iconComponent && !iconRegistry[`${prefix}${name}`]) {
        cacheLoadedIcon(prefix, name, iconComponent)
      }
      return `${prefix}${name}`
    })
}
