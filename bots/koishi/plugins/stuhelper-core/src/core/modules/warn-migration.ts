import type { WarnModule } from './warn.module'

type WarnMigrationData = Record<string, Record<string, { count: number; timestamp: number }>>
type LegacyWarnRecord = { count: number; timestamp?: number }

interface NumericWarnInput {
  migratedData: WarnMigrationData
  guildId: string
  userId: string
  count: number
}

export function migrateWarnData(host: WarnModule): void {
  const allWarns = host.data.warns.getAll()
  const migratedData: WarnMigrationData = {}
  const migratedKeys = new Set<string>()
  let hasMigration = false

  for (const [key, record] of Object.entries(allWarns)) {
    if (key.includes(':')) {
      if (migrateCompositeKeyRecord(migratedData, key, record)) {
        migratedKeys.add(key)
        hasMigration = true
      }
      continue
    }
    if (migrateGuildRecord(migratedData, key, record)) {
      migratedKeys.add(key)
      hasMigration = true
    }
  }

  if (!hasMigration) return
  deleteOldWarnRecords(host, migratedKeys)
  writeMigratedWarnRecords(host, migratedData)
  host.data.warns.flush()
  host.ctx.logger('stuhelperGroupCenter').info('警告数据已迁移到新格式')
}

function migrateCompositeKeyRecord(migratedData: WarnMigrationData, key: string, record: unknown): boolean {
  const [guildId, userId] = key.split(':')
  if (!guildId || !userId || !isRecord(record) || !isRecord(record.groups)) return false

  const oldRecord = record.groups[guildId]
  if (!isLegacyWarnRecord(oldRecord)) return false

  ensureGuild(migratedData, guildId)
  migratedData[guildId][userId] = {
    count: oldRecord.count,
    timestamp: oldRecord.timestamp || Date.now(),
  }
  return true
}

function migrateGuildRecord(migratedData: WarnMigrationData, guildId: string, record: unknown): boolean {
  if (!isRecord(record)) return false

  let hasMigration = false
  for (const [userId, value] of Object.entries(record)) {
    if (typeof value === 'number') {
      migrateNumericWarn({ migratedData, guildId, userId, count: value })
      hasMigration = true
      continue
    }
    if (isLegacyWarnRecord(value)) {
      ensureGuild(migratedData, guildId)
      migratedData[guildId][userId] = normalizeWarnRecord(value)
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

function deleteOldWarnRecords(host: WarnModule, migratedKeys: ReadonlySet<string>): void {
  for (const key of migratedKeys) {
    host.data.warns.delete(key)
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

function normalizeWarnRecord(record: LegacyWarnRecord): { count: number; timestamp: number } {
  return {
    count: record.count,
    timestamp: record.timestamp || Date.now(),
  }
}

function isLegacyWarnRecord(value: unknown): value is LegacyWarnRecord {
  return isRecord(value)
    && typeof value.count === 'number'
    && (value.timestamp === undefined || typeof value.timestamp === 'number')
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
