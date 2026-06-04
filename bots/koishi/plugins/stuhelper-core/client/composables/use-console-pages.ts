import { computed } from 'vue'

import type { ConsoleViewId } from '../models/views'
import DashboardHubView from '../components/DashboardHubView.vue'
import AdmissionView from '../components/AdmissionView.vue'
import ConfigCenterView from '../components/ConfigCenterView.vue'
import WarnsView from '../components/WarnsView.vue'
import BlacklistView from '../components/BlacklistView.vue'
import IdentityView from '../components/IdentityView.vue'
import ReviewView from '../components/ReviewView.vue'
import RolesView from '../components/RolesView.vue'
import LogsView from '../components/LogsView.vue'
import ChatView from '../components/ChatView.vue'
import SubscriptionView from '../components/SubscriptionView.vue'
import SettingsView from '../components/SettingsView.vue'
import SystemView from '../components/SystemView.vue'

const VIEW_COMPONENTS = {
  dashboard: DashboardHubView,
  admission: AdmissionView,
  config: ConfigCenterView,
  warns: WarnsView,
  blacklist: BlacklistView,
  identity: IdentityView,
  review: ReviewView,
  roles: RolesView,
  logs: LogsView,
  chat: ChatView,
  subscriptions: SubscriptionView,
  settings: SettingsView,
  system: SystemView,
} as const

export function useConsolePages() {
  const resolve = (view: ConsoleViewId) => VIEW_COMPONENTS[view]

  return {
    views: computed(() => VIEW_COMPONENTS),
    resolve,
  }
}
