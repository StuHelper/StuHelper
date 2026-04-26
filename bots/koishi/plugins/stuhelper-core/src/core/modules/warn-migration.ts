import type { WarnModule } from './warn.module'

type WarnMigrationData = Record<string, Record<string, { count: number; timestamp: number }>>

interface NumericWarnInput {
  migratedData: WarnMigrationData
  guildId: string
  userId: string
  count: number
}

export function migrateWarnData(host: WarnModule): void {
  const allWarns = host.data.warns.getAll()
  const migratedData: WarnMigrationData = {}
  let hasMigration = false

  for (const [key, record] of Object.entries(allWarns)) {
    if (key.includes(':')) {
      hasMigration = migrateCompositeKeyRecord(migratedData, key, record) || hasMigration
      continue
    }
    hasMigration = migrateGuildRecord(migratedData, key, record) || hasMigration
  }

  if (!hasMigration) return
  deleteOldWarnRecords(host, allWarns)
  writeMigratedWarnRecords(host, migratedData)
  host.data.warns.flush()
  host.ctx.logger('stuhelperGroupCenter').info('警告数据已迁移到新格式')
}

function migrateCompositeKeyRecord(migratedData: WarnMigrationData, key: string, record: any): boolean {
  const [guildId, userId] = key.split(':')
  if (!record?.groups?.[guildId]) return false

  const oldRecord = record.groups[guildId]
  ensureGuild(migratedData, guildId)
  migratedData[guildId][userId] = {
    count: oldRecord.count,
    timestamp: oldRecord.timestamp || Date.now(),
  }
  return true
}

function migrateGuildRecord(migratedData: WarnMigrationData, guildId: string, record: any): boolean {
  let hasMigration = false
  for (const [userId, value] of Object.entries(record)) {
    if (typeof value === 'number') {
      migrateNumericWarn({ migratedData, guildId, userId, count: value })
      hasMigration = true
      continue
    }
    if (isWarnRecord(value)) {
      ensureGuild(migratedData, guildId)
      migratedData[guildId][userId] = value
    }
  }
  return hasMigration
}

function migrateNumericWarn(input: NumericWarnInput): void {
  const { migratedData, guildId, userId, count } = input
  ensureGuild(migratedData, guildId)
  if (migratedData[guildId][userId]) return
  migratedData[guildId][userId] = { count, timestamp: Date.now() }
}

function deleteOldWarnRecords(host: WarnModule, allWarns: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(allWarns)) {
    if (key.includes(':') || hasNumericWarnValue(value)) {
      host.data.warns.delete(key)
    }
  }
}

function writeMigratedWarnRecords(host: WarnModule, migratedData: WarnMigrationData): void {
  for (const [guildId, records] of Object.entries(migratedData)) {
    host.data.warns.set(guildId, records)
  }
}

function ensureGuild(migratedData: WarnMigrationData, guildId: string): void {
  if (!migratedData[guildId]) migratedData[guildId] = {}
}

function isWarnRecord(value: unknown): value is { count: number; timestamp: number } {
  return typeof value === 'object' && value !== null && 'count' in value
}

function hasNumericWarnValue(value: unknown): boolean {
  for (const item of Object.values(value as any)) {
    if (typeof item === 'number') return true
  }
  return false
}
