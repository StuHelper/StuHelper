<template>
    <Transition name="overlay">
        <div
            v-if="countdown > 0"
            class="absolute inset-0 bg-bg-card/95 backdrop-blur-sm rounded-xl flex flex-col items-center justify-center gap-4 z-10"
        >
            <div
                class="w-14 h-14 rounded-full border-3 border-primary flex items-center justify-center text-2xl font-bold text-primary animate-pulse"
            >
                {{ countdown }}
            </div>
            <p class="text-sm text-text-primary font-medium">
                {{ t("review.draft.savedAndLogin") }}
            </p>
            <p class="text-xs text-text-muted">
                {{ t("review.draft.redirectCountdown", { n: countdown }) }}
            </p>
        </div>
    </Transition>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

defineProps<{ countdown: number }>();

const { t } = useI18n();
</script>
