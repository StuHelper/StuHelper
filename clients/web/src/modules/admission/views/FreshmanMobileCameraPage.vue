<template>
    <main class="mobile-camera join-surface">
        <section class="mobile-camera__frame">
            <header class="mobile-camera__header join-glass-heavy">
                <p class="mobile-camera__eyebrow join-eyebrow">StuHelper</p>
                <h1 class="mobile-camera__title">新生材料拍照</h1>
            </header>

            <section class="mobile-camera__panel join-glass">
                <div
                    v-if="pageState === 'loading'"
                    class="mobile-camera__loading"
                    data-state="loading"
                >
                    <span class="mobile-camera__spinner" aria-hidden="true" />
                    <p class="mobile-camera__loading-text">
                        正在打开手机拍照链接...
                    </p>
                </div>

                <div
                    v-else-if="pageState === 'ready'"
                    class="mobile-camera__ready"
                    data-state="ready"
                >
                    <div class="mobile-camera__viewport">
                        <video
                            ref="videoRef"
                            autoplay
                            class="mobile-camera__video"
                            muted
                            playsinline
                        />
                        <span
                            class="mobile-camera__viewfinder"
                            aria-hidden="true"
                        />
                    </div>

                    <img
                        v-if="previewDataURL"
                        :src="previewDataURL"
                        alt="录取材料预览"
                        class="mobile-camera__preview"
                    />

                    <div class="mobile-camera__actions">
                        <button
                            v-if="!streamActive"
                            class="secondary-button mobile-camera__open"
                            data-mobile-camera-open-button
                            type="button"
                            :disabled="!cameraSupported"
                            @click="openCamera"
                        >
                            {{
                                cameraSupported
                                    ? "打开摄像头"
                                    : "当前浏览器不支持摄像头"
                            }}
                        </button>
                        <button
                            v-if="streamActive"
                            class="secondary-button"
                            data-mobile-camera-capture-button
                            type="button"
                            @click="captureMaterial"
                        >
                            拍摄
                        </button>
                        <button
                            v-if="capturedPayload"
                            class="secondary-button"
                            data-mobile-camera-retake-button
                            type="button"
                            @click="retake"
                        >
                            重拍
                        </button>
                        <button
                            class="primary-button"
                            data-mobile-camera-upload-button
                            type="button"
                            :disabled="uploading || !capturedPayload"
                            @click="uploadCapture"
                        >
                            {{ uploading ? "上传中..." : "上传材料" }}
                        </button>
                    </div>
                </div>

                <div
                    v-else-if="pageState === 'uploaded'"
                    class="mobile-camera__state"
                    data-state="uploaded"
                >
                    <span
                        class="mobile-camera__state-icon join-tone-success"
                        aria-hidden="true"
                    >
                        <svg
                            class="mobile-camera__state-glyph"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <path d="M20 6 9 17l-5-5" />
                        </svg>
                    </span>
                    <h2 class="mobile-camera__state-title">材料已上传</h2>
                    <p class="mobile-camera__state-text">
                        请选择后续在哪一端继续。选择后另一端会被锁定，避免重复提交。
                    </p>
                    <div class="mobile-camera__actions">
                        <button
                            class="primary-button"
                            data-mobile-camera-continue-desktop-button
                            type="button"
                            :disabled="choosing"
                            @click="chooseContinuation('desktop')"
                        >
                            回到电脑端继续
                        </button>
                        <button
                            class="secondary-button"
                            data-mobile-camera-continue-mobile-button
                            type="button"
                            :disabled="choosing"
                            @click="chooseContinuation('mobile')"
                        >
                            在手机端继续
                        </button>
                    </div>
                </div>

                <div
                    v-else-if="pageState === 'desktop'"
                    class="mobile-camera__state"
                    data-state="desktop"
                >
                    <span
                        class="mobile-camera__state-icon join-tone-info"
                        aria-hidden="true"
                    >
                        <svg
                            class="mobile-camera__state-glyph"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <rect x="3" y="4" width="18" height="12" rx="2" />
                            <path d="M8 20h8" />
                            <path d="M12 16v4" />
                        </svg>
                    </span>
                    <h2 class="mobile-camera__state-title">请回到电脑端继续</h2>
                    <p class="mobile-camera__state-text">
                        材料已上传，电脑端会自动刷新状态。
                    </p>
                </div>

                <div
                    v-else-if="pageState === 'mobile'"
                    class="mobile-camera__state"
                    data-state="mobile"
                >
                    <span
                        class="mobile-camera__state-icon join-tone-success"
                        aria-hidden="true"
                    >
                        <svg
                            class="mobile-camera__state-glyph"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <path d="M20 6 9 17l-5-5" />
                        </svg>
                    </span>
                    <h2 class="mobile-camera__state-title">材料已提交</h2>
                    <p class="mobile-camera__state-text">请等待管理员审核。</p>
                </div>

                <div
                    v-else-if="pageState === 'expired'"
                    class="mobile-camera__state"
                    data-state="expired"
                >
                    <span
                        class="mobile-camera__state-icon join-tone-warning"
                        aria-hidden="true"
                    >
                        <svg
                            class="mobile-camera__state-glyph"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <circle cx="12" cy="12" r="10" />
                            <path d="M12 6v6l4 2" />
                        </svg>
                    </span>
                    <h2 class="mobile-camera__state-title">链接已过期</h2>
                    <p class="mobile-camera__state-text">
                        请回到电脑端重新生成手机拍照二维码。
                    </p>
                </div>

                <div v-else class="mobile-camera__state" data-state="error">
                    <span
                        class="mobile-camera__state-icon join-tone-danger"
                        aria-hidden="true"
                    >
                        <svg
                            class="mobile-camera__state-glyph"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <path
                                d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
                            />
                            <path d="M12 9v4" />
                            <path d="M12 17h.01" />
                        </svg>
                    </span>
                    <h2 class="mobile-camera__state-title">无法打开拍照链接</h2>
                    <p class="mobile-camera__state-text">
                        {{ errorMessage }}
                    </p>
                </div>

                <p
                    v-if="errorMessage && pageState !== 'error'"
                    class="mobile-camera__error"
                >
                    {{ errorMessage }}
                </p>
            </section>
        </section>
    </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRoute } from "vue-router";

