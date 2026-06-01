// @vitest-environment jsdom

import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import FreshmanMobileCameraPage from "../views/FreshmanMobileCameraPage.vue";

const mockAdmissionApi = vi.hoisted(() => ({
    previewFreshmanMobileCameraHandoff: vi.fn(),
    uploadFreshmanMobileCameraCapture: vi.fn(),
    chooseFreshmanMobileCameraContinuation: vi.fn(),
}));

const mockCaptureFrameAsBase64 = vi.hoisted(() => vi.fn());
const mockStartCameraStream = vi.hoisted(() => vi.fn());
const mockStopCameraStream = vi.hoisted(() => vi.fn());

vi.mock("../api", () => ({
    admissionApi: mockAdmissionApi,
}));

vi.mock("vue-router", () => ({
    useRoute: () => ({
        params: {
            token: "mobile-token",
        },
    }),
}));

vi.mock("../cameraCapture", () => ({
    captureFrameAsBase64: mockCaptureFrameAsBase64,
    startCameraStream: mockStartCameraStream,
    stopCameraStream: mockStopCameraStream,
    supportsCameraCapture: () => true,
}));

const pendingHandoff = {
    id: "handoff-1",
    applicationID: "application-1",
    userID: "42",
    status: "pending",
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

describe("FreshmanMobileCameraPage", () => {
    beforeEach(() => {
        vi.clearAllMocks();
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
});
