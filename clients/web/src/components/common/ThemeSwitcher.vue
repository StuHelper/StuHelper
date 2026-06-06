<template>
    <button
        type="button"
        class="group relative inline-flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-transparent text-text-secondary transition-all duration-fast hover:border-white/20 hover:bg-bg-card hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
        :aria-label="nextLabel"
        :title="nextLabel"
        @click="toggleTheme"
    >
        <Sun v-if="themeStore.isDark" class="size-[18px]" aria-hidden="true" />
        <Moon v-else class="size-[18px]" aria-hidden="true" />
    </button>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Moon, Sun } from "lucide-vue-next";
import { useThemeStore } from "@/stores/theme";

const { t } = useI18n();
const themeStore = useThemeStore();

const nextLabel = computed(() =>
    themeStore.isDark
        ? t("common.theme.switchToLight")
        : t("common.theme.switchToDark"),
);

function toggleTheme() {
    themeStore.setMode(themeStore.isDark ? "light" : "dark");
}
</script>