import type {
    CameraCaptureRequest,
    FreshmanCameraHandoff,
    FreshmanCameraHandoffContinuationRequest,
} from "@stuhelper/shared/api";

import { admissionApi } from "../api";
import { isAdmissionSessionExpiredError } from "../admissionToken";
import {
    captureFrameAsBase64,
    describeCameraCaptureError,
    startCameraStream,
    stopCameraStream,
    supportsCameraCapture,
} from "../cameraCapture";

type MobilePageState =
    | "loading"
    | "ready"
    | "uploaded"
    | "desktop"
    | "mobile"
    | "expired"
    | "error";

const route = useRoute();
const videoRef = ref<HTMLVideoElement | null>(null);
const stream = ref<MediaStream | null>(null);
const capturedPayload = ref<CameraCaptureRequest | null>(null);
const previewDataURL = ref("");
const loading = ref(true);
const uploading = ref(false);
const choosing = ref(false);
const cameraSupported = ref(supportsCameraCapture());
const handoff = ref<FreshmanCameraHandoff | null>(null);
const errorMessage = ref("");
const forceExpired = ref(false);
let handoffLoadSeq = 0;

const token = computed(() => String(route.params.token ?? ""));
const streamActive = computed(() => stream.value !== null);
const pageState = computed<MobilePageState>(() => {
    if (loading.value) return "loading";
    if (forceExpired.value) return "expired";
    const current = handoff.value;
    if (!current) return "error";
    if (current.status === "expired") return "expired";
    if (current.status === "uploaded") return "uploaded";
    if (current.status === "locked" && current.continueOn === "desktop")
        return "desktop";
    if (current.status === "locked" && current.continueOn === "mobile")
        return "mobile";
    if (current.status === "pending") return "ready";
    return "error";
});

watch(
    token,
    (nextToken) => {
        void loadHandoff(nextToken);
    },
    { immediate: true },
);

