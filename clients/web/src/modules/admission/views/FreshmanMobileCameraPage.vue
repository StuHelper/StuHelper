<template>
    <main class="min-h-screen bg-slate-50 px-4 py-6 text-slate-950">
        <section class="mx-auto grid w-full max-w-xl gap-4">
            <header
                class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
            >
                <p class="text-sm font-medium text-slate-500">StuHelper</p>
                <h1 class="mt-2 text-2xl font-semibold">新生材料拍照</h1>
            </header>

            <section
                class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
            >
                <div v-if="pageState === 'loading'" data-state="loading">
                    <p class="text-sm text-slate-600">
                        正在打开手机拍照链接...
                    </p>
                </div>

                <div
                    v-else-if="pageState === 'ready'"
                    class="grid gap-4"
                    data-state="ready"
                >
                    <video
                        ref="videoRef"
                        autoplay
                        class="aspect-video w-full rounded-lg bg-slate-950 object-cover"
                        muted
                        playsinline
                    />

                    <img
                        v-if="previewDataURL"
                        :src="previewDataURL"
                        alt="录取材料预览"
                        class="aspect-video w-full rounded-lg border border-slate-200 object-contain"
                    />

                    <div class="flex flex-wrap gap-3">
                        <button
                            v-if="!streamActive"
                            class="secondary-button"
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
                    class="grid gap-4"
                    data-state="uploaded"
                >
                    <h2 class="text-lg font-semibold">材料已上传</h2>
                    <p class="text-sm text-slate-600">
                        请选择后续在哪一端继续。选择后另一端会被锁定，避免重复提交。
                    </p>
                    <div class="flex flex-wrap gap-3">
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

                <div v-else-if="pageState === 'desktop'" data-state="desktop">
                    <h2 class="text-lg font-semibold">请回到电脑端继续</h2>
                    <p class="mt-2 text-sm text-slate-600">
                        材料已上传，电脑端会自动刷新状态。
                    </p>
                </div>

                <div v-else-if="pageState === 'mobile'" data-state="mobile">
                    <h2 class="text-lg font-semibold">材料已提交</h2>
                    <p class="mt-2 text-sm text-slate-600">
                        请等待管理员审核。
                    </p>
                </div>

                <div v-else-if="pageState === 'expired'" data-state="expired">
                    <h2 class="text-lg font-semibold">链接已过期</h2>
                    <p class="mt-2 text-sm text-slate-600">
                        请回到电脑端重新生成手机拍照二维码。
                    </p>
                </div>

                <div v-else data-state="error">
                    <h2 class="text-lg font-semibold">无法打开拍照链接</h2>
                    <p class="mt-2 text-sm text-slate-600">
                        {{ errorMessage }}
                    </p>
                </div>

                <p
                    v-if="errorMessage && pageState !== 'error'"
                    class="mt-4 text-sm text-red-600"
                >
                    {{ errorMessage }}
                </p>
            </section>
        </section>
    </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
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

onMounted(() => {
    void loadHandoff();
});

async function loadHandoff(): Promise<void> {
    loading.value = true;
    errorMessage.value = "";
    try {
        handoff.value = await admissionApi.previewFreshmanMobileCameraHandoff(
            token.value,
        );
    } catch (error) {
        if (handleAdmissionExpiredError(error)) return;
        errorMessage.value = readErrorMessage(error, "拍照链接暂时无法打开。");
        handoff.value = null;
    } finally {
        loading.value = false;
    }
}

async function openCamera(): Promise<void> {
    errorMessage.value = "";
    let openedStream: MediaStream | null = null;
    try {
        openedStream = await startCameraStream();
        stream.value = openedStream;
        if (videoRef.value) {
            videoRef.value.srcObject = stream.value;
            await videoRef.value.play();
        }
    } catch (error) {
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
    uploading.value = true;
    errorMessage.value = "";
    try {
        handoff.value = await admissionApi.uploadFreshmanMobileCameraCapture(
            token.value,
            capturedPayload.value,
        );
        stopCameraStream(stream.value);
        stream.value = null;
    } catch (error) {
        if (handleAdmissionExpiredError(error)) return;
        errorMessage.value = readErrorMessage(error, "材料上传失败。");
    } finally {
        uploading.value = false;
    }
}

async function chooseContinuation(
    continueOn: FreshmanCameraHandoffContinuationRequest["continueOn"],
): Promise<void> {
    if (choosing.value) return;
    choosing.value = true;
    errorMessage.value = "";
    try {
        handoff.value =
            await admissionApi.chooseFreshmanMobileCameraContinuation(
                token.value,
                { continueOn },
            );
    } catch (error) {
        if (handleAdmissionExpiredError(error)) return;
        errorMessage.value = readErrorMessage(error, "继续方式保存失败。");
    } finally {
        choosing.value = false;
    }
}

function readErrorMessage(error: unknown, fallback: string): string {
    return error instanceof Error && error.message ? error.message : fallback;
}

function handleAdmissionExpiredError(error: unknown): boolean {
    if (!isAdmissionSessionExpiredError(error)) return false;
    forceExpired.value = true;
    errorMessage.value = "";
    stopCameraStream(stream.value);
    stream.value = null;
    return true;
}

onBeforeUnmount(() => {
    stopCameraStream(stream.value);
});
</script>
