<template>
  <div class="max-w-[600px] mx-auto p-6 animate-fade-in max-sm:p-4">
    <header class="flex items-center gap-3 mb-6">
      <button
        class="p-2 bg-transparent rounded-lg text-text-muted cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
        :aria-label="t('common.actions.back')"
        @click="router.back()"
      >
        <ArrowLeft class="size-5" />
      </button>
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">
        {{ t('user.verification.phone.title') }}
      </h1>
    </header>

    <!-- Already bound -->
    <div
      v-if="alreadyBound"
      class="bg-bg-card border border-green-500/30 rounded-xl p-5 shadow-card"
    >
      <div class="flex items-center gap-3 mb-2">
        <div class="p-2 bg-green-500/10 rounded-lg">
          <Phone class="size-6 text-green-500" />
        </div>
        <div>
          <h2 class="text-base font-bold text-text-primary m-0">
            {{ t('user.verification.phone.bound') }}
          </h2>
          <p v-if="maskedPhone" class="text-sm text-text-muted m-0 mt-1">
            {{ maskedPhone }}
          </p>
        </div>
      </div>
    </div>

    <!-- Bind form -->
    <div v-else class="bg-bg-card rounded-xl p-5 shadow-card">
      <div class="mb-5">
        <label
          class="block text-sm font-semibold text-text-primary mb-2"
          for="phone-binding-input"
        >
          {{ t('user.verification.phone.phoneNumber') }}
        </label>
        <input
          id="phone-binding-input"
          v-model="phone"
          type="tel"
          maxlength="11"
          inputmode="numeric"
          :placeholder="t('user.verification.phone.phoneNumber')"
          :disabled="loading"
          class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
        />
      </div>

      <div class="mb-5">
        <label
          class="block text-sm font-semibold text-text-primary mb-2"
          for="phone-binding-otp"
        >
          {{ t('user.verification.phone.verifyCode') }}
        </label>
        <div class="flex gap-2">
          <input
            id="phone-binding-otp"
            v-model="otpCode"
            type="text"
            inputmode="numeric"
            maxlength="6"
            :placeholder="t('user.verification.phone.verifyCode')"
            :disabled="loading"
            class="flex-1 px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
          />
          <button
            type="button"
            class="px-4 py-2 bg-accent text-white rounded-lg text-sm font-medium border-0 cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed whitespace-nowrap"
            :disabled="!canSendCode"
            @click="onSendCode"
          >
            {{ sendButtonText }}
          </button>
        </div>
      </div>

      <button
        type="button"
        class="w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white border-0 disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="!canSubmit"
        @click="onSubmit"
      >
        {{ loading ? t('common.actions.submit') : t('user.verification.phone.bind') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ArrowLeft, Phone } from 'lucide-vue-next'
import { useVerificationStore } from '@/stores/verification'
import { useToast } from '@/composables/useToast'
import { getErrorStatus } from '@/api/errors'

const { t } = useI18n()
const router = useRouter()
const toast = useToast()
const verificationStore = useVerificationStore()

const phone = ref('')
const otpCode = ref('')
const loading = ref(false)
const sending = ref(false)
const cooldown = ref(0)
let cooldownTimer: ReturnType<typeof setInterval> | null = null

const phonePattern = /^1[3-9]\d{9}$/

const alreadyBound = computed(() => verificationStore.profile?.phoneVerified === true)
const maskedPhone = computed(() => verificationStore.profile?.phone ?? '')

const isPhoneValid = computed(() => phonePattern.test(phone.value))
const isCodeValid = computed(() => /^\d{6}$/.test(otpCode.value))
const canSendCode = computed(
  () => isPhoneValid.value && cooldown.value === 0 && !sending.value && !loading.value,
)
const canSubmit = computed(() => isPhoneValid.value && isCodeValid.value && !loading.value)

const sendButtonText = computed(() => {
  if (sending.value) return t('user.verification.phone.sending')
  if (cooldown.value > 0)
    return t('user.verification.phone.resend', { seconds: cooldown.value })
  return t('user.verification.phone.sendCode')
})

function startCooldown() {
  cooldown.value = 60
  cooldownTimer = setInterval(() => {
    cooldown.value -= 1
    if (cooldown.value <= 0 && cooldownTimer) {
      clearInterval(cooldownTimer)
      cooldownTimer = null
    }
  }, 1000)
}

async function onSendCode() {
  if (!isPhoneValid.value) {
    toast.error(t('user.verification.phone.invalidPhone'))
    return
  }
  sending.value = true
  try {
    await verificationStore.requestBindPhoneOTP(phone.value)
    toast.success(t('user.verification.phone.codeSent'))
    startCooldown()
  } catch (err: unknown) {
    const status = getErrorStatus(err)
    if (status === 429) {
      toast.error(t('user.verification.phone.tooManyRequests'))
    } else if (status === 400) {
      toast.error(t('user.verification.phone.invalidPhone'))
    } else if (status === 503) {
      toast.error(t('user.verification.phone.serviceUnavailable'))
    } else {
      toast.error(t('common.actions.operationFailed'))
    }
  } finally {
    sending.value = false
  }
}

async function onSubmit() {
  if (!canSubmit.value) return
  loading.value = true
  try {
    await verificationStore.bindPhone({ phone: phone.value, otpCode: otpCode.value })
    toast.success(t('user.verification.phone.bindSuccess'))
    router.push('/user/reviews')
  } catch (err: unknown) {
    const status = getErrorStatus(err)
    if (status === 401) {
      toast.error(t('user.verification.phone.invalidCode'))
    } else if (status === 409) {
      toast.error(t('user.verification.phone.alreadyBound'))
    } else if (status === 429) {
      toast.error(t('user.verification.phone.tooManyAttempts'))
    } else if (status === 503) {
      toast.error(t('user.verification.phone.serviceUnavailable'))
    } else {
      toast.error(t('common.actions.operationFailed'))
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (!verificationStore.profile) {
    void verificationStore.fetchStatus().catch(() => {})
  }
})

onUnmounted(() => {
  if (cooldownTimer) clearInterval(cooldownTimer)
})
</script>