async function loadHandoff(requestToken: string): Promise<void> {
    const requestSeq = ++handoffLoadSeq;
    loading.value = true;
    resetHandoffStateForTokenLoad();
    errorMessage.value = "";
    try {
        const nextHandoff =
            await admissionApi.previewFreshmanMobileCameraHandoff(requestToken);
        if (!isCurrentHandoffLoad(requestSeq, requestToken)) {
            return;
        }
        handoff.value = nextHandoff;
    } catch (error) {
        if (!isCurrentHandoffLoad(requestSeq, requestToken)) {
            return;
        }
        if (handleAdmissionExpiredError(error, requestToken)) return;
        errorMessage.value = readErrorMessage(error, "拍照链接暂时无法打开。");
        handoff.value = null;
    } finally {
        if (isCurrentHandoffLoad(requestSeq, requestToken)) {
            loading.value = false;
        }
    }
}

async function openCamera(): Promise<void> {
    const requestToken = token.value;
    errorMessage.value = "";
    let openedStream: MediaStream | null = null;
    try {
        openedStream = await startCameraStream();
        if (
            !isCurrentMobileToken(requestToken) ||
            pageState.value !== "ready"
        ) {
            stopCameraStream(openedStream);
            return;
        }
        stream.value = openedStream;
        if (videoRef.value) {
            videoRef.value.srcObject = stream.value;
            await videoRef.value.play();
        }
    } catch (error) {
        if (!isCurrentMobileToken(requestToken)) {
            if (openedStream) stopCameraStream(openedStream);
            return;
        }
        if (openedStream) {
            stopCameraStream(openedStream);
            stream.value = null;
        }
        errorMessage.value = describeCameraCaptureError(
            error,
            "无法打开摄像头。",
        );
    }
}

function captureMaterial(): void {
    errorMessage.value = "";
    if (!videoRef.value) {
        throw new Error("Camera video element is not mounted");
    }
    try {
        const payload = captureFrameAsBase64(videoRef.value, {
            maxBytes: handoff.value?.maxMaterialBytes,
        });
        capturedPayload.value = payload;
        previewDataURL.value = `data:${payload.contentType};base64,${payload.imageBase64}`;
    } catch (error) {
        errorMessage.value = describeCameraCaptureError(
            error,
            "拍摄失败，请重试。",
        );
    }
}

function retake(): void {
    capturedPayload.value = null;
    previewDataURL.value = "";
}

async function uploadCapture(): Promise<void> {
    if (uploading.value || !capturedPayload.value) return;
    const requestToken = token.value;
    uploading.value = true;
    errorMessage.value = "";
    try {
        const nextHandoff =
            await admissionApi.uploadFreshmanMobileCameraCapture(
                requestToken,
                capturedPayload.value,
            );
        if (!isCurrentMobileToken(requestToken)) {
            return;
        }
        handoff.value = nextHandoff;
        stopCameraStream(stream.value);
        stream.value = null;
    } catch (error) {
        if (!isCurrentMobileToken(requestToken)) {
            return;
        }
        if (handleAdmissionExpiredError(error, requestToken)) return;
        errorMessage.value = readErrorMessage(error, "材料上传失败。");
    } finally {
        if (isCurrentMobileToken(requestToken)) {
            uploading.value = false;
        }
    }
}

async function chooseContinuation(
    continueOn: FreshmanCameraHandoffContinuationRequest["continueOn"],
): Promise<void> {
    if (choosing.value) return;
    const requestToken = token.value;
    choosing.value = true;
    errorMessage.value = "";
    try {
        const nextHandoff =
            await admissionApi.chooseFreshmanMobileCameraContinuation(
                requestToken,
                { continueOn },
            );
        if (!isCurrentMobileToken(requestToken)) {
            return;
        }
        handoff.value = nextHandoff;
    } catch (error) {
        if (!isCurrentMobileToken(requestToken)) {
            return;
        }
        if (handleAdmissionExpiredError(error, requestToken)) return;
        errorMessage.value = readErrorMessage(error, "继续方式保存失败。");
    } finally {
        if (isCurrentMobileToken(requestToken)) {
            choosing.value = false;
        }
    }
}

