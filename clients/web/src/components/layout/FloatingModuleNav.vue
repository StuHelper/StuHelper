<template>
    <div
        class="fixed z-[var(--z-sticky)] flex flex-col items-center gap-1.5 transition-opacity duration-base"
        :style="positionStyle"
        :class="isDragging ? 'cursor-grabbing' : 'cursor-grab'"
        @click.capture="suppressClickAfterDrag"
        @mouseenter="expanded = true"
        @mouseleave="expanded = false"
        @mousedown.prevent="startDrag"
        @touchstart="startDrag"
    >
        <!-- 当前模块图标（始终显示） -->
        <div class="relative group">
            <router-link
                :to="activeTab.to"
                data-testid="floating-module-nav-active"
                class="w-10 h-10 rounded-full bg-bg-glass-heavy backdrop-blur-xl backdrop-saturate-150 border border-white/15 dark:border-white/8 shadow-md flex items-center justify-center no-underline transition-all duration-base hover:shadow-lg"
                aria-describedby="tooltip-active"
                @click.stop
            >
                <component
                    :is="activeTab.icon"
                    :size="20"
                    class="text-primary"
                />
            </router-link>
            <span
                id="tooltip-active"
                role="tooltip"
                class="absolute right-full mr-2.5 top-1/2 -translate-y-1/2 px-2 py-1 text-xs font-medium text-text-primary bg-bg-glass-heavy backdrop-blur-lg rounded-md shadow-sm border border-white/15 dark:border-white/8 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity duration-fast pointer-events-none"
            >
                {{ activeTab.label }}
            </span>
        </div>

        <!-- 展开的其他模块 -->
        <transition-group
            enter-active-class="transition-all duration-base ease-out"
            enter-from-class="opacity-0 scale-75"
            enter-to-class="opacity-100 scale-100"
            leave-active-class="transition-all duration-fast ease-in"
            leave-from-class="opacity-100 scale-100"
            leave-to-class="opacity-0 scale-75"
        >
            <template v-if="expanded">
                <div
                    v-for="tab in otherTabs"
                    :key="tab.to"
                    class="relative group"
                >
                    <router-link
                        :to="tab.to"
                        class="w-9 h-9 rounded-full bg-bg-glass backdrop-blur-lg backdrop-saturate-150 border border-white/15 dark:border-white/8 shadow-sm flex items-center justify-center no-underline transition-colors duration-fast hover:bg-primary hover:text-white hover:border-primary"
                        :aria-describedby="`tooltip-tab-${tab.to}`"
                        @click.stop
                    >
                        <component
                            :is="tab.icon"
                            :size="16"
                            class="text-text-secondary"
                        />
                    </router-link>
                    <span
                        :id="`tooltip-tab-${tab.to}`"
                        role="tooltip"
                        class="absolute right-full mr-2.5 top-1/2 -translate-y-1/2 px-2 py-1 text-xs font-medium text-text-primary bg-bg-glass-heavy backdrop-blur-lg rounded-md shadow-sm border border-white/15 dark:border-white/8 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity duration-fast pointer-events-none"
                    >
                        {{ tab.label }}
                    </span>
                </div>
            </template>
        </transition-group>
    </div>
</template>

<script setup lang="ts">
import {
    ref,
    computed,
    onMounted,
    onUnmounted,
    markRaw,
    type Component,
} from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { Pencil, GraduationCap } from "lucide-vue-next";

interface ModuleTab {
    to: string;
    icon: Component;
    labelKey: string;
}

const { t } = useI18n();

const tabDefs: ModuleTab[] = [
    { to: "/courses/reviews", icon: markRaw(Pencil), labelKey: "nav.review" },
    { to: "/teachers", icon: markRaw(GraduationCap), labelKey: "nav.teacher" },
];

const tabs = computed(() =>
    tabDefs.map((d) => ({
        to: d.to,
        icon: d.icon,
        label: t(d.labelKey),
    })),
);

const route = useRoute();
const expanded = ref(false);
const isDragging = ref(false);

