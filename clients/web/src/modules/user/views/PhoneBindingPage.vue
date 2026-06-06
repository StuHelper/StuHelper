<template>
    <main class="mx-auto max-w-[640px] p-6 animate-fade-in max-sm:p-4">
        <header class="mb-6 flex items-center gap-3">
            <button
                type="button"
                class="rounded-lg bg-transparent p-2 text-text-muted transition-colors duration-fast hover:text-text-primary"
                :aria-label="t('common.actions.back')"
                @click="goBack"
            >
                <ArrowLeft class="size-5" aria-hidden="true" />
            </button>
            <h1
                class="m-0 text-xl font-extrabold tracking-tight text-text-primary"
            >
                {{ t("user.verification.phone.title") }}
            </h1>
        </header>

        <section
            class="rounded-xl border border-border bg-bg-card p-5 shadow-card"
        >
            <div class="flex items-start gap-4">
                <span
                    class="grid size-12 shrink-0 place-items-center rounded-lg"
                    :class="
                        alreadyBound
                            ? 'bg-green-500/10 text-green-600'
                            : 'bg-bg-base text-text-muted'
                    "
                >
                    <CheckCircle2
                        v-if="alreadyBound"
                        class="size-6"
                        aria-hidden="true"
                    />
                    <Phone v-else class="size-6" aria-hidden="true" />
                </span>
                <div class="min-w-0">
                    <h2 class="m-0 text-base font-bold text-text-primary">
                        {{
                            alreadyBound
                                ? t("user.verification.phone.bound")
                                : t("user.verification.phone.unbound")
                        }}
                    </h2>
                    <p
                        v-if="maskedPhone"
                        class="m-0 mt-1 text-sm text-text-secondary"
                    >
                        {{ maskedPhone }}
                    </p>
                    <p
                        class="m-0 mt-3 text-sm leading-relaxed text-text-secondary"
                    >
                        {{ t("user.verification.phone.ssoManaged") }}
                    </p>
                </div>
            </div>

            <a
                v-if="accountSettingsUrl"
                :href="accountSettingsUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="mt-5 inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border bg-bg-base px-4 text-sm font-semibold text-text-primary no-underline transition-colors duration-fast hover:border-primary/40 hover:text-primary"
            >
                {{ t("user.verification.phone.openSSOSettings") }}
                <ExternalLink class="size-4" aria-hidden="true" />
            </a>
            <p
                v-else
                class="m-0 mt-5 rounded-md border border-warning/20 bg-warning/10 p-3 text-sm text-warning"
            >
                {{ t("user.verification.phone.ssoSettingsUnavailable") }}
            </p>
        </section>
    </main>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { ArrowLeft, CheckCircle2, ExternalLink, Phone } from "lucide-vue-next";
import { useAuthStore } from "@/stores/auth";
import { useVerificationStore } from "@/stores/verification";
import { useToast } from "@/composables/useToast";

const { t } = useI18n();
const router = useRouter();
const toast = useToast();
const authStore = useAuthStore();
const verificationStore = useVerificationStore();

const alreadyBound = computed(
    () => verificationStore.profile?.phoneVerified === true,
);
const maskedPhone = computed(() => verificationStore.profile?.phone ?? "");
const accountSettingsUrl = computed(
    () => authStore.user?.accountSettingsUrl ?? "",
);

function goBack() {
    void router.push("/identity");
}

onMounted(() => {
    if (!verificationStore.profile) {
        void verificationStore.fetchStatus().catch(() => {
            toast.error(t("common.loadFailed"));
        });
    }
});
</script>
