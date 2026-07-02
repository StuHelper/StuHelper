/** 入群认证提醒/管理命令的重复抑制窗口（毫秒），全插件共用同一窗口。 */
export const ADMISSION_DEDUPE_WINDOW_MS = 30_000

export class AdmissionReminderDeduper {
  private readonly sentAtBySession = new Map<string, number>()

  constructor(private readonly windowMs = ADMISSION_DEDUPE_WINDOW_MS) {}

  remember(sessionID: string, now = new Date()) {
    if (!sessionID) return
    const current = now.getTime()
    this.sentAtBySession.set(sessionID, current)
    this.prune(current)
  }

  forget(sessionID: string) {
    this.sentAtBySession.delete(sessionID)
  }

  wasRecentlySent(sessionID: string, now = new Date()) {
    const sentAt = this.sentAtBySession.get(sessionID)
    if (sentAt === undefined) return false
    const current = now.getTime()
    if (current - sentAt <= this.windowMs) return true
    this.sentAtBySession.delete(sessionID)
    return false
  }

  private prune(nowMs: number) {
    for (const [sessionID, sentAt] of this.sentAtBySession) {
      if (nowMs - sentAt > this.windowMs) {
        this.sentAtBySession.delete(sessionID)
      }
    }
  }
}
