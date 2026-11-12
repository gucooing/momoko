// GM 组件共用的类型。

/** 一项奖励（与 MGP gift.send / mail.broadcast 的 rewards 相同；货币的 resourceId 为空串）。 */
export interface GmReward {
  resourceType: number
  resourceId: string
  quantity: number
}

/** 指令目标（玩家 pid 与显示名）。 */
export interface GmTargetRef {
  pid: string
  name?: string
}
