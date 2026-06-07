// @vitest-environment jsdom

import { flushPromises, mount } from "@vue/test-utils";
import { nextTick, reactive } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/api/errors";
import FreshmanMobileCameraPage from "../views/FreshmanMobileCameraPage.vue";

const mockAdmissionApi = vi.hoisted(() => ({
    previewFreshmanMobileCameraHandoff: vi.fn(),
    uploadFreshmanMobileCameraCapture: vi.fn(),
    chooseFreshmanMobileCameraContinuation: vi.fn(),
}));

const mockCaptureFrameAsBase64 = vi.hoisted(() => vi.fn());
const mockStartCameraStream = vi.hoisted(() => vi.fn());
const mockStopCameraStream = vi.hoisted(() => vi.fn());
const mockRouteContainer = vi.hoisted(() => ({
    route: null as null | { params: { token: string } },
}));

vi.mock("../api", () => ({
    admissionApi: mockAdmissionApi,
}));

vi.mock("vue-router", async () => {
    const { reactive } = await vi.importActual<typeof import("vue")>("vue");
    mockRouteContainer.route = reactive({
        params: {
            token: "mobile-token",
        },
    });
    return {
        useRoute: () => mockRouteContainer.route,
    };
});

vi.mock("../cameraCapture", () => ({
    captureFrameAsBase64: mockCaptureFrameAsBase64,
    describeCameraCaptureError: (error: unknown, fallback: string) => {
        const message = error instanceof Error ? error.message : "";
        if (/capture exceeds the admission material size limit/i.test(message)) {
            return "拍摄图片超过材料大小限制。请调整距离重新拍摄，或使用更低分辨率的设备重试。";
        }
        return message || fallback;
    },
    startCameraStream: mockStartCameraStream,
    stopCameraStream: mockStopCameraStream,
    supportsCameraCapture: () => true,
}));

const pendingHandoff = {
    id: "handoff-1",
    applicationID: "application-1",
    userID: "42",
    status: "pending",
    maxMaterialBytes: 1024,
    mobileURL:
        "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
    expiresAt: "2026-06-01T10:30:00Z",
    createdAt: "2026-06-01T10:00:00Z",
};

const uploadedHandoff = {
    ...pendingHandoff,
    status: "uploaded",
    uploadedAt: "2026-06-01T10:05:00Z",
};

function createDeferred<T>() {
    let resolve!: (value: T) => void;
    let reject!: (reason?: unknown) => void;
    const promise = new Promise<T>((promiseResolve, promiseReject) => {
        resolve = promiseResolve;
        reject = promiseReject;
    });
    return { promise, reject, resolve };
}

function setRouteToken(token: string): void {
    if (!mockRouteContainer.route) {
        throw new Error("Route mock is not initialized");
    }
    mockRouteContainer.route.params.token = token;
}

