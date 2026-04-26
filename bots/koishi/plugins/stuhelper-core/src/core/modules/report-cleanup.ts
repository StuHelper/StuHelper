import type { ReportModule } from './report.module'

const CLEANUP_INTERVAL_MS = 10 * 60 * 1000
const REPORTED_MESSAGE_TTL_MS = 24 * 60 * 60 * 1000

export function setupReportCleanupTask(host: ReportModule): void {
  host.ctx.setInterval(() => cleanupReportState(host, Date.now()), CLEANUP_INTERVAL_MS)
}

function cleanupReportState(host: ReportModule, now: number): void {
  for (const key in host.reportBans) {
    if (host.reportBans[key].expireTime <= now) delete host.reportBans[key]
  }

  for (const key in host.reportedMessages) {
    if (now - host.reportedMessages[key].timestamp > REPORTED_MESSAGE_TTL_MS) {
      delete host.reportedMessages[key]
    }
  }
}
