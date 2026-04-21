import { computed, reactive, ref, watch, onBeforeUnmount, onMounted } from 'vue'

import type { ConsoleNavigationController } from './use-console-navigation'
import { consolePageApi } from '../page-api'
import type { ConfigGovernancePageData } from '../page-types'
import {
  assignBindingForm,
  assignPolicyForm,
  assignTemplateForm,
  splitCommaTokens,
  type BindingFormState,
  type PolicyFormState,
  type TemplateFormState,
} from '../models/config-forms'
import {
  cloneBindingForm,
  clonePolicyForm,
  cloneTemplateForm,
  confirmDiscardChanges,
  isBindingFormDirty,
  isPolicyFormDirty,
  isTemplateFormDirty,
} from '../models/config-editor'
import { buildConfigGovernanceModel } from '../models/config'

type WorkspaceId = 'guild-config' | 'templates' | 'bindings' | 'command-policies'

export function useConfigGovernance(navigation?: ConsoleNavigationController) {
  const loading = ref(false)
  const error = ref('')
  const data = ref<ConfigGovernancePageData | null>(null)
  const currentWorkspace = ref<WorkspaceId>('guild-config')
  const notice = ref('')
  const submittingTemplate = ref(false)
  const submittingBinding = ref(false)
  const submittingPolicy = ref(false)
  const templateSnapshot = ref<TemplateFormState | null>(null)
  const bindingSnapshot = ref<BindingFormState | null>(null)
  const policySnapshot = ref<PolicyFormState | null>(null)
  const templateForm = reactive<TemplateFormState>({
    id: '',
    name: '',
    muteDurationSeconds: 1800,
    kickAfterMinutes: 30,
    reminderTemplate: '',
    exemptUsersText: '',
    enabled: true,
  })
  const bindingForm = reactive<BindingFormState>({
    platform: 'onebot',
    guildId: '',
    templateId: '',
    note: '',
    enabled: true,
  })
  const policyForm = reactive<PolicyFormState>({
    commandId: '',
    minAuthority: 3,
    rolesText: '',
  })

  const configModel = computed(() => data.value ? buildConfigGovernanceModel(data.value, { workspace: currentWorkspace.value }) : null)
  const templateDirty = computed(() => isTemplateFormDirty(templateForm, templateSnapshot.value))
  const bindingDirty = computed(() => isBindingFormDirty(bindingForm, bindingSnapshot.value))
  const policyDirty = computed(() => isPolicyFormDirty(policyForm, policySnapshot.value))
  const currentDirty = computed(() => {
    if (currentWorkspace.value === 'templates') return templateDirty.value
    if (currentWorkspace.value === 'bindings') return bindingDirty.value
    if (currentWorkspace.value === 'command-policies') return policyDirty.value
    return false
  })
  const hasUnsavedChanges = computed(() => templateDirty.value || bindingDirty.value || policyDirty.value)

  watch(
    () => navigation?.state.value.workspace,
    (workspace) => {
      if (workspace === 'templates' || workspace === 'bindings' || workspace === 'command-policies') {
        currentWorkspace.value = workspace
        return
      }
      currentWorkspace.value = 'guild-config'
    },
    { immediate: true },
  )

  onMounted(() => {
    window.addEventListener('beforeunload', handleBeforeUnload)
  })
  onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', handleBeforeUnload)
  })

  void loadData()

  return {
    loading,
    error,
    data,
    currentWorkspace,
    notice,
    submittingTemplate,
    submittingBinding,
    submittingPolicy,
    templateForm,
    bindingForm,
    policyForm,
    configModel,
    currentDirty,
    loadData,
    selectWorkspace,
    loadTemplate,
    loadBinding,
    loadPolicy,
    submitTemplate,
    submitBinding,
    submitPolicy,
  }

  function handleBeforeUnload(event: BeforeUnloadEvent) {
    if (!hasUnsavedChanges.value) return
    event.preventDefault()
    event.returnValue = ''
  }

  async function loadData() {
    loading.value = true
    error.value = ''
    try {
      data.value = await consolePageApi.configGovernance()
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading.value = false
    }
  }

  function selectWorkspace(workspace: WorkspaceId) {
    if (!confirmDiscardChanges(currentDirty.value, window.confirm)) return
    currentWorkspace.value = workspace
    navigation?.replaceState({ workspace })
  }

  function loadTemplate(item: ConfigGovernancePageData['templates'][number]) {
    if (templateDirty.value && templateForm.id !== item.id && !confirmDiscardChanges(true, window.confirm)) return
    assignTemplateForm(templateForm, item)
    templateSnapshot.value = cloneTemplateForm(templateForm)
  }

  function loadBinding(item: ConfigGovernancePageData['bindings'][number]) {
    const isSame = bindingForm.guildId === item.guildId && bindingForm.platform === item.platform
    if (bindingDirty.value && !isSame && !confirmDiscardChanges(true, window.confirm)) return
    assignBindingForm(bindingForm, item)
    bindingSnapshot.value = cloneBindingForm(bindingForm)
  }

  function loadPolicy(item: ConfigGovernancePageData['commandPolicies'][number]) {
    if (policyDirty.value && policyForm.commandId !== item.commandId && !confirmDiscardChanges(true, window.confirm)) return
    assignPolicyForm(policyForm, item)
    policySnapshot.value = clonePolicyForm(policyForm)
  }

  async function submitTemplate() {
    submittingTemplate.value = true
    await submit(async () => {
      notice.value = await consolePageApi.saveGuardTemplate({
        id: templateForm.id,
        name: templateForm.name,
        muteDurationSeconds: Number(templateForm.muteDurationSeconds),
        kickAfterMinutes: Number(templateForm.kickAfterMinutes),
        reminderTemplate: templateForm.reminderTemplate,
        exemptUsers: splitCommaTokens(templateForm.exemptUsersText),
        enabled: templateForm.enabled,
      })
      await loadData()
      templateSnapshot.value = cloneTemplateForm(templateForm)
    }, () => {
      submittingTemplate.value = false
    })
  }

  async function submitBinding() {
    submittingBinding.value = true
    await submit(async () => {
      notice.value = await consolePageApi.saveGuardBinding({
        platform: bindingForm.platform,
        guildId: bindingForm.guildId,
        templateId: bindingForm.templateId,
        enabled: bindingForm.enabled,
        note: bindingForm.note || undefined,
      })
      await loadData()
      bindingSnapshot.value = cloneBindingForm(bindingForm)
    }, () => {
      submittingBinding.value = false
    })
  }

  async function submitPolicy() {
    submittingPolicy.value = true
    await submit(async () => {
      notice.value = await consolePageApi.saveCommandPolicy({
        commandId: policyForm.commandId,
        minAuthority: Number(policyForm.minAuthority),
        roles: splitCommaTokens(policyForm.rolesText),
      })
      await loadData()
      policySnapshot.value = clonePolicyForm(policyForm)
    }, () => {
      submittingPolicy.value = false
    })
  }

  async function submit(action: () => Promise<void>, done: () => void) {
    notice.value = ''
    error.value = ''
    try {
      await action()
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : String(cause)
    } finally {
      done()
    }
  }
}
