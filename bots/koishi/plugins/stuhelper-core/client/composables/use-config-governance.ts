import { computed, reactive, ref, watch, onBeforeUnmount, onMounted } from 'vue'

import type { ConsoleNavigationController } from './use-console-navigation'
import { consolePageApi } from '../page-api'
import type { ConfigGovernancePageData } from '../page-types'
import {
  assignBindingForm,
  assignBindingFormState,
  assignPolicyForm,
  assignPolicyFormState,
  assignTemplateForm,
  assignTemplateFormState,
  createBindingForm,
  createPolicyForm,
  createTemplateForm,
  splitCommaTokens,
  validateBindingForm,
  validatePolicyForm,
  validateTemplateForm,
  type BindingFormState,
  type PolicyFormState,
  type TemplateFormState,
} from '../models/config-forms'
import {
  cloneBindingForm,
  clonePolicyForm,
  cloneTemplateForm,
  DISCARD_CHANGES_MESSAGE,
  isBindingFormDirty,
  isPolicyFormDirty,
  isTemplateFormDirty,
} from '../models/config-editor'
import { buildConfigGovernanceModel } from '../models/config'
import { useActionError } from './use-action-error'

type WorkspaceId = 'guild-config' | 'templates' | 'bindings' | 'command-policies'
type ConfirmDiscardChanges = (message: string) => boolean | Promise<boolean>

export function useConfigGovernance(
  navigation: ConsoleNavigationController | undefined,
  confirmDiscardChanges: ConfirmDiscardChanges,
) {
  const loading = ref(false)
  const error = ref('')
  const {
    actionError,
    actionErrorTitle,
    setActionError,
    clearActionError,
    errorMessage,
  } = useActionError()
  const data = ref<ConfigGovernancePageData | null>(null)
  const currentWorkspace = ref<WorkspaceId>('guild-config')
  const notice = ref('')
  const confirmingDiscard = ref(false)
  const submittingTemplate = ref(false)
  const submittingBinding = ref(false)
  const submittingPolicy = ref(false)
  const templateSnapshot = ref<TemplateFormState | null>(createTemplateForm())
  const bindingSnapshot = ref<BindingFormState | null>(createBindingForm())
  const policySnapshot = ref<PolicyFormState | null>(createPolicyForm())
  const templateForm = reactive<TemplateFormState>(createTemplateForm())
  const bindingForm = reactive<BindingFormState>(createBindingForm())
  const policyForm = reactive<PolicyFormState>(createPolicyForm())
  let loadRequestSeq = 0

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
    actionError,
    actionErrorTitle,
    data,
    currentWorkspace,
    notice,
    confirmingDiscard,
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
    clearActionError,
  }

  function handleBeforeUnload(event: BeforeUnloadEvent) {
    if (!hasUnsavedChanges.value) return
    event.preventDefault()
    event.returnValue = ''
  }

  async function loadData() {
    const requestSeq = ++loadRequestSeq
    loading.value = true
    error.value = ''
    clearActionError()
    try {
      const next = await consolePageApi.configGovernance()
      if (requestSeq !== loadRequestSeq) return
      data.value = next
    } catch (cause) {
      if (requestSeq !== loadRequestSeq) return
      error.value = errorMessage(cause, '加载配置治理数据失败')
    } finally {
      if (requestSeq === loadRequestSeq) {
        loading.value = false
      }
    }
  }

  async function selectWorkspace(workspace: WorkspaceId) {
    if (workspace === currentWorkspace.value) return
    if (!(await confirmCurrentDiscard())) return
    currentWorkspace.value = workspace
    navigation?.replaceState({ workspace })
  }

  async function confirmCurrentDiscard() {
    if (!currentDirty.value) return true
    if (!(await requestDiscardConfirmation())) return false
    discardCurrentChanges()
    return true
  }

  async function requestDiscardConfirmation() {
    if (confirmingDiscard.value) return false
    confirmingDiscard.value = true
    try {
      return await confirmDiscardChanges(DISCARD_CHANGES_MESSAGE)
    } finally {
      confirmingDiscard.value = false
    }
  }

  function discardCurrentChanges() {
    if (currentWorkspace.value === 'templates') restoreTemplateForm()
    if (currentWorkspace.value === 'bindings') restoreBindingForm()
    if (currentWorkspace.value === 'command-policies') restorePolicyForm()
  }

  function restoreTemplateForm() {
    if (!templateSnapshot.value) return
    assignTemplateFormState(templateForm, templateSnapshot.value)
  }

  function restoreBindingForm() {
    if (!bindingSnapshot.value) return
    assignBindingFormState(bindingForm, bindingSnapshot.value)
  }

  function restorePolicyForm() {
    if (!policySnapshot.value) return
    assignPolicyFormState(policyForm, policySnapshot.value)
  }

  async function loadTemplate(item: ConfigGovernancePageData['templates'][number]) {
    if (templateDirty.value && templateForm.id !== item.id && !(await requestDiscardConfirmation())) return
    assignTemplateForm(templateForm, item)
    templateSnapshot.value = cloneTemplateForm(templateForm)
  }

  async function loadBinding(item: ConfigGovernancePageData['bindings'][number]) {
    const isSame = bindingForm.guildId === item.guildId && bindingForm.platform === item.platform
    if (bindingDirty.value && !isSame && !(await requestDiscardConfirmation())) return
    assignBindingForm(bindingForm, item)
    bindingSnapshot.value = cloneBindingForm(bindingForm)
  }

  async function loadPolicy(item: ConfigGovernancePageData['commandPolicies'][number]) {
    if (policyDirty.value && policyForm.commandId !== item.commandId && !(await requestDiscardConfirmation())) return
    assignPolicyForm(policyForm, item)
    policySnapshot.value = clonePolicyForm(policyForm)
  }

  async function submitTemplate() {
    if (!validateBeforeSubmit('保存模板失败', validateTemplateForm(templateForm))) return
    submittingTemplate.value = true
    await submit('保存模板失败', async () => {
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
    if (!validateBeforeSubmit('保存绑定失败', validateBindingForm(bindingForm))) return
    submittingBinding.value = true
    await submit('保存绑定失败', async () => {
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
    if (!validateBeforeSubmit('保存策略失败', validatePolicyForm(policyForm))) return
    submittingPolicy.value = true
    await submit('保存策略失败', async () => {
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

  async function submit(title: string, action: () => Promise<void>, done: () => void) {
    notice.value = ''
    clearActionError()
    try {
      await action()
    } catch (cause) {
      setActionError(title, cause, title)
    } finally {
      done()
    }
  }

  function validateBeforeSubmit(title: string, validationError: string) {
    notice.value = ''
    clearActionError()
    if (!validationError) return true
    setActionError(title, validationError, title)
    return false
  }

}
