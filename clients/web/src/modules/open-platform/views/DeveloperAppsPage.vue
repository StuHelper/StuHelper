<template>
  <main class="mx-auto max-w-[1120px] p-6 animate-fade-in max-sm:p-4">
    <header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div class="min-w-0">
        <p class="m-0 text-xs font-semibold uppercase tracking-[0.12em] text-primary">
          {{ t('developer.apps.eyebrow') }}
        </p>
        <h1 class="m-0 mt-2 text-2xl font-bold text-text-primary">
          {{ t('developer.apps.title') }}
        </h1>
      </div>

      <button
        type="button"
        class="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border bg-bg-card px-3 text-sm font-medium text-text-secondary transition-colors duration-fast hover:border-primary/40 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50"
        :disabled="loading"
        @click="loadApps"
      >
        <RefreshCw class="h-4 w-4" :class="loading && 'animate-spin'" aria-hidden="true" />
        {{ t('developer.apps.refresh') }}
      </button>
    </header>

    <section
      v-if="rotatedSecret"
      class="mb-6 rounded-lg border border-success/30 bg-success/10 p-4 text-sm text-text-secondary"
    >
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div class="min-w-0">
          <h2 class="m-0 text-sm font-semibold text-success">
            {{ t('developer.apps.secretRotated', { app: rotatedSecret.appName }) }}
          </h2>
          <p class="m-0 mt-1 break-all text-xs text-text-muted">
            {{ rotatedSecret.clientID }}
          </p>
        </div>
        <button
          type="button"
          class="inline-flex h-8 shrink-0 items-center justify-center rounded-md px-2 text-xs font-medium text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary"
          @click="rotatedSecret = null"
        >
          {{ t('common.actions.close') }}
        </button>
      </div>
      <code class="mt-3 block break-all rounded-md bg-bg-base px-3 py-2 text-xs text-success">
        {{ rotatedSecret.secret }}
      </code>
    </section>

    <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_390px] lg:items-start">
      <section class="min-w-0">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 items-center gap-3">
            <h2 class="m-0 text-base font-semibold text-text-primary">
              {{ t('developer.apps.listTitle') }}
            </h2>
            <span class="text-sm text-text-muted">
              {{ t('developer.apps.total', { total }) }}
            </span>
          </div>

          <label class="flex items-center gap-2 text-sm text-text-secondary">
            <span class="shrink-0">{{ t('developer.apps.statusFilter') }}</span>
            <select
              v-model="statusFilter"
              class="h-9 rounded-md border border-border bg-bg-card px-2 text-sm text-text-primary outline-none transition-colors focus:border-primary/60 focus:ring-2 focus:ring-primary/15"
            >
              <option v-for="status in statusOptions" :key="status" :value="status">
                {{ statusLabel(status) }}
              </option>
            </select>
          </label>
        </div>

        <div v-if="loading" class="grid gap-4">
          <SkeletonCard v-for="i in 3" :key="i" variant="course" />
        </div>

        <EmptyState
          v-else-if="errorMessage"
          :title="t('common.loadFailed')"
          :description="errorMessage"
        >
          <template #action>
            <button
              type="button"
              class="inline-flex items-center justify-center gap-2 rounded-md bg-text-primary px-4 py-2 text-sm font-medium text-bg-base transition-colors duration-fast hover:bg-accent hover:text-white"
              @click="loadApps"
            >
              <RefreshCw class="h-4 w-4" aria-hidden="true" />
              {{ t('common.actions.retry') }}
            </button>
          </template>
        </EmptyState>

        <EmptyState
          v-else-if="apps.length === 0"
          :title="t('developer.apps.empty')"
          :description="t('developer.apps.emptyDesc')"
        >
          <template #icon>
            <KeyRound class="h-full w-full" aria-hidden="true" />
          </template>
        </EmptyState>

        <div v-else class="grid gap-4">
          <article
            v-for="item in apps"
            :key="item.app.id"
            class="rounded-lg border border-border bg-bg-card p-5 shadow-sm"
          >
            <header class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <div class="flex min-w-0 items-start gap-3">
                  <div class="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-primary-alpha text-primary">
                    <KeyRound class="h-5 w-5" aria-hidden="true" />
                  </div>
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <h3 class="m-0 max-w-full truncate text-base font-semibold text-text-primary">
                        {{ item.app.displayName }}
                      </h3>
                      <span
                        class="rounded-full px-2 py-0.5 text-xs font-medium"
                        :class="appStatusClass(item.app.status)"
                      >
                        {{ statusLabel(item.app.status) }}
                      </span>
                    </div>
                    <p class="m-0 mt-1 break-all text-xs text-text-muted">
                      {{ t('developer.apps.clientID') }}: {{ item.app.clientID }}
                    </p>
                  </div>
                </div>

                <p class="mb-0 mt-3 text-sm leading-relaxed text-text-secondary">
                  {{ item.app.description || t('developer.apps.noDescription') }}
                </p>
              </div>

              <div class="flex shrink-0 flex-wrap gap-2">
                <button
                  type="button"
                  class="developer-action"
                  :disabled="auditLoading && auditPanelAppID === item.app.id"
                  @click="toggleAuditPanel(item)"
                >
                  <Activity
                    class="h-3.5 w-3.5"
                    :class="auditLoading && auditPanelAppID === item.app.id && 'animate-pulse'"
                    aria-hidden="true"
                  />
                  {{ auditPanelAppID === item.app.id ? t('developer.apps.hideAudit') : t('developer.apps.viewAudit') }}
                </button>
                <button
                  v-if="canUpdateProfile(item)"
                  type="button"
                  class="developer-action"
                  :disabled="profileSubmittingAppID === item.app.id"
                  @click="startProfileEdit(item)"
                >
                  <PencilLine class="h-3.5 w-3.5" aria-hidden="true" />
                  {{ t('developer.apps.editProfile') }}
                </button>
                <button
                  v-if="canRequestScopeChange(item)"
                  type="button"
                  class="developer-action"
                  :disabled="scopeSubmittingAppID === item.app.id"
                  @click="startScopeChange(item)"
                >
                  <Plus class="h-3.5 w-3.5" aria-hidden="true" />
                  {{ t('developer.apps.requestScopes') }}
                </button>
                <button
                  v-if="canWithdrawApp(item)"
                  type="button"
                  class="developer-action"
                  :disabled="withdrawingKey === appWithdrawKey(item.app.id)"
                  @click="openWithdrawAppDialog(item)"
                >
                  <Undo2 class="h-3.5 w-3.5" aria-hidden="true" />
                  {{ t('developer.apps.withdrawApp') }}
                </button>
                <button
                  v-if="item.app.status === 'approved'"
                  type="button"
                  class="developer-action"
                  :disabled="rotatingAppID === item.app.id"
                  @click="openRotateSecretDialog(item)"
                >
                  <RefreshCw
                    class="h-3.5 w-3.5"
                    :class="rotatingAppID === item.app.id && 'animate-spin'"
                    aria-hidden="true"
                  />
                  {{ t('developer.apps.rotateSecret') }}
                </button>
                <button
                  v-if="canRequestRedirectChange(item)"
                  type="button"
                  class="developer-action"
                  :disabled="redirectSubmittingAppID === item.app.id"
                  @click="startRedirectChange(item)"
                >
                  <PencilLine class="h-3.5 w-3.5" aria-hidden="true" />
                  {{ t('developer.apps.changeRedirects') }}
                </button>
                <a
                  class="developer-link"
                  :href="item.app.homepageURL"
                  target="_blank"
                  rel="noreferrer"
                >
                  {{ t('developer.apps.homepage') }}
                  <ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
                </a>
                <a
                  class="developer-link"
                  :href="item.app.privacyPolicyURL"
                  target="_blank"
                  rel="noreferrer"
                >
                  {{ t('developer.apps.privacy') }}
                  <ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
                </a>
              </div>
            </header>

            <dl class="mt-4 grid gap-3 border-t border-border pt-4 text-sm sm:grid-cols-2">
              <div class="min-w-0">
                <dt class="text-xs text-text-muted">{{ t('developer.apps.createdAt') }}</dt>
                <dd class="m-0 mt-1 text-text-secondary">{{ formatDate(item.app.createdAt) }}</dd>
              </div>
              <div class="min-w-0">
                <dt class="text-xs text-text-muted">{{ t('developer.apps.updatedAt') }}</dt>
                <dd class="m-0 mt-1 text-text-secondary">{{ formatDate(item.app.updatedAt) }}</dd>
              </div>
            </dl>

            <form
              v-if="profileEditingAppID === item.app.id"
              class="mt-4 grid gap-3 rounded-md border border-border bg-bg-elevated p-3"
              @submit.prevent="submitProfileUpdate(item)"
            >
              <h4 class="m-0 text-sm font-semibold text-text-primary">
                {{ t('developer.apps.profileEditTitle') }}
              </h4>

              <label class="grid gap-1.5">
                <span class="text-sm font-medium text-text-secondary">
                  {{ t('developer.apps.fields.displayName') }}
                </span>
                <input
                  v-model.trim="profileForm.displayName"
                  class="developer-input"
                  type="text"
                  autocomplete="off"
                />
              </label>

              <label class="grid gap-1.5">
                <span class="text-sm font-medium text-text-secondary">
                  {{ t('developer.apps.fields.description') }}
                </span>
                <textarea
                  v-model.trim="profileForm.description"
                  class="developer-input min-h-20 resize-y py-2"
                  rows="3"
                />
              </label>

              <div class="grid gap-3 sm:grid-cols-2">
                <label class="grid gap-1.5">
                  <span class="text-sm font-medium text-text-secondary">
                    {{ t('developer.apps.fields.homepageURL') }}
                  </span>
                  <input
                    v-model.trim="profileForm.homepageURL"
                    class="developer-input"
                    type="url"
                    inputmode="url"
                    placeholder="https://"
                  />
                </label>

                <label class="grid gap-1.5">
                  <span class="text-sm font-medium text-text-secondary">
                    {{ t('developer.apps.fields.privacyPolicyURL') }}
                  </span>
                  <input
                    v-model.trim="profileForm.privacyPolicyURL"
                    class="developer-input"
                    type="url"
                    inputmode="url"
                    placeholder="https://"
                  />
                </label>
              </div>

              <textarea
                v-model.trim="profileForm.reason"
                class="developer-input min-h-16 resize-y py-2"
                rows="2"
                :aria-label="t('developer.apps.profileReasonLabel')"
              />

              <p
                v-if="profileForm.error"
                class="m-0 rounded-md bg-danger/10 px-3 py-2 text-sm text-danger"
              >
                {{ profileForm.error }}
              </p>

              <div class="flex flex-wrap gap-2">
                <button
                  type="submit"
                  class="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-text-primary px-3 text-sm font-semibold text-bg-base transition-colors hover:bg-primary hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="profileSubmittingAppID === item.app.id"
                >
                  <Send class="h-4 w-4" aria-hidden="true" />
                  {{ t('developer.apps.updateProfile') }}
                </button>
                <button
                  type="button"
                  class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
                  @click="cancelProfileEdit"
                >
                  <X class="h-4 w-4" aria-hidden="true" />
                  {{ t('common.actions.cancel') }}
                </button>
              </div>
            </form>

            <section class="mt-4">
              <h4 class="m-0 text-sm font-semibold text-text-primary">
                {{ t('developer.apps.redirectURIs') }}
              </h4>
              <ul class="m-0 mt-2 grid list-none gap-2 p-0">
                <li
                  v-for="uri in item.app.redirectURIs"
                  :key="uri"
                  class="break-all rounded-md bg-bg-elevated px-3 py-2 text-xs text-text-secondary"
                >
                  {{ uri }}
                </li>
              </ul>
              <div
                v-if="pendingRedirectURIRequests(item).length > 0"
                class="mt-3 grid gap-2 rounded-md border border-warning/30 bg-warning/10 p-3"
              >
                <div
                  v-for="request in pendingRedirectURIRequests(item)"
                  :key="request.id"
                  class="grid gap-2"
                >
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <div class="flex flex-wrap items-center gap-2">
                      <span
                        class="rounded-full bg-warning/10 px-2 py-0.5 text-xs font-medium text-warning"
                      >
                        {{ redirectRequestStatusLabel(request.status) }}
                      </span>
                      <span class="text-xs text-text-muted">
                        {{ formatDate(request.updatedAt) }}
                      </span>
                    </div>
                    <button
                      type="button"
                      class="inline-flex h-7 items-center justify-center gap-1 rounded-md border border-warning/30 px-2 text-xs font-medium text-warning transition-colors hover:bg-warning/10 disabled:cursor-not-allowed disabled:opacity-50"
                      :disabled="withdrawingKey === redirectWithdrawKey(item.app.id, request.id)"
                      @click="openWithdrawRedirectDialog(item, request)"
                    >
                      <Undo2 class="h-3.5 w-3.5" aria-hidden="true" />
                      {{ t('developer.apps.withdrawRequest') }}
                    </button>
                  </div>
                  <ul class="m-0 grid list-none gap-1 p-0">
                    <li
                      v-for="uri in request.redirectURIs"
                      :key="`${request.id}-${uri}`"
                      class="break-all text-xs text-text-secondary"
                    >
                      {{ uri }}
                    </li>
                  </ul>
                  <p v-if="request.reason" class="m-0 text-xs text-text-muted">
                    {{ request.reason }}
                  </p>
                </div>
              </div>

              <form
                v-if="redirectEditingAppID === item.app.id"
                class="mt-3 grid gap-3 rounded-md border border-border bg-bg-elevated p-3"
                @submit.prevent="submitRedirectChange(item)"
              >
                <div
                  v-for="(_, index) in redirectChangeForm.redirectURIs"
                  :key="index"
                  class="flex gap-2"
                >
                  <input
                    v-model.trim="redirectChangeForm.redirectURIs[index]"
                    class="developer-input min-w-0 flex-1"
                    type="url"
                    inputmode="url"
                    placeholder="https://"
                    :aria-label="t('developer.apps.newRedirectLabel', { index: index + 1 })"
                  />
                  <button
                    type="button"
                    class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md border border-border text-text-secondary transition-colors hover:border-danger/40 hover:text-danger disabled:cursor-not-allowed disabled:opacity-40"
                    :aria-label="t('developer.apps.removeRedirect')"
                    :disabled="redirectChangeForm.redirectURIs.length === 1"
                    @click="removeRedirectChangeURI(index)"
                  >
                    <Trash2 class="h-4 w-4" aria-hidden="true" />
                  </button>
                </div>
                <button
                  type="button"
                  class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium text-text-secondary transition-colors hover:border-primary/40 hover:text-primary"
                  @click="addRedirectChangeURI"
                >
                  <Plus class="h-4 w-4" aria-hidden="true" />
                  {{ t('developer.apps.addRedirect') }}
                </button>
                <textarea
                  v-model.trim="redirectChangeForm.reason"
                  class="developer-input min-h-16 resize-y py-2"
                  rows="2"
                  :aria-label="t('developer.apps.redirectChangeReasonLabel')"
                />
                <p
                  v-if="redirectChangeForm.error"
                  class="m-0 rounded-md bg-danger/10 px-3 py-2 text-sm text-danger"
                >
                  {{ redirectChangeForm.error }}
                </p>
                <div class="flex flex-wrap gap-2">
                  <button
                    type="submit"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-text-primary px-3 text-sm font-semibold text-bg-base transition-colors hover:bg-primary hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                    :disabled="redirectSubmittingAppID === item.app.id"
                  >
                    <Send class="h-4 w-4" aria-hidden="true" />
                    {{ t('developer.apps.submitRedirectChange') }}
                  </button>
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
                    @click="cancelRedirectChange"
                  >
                    <X class="h-4 w-4" aria-hidden="true" />
                    {{ t('common.actions.cancel') }}
                  </button>
                </div>
              </form>
            </section>

            <section class="mt-4">
              <h4 class="m-0 text-sm font-semibold text-text-primary">
                {{ t('developer.apps.scopeRequests') }}
              </h4>
              <ul class="m-0 mt-2 divide-y divide-border rounded-md border border-border p-0">
                <li
                  v-for="scope in item.scopes"
                  :key="scope.scope"
                  class="grid gap-2 p-3 sm:grid-cols-[1fr_auto] sm:items-start"
                >
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="text-sm font-medium text-text-primary">{{ scope.displayName }}</span>
                      <span
                        class="rounded-full px-2 py-0.5 text-xs font-medium"
                        :class="scopeStatusClass(scope.status)"
                      >
                        {{ scopeStatusLabel(scope.status) }}
                      </span>
                      <span
                        class="rounded-full px-2 py-0.5 text-xs font-medium"
                        :class="sensitivityClass(scope.sensitivity)"
                      >
                        {{ sensitivityLabel(scope.sensitivity) }}
                      </span>
                    </div>
                    <p class="mb-0 mt-1 break-all text-xs text-text-muted">{{ scope.scope }}</p>
                    <p v-if="scope.reason" class="mb-0 mt-2 text-sm text-text-secondary">
                      {{ scope.reason }}
                    </p>
                    <p v-if="scope.decisionNote" class="mb-0 mt-2 text-xs text-text-muted">
                      {{ scope.decisionNote }}
                    </p>
                  </div>
                  <div class="flex flex-wrap items-center gap-2 sm:justify-end">
                    <span class="text-xs text-text-muted">
                      {{ formatDate(scope.updatedAt) }}
                    </span>
                    <button
                      v-if="scope.status === 'pending'"
                      type="button"
                      class="inline-flex h-7 items-center justify-center gap-1 rounded-md border border-warning/30 px-2 text-xs font-medium text-warning transition-colors hover:bg-warning/10 disabled:cursor-not-allowed disabled:opacity-50"
                      :disabled="withdrawingKey === scopeWithdrawKey(item.app.id, scope.scope)"
                      @click="openWithdrawScopeDialog(item, scope)"
                    >
                      <Undo2 class="h-3.5 w-3.5" aria-hidden="true" />
                      {{ t('developer.apps.withdrawRequest') }}
                    </button>
                  </div>
                </li>
              </ul>

              <form
                v-if="scopeEditingAppID === item.app.id"
                class="mt-3 grid gap-3 rounded-md border border-border bg-bg-elevated p-3"
                @submit.prevent="submitScopeChange(item)"
              >
                <h5 class="m-0 text-sm font-semibold text-text-primary">
                  {{ t('developer.apps.scopeChangeTitle') }}
                </h5>
                <div class="divide-y divide-border rounded-md border border-border bg-bg-card">
                  <div
                    v-for="option in requestableScopeOptions(item)"
                    :key="option.scope"
                    class="p-3"
                  >
                    <label
                      class="flex cursor-pointer items-start gap-3"
                      :for="scopeChangeInputID(item, option.scope)"
                    >
                      <input
                        :id="scopeChangeInputID(item, option.scope)"
                        v-model="scopeChangeSelectedScopes"
                        class="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary/30"
                        type="checkbox"
                        :value="option.scope"
                      />
                      <span class="min-w-0 flex-1">
                        <span class="flex flex-wrap items-center gap-2">
                          <span class="text-sm font-medium text-text-primary">{{ option.label }}</span>
                          <span
                            class="rounded-full px-2 py-0.5 text-xs font-medium"
                            :class="sensitivityClass(option.sensitivity)"
                          >
                            {{ sensitivityLabel(option.sensitivity) }}
                          </span>
                        </span>
                        <span class="mt-1 block break-all text-xs text-text-muted">{{ option.scope }}</span>
                        <span class="mt-1 block text-xs text-text-muted">
                          {{ option.fields.join(' / ') }}
                        </span>
                      </span>
                    </label>

                    <textarea
                      v-if="isScopeChangeSelected(option.scope)"
                      v-model.trim="scopeChangeReasons[option.scope]"
                      class="developer-input mt-3 min-h-16 resize-y py-2"
                      rows="2"
                      :aria-label="t('developer.apps.scopeChangeReasonLabel', { scope: option.scope })"
                    />
                  </div>
                </div>
                <p
                  v-if="scopeChangeError"
                  class="m-0 rounded-md bg-danger/10 px-3 py-2 text-sm text-danger"
                >
                  {{ scopeChangeError }}
                </p>
                <div class="flex flex-wrap gap-2">
                  <button
                    type="submit"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-text-primary px-3 text-sm font-semibold text-bg-base transition-colors hover:bg-primary hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                    :disabled="scopeSubmittingAppID === item.app.id"
                  >
                    <Send class="h-4 w-4" aria-hidden="true" />
                    {{ t('developer.apps.submitScopeChange') }}
                  </button>
                  <button
                    type="button"
                    class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary"
                    @click="cancelScopeChange"
                  >
                    <X class="h-4 w-4" aria-hidden="true" />
                    {{ t('common.actions.cancel') }}
                  </button>
                </div>
              </form>
            </section>

            <section
              v-if="auditPanelAppID === item.app.id"
              class="mt-4 rounded-md border border-border bg-bg-elevated p-4"
            >
              <header class="flex flex-wrap items-center justify-between gap-2">
                <div class="min-w-0">
                  <h4 class="m-0 text-sm font-semibold text-text-primary">
                    {{ t('developer.apps.auditTitle') }}
                  </h4>
                  <p class="mb-0 mt-1 text-xs text-text-muted">
                    {{ t('developer.apps.auditTotal', { total: auditTotal }) }}
                  </p>
                </div>
                <button
                  type="button"
                  class="inline-flex h-8 items-center justify-center gap-2 rounded-md border border-border bg-bg-card px-2 text-xs font-medium text-text-secondary transition-colors hover:border-primary/40 hover:text-primary disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="auditLoading"
                  @click="loadAppAuditEvents(item.app.id)"
                >
                  <RefreshCw class="h-3.5 w-3.5" :class="auditLoading && 'animate-spin'" aria-hidden="true" />
                  {{ t('developer.apps.refresh') }}
                </button>
              </header>

              <div v-if="auditLoading" class="mt-3 grid gap-3">
                <SkeletonCard v-for="i in 2" :key="i" />
              </div>

              <p v-else-if="auditError" class="mb-0 mt-3 text-sm text-danger">
                {{ auditError }}
              </p>

              <p v-else-if="auditEvents.length === 0" class="mb-0 mt-3 text-sm text-text-muted">
                {{ t('developer.apps.auditEmpty') }}
              </p>

              <ul v-else class="m-0 mt-3 grid list-none gap-2 p-0">
                <li
                  v-for="event in auditEvents"
                  :key="event.id"
                  class="grid gap-2 rounded-md bg-bg-card p-3 text-sm sm:grid-cols-[1fr_auto] sm:items-start"
                >
                  <div class="min-w-0">
                    <p class="m-0 font-medium text-text-primary">
                      {{ appAuditLabel(event) }}
                    </p>
                    <p class="mb-0 mt-1 break-all text-xs text-text-muted">
                      {{ event.eventType }}
                    </p>
                    <p v-if="event.scopes.length" class="mb-0 mt-2 text-xs text-text-secondary">
                      {{ t('developer.apps.auditScopes', { scopes: event.scopes.join(' / ') }) }}
                    </p>
                    <p v-if="appAuditDetail(event)" class="mb-0 mt-1 text-xs text-text-muted">
                      {{ appAuditDetail(event) }}
                    </p>
                  </div>
                  <time class="text-xs text-text-muted sm:text-right" :datetime="event.createdAt">
                    {{ formatDate(event.createdAt) }}
                  </time>
                </li>
              </ul>
            </section>
          </article>

          <nav
            v-if="totalPages > 1"
            class="flex items-center justify-between rounded-lg border border-border bg-bg-card px-3 py-2"
            :aria-label="t('common.pagination.nav')"
          >
            <button
              type="button"
              class="inline-flex h-9 items-center justify-center rounded-md px-3 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-hover disabled:cursor-not-allowed disabled:opacity-50"
              :disabled="page <= 1 || loading"
              @click="goToPage(page - 1)"
            >
              {{ t('developer.apps.previousPage') }}
            </button>
            <span class="text-sm text-text-muted">
              {{ t('common.pagination.page', { page }) }}
            </span>
            <button
              type="button"
              class="inline-flex h-9 items-center justify-center rounded-md px-3 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-hover disabled:cursor-not-allowed disabled:opacity-50"
              :disabled="page >= totalPages || loading"
              @click="goToPage(page + 1)"
            >
              {{ t('developer.apps.nextPage') }}
            </button>
          </nav>
        </div>
      </section>

      <aside class="rounded-lg border border-border bg-bg-card p-5 shadow-sm">
        <h2 class="m-0 text-base font-semibold text-text-primary">
          {{ t('developer.apps.createTitle') }}
        </h2>

        <form class="mt-5 grid gap-4" @submit.prevent="submitApp">
          <label class="grid gap-1.5">
            <span class="text-sm font-medium text-text-secondary">
              {{ t('developer.apps.fields.displayName') }}
            </span>
            <input
              v-model.trim="form.displayName"
              class="developer-input"
              type="text"
              autocomplete="off"
            />
          </label>

          <label class="grid gap-1.5">
            <span class="text-sm font-medium text-text-secondary">
              {{ t('developer.apps.fields.description') }}
            </span>
            <textarea
              v-model.trim="form.description"
              class="developer-input min-h-20 resize-y py-2"
              rows="3"
            />
          </label>

          <label class="grid gap-1.5">
            <span class="text-sm font-medium text-text-secondary">
              {{ t('developer.apps.fields.homepageURL') }}
            </span>
            <input
              v-model.trim="form.homepageURL"
              class="developer-input"
              type="url"
              inputmode="url"
              placeholder="https://"
            />
          </label>

          <label class="grid gap-1.5">
            <span class="text-sm font-medium text-text-secondary">
              {{ t('developer.apps.fields.privacyPolicyURL') }}
            </span>
            <input
              v-model.trim="form.privacyPolicyURL"
              class="developer-input"
              type="url"
              inputmode="url"
              placeholder="https://"
            />
          </label>

          <fieldset class="m-0 grid gap-2 border-0 p-0">
            <legend class="mb-1 text-sm font-medium text-text-secondary">
              {{ t('developer.apps.fields.redirectURIs') }}
            </legend>
            <div
              v-for="(_, index) in form.redirectURIs"
              :key="index"
              class="flex gap-2"
            >
              <input
                v-model.trim="form.redirectURIs[index]"
                class="developer-input min-w-0 flex-1"
                type="url"
                inputmode="url"
                placeholder="https://"
                :aria-label="t('developer.apps.redirectLabel', { index: index + 1 })"
              />
              <button
                type="button"
                class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md border border-border text-text-secondary transition-colors hover:border-danger/40 hover:text-danger disabled:cursor-not-allowed disabled:opacity-40"
                :aria-label="t('developer.apps.removeRedirect')"
                :disabled="form.redirectURIs.length === 1"
                @click="removeRedirectURI(index)"
              >
                <Trash2 class="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
            <button
              type="button"
              class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium text-text-secondary transition-colors hover:border-primary/40 hover:text-primary"
              @click="addRedirectURI"
            >
              <Plus class="h-4 w-4" aria-hidden="true" />
              {{ t('developer.apps.addRedirect') }}
            </button>
          </fieldset>

          <fieldset class="m-0 border-0 p-0">
            <legend class="mb-2 text-sm font-medium text-text-secondary">
              {{ t('developer.apps.fields.scopes') }}
            </legend>
            <div class="divide-y divide-border rounded-md border border-border">
              <div
                v-for="option in availableScopes"
                :key="option.scope"
                class="p-3"
              >
                <label
                  class="flex cursor-pointer items-start gap-3"
                  :for="scopeInputID(option.scope)"
                >
                  <input
                    :id="scopeInputID(option.scope)"
                    v-model="selectedScopes"
                    class="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary/30"
                    type="checkbox"
                    :value="option.scope"
                  />
                  <span class="min-w-0 flex-1">
                    <span class="flex flex-wrap items-center gap-2">
                      <span class="text-sm font-medium text-text-primary">{{ option.label }}</span>
                      <span
                        class="rounded-full px-2 py-0.5 text-xs font-medium"
                        :class="sensitivityClass(option.sensitivity)"
                      >
                        {{ sensitivityLabel(option.sensitivity) }}
                      </span>
                    </span>
                    <span class="mt-1 block break-all text-xs text-text-muted">{{ option.scope }}</span>
                    <span class="mt-1 block text-xs text-text-muted">
                      {{ option.fields.join(' / ') }}
                    </span>
                  </span>
                </label>

                <textarea
                  v-if="isScopeSelected(option.scope)"
                  v-model.trim="form.scopeReasons[option.scope]"
                  class="developer-input mt-3 min-h-16 resize-y py-2"
                  rows="2"
                  :aria-label="t('developer.apps.scopeReasonLabel', { scope: option.scope })"
                />
              </div>
            </div>
          </fieldset>

          <p v-if="formError" class="m-0 rounded-md bg-danger/10 px-3 py-2 text-sm text-danger">
            {{ formError }}
          </p>

          <button
            type="submit"
            class="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-text-primary px-4 text-sm font-semibold text-bg-base transition-colors duration-fast hover:bg-primary hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="submitting"
          >
            <Send class="h-4 w-4" aria-hidden="true" />
            {{ submitting ? t('developer.apps.submitting') : t('developer.apps.submit') }}
          </button>
        </form>
      </aside>
    </div>
  </main>

  <Teleport to="body">
    <div
      v-if="reasonDialog.open"
      class="fixed inset-0 z-50 grid place-items-center bg-black/45 p-4"
      role="presentation"
      @click.self="closeReasonDialog"
    >
      <form
        class="w-full max-w-md rounded-lg border border-border bg-bg-card p-5 shadow-xl"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="reasonDialogTitleID"
        @submit.prevent="submitReasonDialog"
      >
        <header class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <h2 :id="reasonDialogTitleID" class="m-0 text-base font-semibold text-text-primary">
              {{ reasonDialogTitle }}
            </h2>
            <p class="m-0 mt-1 text-sm leading-relaxed text-text-secondary">
              {{ reasonDialogDescription }}
            </p>
          </div>
          <button
            type="button"
            class="grid h-8 w-8 shrink-0 place-items-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
            :aria-label="t('common.actions.close')"
            :disabled="reasonDialogSubmitting"
            @click="closeReasonDialog"
          >
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </header>

        <label class="mt-4 grid gap-1.5">
          <span class="text-sm font-medium text-text-secondary">
            {{ t('developer.apps.reasonLabel') }}
          </span>
          <textarea
            v-model.trim="reasonDialog.reason"
            class="developer-input min-h-24 resize-y py-2"
            rows="4"
            autocomplete="off"
          />
        </label>

        <p
          v-if="reasonDialog.error"
          class="m-0 mt-3 rounded-md bg-danger/10 px-3 py-2 text-sm text-danger"
        >
          {{ reasonDialog.error }}
        </p>

        <div class="mt-5 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="reasonDialogSubmitting"
            @click="closeReasonDialog"
          >
            <X class="h-4 w-4" aria-hidden="true" />
            {{ t('common.actions.cancel') }}
          </button>
          <button
            type="submit"
            class="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-text-primary px-3 text-sm font-semibold text-bg-base transition-colors hover:bg-primary hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="reasonDialogSubmitting"
          >
            <Send class="h-4 w-4" aria-hidden="true" />
            {{ reasonDialogConfirmLabel }}
          </button>
        </div>
      </form>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Activity, ExternalLink, KeyRound, PencilLine, Plus, RefreshCw, Send, Trash2, Undo2, X } from 'lucide-vue-next'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import EmptyState from '@/components/common/EmptyState.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import { developerOpenPlatformAuditTypeKey } from '../auditEvents'