function resetHandoffStateForTokenLoad(): void {
    forceExpired.value = false;
    handoff.value = null;
    capturedPayload.value = null;
    previewDataURL.value = "";
    uploading.value = false;
    choosing.value = false;
    stopCameraStream(stream.value);
    stream.value = null;
}

function isCurrentHandoffLoad(
    requestSeq: number,
    requestToken: string,
): boolean {
    return handoffLoadSeq === requestSeq && isCurrentMobileToken(requestToken);
}

function isCurrentMobileToken(requestToken: string): boolean {
    return token.value === requestToken;
}

function readErrorMessage(error: unknown, fallback: string): string {
    if (isMobileCameraHandoffNotFoundError(error)) {
        return "拍照链接不存在或已失效。请回到电脑端重新生成手机拍照二维码。";
    }
    return error instanceof Error && error.message ? error.message : fallback;
}

function isMobileCameraHandoffNotFoundError(error: unknown): boolean {
    if (!(error instanceof Error)) return false;
    const status =
        "status" in error ? (error as { status?: unknown }).status : undefined;
    if (status === 404) return true;
    return /admission camera handoff not found|camera handoff not found|handoff not found/i.test(
        error.message,
    );
}

function handleAdmissionExpiredError(
    error: unknown,
    requestToken = token.value,
): boolean {
    if (!isAdmissionSessionExpiredError(error)) return false;
    if (!isCurrentMobileToken(requestToken)) return true;
    forceExpired.value = true;
    errorMessage.value = "";
    stopCameraStream(stream.value);
    stream.value = null;
    return true;
}

onBeforeUnmount(() => {
    handoffLoadSeq += 1;
    stopCameraStream(stream.value);
});
</script>

<style src="./join-theme.css"></style>

<style scoped>
/*
 * 独立免登录路由（layout:'none'），根元素自带 .join-surface 并直接引入
 * join-theme.css（与 AdmissionShell 同一套品牌玻璃风原语）。
 * 测试契约（勿动）：data-state 七态、data-mobile-camera-* 选择器、
 * .primary-button/.secondary-button 类名、h1/h2 标题文案、
 * video 的 autoplay/muted/playsinline 且仅存在于 ready 分支。
 */
.mobile-camera {
    min-height: 100dvh;
    padding: 18px 12px max(28px, env(safe-area-inset-bottom));
}

.mobile-camera__frame {
    display: grid;
    gap: 14px;
    margin: 0 auto;
    max-width: 560px;
    width: 100%;
}

/* ── 玻璃头部卡 ─────────────────────────────────── */
.mobile-camera__header {
    display: grid;
    gap: 6px;
    padding: 18px 20px;
}

.mobile-camera__eyebrow {
    margin: 0;
}

.mobile-camera__title {
    color: var(--join-ink);
    font-size: 24px;
    font-weight: 800;
    letter-spacing: -0.01em;
    line-height: 32px;
    margin: 0;
}

/* ── 主玻璃面板 ─────────────────────────────────── */
.mobile-camera__panel {
    padding: 20px 16px;
}

/* ── 加载态 ─────────────────────────────────────── */
.mobile-camera__loading {
    align-items: center;
    display: flex;
    gap: 12px;
    padding: 8px 0;
}

.mobile-camera__spinner {
    animation: mobile-camera-spin 0.9s linear infinite;
    border: 3px solid var(--join-glass-border);
    border-radius: 999px;
    border-top-color: var(--color-primary);
    flex-shrink: 0;
    height: 22px;
    width: 22px;
}

@keyframes mobile-camera-spin {
    to {
        transform: rotate(360deg);
    }
}

.mobile-camera__loading-text {
    color: var(--join-ink-soft);
    font-size: 14px;
    line-height: 22px;
    margin: 0;
}

/* ── 拍照就绪态 ─────────────────────────────────── */
.mobile-camera__ready {
    display: grid;
    gap: 14px;
}

