import { assertConsoleGuildAccess, assertGlobalConsoleScope } from './console-guild-scope'
import { error, success } from './api-response'
import {
  filterGuildEntries,
  type WebSocketAPIContext,
} from './websocket-api-context'

export function registerConfigAPI(api: WebSocketAPIContext) {
  const { ctx, data, addAuthorityListener, resolveConsoleScope } = api

  addAuthorityListener('stuhelperGroupCenter/config/reload', async () => {
    try {
      data.groupConfig.reload()
      ctx.logger('stuhelperGroupCenter').info('群组配置已重新加载，共 %d 条', Object.keys(data.groupConfig.getAll()).length)
      return success({
        success: true,
        count: Object.keys(data.groupConfig.getAll()).length,
      })
    } catch (e) {
      return error(e instanceof Error ? e.message : '重新加载配置失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/config/list', async function (params?: { fetchNames?: boolean }) {
    try {
      const scope = await resolveConsoleScope(this)
      const allConfigs = data.groupConfig.getAll()
      const entries = filterGuildEntries(allConfigs, scope)

      if (!params?.fetchNames) {
        return success(Object.fromEntries(entries))
      }

      const enrichedEntries = await Promise.all(entries.map(async ([guildId, config]) => {
        let name = guildId
        let avatar = ''
        try {
          const bot = ctx.bots.find(b => b.platform === 'onebot' || b.platform === 'red')
          if (bot?.getGuild) {
            const guild = await bot.getGuild(guildId)
            if (guild?.name) name = guild.name
            if (guild?.avatar) avatar = guild.avatar
          }
        } catch (e) {
          ctx.logger('stuhelperGroupCenter').debug('获取群名称失败: %s', e)
        }
        return [guildId, { ...config, guildName: name, guildAvatar: avatar }]
      }))
      return success(Object.fromEntries(enrichedEntries))
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取配置列表失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/config/get', async function (params: { guildId: string }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertConsoleGuildAccess(scope, params.guildId, 'group config')
      const config = data.groupConfig.get(params.guildId)
      return success(config)
    } catch (e) {
      return error(e instanceof Error ? e.message : '获取配置失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/config/update', async function (params: { guildId: string, config: any }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertConsoleGuildAccess(scope, params.guildId, 'group config')
      data.groupConfig.set(params.guildId, params.config)
      await data.groupConfig.flush()
      return success({ success: true })
    } catch (e) {
      return error(e instanceof Error ? e.message : '更新配置失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/config/create', async function (params: { guildId: string }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertConsoleGuildAccess(scope, params.guildId, 'group config')
      if (data.groupConfig.get(params.guildId)) {
        return error('配置已存在')
      }
      data.groupConfig.set(params.guildId, {
        command: '绑定',
        enableWelcome: true,
        welcomeMessage: '欢迎新成员！',
        enableLeave: false,
        leaveMessage: '',
        enableMute: false,
        muteThreshold: 3,
      } as any)
      await data.groupConfig.flush()
      return success({ success: true })
    } catch (e) {
      return error(e instanceof Error ? e.message : '创建配置失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/config/delete', async function (params: { guildId: string }) {
    const scope = await resolveConsoleScope(this)
    assertConsoleGuildAccess(scope, params.guildId, 'group config')
    data.groupConfig.delete(params.guildId)
    await data.groupConfig.flush()
    return success({ success: true })
  })
}
