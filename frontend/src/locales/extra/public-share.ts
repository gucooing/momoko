// 公开分享页（/public/share/:token）补充文案：递归文件数、文件夹子项数、下载方式说明。
export default {
  'zh-CN': {
    publicShare: {
      all: '全部',
      items: '{n} 项',
      noteStorage: '由存储节点直接下载，不限速',
      noteProxy: '经由分享者的服务器下载，不限速',
      noteStorageShort: '由存储节点直接下载 · 每次下载计 1 次',
      noteProxyShort: '经由分享者的服务器下载 · 每次下载计 1 次',
    },
  },
  'zh-HK': {
    publicShare: {
      all: '全部',
      items: '{n} 項',
      noteStorage: '由儲存節點直接下載，不限速',
      noteProxy: '經由分享者的伺服器下載，不限速',
      noteStorageShort: '由儲存節點直接下載 · 每次下載計 1 次',
      noteProxyShort: '經由分享者的伺服器下載 · 每次下載計 1 次',
    },
  },
  'en-US': {
    publicShare: {
      all: 'All',
      items: '{n} items',
      noteStorage: 'Downloads come straight from the storage node, no speed limit',
      noteProxy: 'Downloads go through the sharer server, no speed limit',
      noteStorageShort: 'Straight from the storage node · each download counts once',
      noteProxyShort: 'Through the sharer server · each download counts once',
    },
  },
}