// 位置状态
const STORAGE_KEY = "floating-nav-position";
const DRAG_THRESHOLD = 5;
const position = ref({ x: 16, y: Math.round(window.innerHeight / 2 - 20) });

// 验证位置值是否合法
function isValidPosition(v: unknown): v is { x: number; y: number } {
    if (!v || typeof v !== "object") return false;
    const p = v as Record<string, unknown>;
    return (
        typeof p.x === "number" &&
        typeof p.y === "number" &&
        Number.isFinite(p.x) &&
        Number.isFinite(p.y)
    );
}

// 将位置约束到当前视口范围内
function clampPosition(x: number, y: number): { x: number; y: number } {
    return {
        x: Math.max(8, Math.min(window.innerWidth - 56, x)),
        y: Math.max(8, Math.min(window.innerHeight - 56, y)),
    };
}

function loadPosition() {
    try {
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved) {
            const parsed: unknown = JSON.parse(saved);
            if (!isValidPosition(parsed)) return;
            position.value = clampPosition(parsed.x, parsed.y);
        }
    } catch (_error) { void _error;
        /* ignore malformed data */
    }
}

// 保存位置防抖
let saveTimer: ReturnType<typeof setTimeout> | null = null;
function savePosition() {
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(() => {
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify(position.value));
        } catch (_error) { void _error;
            /* ignore */
        }
        saveTimer = null;
    }, 200);
}

const positionStyle = computed(() => ({
    right: `${position.value.x}px`,
    top: `${position.value.y}px`,
}));

const activeTab = computed(() => {
    return tabs.value.find((t) => route.path.startsWith(t.to)) || tabs.value[0];
});

const otherTabs = computed(() => {
    return tabs.value.filter((t) => t !== activeTab.value);
});

// 拖拽逻辑
let dragStartX = 0;
let dragStartY = 0;
let startPosX = 0;
let startPosY = 0;
let hasMoved = false;
let dragController: AbortController | null = null;

function startDrag(e: MouseEvent | TouchEvent) {
    // 清理之前可能残留的监听器，防止重复注册
    dragController?.abort();

    isDragging.value = true;
    hasMoved = false;
    const point = "touches" in e ? e.touches[0] : e;
    dragStartX = point.clientX;
    dragStartY = point.clientY;
    startPosX = position.value.x;
    startPosY = position.value.y;

    dragController = new AbortController();
    const { signal } = dragController;
    document.addEventListener("mousemove", onDrag, { signal });
    document.addEventListener("mouseup", stopDrag, { signal });
    document.addEventListener("touchmove", onDrag, { signal });
    document.addEventListener("touchend", stopDrag, { signal });
}

function onDrag(e: MouseEvent | TouchEvent) {
    const point = "touches" in e ? e.touches[0] : e;
    const dx = dragStartX - point.clientX;
    const dy = point.clientY - dragStartY;

    // 使用常量化的拖拽阈值
    if (Math.abs(dx) > DRAG_THRESHOLD || Math.abs(dy) > DRAG_THRESHOLD) {
        hasMoved = true;
        if ("touches" in e && e.cancelable) {
            e.preventDefault();
        }
    }

    // 使用 clampPosition 约束到视口范围
    const clamped = clampPosition(startPosX + dx, startPosY + dy);
    position.value = clamped;
}

function stopDrag() {
    isDragging.value = false;
    dragController?.abort();
    dragController = null;

    // 靠边吸附：靠近右侧吸附到右边，靠近左侧吸附到左边
    if (position.value.x < 40) {
        position.value.x = 8;
    } else if (position.value.x > window.innerWidth - 80) {
        position.value.x = window.innerWidth - 56;
    }

    if (hasMoved) {
        savePosition();
    }
}

function suppressClickAfterDrag(e: MouseEvent) {
    if (!hasMoved) return;
    e.preventDefault();
    e.stopPropagation();
    hasMoved = false;
}

onMounted(loadPosition);
onUnmounted(() => {
    dragController?.abort();
    dragController = null;
    if (saveTimer) {
        clearTimeout(saveTimer);
        saveTimer = null;
    }
});
</script>
