<template>
  <div class="bg-bg-card border border-border rounded-xl p-5 shadow-card mb-6">
    <!-- User info header -->
    <div class="flex items-center gap-4 mb-5">
      <div class="p-[3px] bg-gradient-to-br from-primary to-accent rounded-full shrink-0">
        <div class="size-14 bg-bg-card rounded-full flex items-center justify-center overflow-hidden">
          <img
            v-if="user?.avatar"
            :src="user.avatar"
            :alt="displayName"
            class="size-full object-cover"
          />
          <User v-else class="size-7 text-text-muted" />
        </div>
      </div>
      <div class="min-w-0">
        <h2 class="text-base font-bold text-text-primary m-0 truncate">
          {{ displayName }}
        </h2>
        <p class="text-sm text-text-muted m-0 mt-0.5 truncate">
          {{ user?.email ?? '' }}
        </p>
      </div>
    </div>

    <!-- Status cards grid -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <!-- Identity verification card -->
      <div class="border border-border rounded-lg p-4 transition-all duration-fast">
        <div class="flex items-center gap-3 mb-3">
          <div
            class="p-2 rounded-lg"
            :class="identityVerified ? 'bg-green-500/10' : 'bg-bg-base'"
          >
            <ShieldCheck
              class="size-5"
              :class="identityVerified ? 'text-green-500' : 'text-text-muted'"
            />
          </div>
          <div class="min-w-0">
            <p class="text-sm font-semibold text-text-primary m-0">
              {{ t('user.verification.identity.title') }}
            </p>
            <span
              class="inline-block mt-1 px-2 py-0.5 rounded-full text-xs font-medium"
              :class="identityStatusClass"
            >
              {{ identityStatusLabel }}
            </span>
          </div>
        </div>
        <router-link
          v-if="!identityVerified"
          to="/user/identity-verification"
          class="block w-full py-2 bg-text-primary text-bg-base rounded-lg text-xs font-medium text-center no-underline transition-all duration-fast hover:bg-accent hover:text-white"
        >
          {{ t('user.verification.identity.unverified') }}
        </router-link>
      </div>

      <!-- Student verification card -->
      <div class="border border-border rounded-lg p-4 transition-all duration-fast">
        <div class="flex items-center gap-3 mb-3">
          <div
            class="p-2 rounded-lg"
            :class="studentVerified ? 'bg-green-500/10' : 'bg-bg-base'"
          >
            <GraduationCap
              class="size-5"
              :class="studentVerified ? 'text-green-500' : 'text-text-muted'"
            />
          </div>
          <div class="min-w-0">
            <p class="text-sm font-semibold text-text-primary m-0">
              {{ t('user.verification.student.title') }}
            </p>
            <span
              class="inline-block mt-1 px-2 py-0.5 rounded-full text-xs font-medium"
              :class="studentStatusClass"
            >
              {{ studentStatusLabel }}
            </span>
          </div>
        </div>
        <template v-if="!studentVerified">
          <router-link
            v-if="identityVerified"
            to="/user/student-verification"
            class="block w-full py-2 bg-text-primary text-bg-base rounded-lg text-xs font-medium text-center no-underline transition-all duration-fast hover:bg-accent hover:text-white"
          >
            {{ t('user.verification.student.unverified') }}
          </router-link>
          <p v-else class="text-xs text-text-muted m-0 text-center">
            {{ t('user.verification.student.identityRequired') }}
          </p>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShieldCheck, GraduationCap, User } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'

const { t } = useI18n()
const authStore = useAuthStore()
const verificationStore = useVerificationStore()

const user = computed(() => authStore.user)
const displayName = computed(() => user.value?.displayName ?? user.value?.name ?? '')
const identityVerified = computed(() => verificationStore.identityVerified)
const studentVerified = computed(() => verificationStore.studentVerified)

const identity = computed(() => verificationStore.identity)
const profile = computed(() => verificationStore.profile)

// Identity status helpers
type StatusVariant = 'verified' | 'pending' | 'rejected' | 'unverified'

const identityStatus = computed((): StatusVariant => {
  if (!identity.value) return 'unverified'
  if (identity.value.verified) return 'verified'
  if (identity.value.reviewedAt) return 'rejected'
  return 'pending'
})

const identityStatusLabel = computed(() => {
  return t(`user.verification.identity.${identityStatus.value}`)
})

const studentStatus = computed((): StatusVariant => {
  if (!profile.value) return 'unverified'
  const status = profile.value.verificationStatus
  if (status === 'verified') return 'verified'
  if (status === 'rejected') return 'rejected'
  if (status === 'pending') return 'pending'
  return 'unverified'
})

const studentStatusLabel = computed(() => {
  return t(`user.verification.student.${studentStatus.value}`)
})

const statusClassMap: Record<StatusVariant, string> = {
  verified: 'bg-green-500/10 text-green-600',
  pending: 'bg-yellow-500/10 text-yellow-600',
  rejected: 'bg-red-500/10 text-red-600',
  unverified: 'bg-bg-base text-text-muted',
}

const identityStatusClass = computed(() => statusClassMap[identityStatus.value])
const studentStatusClass = computed(() => statusClassMap[studentStatus.value])

onMounted(() => {
  if (authStore.isAuthenticated) {
    verificationStore.fetchStatus()
  }
})
</script>
