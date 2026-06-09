export interface AdmissionSubjectRef {
  readonly platform: string
  readonly botSelfId: string
  readonly guildId: string
  readonly memberId: string
}

export class AdmissionSubjectCoordinator {
  private readonly cancelledAtBySession = new Map<string, number>()
  private readonly cancelledAtBySubject = new Map<string, number>()
  private readonly tailsBySubject = new Map<string, Promise<unknown>>()

  cancelSubject(ref: AdmissionSubjectRef, now = new Date()) {
    const current = now.getTime()
    this.cancelledAtBySubject.set(admissionSubjectKey(ref), current)
    this.pruneCancelled(current)
  }

  clearSubjectCancellation(ref: AdmissionSubjectRef) {
    this.cancelledAtBySubject.delete(admissionSubjectKey(ref))
  }

  cancel(ref: AdmissionSubjectRef, admissionSessionID: string, now = new Date()) {
    if (!admissionSessionID) return
    const current = now.getTime()
    this.cancelledAtBySession.set(admissionSubjectSessionKey(ref, admissionSessionID), current)
    this.pruneCancelled(current)
  }

  isCancelled(ref: AdmissionSubjectRef, admissionSessionID: string, now = new Date()) {
    if (!admissionSessionID) return false
    if (this.isSubjectCancelled(ref, now)) return true
    const key = admissionSubjectSessionKey(ref, admissionSessionID)
    const cancelledAt = this.cancelledAtBySession.get(key)
    if (cancelledAt === undefined) return false
    const current = now.getTime()
    if (current - cancelledAt <= CANCELLATION_TTL_MS) return true
    this.cancelledAtBySession.delete(key)
    return false
  }

  async runExclusive<T>(ref: AdmissionSubjectRef, run: () => Promise<T>) {
    const key = admissionSubjectKey(ref)
    const previous = this.tailsBySubject.get(key) ?? Promise.resolve()
    let release!: () => void
    const current = new Promise<void>((resolve) => {
      release = resolve
    })
    const tail = previous.then(() => current, () => current)
    this.tailsBySubject.set(key, tail)
    await previous.catch(() => {})
    try {
      return await run()
    } finally {
      release()
      if (this.tailsBySubject.get(key) === tail) {
        this.tailsBySubject.delete(key)
      }
    }
  }

  private pruneCancelled(nowMs: number) {
    for (const [key, cancelledAt] of this.cancelledAtBySubject) {
      if (nowMs - cancelledAt > SUBJECT_CANCELLATION_TTL_MS) {
        this.cancelledAtBySubject.delete(key)
      }
    }
    for (const [key, cancelledAt] of this.cancelledAtBySession) {
      if (nowMs - cancelledAt > CANCELLATION_TTL_MS) {
        this.cancelledAtBySession.delete(key)
      }
    }
  }

  private isSubjectCancelled(ref: AdmissionSubjectRef, now: Date) {
    const key = admissionSubjectKey(ref)
    const cancelledAt = this.cancelledAtBySubject.get(key)
    if (cancelledAt === undefined) return false
    const current = now.getTime()
    if (current - cancelledAt <= SUBJECT_CANCELLATION_TTL_MS) return true
    this.cancelledAtBySubject.delete(key)
    return false
  }
}

export function admissionSubjectKey(ref: AdmissionSubjectRef) {
  return JSON.stringify([ref.platform, ref.botSelfId, ref.guildId, ref.memberId])
}

function admissionSubjectSessionKey(ref: AdmissionSubjectRef, admissionSessionID: string) {
  return `${admissionSubjectKey(ref)}:${admissionSessionID}`
}

const SUBJECT_CANCELLATION_TTL_MS = 60_000
const CANCELLATION_TTL_MS = 10 * 60_000
