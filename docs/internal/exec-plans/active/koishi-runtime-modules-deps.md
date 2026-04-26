---
type: internal
audience: maintainers
status: current
authoritative-source: source scan
last-verified: 2026-04-26
---

# Koishi Runtime Modules Dependency Audit

Scope: `bots/koishi/plugins/stuhelper-core/src/core/modules/`.

Purpose: P4a moves module construction into `src/runtime/registry.ts`. This file records the
current dependency shape for P4b migration planning only; runtime code must not consume it.

Audit method:

- Searched module files for `getModule`, `stuhelperGroupCenter`, `this.data`, `this.config`,
  `ctx.database`, `ctx.http`, `ctx.puppeteer`, event listeners, middleware, timers, and direct
  relative imports.
- Treated `BaseModule` dependencies as common to all adapted modules: `Context`, `DataManager`,
  plugin `Config`, and `ctx.stuhelperGroupCenter.auth` for command permission registration.

## Registry Order

The P4a registry order intentionally matches the P3 `MODULE_CLASSES` order.

| Order | Runtime id | Class | Direct module dependency | Service / data dependency notes |
|------:|------------|-------|--------------------------|---------------------------------|
| 0 | `warn` | `WarnModule` | None | `warns`, `mutes`, group config, `pushMessage`, `warnLimit`, `banTimes` |
| 1 | `keyword` | `KeywordModule` | None | group config, `mutes`, `forbidden` config, middleware |
| 2 | `manage-member` | `MemberManageModule` | None | `blacklist`, `mutes`, `pushMessage`, `setTitle` config |
| 3 | `manage-message` | `MessageManageModule` | None | `setEssenceMsg` config, bot message APIs |
| 4 | `manage-order` | `OrderManageModule` | None | `mutes`, bot moderation APIs |
| 5 | `antirepeat` | `AntirepeatModule` | None | group config, `antiRepeat` config |
| 6 | `welcome` | `WelcomeModule` | None | group config, `defaultWelcome`, `defaultGoodbye`, member events |
| 7 | `repeat` | `RepeatModule` | None | group config, middleware, cleanup timer |
| 8 | `dice` | `DiceModule` | None | group config, dice config |
| 9 | `banme` | `BanmeModule` | None | `banmeRecords`, `mutes`, group config, `banme` config |
| 10 | `antirecall` | `AntiRecallModule` | None | `recallRecords`, group config, `pushMessage`, recall events |
| 11 | `ai` | `AIModule` | None | `ctx.http`, AI context file, group config, OpenAI-compatible config |
| 12 | `config` | `ConfigModule` | None | `auth` service, `warns`, `blacklist`, group config, `mutes` |
| 13 | `log` | `LogModule` | None | command events, middleware, `ctx.database`, log files |
| 14 | `subscription` | `SubscriptionModule` | None | subscriptions store, `mutes`, bot lookup, dispose listener |
| 15 | `help` | `HelpModule` | None | `auth.getCommandsByModule()` and permission checks at command execution time |
| 16 | `report` | `ReportModule` | `ai` | `getModule<AIModule>('ai')`, `ctx.database`, settings, command logs, `pushMessage` |
| 17 | `getauth` | `GetAuthModule` | None | `ctx.database`, `auth` roles, bot member info |
| 18 | `auth` | `AuthModule` | None | `auth` roles and assignments |
| 19 | `event` | `EventModule` | None | request/member events, `blacklist`, `leaveRecords`, group config, `mutes`, `pushMessage` |
| 20 | `status` | `StatusModule` | None | optional `ctx.puppeteer`, registry stats, group config, command logs |
| 21 | `manage-cross-group` | `crossGroupModule` | None | bot group leave/send APIs |

## P4b Constraints

- `report` must continue to load after `ai`; `ReportModule.callModeration()` calls
  `ctx.stuhelperGroupCenter.getModule<AIModule>('ai')`.
- Modules that call `BaseModule.registerCommand()` depend on `StuhelperGroupCenterService.auth`,
  not on `AuthModule`.
- `help` does not need to run after every command module for initialization safety because it reads
  `auth.getCommandsByModule()` when the help command executes, but keeping the current order avoids
  changing observable command grouping behavior.
- `help`, `dice`, `banme`, `config`, `status`, `event`, `antirepeat`, `manage-message`,
  `getauth`, `manage-cross-group`, `auth`, `repeat`, `welcome`, and `log` have been migrated to native
  `RuntimeModule` implementations in P4b-1 through P4b-14; the remaining rows still use
  `BaseModuleAdapter`.
- `log`, `keyword`, `repeat`, `ai`, and `subscription` install listeners, middleware, or timers.
  P4b native modules must preserve their dispose behavior instead of relying on silent fallback
  cleanup. The migrated `event` module preserves listener and interval registration through the
  Koishi plugin context.