/* 取景器：渐变描边玻璃相框 + 固定深色画面（明暗主题不翻转） */
.mobile-camera__viewport {
    background:
        linear-gradient(var(--join-glass-bg-heavy), var(--join-glass-bg-heavy))
            padding-box,
        var(--join-gradient-cta) border-box;
    border: 1px solid transparent;
    border-radius: var(--join-radius-card);
    box-shadow: var(--shadow-card);
    padding: 6px;
    position: relative;
}

.mobile-camera__video {
    aspect-ratio: 3 / 4;
    background: #10121d;
    border-radius: calc(var(--join-radius-card) - 6px);
    display: block;
    max-height: 56dvh;
    object-fit: cover;
    width: 100%;
}

/* 对角取景框角标：纯装饰，固定浅色以浮在深色画面上 */
.mobile-camera__viewfinder {
    inset: 20px;
    pointer-events: none;
    position: absolute;
}

.mobile-camera__viewfinder::before,
.mobile-camera__viewfinder::after {
    content: "";
    height: 22px;
    position: absolute;
    width: 22px;
}

.mobile-camera__viewfinder::before {
    border-left: 2px solid rgba(255, 255, 255, 0.55);
    border-top: 2px solid rgba(255, 255, 255, 0.55);
    border-top-left-radius: 6px;
    left: 0;
    top: 0;
}

.mobile-camera__viewfinder::after {
    border-bottom: 2px solid rgba(255, 255, 255, 0.55);
    border-right: 2px solid rgba(255, 255, 255, 0.55);
    border-bottom-right-radius: 6px;
    bottom: 0;
    right: 0;
}

.mobile-camera__preview {
    aspect-ratio: 3 / 4;
    background: var(--join-chip-bg);
    border: 1px solid var(--join-glass-border);
    border-radius: var(--join-radius-card);
    max-height: 56dvh;
    object-fit: contain;
    width: 100%;
}

/* ── 操作区：整列堆叠，单一主 CTA ───────────────── */
.mobile-camera__actions {
    display: grid;
    gap: 10px;
    width: 100%;
}

/* 入口动作升级为渐变 CTA（保留 .secondary-button 测试选择器） */
.mobile-camera__open.secondary-button:not(:disabled) {
    background: var(--join-gradient-cta);
    border-color: transparent;
    box-shadow: var(--join-cta-glow);
    color: #ffffff;
}

.mobile-camera__open.secondary-button:hover:not(:disabled) {
    background: var(--join-gradient-cta);
    border-color: transparent;
    box-shadow: var(--join-cta-glow-hover);
    filter: brightness(1.06);
}

/* ── 终态屏：色调图标气泡 + 居中文案 ────────────── */
.mobile-camera__state {
    display: grid;
    gap: 12px;
    justify-items: center;
    padding: 18px 4px 8px;
    text-align: center;
}

.mobile-camera__state-icon {
    align-items: center;
    border-radius: 999px;
    display: flex;
    height: 56px;
    justify-content: center;
    width: 56px;
}

.mobile-camera__state-glyph {
    height: 26px;
    width: 26px;
}

.mobile-camera__state-title {
    color: var(--join-ink);
    font-size: 20px;
    font-weight: 700;
    line-height: 28px;
    margin: 0;
}

.mobile-camera__state-text {
    color: var(--join-ink-soft);
    font-size: 14px;
    line-height: 22px;
    margin: 0;
    max-width: 36ch;
}

.mobile-camera__state .mobile-camera__actions {
    margin-top: 8px;
}

/* ── 错误反馈气泡 ───────────────────────────────── */
.mobile-camera__error {
    background: var(--join-tone-danger-bg);
    border-radius: var(--radius-lg);
    color: var(--join-tone-danger-fg);
    font-size: 13px;
    font-weight: 600;
    line-height: 20px;
    margin: 14px 0 0;
    padding: 10px 14px;
}

/* ── ≥640px：留白放大 ──────────────────────────── */
@media (min-width: 640px) {
    .mobile-camera {
        padding: 28px 16px max(36px, env(safe-area-inset-bottom));
    }

    .mobile-camera__panel {
        padding: 24px;
    }
}
</style>