describe("FreshmanMobileCameraPage", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockRouteContainer.route = reactive({
            params: {
                token: "mobile-token",
            },
        });
        mockAdmissionApi.previewFreshmanMobileCameraHandoff.mockResolvedValue(
            pendingHandoff,
        );
        mockAdmissionApi.uploadFreshmanMobileCameraCapture.mockResolvedValue(
            uploadedHandoff,
        );
        mockAdmissionApi.chooseFreshmanMobileCameraContinuation.mockResolvedValue(
            {
                ...uploadedHandoff,
                status: "locked",
                continueOn: "mobile",
                chosenAt: "2026-06-01T10:06:00Z",
            },
        );
        mockStartCameraStream.mockResolvedValue({ id: "stream-1" });
        mockCaptureFrameAsBase64.mockReturnValue({
            contentType: "image/jpeg",
            imageBase64: "AA==",
        });
        vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(
            undefined,
        );
    });

    it("lets a phone preview, capture, upload, and choose continuation by token", async () => {
        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        expect(
            mockAdmissionApi.previewFreshmanMobileCameraHandoff,
        ).toHaveBeenCalledWith("mobile-token");
        expect(wrapper.find('[data-state="ready"]').exists()).toBe(true);

        await wrapper.find("[data-mobile-camera-open-button]").trigger("click");
        await flushPromises();
        await wrapper
            .find("[data-mobile-camera-capture-button]")
            .trigger("click");
        await wrapper
            .find("[data-mobile-camera-upload-button]")
            .trigger("click");
        await flushPromises();

        expect(
            mockAdmissionApi.uploadFreshmanMobileCameraCapture,
        ).toHaveBeenCalledWith("mobile-token", {
            contentType: "image/jpeg",
            imageBase64: "AA==",
        });
        expect(mockStopCameraStream).toHaveBeenCalledWith({ id: "stream-1" });
        expect(wrapper.find('[data-state="uploaded"]').exists()).toBe(true);

        await wrapper
            .find("[data-mobile-camera-continue-mobile-button]")
            .trigger("click");
        await flushPromises();

        expect(
            mockAdmissionApi.chooseFreshmanMobileCameraContinuation,
        ).toHaveBeenCalledWith("mobile-token", { continueOn: "mobile" });
        expect(wrapper.find('[data-state="mobile"]').exists()).toBe(true);
    });

    it("lets the phone choose to continue on the desktop after upload", async () => {
        mockAdmissionApi.chooseFreshmanMobileCameraContinuation.mockResolvedValue(
            {
                ...uploadedHandoff,
                status: "locked",
                continueOn: "desktop",
                chosenAt: "2026-06-01T10:06:00Z",
            },
        );

        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        await wrapper.find("[data-mobile-camera-open-button]").trigger("click");
        await flushPromises();
        await wrapper
            .find("[data-mobile-camera-capture-button]")
            .trigger("click");
        await wrapper
            .find("[data-mobile-camera-upload-button]")
            .trigger("click");
        await flushPromises();

        await wrapper
            .find("[data-mobile-camera-continue-desktop-button]")
            .trigger("click");
        await flushPromises();

        expect(
            mockAdmissionApi.chooseFreshmanMobileCameraContinuation,
        ).toHaveBeenCalledWith("mobile-token", { continueOn: "desktop" });
        expect(wrapper.find('[data-state="desktop"]').exists()).toBe(true);
    });

    it("prevents duplicate mobile uploads while the first upload is pending", async () => {
        const uploadDeferred = createDeferred<typeof uploadedHandoff>();
        mockAdmissionApi.uploadFreshmanMobileCameraCapture.mockReturnValue(
            uploadDeferred.promise,
        );

        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        await wrapper.find("[data-mobile-camera-open-button]").trigger("click");
        await flushPromises();
        await wrapper
            .find("[data-mobile-camera-capture-button]")
            .trigger("click");

        const uploadButton = wrapper.find("[data-mobile-camera-upload-button]");
        await uploadButton.trigger("click");
        await uploadButton.trigger("click");
        await flushPromises();

        expect(
            mockAdmissionApi.uploadFreshmanMobileCameraCapture,
        ).toHaveBeenCalledTimes(1);
        expect(
            wrapper.find<HTMLButtonElement>(
                "[data-mobile-camera-upload-button]",
            ).element.disabled,
        ).toBe(true);

        uploadDeferred.resolve(uploadedHandoff);
        await flushPromises();

        expect(wrapper.find('[data-state="uploaded"]').exists()).toBe(true);
    });

    it("reloads the current mobile token and ignores stale preview responses", async () => {
        const stalePreview = createDeferred<typeof pendingHandoff>();
        const currentPreview = createDeferred<typeof uploadedHandoff>();
        mockAdmissionApi.previewFreshmanMobileCameraHandoff
            .mockReturnValueOnce(stalePreview.promise)
            .mockReturnValueOnce(currentPreview.promise);

        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        expect(
            mockAdmissionApi.previewFreshmanMobileCameraHandoff,
        ).toHaveBeenCalledWith("mobile-token");

        setRouteToken("next-token");
        await nextTick();

        expect(
            mockAdmissionApi.previewFreshmanMobileCameraHandoff,
        ).toHaveBeenCalledWith("next-token");

        currentPreview.resolve({
            ...uploadedHandoff,
            id: "handoff-next",
            mobileURL:
                "https://join.stuhelper.com/admission/freshman/camera/next-token",
        });
        await flushPromises();

        expect(wrapper.find('[data-state="uploaded"]').exists()).toBe(true);

        stalePreview.resolve({
            ...pendingHandoff,
            id: "handoff-stale",
        });
        await flushPromises();

        expect(wrapper.find('[data-state="uploaded"]').exists()).toBe(true);
        expect(wrapper.find('[data-state="ready"]').exists()).toBe(false);
    });

    it("ignores stale mobile upload responses after switching tokens", async () => {
        const uploadDeferred = createDeferred<typeof uploadedHandoff>();
        mockAdmissionApi.uploadFreshmanMobileCameraCapture.mockReturnValueOnce(
            uploadDeferred.promise,
        );

        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        await wrapper.find("[data-mobile-camera-open-button]").trigger("click");
        await flushPromises();
        await wrapper
            .find("[data-mobile-camera-capture-button]")
            .trigger("click");
        await wrapper
            .find("[data-mobile-camera-upload-button]")
            .trigger("click");
        await flushPromises();

        expect(
            mockAdmissionApi.uploadFreshmanMobileCameraCapture,
        ).toHaveBeenCalledWith("mobile-token", {
            contentType: "image/jpeg",
            imageBase64: "AA==",
        });

        setRouteToken("next-token");
        await nextTick();
        await flushPromises();

        expect(
            mockAdmissionApi.previewFreshmanMobileCameraHandoff,
        ).toHaveBeenCalledWith("next-token");
        expect(wrapper.find('[data-state="ready"]').exists()).toBe(true);

        uploadDeferred.resolve(uploadedHandoff);
        await flushPromises();

        expect(wrapper.find('[data-state="ready"]').exists()).toBe(true);
        expect(wrapper.find('[data-state="uploaded"]').exists()).toBe(false);
    });

    it("closes the acquired camera stream when video playback fails", async () => {
        vi.spyOn(HTMLMediaElement.prototype, "play").mockRejectedValueOnce(
            new Error("play blocked"),
        );

        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        await wrapper.find("[data-mobile-camera-open-button]").trigger("click");
        await flushPromises();

        expect(mockStartCameraStream).toHaveBeenCalledTimes(1);
        expect(mockStopCameraStream).toHaveBeenCalledWith({ id: "stream-1" });
        expect(wrapper.find("[data-mobile-camera-open-button]").exists()).toBe(
            true,
        );
        expect(
            wrapper.find("[data-mobile-camera-capture-button]").exists(),
        ).toBe(false);
        expect(wrapper.text()).toContain("play blocked");
    });

    it("prevents duplicate continuation choices while the first choice is pending", async () => {
        const continuationDeferred = createDeferred<{
            status: string;
            continueOn: string;
        }>();
        mockAdmissionApi.chooseFreshmanMobileCameraContinuation.mockReturnValue(
            continuationDeferred.promise,
        );

        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        await wrapper.find("[data-mobile-camera-open-button]").trigger("click");
        await flushPromises();
        await wrapper
            .find("[data-mobile-camera-capture-button]")
            .trigger("click");
        await wrapper
            .find("[data-mobile-camera-upload-button]")
            .trigger("click");
        await flushPromises();

        const continueButton = wrapper.find(
            "[data-mobile-camera-continue-mobile-button]",
        );
        await continueButton.trigger("click");
        await continueButton.trigger("click");
        await flushPromises();

        expect(
            mockAdmissionApi.chooseFreshmanMobileCameraContinuation,
        ).toHaveBeenCalledTimes(1);
        expect(
            mockAdmissionApi.chooseFreshmanMobileCameraContinuation,
        ).toHaveBeenCalledWith("mobile-token", { continueOn: "mobile" });
        expect(
            wrapper.find<HTMLButtonElement>(
                "[data-mobile-camera-continue-mobile-button]",
            ).element.disabled,
        ).toBe(true);

        continuationDeferred.resolve({
            ...uploadedHandoff,
            status: "locked",
            continueOn: "mobile",
            chosenAt: "2026-06-01T10:06:00Z",
        });
        await flushPromises();

        expect(wrapper.find('[data-state="mobile"]').exists()).toBe(true);
    });

    it("shows expired state when the admission session expires during mobile upload", async () => {
        mockAdmissionApi.uploadFreshmanMobileCameraCapture.mockRejectedValueOnce(
            new ApiError({ code: "admission.token_expired", message: "expired" }),
        );

        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        await wrapper.find("[data-mobile-camera-open-button]").trigger("click");
        await flushPromises();
        await wrapper
            .find("[data-mobile-camera-capture-button]")
            .trigger("click");
        await wrapper
            .find("[data-mobile-camera-upload-button]")
            .trigger("click");
        await flushPromises();

        expect(wrapper.find('[data-state="expired"]').exists()).toBe(true);
        expect(wrapper.text()).toContain("链接已过期");
        expect(mockStopCameraStream).toHaveBeenCalledWith({ id: "stream-1" });
    });

    it("uses the handoff material size limit when capturing on mobile", async () => {
        mockCaptureFrameAsBase64.mockImplementationOnce(() => {
            throw new Error(
                "Camera capture exceeds the admission material size limit",
            );
        });

        const wrapper = mount(FreshmanMobileCameraPage);
        await flushPromises();

        await wrapper.find("[data-mobile-camera-open-button]").trigger("click");
        await flushPromises();
        await wrapper
            .find("[data-mobile-camera-capture-button]")
            .trigger("click");

        expect(mockCaptureFrameAsBase64).toHaveBeenCalledWith(
            expect.any(HTMLVideoElement),
            { maxBytes: 1024 },
        );
        expect(wrapper.text()).toContain("拍摄图片超过材料大小限制");
        expect(
            wrapper.find<HTMLButtonElement>(
                "[data-mobile-camera-upload-button]",
            ).element.disabled,
        ).toBe(true);
    });
});
