import type { OrderManageModule } from './orderManage.module'
import { registerOrderBanCommands } from './order-manage-ban-commands'
import { registerOrderGroupCommands } from './order-manage-group-commands'
import { registerOrderUnbanCommands } from './order-manage-unban-commands'

export function registerOrderManageCommands(host: OrderManageModule): void {
  registerOrderBanCommands(host)
  registerOrderGroupCommands(host)
  registerOrderUnbanCommands(host)
}
