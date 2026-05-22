import VxeTable, { VxeGrid } from 'vxe-table'
import {
  VxeButton,
  VxeButtonGroup,
  VxeTooltip,
  VxeNumberInput,
  VxeInput,
  VxeSelect,
} from 'vxe-pc-ui'
import 'vxe-pc-ui/es/style.css'
import 'vxe-table/lib/style.css'
import '@/styles/vxeGrid.css'

let hasConfigured = false

const ensureVxeGridConfig = () => {
  if (hasConfigured) return

  VxeTable.setConfig({
    table: {
      showHeaderOverflow: true,
      border: true,
      headerCellConfig: {
        height: 40,
      },
      cellConfig: {
        height: 40,
      },
      columnConfig: {
        resizable: true,
      },
    },
  })

  hasConfigured = true
}

ensureVxeGridConfig()

// 导入即注册，保持页面按需接入 VXE 能力。
void [VxeTooltip, VxeButtonGroup, VxeNumberInput, VxeInput, VxeSelect]

export { VxeButton, VxeGrid, ensureVxeGridConfig }