import type {
  OpenPlatformAppRedirectURIRequest,
  OpenPlatformAppListParams,
  OpenPlatformAppScopeRequest,
  OpenPlatformAppWithScopes,
  OpenPlatformDeveloperAppAuditEvent,
  OpenPlatformRedirectURIChangeRequest,
  OpenPlatformRegisterAppRequest,
} from '@stuhelper/shared/api'

type AppStatus = OpenPlatformAppWithScopes['app']['status']
type AppStatusFilter = NonNullable<OpenPlatformAppListParams['status']>
type RedirectURIRequestStatus = OpenPlatformAppRedirectURIRequest['status']
type ScopeStatus = OpenPlatformAppScopeRequest['status']
type ScopeValue = OpenPlatformRegisterAppRequest['scopes'][number]['scope']
type Sensitivity = OpenPlatformAppScopeRequest['sensitivity']

interface ScopeOption {
  scope: ScopeValue
  label: string
  sensitivity: Sensitivity
  fields: string[]
}

interface AppForm {
  displayName: string
  description: string
  homepageURL: string
  privacyPolicyURL: string
  redirectURIs: string[]
  scopeReasons: Record<ScopeValue, string>
}

interface RedirectChangeForm {
  redirectURIs: string[]
  reason: string
  error: string
}

