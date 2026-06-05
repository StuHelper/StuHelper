import { onBeforeUnmount, onMounted, reactive } from 'vue'

import { consolePageApi } from '../page-api'
import { errorMessage } from '../utils/error-message'

export interface PulseState {
  pendingReviews: number
  pendingAdmissions: number
  openReports: number
  highRiskEvents: number
  policyItems: number
  generatedAt: string | null
  loading: boolean
  error: string
}

const POLL_INTERVAL_MS = 30_000

export interface PulseHandle {
  readonly state: PulseState
  refresh(): Promise<void>
  stop(): void
}

export function usePulse(): PulseHandle {
  const state = reactive<PulseState>({
    pendingReviews: 0,
    pendingAdmissions: 0,
    openReports: 0,
    highRiskEvents: 0,
    policyItems: 0,
    generatedAt: null,
    loading: false,
    error: '',
  })

  let timer: ReturnType<typeof setInterval> | null = null

  async function refresh(): Promise<void> {
    state.loading = true
    try {
      const data = await consolePageApi.dashboard()
      state.pendingReviews = data.overview.pendingReviews
      state.pendingAdmissions = data.overview.pendingAdmissions
      state.openReports = data.overview.openReports
      state.highRiskEvents = data.overview.highRiskEvents
      state.policyItems = data.overview.policyItems
      state.generatedAt = data.generatedAt
      state.error = ''
    } catch (cause) {
      state.error = errorMessage(cause, '脉冲数据加载失败')
    } finally {
      state.loading = false
    }
  }

  function stop() {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  onMounted(() => {
    void refresh()
    timer = setInterval(() => void refresh(), POLL_INTERVAL_MS)
  })

  onBeforeUnmount(stop)

  return { state, refresh, stop }
}
