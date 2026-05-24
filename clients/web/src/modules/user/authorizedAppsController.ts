import { computed, reactive, ref } from 'vue'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import type {
  OpenPlatformUserAuthorizedApp,
  OpenPlatformUserConsentAuditEvent,
  OpenPlatformUserConsentScope,
} from '@stuhelper/shared/api'

type Translate = (key: string, params?: Record<string, unknown>) => string
type RevokeDialogMode = 'app' | 'scope'

interface RevokeDialogState {
  open: boolean
  mode: RevokeDialogMode | null
  app: OpenPlatformUserAuthorizedApp | null
  scope: OpenPlatformUserConsentScope | null
  error: string
}

export function useAuthorizedAppsController(t: Translate) {
  const toast = useToast()

  const apps = ref<OpenPlatformUserAuthorizedApp[]>([])
  const activities = ref<OpenPlatformUserConsentAuditEvent[]>([])
  const loading = ref(true)
  const activityLoading = ref(true)
  const errorMessage = ref('')
  const activityErrorMessage = ref('')
  const revokingKey = ref('')
  const revokeDialog = reactive<RevokeDialogState>({
    open: false,
    mode: null,
    app: null,
    scope: null,
    error: '',
  })

  const revokeDialogTitle = computed(() => {
    if (revokeDialog.mode === 'scope') {
      return t('user.authorizedApps.revokeScopeDialogTitle', {
        scope: revokeDialog.scope?.displayName ?? '',
      })
    }
    return t('user.authorizedApps.revokeAppDialogTitle')
  })

  const revokeDialogDescription = computed(() => {
    if (revokeDialog.mode === 'scope') {
      return t('user.authorizedApps.revokeScopeDialogDescription', {
        app: revokeDialog.app?.app.displayName ?? '',
        scope: revokeDialog.scope?.displayName ?? '',
      })
    }
    return t('user.authorizedApps.revokeAppDialogDescription', {
      app: revokeDialog.app?.app.displayName ?? '',
    })
  })

  const revokeDialogSubmitting = computed(() => {
    if (!revokeDialog.app) return false
    if (revokeDialog.mode === 'scope' && revokeDialog.scope) {
      return revokingKey.value === scopeRevokeKey(revokeDialog.app.app.id, revokeDialog.scope.scope)
    }
    return revokingKey.value === appRevokeKey(revokeDialog.app.app.id)
  })

  async function loadConsents() {
    loading.value = true
    errorMessage.value = ''
    try {
      const response = await api.openPlatform.listConsents()
      apps.value = response.data?.data?.apps ?? []
    } catch (err) {
      errorMessage.value = getErrorMessage(err, t('common.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  async function loadActivities() {
    activityLoading.value = true
    activityErrorMessage.value = ''
    try {
      const response = await api.openPlatform.listConsentAuditEvents({ pageSize: 10 })
      activities.value = response.data?.data?.list ?? []
    } catch (err) {
      activityErrorMessage.value = getErrorMessage(err, t('user.authorizedApps.activityLoadFailed'))
    } finally {
      activityLoading.value = false
    }
  }

  function openRevokeAppDialog(authorizedApp: OpenPlatformUserAuthorizedApp) {
    revokeDialog.open = true
    revokeDialog.mode = 'app'
    revokeDialog.app = authorizedApp
    revokeDialog.scope = null
    revokeDialog.error = ''
  }

  function openRevokeScopeDialog(
    authorizedApp: OpenPlatformUserAuthorizedApp,
    scope: OpenPlatformUserConsentScope,
  ) {
    revokeDialog.open = true
    revokeDialog.mode = 'scope'
    revokeDialog.app = authorizedApp
    revokeDialog.scope = scope
    revokeDialog.error = ''
  }

  function closeRevokeDialog() {
    if (revokeDialogSubmitting.value) return
    revokeDialog.open = false
    revokeDialog.mode = null
    revokeDialog.app = null
    revokeDialog.scope = null
    revokeDialog.error = ''
  }

  async function submitRevokeDialog() {
    if (!revokeDialog.app || !revokeDialog.mode) return
    let succeeded = false
    if (revokeDialog.mode === 'scope' && revokeDialog.scope) {
      succeeded = await revokeScope(revokeDialog.app, revokeDialog.scope)
    } else {
      succeeded = await revokeApp(revokeDialog.app)
    }
    if (succeeded) {
      closeRevokeDialog()
    }
  }

  async function revokeApp(authorizedApp: OpenPlatformUserAuthorizedApp) {
    const key = appRevokeKey(authorizedApp.app.id)
    revokingKey.value = key
    try {
      await api.openPlatform.revokeConsent({ appID: authorizedApp.app.id })
      apps.value = apps.value.filter(item => item.app.id !== authorizedApp.app.id)
      await loadActivities()
      toast.success(t('user.authorizedApps.revokeSuccess'))
      return true
    } catch (err) {
      revokeDialog.error = getErrorMessage(err, t('user.authorizedApps.revokeFailed'))
      toast.error(revokeDialog.error)
      return false
    } finally {
      if (revokingKey.value === key) revokingKey.value = ''
    }
  }

  async function revokeScope(
    authorizedApp: OpenPlatformUserAuthorizedApp,
    scope: OpenPlatformUserConsentScope,
  ) {
    const key = scopeRevokeKey(authorizedApp.app.id, scope.scope)
    revokingKey.value = key
    try {
      await api.openPlatform.revokeConsent({
        appID: authorizedApp.app.id,
        scopes: [scope.scope],
      })
      apps.value = apps.value
        .map((item) => {
          if (item.app.id !== authorizedApp.app.id) return item
          return {
            ...item,
            scopes: item.scopes.filter(consentScope => consentScope.scope !== scope.scope),
          }
        })
        .filter(item => item.scopes.length > 0)
      await loadActivities()
      toast.success(t('user.authorizedApps.revokeSuccess'))
      return true
    } catch (err) {
      revokeDialog.error = getErrorMessage(err, t('user.authorizedApps.revokeFailed'))
      toast.error(revokeDialog.error)
      return false
    } finally {
      if (revokingKey.value === key) revokingKey.value = ''
    }
  }

  return {
    apps,
    activities,
    loading,
    activityLoading,
    errorMessage,
    activityErrorMessage,
    revokingKey,
    revokeDialog,
    revokeDialogTitle,
    revokeDialogDescription,
    revokeDialogSubmitting,
    loadConsents,
    loadActivities,
    openRevokeAppDialog,
    openRevokeScopeDialog,
    closeRevokeDialog,
    submitRevokeDialog,
    revokeApp,
    revokeScope,
    appRevokeKey,
    scopeRevokeKey,
  }
}

function appRevokeKey(appID: number) {
  return `app:${appID}`
}

function scopeRevokeKey(appID: number, scope: string) {
  return `scope:${appID}:${scope}`
}