interface AppProfileForm {
  displayName: string
  description: string
  homepageURL: string
  privacyPolicyURL: string
  reason: string
  error: string
}

type ReasonDialogAction = 'rotate-secret' | 'withdraw-app' | 'withdraw-redirect' | 'withdraw-scope'

interface ReasonDialogState {
  open: boolean
  action: ReasonDialogAction | null
  app: OpenPlatformAppWithScopes | null
  redirectRequest: OpenPlatformAppRedirectURIRequest | null
  scopeRequest: OpenPlatformAppScopeRequest | null
  reason: string
  error: string
}

const PAGE_SIZE = 10
const APP_AUDIT_PAGE_SIZE = 10
const reasonDialogTitleID = 'developer-apps-reason-dialog-title'

const { t } = useI18n()
const toast = useToast()

const apps = ref<OpenPlatformAppWithScopes[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const submitting = ref(false)
const profileEditingAppID = ref<number | null>(null)
const profileSubmittingAppID = ref<number | null>(null)
const rotatingAppID = ref<number | null>(null)
const withdrawingKey = ref<string | null>(null)
const redirectEditingAppID = ref<number | null>(null)
const redirectSubmittingAppID = ref<number | null>(null)
const scopeEditingAppID = ref<number | null>(null)
const scopeSubmittingAppID = ref<number | null>(null)
const auditPanelAppID = ref<number | null>(null)
const auditEvents = ref<OpenPlatformDeveloperAppAuditEvent[]>([])
const auditTotal = ref(0)
const auditLoading = ref(false)
const auditError = ref('')
const errorMessage = ref('')
const formError = ref('')
const statusFilter = ref<AppStatusFilter>('all')
const selectedScopes = ref<ScopeValue[]>(['profile.basic.read'])
const scopeChangeSelectedScopes = ref<ScopeValue[]>([])
const scopeChangeReasons = ref<Record<ScopeValue, string>>(defaultScopeReasons())
const scopeChangeError = ref('')
const rotatedSecret = ref<null | {
  appName: string
  clientID: string
  secret: string
}>(null)
const reasonDialog = reactive<ReasonDialogState>({
  open: false,
  action: null,
  app: null,
  redirectRequest: null,
  scopeRequest: null,
  reason: '',
  error: '',
})

const statusOptions: AppStatusFilter[] = ['all', 'pending', 'approved', 'suspended', 'revoked']

const form = reactive<AppForm>({
  displayName: '',
  description: '',
  homepageURL: '',
  privacyPolicyURL: '',
  redirectURIs: [''],
  scopeReasons: defaultScopeReasons(),
})

const redirectChangeForm = reactive<RedirectChangeForm>({
  redirectURIs: [''],
  reason: '',
  error: '',
})

const profileForm = reactive<AppProfileForm>({
  displayName: '',
  description: '',
  homepageURL: '',
  privacyPolicyURL: '',
  reason: '',
  error: '',
})

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

const reasonDialogTitle = computed(() => {
  switch (reasonDialog.action) {
    case 'rotate-secret':
      return t('developer.apps.rotateSecretDialogTitle')
    case 'withdraw-app':
      return t('developer.apps.withdrawAppDialogTitle')
    case 'withdraw-redirect':
      return t('developer.apps.withdrawRedirectDialogTitle')
    case 'withdraw-scope':
      return t('developer.apps.withdrawScopeDialogTitle', {
        scope: reasonDialog.scopeRequest?.displayName ?? '',
      })
    default:
      return ''
  }
})

const reasonDialogDescription = computed(() => {
  switch (reasonDialog.action) {
    case 'rotate-secret':
      return t('developer.apps.rotateSecretDialogDescription')
    case 'withdraw-app':
      return t('developer.apps.withdrawAppDialogDescription')
    case 'withdraw-redirect':
      return t('developer.apps.withdrawRedirectDialogDescription')
    case 'withdraw-scope':
      return t('developer.apps.withdrawScopeDialogDescription')
    default:
      return ''
  }
})

const reasonDialogConfirmLabel = computed(() => {
  switch (reasonDialog.action) {
    case 'rotate-secret':
      return t('developer.apps.confirmRotateSecret')
    case 'withdraw-app':
      return t('developer.apps.confirmWithdrawApp')
    case 'withdraw-redirect':
    case 'withdraw-scope':
      return t('developer.apps.confirmWithdrawRequest')
    default:
      return t('common.actions.confirm')
  }
})

const reasonDialogSubmitting = computed(() => {
  const appID = reasonDialog.app?.app.id
  if (!appID) return false
  switch (reasonDialog.action) {
    case 'rotate-secret':
      return rotatingAppID.value === appID
    case 'withdraw-app':
      return withdrawingKey.value === appWithdrawKey(appID)
    case 'withdraw-redirect':
      return Boolean(
        reasonDialog.redirectRequest &&
        withdrawingKey.value === redirectWithdrawKey(appID, reasonDialog.redirectRequest.id),
      )
    case 'withdraw-scope':
      return Boolean(
        reasonDialog.scopeRequest &&
        withdrawingKey.value === scopeWithdrawKey(appID, reasonDialog.scopeRequest.scope),
      )
    default:
      return false
  }
})

const availableScopes = computed<ScopeOption[]>(() => [
  {
    scope: 'profile.basic.read',
    label: t('developer.apps.scopes.profileBasic'),
    sensitivity: 'low',
    fields: [
      t('developer.apps.scopeFields.username'),
      t('developer.apps.scopeFields.displayName'),
      t('developer.apps.scopeFields.avatar'),
    ],
  },
  {
    scope: 'email.read',
    label: t('developer.apps.scopes.email'),
    sensitivity: 'medium',
    fields: [t('developer.apps.scopeFields.email')],
  },
  {
    scope: 'phone.read',
    label: t('developer.apps.scopes.phone'),
    sensitivity: 'high',
    fields: [
      t('developer.apps.scopeFields.phone'),
      t('developer.apps.scopeFields.phoneVerified'),
    ],
  },
  {
    scope: 'stu.identity.status.read',
    label: t('developer.apps.scopes.identityStatus'),
    sensitivity: 'high',
    fields: [t('developer.apps.scopeFields.identityVerified')],
  },
  {
    scope: 'stu.identity.type.read',
    label: t('developer.apps.scopes.identityType'),
    sensitivity: 'high',
    fields: [t('developer.apps.scopeFields.identityType')],
  },
  {
    scope: 'stu.student.status.read',
    label: t('developer.apps.scopes.studentStatus'),
    sensitivity: 'high',
    fields: [t('developer.apps.scopeFields.studentVerified')],
  },
  {
    scope: 'stu.student.school.read',
    label: t('developer.apps.scopes.studentSchool'),
    sensitivity: 'high',
    fields: [
      t('developer.apps.scopeFields.schoolID'),
      t('developer.apps.scopeFields.schoolName'),
    ],
  },
  {
    scope: 'resource.read',
    label: t('developer.apps.scopes.resourceRead'),
    sensitivity: 'high',
    fields: [t('developer.apps.scopeFields.resourceRead')],
  },
  {
    scope: 'resource.write',
    label: t('developer.apps.scopes.resourceWrite'),
    sensitivity: 'very_high',
    fields: [t('developer.apps.scopeFields.resourceWrite')],
  },
  {
    scope: 'offline_access',
    label: t('developer.apps.scopes.offlineAccess'),
    sensitivity: 'high',
    fields: [t('developer.apps.scopeFields.offlineAccess')],
  },
])

async function loadApps() {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await api.openPlatform.listApps({
      page: page.value,
      pageSize: PAGE_SIZE,
      status: statusFilter.value,
    })
    const data = response.data?.data
    apps.value = data?.list ?? []
    total.value = data?.total ?? 0
  } catch (err) {
    errorMessage.value = getErrorMessage(err, t('developer.apps.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function toggleAuditPanel(item: OpenPlatformAppWithScopes) {
  if (auditPanelAppID.value === item.app.id) {
    auditPanelAppID.value = null
    auditEvents.value = []
    auditTotal.value = 0
    auditError.value = ''
    return
  }
  auditPanelAppID.value = item.app.id
  await loadAppAuditEvents(item.app.id)
}

async function loadAppAuditEvents(appID = auditPanelAppID.value) {
  if (!appID) return
  auditLoading.value = true
  auditError.value = ''
  try {
    const response = await api.openPlatform.listAppAuditEvents({
      appID,
      params: { pageSize: APP_AUDIT_PAGE_SIZE },
    })
    const data = response.data?.data
    auditEvents.value = data?.list ?? []
    auditTotal.value = data?.total ?? 0
  } catch (err) {
    auditError.value = getErrorMessage(err, t('developer.apps.auditLoadFailed'))
  } finally {
    auditLoading.value = false
  }
}

async function submitApp() {
  const validationError = validateForm()
  if (validationError) {
    formError.value = validationError
    return
  }

  submitting.value = true
  formError.value = ''
  try {
    await api.openPlatform.registerApp({
      displayName: form.displayName.trim(),
      description: form.description.trim(),
      homepageURL: form.homepageURL.trim(),
      privacyPolicyURL: form.privacyPolicyURL.trim(),
      redirectURIs: normalizedRedirectURIs(),
      scopes: selectedScopes.value.map(scope => ({
        scope,
        reason: form.scopeReasons[scope].trim(),
      })),
    })
    toast.success(t('developer.apps.createSuccess'))
    resetForm()
    page.value = 1
    statusFilter.value = 'all'
    await loadApps()
  } catch (err) {
    formError.value = getErrorMessage(err, t('developer.apps.createFailed'))
  } finally {
    submitting.value = false
  }
}

function openReasonDialog(
  action: ReasonDialogAction,
  item: OpenPlatformAppWithScopes,
  options: {
    redirectRequest?: OpenPlatformAppRedirectURIRequest
    scopeRequest?: OpenPlatformAppScopeRequest
  } = {},
) {
  reasonDialog.open = true
  reasonDialog.action = action
  reasonDialog.app = item
  reasonDialog.redirectRequest = options.redirectRequest ?? null
  reasonDialog.scopeRequest = options.scopeRequest ?? null
  reasonDialog.reason = ''
  reasonDialog.error = ''
}

function closeReasonDialog() {
  if (reasonDialogSubmitting.value) return
  reasonDialog.open = false
  reasonDialog.action = null
  reasonDialog.app = null
  reasonDialog.redirectRequest = null
  reasonDialog.scopeRequest = null
  reasonDialog.reason = ''
  reasonDialog.error = ''
}

function openRotateSecretDialog(item: OpenPlatformAppWithScopes) {
  openReasonDialog('rotate-secret', item)
}

function openWithdrawAppDialog(item: OpenPlatformAppWithScopes) {
  openReasonDialog('withdraw-app', item)
}

function openWithdrawRedirectDialog(
  item: OpenPlatformAppWithScopes,
  request: OpenPlatformAppRedirectURIRequest,
) {
  openReasonDialog('withdraw-redirect', item, { redirectRequest: request })
}

function openWithdrawScopeDialog(
  item: OpenPlatformAppWithScopes,
  scope: OpenPlatformAppScopeRequest,
) {
  openReasonDialog('withdraw-scope', item, { scopeRequest: scope })
}

async function submitReasonDialog() {
  const item = reasonDialog.app
  if (!item || !reasonDialog.action) return
  const trimmedReason = reasonDialog.reason.trim()
  if (!trimmedReason) {
    reasonDialog.error = reasonDialog.action === 'rotate-secret'
      ? t('developer.apps.validation.rotateReason')
      : t('developer.apps.validation.withdrawReason')
    return
  }

  let succeeded = false
  switch (reasonDialog.action) {
    case 'rotate-secret':
      succeeded = await rotateSecret(item, trimmedReason)
      break
    case 'withdraw-app':
      succeeded = await withdrawApp(item, trimmedReason)
      break
    case 'withdraw-redirect':
      if (reasonDialog.redirectRequest) {
        succeeded = await withdrawRedirectURIRequest(item, reasonDialog.redirectRequest, trimmedReason)
      }
      break
    case 'withdraw-scope':
      if (reasonDialog.scopeRequest) {
        succeeded = await withdrawScopeRequest(item, reasonDialog.scopeRequest, trimmedReason)
      }
      break
  }

  if (succeeded) {
    closeReasonDialog()
  }
}

async function rotateSecret(item: OpenPlatformAppWithScopes, reason: string) {
  rotatingAppID.value = item.app.id
  try {
    const response = await api.openPlatform.rotateAppSecret({
      appID: item.app.id,
      reason,
    })
    const data = response.data?.data
    if (data) {
      rotatedSecret.value = {
        appName: data.app.displayName,
        clientID: data.app.clientID,
        secret: data.clientSecret,
      }
    }
    toast.success(t('developer.apps.rotateSecretSuccess'))
    await loadApps()
    if (auditPanelAppID.value === item.app.id) {
      await loadAppAuditEvents(item.app.id)
    }
    return true
  } catch (err) {
    reasonDialog.error = getErrorMessage(err, t('developer.apps.rotateSecretFailed'))
    toast.error(reasonDialog.error)
    return false
  } finally {
    rotatingAppID.value = null
  }
}

async function withdrawApp(item: OpenPlatformAppWithScopes, reason: string) {
  const key = appWithdrawKey(item.app.id)
  withdrawingKey.value = key
  try {
    await api.openPlatform.withdrawApp({
      appID: item.app.id,
      reason,
    })
    toast.success(t('developer.apps.withdrawAppSuccess'))
    await loadApps()
    if (auditPanelAppID.value === item.app.id) {
      await loadAppAuditEvents(item.app.id)
    }
    return true
  } catch (err) {
    reasonDialog.error = getErrorMessage(err, t('developer.apps.withdrawAppFailed'))
    toast.error(reasonDialog.error)
    return false
  } finally {
    if (withdrawingKey.value === key) withdrawingKey.value = null
  }
}

function startProfileEdit(item: OpenPlatformAppWithScopes) {
  profileEditingAppID.value = item.app.id
  profileForm.displayName = item.app.displayName
  profileForm.description = item.app.description
  profileForm.homepageURL = item.app.homepageURL
  profileForm.privacyPolicyURL = item.app.privacyPolicyURL
  profileForm.reason = ''
  profileForm.error = ''
}

function cancelProfileEdit() {
  profileEditingAppID.value = null
  profileForm.displayName = ''
  profileForm.description = ''
  profileForm.homepageURL = ''
  profileForm.privacyPolicyURL = ''
  profileForm.reason = ''
  profileForm.error = ''
}

async function submitProfileUpdate(item: OpenPlatformAppWithScopes) {
  const validationError = validateProfileForm()
  if (validationError) {
    profileForm.error = validationError
    return
  }

  profileSubmittingAppID.value = item.app.id
  profileForm.error = ''
  try {
    await api.openPlatform.updateAppProfile({
      appID: item.app.id,
      profile: {
        displayName: profileForm.displayName.trim(),
        description: profileForm.description.trim(),
        homepageURL: profileForm.homepageURL.trim(),
        privacyPolicyURL: profileForm.privacyPolicyURL.trim(),
        reason: profileForm.reason.trim(),
      },
    })
    toast.success(t('developer.apps.profileUpdateSuccess'))
    cancelProfileEdit()
    await loadApps()
    if (auditPanelAppID.value === item.app.id) {
      await loadAppAuditEvents(item.app.id)
    }
  } catch (err) {
    profileForm.error = getErrorMessage(err, t('developer.apps.profileUpdateFailed'))
  } finally {
    profileSubmittingAppID.value = null
  }
}

function startRedirectChange(item: OpenPlatformAppWithScopes) {
  redirectEditingAppID.value = item.app.id
  redirectChangeForm.redirectURIs = [...item.app.redirectURIs]
  if (redirectChangeForm.redirectURIs.length === 0) {
    redirectChangeForm.redirectURIs = ['']
  }
  redirectChangeForm.reason = ''
  redirectChangeForm.error = ''
}

function cancelRedirectChange() {
  redirectEditingAppID.value = null
  redirectChangeForm.redirectURIs = ['']
  redirectChangeForm.reason = ''
  redirectChangeForm.error = ''
}

function addRedirectChangeURI() {
  redirectChangeForm.redirectURIs.push('')
}

function removeRedirectChangeURI(index: number) {
  if (redirectChangeForm.redirectURIs.length === 1) return
  redirectChangeForm.redirectURIs.splice(index, 1)
}

async function submitRedirectChange(item: OpenPlatformAppWithScopes) {
  const validationError = validateRedirectChangeForm()
  if (validationError) {
    redirectChangeForm.error = validationError
    return
  }

  redirectSubmittingAppID.value = item.app.id
  redirectChangeForm.error = ''
  try {
    await api.openPlatform.requestRedirectURIChange({
      appID: item.app.id,
      redirectURIs: normalizedRedirectChangeURIs(),
      reason: redirectChangeForm.reason.trim(),
    })
    toast.success(t('developer.apps.redirectChangeSuccess'))
    cancelRedirectChange()
    await loadApps()
    if (auditPanelAppID.value === item.app.id) {
      await loadAppAuditEvents(item.app.id)
    }
  } catch (err) {
    redirectChangeForm.error = getErrorMessage(err, t('developer.apps.redirectChangeFailed'))
  } finally {
    redirectSubmittingAppID.value = null
  }
}

async function withdrawRedirectURIRequest(
  item: OpenPlatformAppWithScopes,
  request: OpenPlatformAppRedirectURIRequest,
  reason: string,
) {
  const key = redirectWithdrawKey(item.app.id, request.id)
  withdrawingKey.value = key
  try {
    await api.openPlatform.withdrawRedirectURIRequest({
      appID: item.app.id,
      requestID: request.id,
      reason,
    })
    toast.success(t('developer.apps.withdrawRequestSuccess'))
    await loadApps()
    if (auditPanelAppID.value === item.app.id) {
      await loadAppAuditEvents(item.app.id)
    }
    return true
  } catch (err) {
    reasonDialog.error = getErrorMessage(err, t('developer.apps.withdrawRequestFailed'))
    toast.error(reasonDialog.error)
    return false
  } finally {
    if (withdrawingKey.value === key) withdrawingKey.value = null
  }
}

function startScopeChange(item: OpenPlatformAppWithScopes) {
  scopeEditingAppID.value = item.app.id
  scopeChangeSelectedScopes.value = []
  scopeChangeReasons.value = defaultScopeReasons()
  for (const scope of item.scopes) {
    if (scope.status === 'rejected' || scope.status === 'withdrawn') {
      scopeChangeReasons.value[scope.scope] = scope.reason
    }
  }
  scopeChangeError.value = ''
}

function cancelScopeChange() {
  scopeEditingAppID.value = null
  scopeChangeSelectedScopes.value = []
  scopeChangeReasons.value = defaultScopeReasons()
  scopeChangeError.value = ''
}

async function submitScopeChange(item: OpenPlatformAppWithScopes) {
  const validationError = validateScopeChangeForm()
  if (validationError) {
    scopeChangeError.value = validationError
    return
  }

  scopeSubmittingAppID.value = item.app.id
  scopeChangeError.value = ''
  try {
    await api.openPlatform.requestScopeChange({
      appID: item.app.id,
      scopes: scopeChangeSelectedScopes.value.map(scope => ({
        scope,
        reason: scopeChangeReasons.value[scope].trim(),
      })),
    })
    toast.success(t('developer.apps.scopeChangeSuccess'))
    cancelScopeChange()
    await loadApps()
    if (auditPanelAppID.value === item.app.id) {
      await loadAppAuditEvents(item.app.id)
    }
  } catch (err) {
    scopeChangeError.value = getErrorMessage(err, t('developer.apps.scopeChangeFailed'))
  } finally {
    scopeSubmittingAppID.value = null
  }
}

async function withdrawScopeRequest(
  item: OpenPlatformAppWithScopes,
  scope: OpenPlatformAppScopeRequest,
  reason: string,
) {
  const key = scopeWithdrawKey(item.app.id, scope.scope)
  withdrawingKey.value = key
  try {
    await api.openPlatform.withdrawScopeRequest({
      appID: item.app.id,
      scope: scope.scope,
      reason,
    })
    toast.success(t('developer.apps.withdrawRequestSuccess'))
    await loadApps()
    if (auditPanelAppID.value === item.app.id) {
      await loadAppAuditEvents(item.app.id)
    }
    return true
  } catch (err) {
    reasonDialog.error = getErrorMessage(err, t('developer.apps.withdrawRequestFailed'))
    toast.error(reasonDialog.error)
    return false
  } finally {
    if (withdrawingKey.value === key) withdrawingKey.value = null
  }
}

function validateForm() {
  if (!form.displayName.trim()) return t('developer.apps.validation.displayName')
  if (!isHTTPSURL(form.homepageURL)) return t('developer.apps.validation.homepageURL')
  if (!isHTTPSURL(form.privacyPolicyURL)) return t('developer.apps.validation.privacyPolicyURL')

  const redirects = normalizedRedirectURIs()
  if (redirects.length === 0) return t('developer.apps.validation.redirectRequired')
  if (redirects.some(uri => !isHTTPSURL(uri) || uri.includes('*'))) {
    return t('developer.apps.validation.redirectURL')
  }

  if (selectedScopes.value.length === 0) return t('developer.apps.validation.scopes')
  if (selectedScopes.value.some(scope => !form.scopeReasons[scope].trim())) {
    return t('developer.apps.validation.scopeReason')
  }

  return ''
}

function validateProfileForm() {
  if (!profileForm.displayName.trim()) return t('developer.apps.validation.displayName')
  if (!isHTTPSURL(profileForm.homepageURL)) return t('developer.apps.validation.homepageURL')
  if (!isHTTPSURL(profileForm.privacyPolicyURL)) return t('developer.apps.validation.privacyPolicyURL')
  if (!profileForm.reason.trim()) return t('developer.apps.validation.profileReason')
  return ''
}

function validateScopeChangeForm() {
  if (scopeChangeSelectedScopes.value.length === 0) return t('developer.apps.validation.scopes')
  if (scopeChangeSelectedScopes.value.some(scope => !scopeChangeReasons.value[scope].trim())) {
    return t('developer.apps.validation.scopeReason')
  }
  return ''
}

function validateRedirectChangeForm() {
  const redirects = normalizedRedirectChangeURIs()
  if (redirects.length === 0) return t('developer.apps.validation.redirectRequired')
  if (redirects.some(uri => !isHTTPSURL(uri) || uri.includes('*'))) {
    return t('developer.apps.validation.redirectURL')
  }
  if (!redirectChangeForm.reason.trim()) return t('developer.apps.validation.redirectReason')
  return ''
}

function isHTTPSURL(value: string) {
  try {
    const parsed = new URL(value.trim())
    return parsed.protocol === 'https:' && Boolean(parsed.host) && !parsed.hash
  } catch (_error) { void _error
    return false
  }
}

function normalizedRedirectURIs() {
  return form.redirectURIs
    .map(uri => uri.trim())
    .filter((uri, index, list) => uri !== '' && list.indexOf(uri) === index)
}

function normalizedRedirectChangeURIs(): OpenPlatformRedirectURIChangeRequest['redirectURIs'] {
  return redirectChangeForm.redirectURIs
    .map(uri => uri.trim())
    .filter((uri, index, list) => uri !== '' && list.indexOf(uri) === index)
}

function addRedirectURI() {
  form.redirectURIs.push('')
}

function removeRedirectURI(index: number) {
  if (form.redirectURIs.length === 1) return
  form.redirectURIs.splice(index, 1)
}

function resetForm() {
  form.displayName = ''
  form.description = ''
  form.homepageURL = ''
  form.privacyPolicyURL = ''
  form.redirectURIs = ['']
  form.scopeReasons = defaultScopeReasons()
  selectedScopes.value = ['profile.basic.read']
}

function defaultScopeReasons(): Record<ScopeValue, string> {
  return {
    'profile.basic.read': '',
    'email.read': '',
    'phone.read': '',
    'stu.identity.status.read': '',
    'stu.identity.type.read': '',
    'stu.student.status.read': '',
    'stu.student.school.read': '',
    'resource.read': '',
    'resource.write': '',
    offline_access: '',
  }
}

function isScopeSelected(scope: ScopeValue) {
  return selectedScopes.value.includes(scope)
}

function isScopeChangeSelected(scope: ScopeValue) {
  return scopeChangeSelectedScopes.value.includes(scope)
}

function scopeInputID(scope: ScopeValue) {
  return `open-platform-scope-${scope.replace(/\./g, '-')}`
}

function scopeChangeInputID(item: OpenPlatformAppWithScopes, scope: ScopeValue) {
  return `open-platform-scope-change-${item.app.id}-${scope.replace(/\./g, '-')}`
}

function goToPage(nextPage: number) {
  if (nextPage < 1 || nextPage > totalPages.value || nextPage === page.value) return
  page.value = nextPage
  void loadApps()
}

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

function statusLabel(status: AppStatus | AppStatusFilter) {
  return t(`developer.apps.status.${status}`)
}

function scopeStatusLabel(status: ScopeStatus) {
  return t(`developer.apps.scopeStatus.${status}`)
}

function redirectRequestStatusLabel(status: RedirectURIRequestStatus) {
  return t(`developer.apps.scopeStatus.${status}`)
}

function pendingRedirectURIRequests(item: OpenPlatformAppWithScopes) {
  return item.redirectURIRequests.filter(request => request.status === 'pending')
}

function requestableScopeOptions(item: OpenPlatformAppWithScopes) {
  const statuses = new Map(item.scopes.map(scope => [scope.scope, scope.status]))
  return availableScopes.value.filter((option) => {
    const status = statuses.get(option.scope)
    return status !== 'approved' && status !== 'pending'
  })
}

function canRequestScopeChange(item: OpenPlatformAppWithScopes) {
  return item.app.status !== 'revoked' && requestableScopeOptions(item).length > 0
}

function canUpdateProfile(item: OpenPlatformAppWithScopes) {
  return item.app.status !== 'revoked'
}

function canRequestRedirectChange(item: OpenPlatformAppWithScopes) {
  return item.app.status === 'approved' || item.app.status === 'suspended'
}

function canWithdrawApp(item: OpenPlatformAppWithScopes) {
  return item.app.status === 'pending'
}

function appWithdrawKey(appID: number) {
  return `app:${appID}`
}

function scopeWithdrawKey(appID: number, scope: string) {
  return `scope:${appID}:${scope}`
}

function redirectWithdrawKey(appID: number, requestID: number) {
  return `redirect:${appID}:${requestID}`
}

function sensitivityLabel(value: Sensitivity) {
  return t(`common.openPlatformConsent.sensitivity.${value}`)
}

function appStatusClass(status: AppStatus) {
  return {
    'bg-warning/10 text-warning': status === 'pending',
    'bg-success/10 text-success': status === 'approved',
    'bg-secondary/10 text-secondary': status === 'suspended',
    'bg-danger/10 text-danger': status === 'revoked',
  }
}

function scopeStatusClass(status: ScopeStatus) {
  return {
    'bg-warning/10 text-warning': status === 'pending',
    'bg-success/10 text-success': status === 'approved',
    'bg-danger/10 text-danger': status === 'rejected',
    'bg-secondary/10 text-secondary': status === 'withdrawn',
  }
}

function sensitivityClass(value: Sensitivity) {
  return {
    'bg-secondary/10 text-secondary': value === 'low',
    'bg-warning/10 text-warning': value === 'medium',
    'bg-danger/10 text-danger': value === 'high' || value === 'very_high',
  }
}

function appAuditLabel(event: OpenPlatformDeveloperAppAuditEvent) {
  const key = developerOpenPlatformAuditTypeKey(event.eventType)
  return key ? t(key) : event.eventType
}

function appAuditDetail(event: OpenPlatformDeveloperAppAuditEvent) {
  const parts: string[] = []
  if (event.endpoint) {
    parts.push(t('developer.apps.auditEndpoint', { endpoint: event.endpoint }))
  }
  if (event.result) {
    parts.push(t('developer.apps.auditResult', { result: event.result }))
  }
  const reason = detailString(event, 'reason')
  if (reason && reason !== event.result) {
    parts.push(t('developer.apps.auditReason', { reason }))
  }
  const resourceType = detailString(event, 'resourceType')
  const resourceID = detailString(event, 'resourceID')
  if (resourceType && resourceID) {
    parts.push(t('developer.apps.auditResource', { type: resourceType, id: resourceID }))
  }
  return parts.join(' · ')
}

function detailString(event: OpenPlatformDeveloperAppAuditEvent, key: string) {
  const value = event.details[key]
  return typeof value === 'string' ? value : ''
}

watch(statusFilter, () => {
  page.value = 1
  void loadApps()
})

onMounted(loadApps)
</script>

<style scoped>
.developer-input {
  min-height: 2.5rem;
  width: 100%;
  border-radius: 0.375rem;
  border: 1px solid var(--color-border);
  background: var(--color-bg-base);
  padding: 0 0.75rem;
  color: var(--color-text-primary);
  font-size: 0.875rem;
  outline: none;
  transition: border-color var(--duration-fast), box-shadow var(--duration-fast);
}

.developer-input:focus {
  border-color: color-mix(in srgb, var(--color-primary) 65%, transparent);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-primary) 15%, transparent);
}

.developer-link {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  min-height: 2rem;
  border-radius: 0.375rem;
  color: var(--color-primary);
  font-size: 0.875rem;
  font-weight: 600;
  text-decoration: none;
}

.developer-action {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  min-height: 2rem;
  padding: 0 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  font-weight: 600;
  background: var(--color-bg-base);
  transition: color var(--duration-fast), border-color var(--duration-fast);
}

.developer-action:hover:not(:disabled) {
  color: var(--color-primary);
  border-color: color-mix(in srgb, var(--color-primary) 45%, transparent);
}

.developer-action:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.developer-link:hover {
  color: var(--color-primary-dark);
}
</style>
