// @vitest-environment jsdom

import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "@/api/errors";
import {
    buildCameraConstraints,
    captureFrameAsBase64,
    describeCameraCaptureError,
    supportsCameraCapture,
} from "../cameraCapture";
import FreshmanCameraFlow from "../views/FreshmanCameraFlow.vue";

const mockAdmissionApi = vi.hoisted(() => ({
    submitFreshmanApplication: vi.fn(),
    uploadCameraCapture: vi.fn(),
    createFreshmanCameraHandoff: vi.fn(),
    getFreshmanCameraHandoff: vi.fn(),
}));

vi.mock("../api", () => ({
    admissionApi: mockAdmissionApi,
}));

vi.mock("qrcode", () => ({
    toDataURL: vi.fn().mockResolvedValue("data:image/png;base64,AA=="),
}));

const freshmanApplication = {
    id: "application-1",
    userID: "42",
    schoolID: 4111010006,
    applicantNameMasked: "张*",
    materialType: "admission_notice",
    status: "pending",
    createdAt: "2026-06-01T10:00:00Z",
};

const freshmanHandoff = {
    id: "handoff-1",
    applicationID: "application-1",
    userID: "42",
    status: "pending",
    mobileURL:
        "https://join.stuhelper.com/admission/freshman/camera/mobile-token",
    expiresAt: "2026-06-01T10:30:00Z",
    createdAt: "2026-06-01T10:00:00Z",
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

describe("camera capture helpers", () => {
    it("reports unsupported when getUserMedia is unavailable", () => {
        expect(
            supportsCameraCapture({ mediaDevices: undefined } as Navigator),
        ).toBe(false);
    });

    it("requests an environment-facing camera", () => {
        expect(buildCameraConstraints()).toEqual({
            audio: false,
            video: {
                facingMode: { ideal: "environment" },
            },
        });
    });

    it("maps browser camera permission errors to actionable Chinese copy", () => {
        expect(
            describeCameraCaptureError(
                new DOMException("Permission denied", "NotAllowedError"),
                "无法打开摄像头。",
            ),
        ).toContain("摄像头权限被浏览器拒绝");
    });

    it("fails when the browser cannot provide a canvas 2D context", () => {
        const video = document.createElement("video");
        const originalCreateElement = document.createElement.bind(document);
        const createElement = vi.spyOn(document, "createElement");
        Object.defineProperty(video, "videoWidth", { value: 1 });
        Object.defineProperty(video, "videoHeight", { value: 1 });
        createElement.mockImplementation((tagName) => {
            if (tagName !== "canvas") return originalCreateElement(tagName);
            return {
                height: 0,
                width: 0,
                getContext: () => null,
                toDataURL: () => "data:image/jpeg;base64,AAAA",
            } as unknown as HTMLCanvasElement;
        });

        expect(() => captureFrameAsBase64(video)).toThrow(
            "Canvas 2D context unavailable",
        );
        createElement.mockRestore();
    });
});

describe("FreshmanCameraFlow material capture UI", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("renders freshman camera flow without a file input", async () => {
        const wrapper = mount(FreshmanCameraFlow, {
            props: {
                maxMaterialBytes: 1024,
                schools: [
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "测试大学",
                        verificationMethod: "manual",
                        enabled: true,
                    },
                ],
            },
        });
        await flushPromises();

        expect(wrapper.find("[data-admission-freshman-flow]").exists()).toBe(
            true,
        );
        expect(wrapper.find('input[type="file"]').exists()).toBe(false);
        expect(wrapper.find("[data-freshman-school-select]").exists()).toBe(
            true,
        );
    });

    it("creates freshman applications by school code and watches handoff status over SSE", async () => {
        const originalEventSource = window.EventSource;
        class FakeEventSource {
            readonly url: string;
            readonly listeners = new Map<
                string,
                Array<(event: MessageEvent) => void>
            >();
            close = vi.fn();

            constructor(url: string) {
                this.url = url;
                sources.push(this);
            }

            addEventListener(
                type: string,
                listener: (event: MessageEvent) => void,
            ) {
                const listeners = this.listeners.get(type) ?? [];
                listeners.push(listener);
                this.listeners.set(type, listeners);
            }

            emit(type: string, data: unknown) {
                const event = new MessageEvent(type, {
                    data: JSON.stringify(data),
                });
                for (const listener of this.listeners.get(type) ?? []) {
                    listener(event);
                }
            }
        }
        const sources: FakeEventSource[] = [];
        Object.defineProperty(window, "EventSource", {
            configurable: true,
            value: FakeEventSource as unknown as typeof EventSource,
        });

        try {
            mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(
                freshmanApplication,
            );
            mockAdmissionApi.createFreshmanCameraHandoff.mockResolvedValue(
                freshmanHandoff,
            );
            const wrapper = mount(FreshmanCameraFlow, {
                props: {
                    admissionSessionId: "session-1",
                    maxMaterialBytes: 1024,
                    schools: [
                        {
                            schoolID: 4111010006,
                            schoolCode: "4111010006",
                            schoolName: "北京航空航天大学",
                            verificationMethod: "manual",
                            enabled: true,
                        },
                    ],
                },
            });
            await flushPromises();
            await wrapper
                .find("[data-freshman-school-select]")
                .setValue("4111010006");
            await wrapper
                .find("[data-freshman-applicant-name-input]")
                .setValue("张三");
            await flushPromises();
            await wrapper
                .find("[data-freshman-mobile-handoff-button]")
                .trigger("click");
            await flushPromises();

            expect(
                mockAdmissionApi.submitFreshmanApplication,
            ).toHaveBeenCalledWith({
                schoolCode: "4111010006",
                admissionSessionID: "session-1",
                applicantName: "张三",
                departmentOrMajor: undefined,
                materialType: "admission_notice",
            });
            expect(sources[0]?.url).toBe(
                "/api/v1/admission/freshman/camera-handoffs/handoff-1/events",
            );

            sources[0]?.emit("handoff", {
                ...freshmanHandoff,
                status: "uploaded",
                uploadedAt: "2026-06-01T10:05:00Z",
            });
            await flushPromises();

            expect(wrapper.text()).toContain("手机已上传材料");

            sources[0]?.emit("handoff", {
                ...freshmanHandoff,
                status: "locked",
                continueOn: "desktop",
                uploadedAt: "2026-06-01T10:05:00Z",
                chosenAt: "2026-06-01T10:06:00Z",
            });
            await flushPromises();

            expect(wrapper.text()).toContain("流程已切回电脑端");
            expect(wrapper.emitted("submitted")?.[0]).toEqual([
                freshmanApplication,
            ]);
        } finally {
            Object.defineProperty(window, "EventSource", {
                configurable: true,
                value: originalEventSource,
            });
        }
    });

    it("ignores stale SSE events after regenerating a mobile handoff", async () => {
        const originalEventSource = window.EventSource;
        class FakeEventSource {
            readonly url: string;
            readonly listeners = new Map<
                string,
                Array<(event: MessageEvent) => void>
            >();
            close = vi.fn();

            constructor(url: string) {
                this.url = url;
                sources.push(this);
            }

            addEventListener(
                type: string,
                listener: (event: MessageEvent) => void,
            ) {
                const listeners = this.listeners.get(type) ?? [];
                listeners.push(listener);
                this.listeners.set(type, listeners);
            }

            emit(type: string, data: unknown) {
                const event = new MessageEvent(type, {
                    data: JSON.stringify(data),
                });
                for (const listener of this.listeners.get(type) ?? []) {
                    listener(event);
                }
            }
        }
        const sources: FakeEventSource[] = [];
        Object.defineProperty(window, "EventSource", {
            configurable: true,
            value: FakeEventSource as unknown as typeof EventSource,
        });

        try {
            const secondHandoff = {
                ...freshmanHandoff,
                id: "handoff-2",
                mobileURL:
                    "https://join.stuhelper.com/admission/freshman/camera/mobile-token-2",
            };
            mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(
                freshmanApplication,
            );
            mockAdmissionApi.createFreshmanCameraHandoff
                .mockResolvedValueOnce(freshmanHandoff)
                .mockResolvedValueOnce(secondHandoff);
            const wrapper = mount(FreshmanCameraFlow, {
                props: {
                    maxMaterialBytes: 1024,
                    schools: [
                        {
                            schoolID: 4111010006,
                            schoolCode: "4111010006",
                            schoolName: "北京航空航天大学",
                            verificationMethod: "manual",
                            enabled: true,
                        },
                    ],
                },
            });
            await flushPromises();
            await wrapper
                .find("[data-freshman-school-select]")
                .setValue("4111010006");
            await wrapper
                .find("[data-freshman-applicant-name-input]")
                .setValue("张三");
            await wrapper
                .find("[data-freshman-mobile-handoff-button]")
                .trigger("click");
            await flushPromises();

            sources[0]?.emit("handoff", {
                ...freshmanHandoff,
                status: "expired",
            });
            await flushPromises();
            expect(sources[0]?.close).toHaveBeenCalled();

            await wrapper
                .find("[data-freshman-mobile-handoff-button]")
                .trigger("click");
            await flushPromises();
            expect(sources).toHaveLength(2);
            expect(sources[1]?.url).toBe(
                "/api/v1/admission/freshman/camera-handoffs/handoff-2/events",
            );

            sources[0]?.emit("handoff", {
                ...freshmanHandoff,
                status: "locked",
                continueOn: "desktop",
                uploadedAt: "2026-06-01T10:05:00Z",
                chosenAt: "2026-06-01T10:06:00Z",
            });
            await flushPromises();

            expect(wrapper.text()).toContain("请用手机扫描二维码");
            expect(wrapper.text()).not.toContain("流程已切回电脑端");
            expect(wrapper.emitted("submitted")).toBeUndefined();
        } finally {
            Object.defineProperty(window, "EventSource", {
                configurable: true,
                value: originalEventSource,
            });
        }
    });

    it("does not create duplicate applications or handoffs while mobile handoff is pending", async () => {
        const applicationDeferred =
            createDeferred<typeof freshmanApplication>();
        const handoffDeferred = createDeferred<typeof freshmanHandoff>();
        mockAdmissionApi.submitFreshmanApplication.mockReturnValue(
            applicationDeferred.promise,
        );
        mockAdmissionApi.createFreshmanCameraHandoff.mockReturnValue(
            handoffDeferred.promise,
        );
        const wrapper = mount(FreshmanCameraFlow, {
            props: {
                maxMaterialBytes: 1024,
                schools: [
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        enabled: true,
                    },
                ],
            },
        });
        await flushPromises();
        await wrapper
            .find("[data-freshman-school-select]")
            .setValue("4111010006");
        await wrapper
            .find("[data-freshman-applicant-name-input]")
            .setValue("张三");

        const button = wrapper.find<HTMLButtonElement>(
            "[data-freshman-mobile-handoff-button]",
        );
        await button.trigger("click");
        await flushPromises();

        expect(button.element.disabled).toBe(true);
        expect(button.text()).toContain("生成中");
        expect(
            mockAdmissionApi.submitFreshmanApplication,
        ).toHaveBeenCalledTimes(1);
        expect(
            mockAdmissionApi.createFreshmanCameraHandoff,
        ).not.toHaveBeenCalled();

        await button.trigger("click");
        await flushPromises();

        expect(
            mockAdmissionApi.submitFreshmanApplication,
        ).toHaveBeenCalledTimes(1);
        expect(
            mockAdmissionApi.createFreshmanCameraHandoff,
        ).not.toHaveBeenCalled();

        applicationDeferred.resolve(freshmanApplication);
        await flushPromises();

        expect(
            mockAdmissionApi.createFreshmanCameraHandoff,
        ).toHaveBeenCalledTimes(1);
        expect(
            mockAdmissionApi.createFreshmanCameraHandoff,
        ).toHaveBeenCalledWith("application-1");

        await button.trigger("click");
        await flushPromises();

        expect(
            mockAdmissionApi.submitFreshmanApplication,
        ).toHaveBeenCalledTimes(1);
        expect(
            mockAdmissionApi.createFreshmanCameraHandoff,
        ).toHaveBeenCalledTimes(1);

        handoffDeferred.resolve(freshmanHandoff);
        await flushPromises();

        expect(
            mockAdmissionApi.submitFreshmanApplication,
        ).toHaveBeenCalledTimes(1);
        expect(
            mockAdmissionApi.createFreshmanCameraHandoff,
        ).toHaveBeenCalledTimes(1);
        expect(wrapper.find("[data-freshman-mobile-handoff]").exists()).toBe(
            true,
        );

        wrapper.unmount();
    });

    it("emits expired when the linked admission session expires while creating material flow", async () => {
        mockAdmissionApi.submitFreshmanApplication.mockRejectedValueOnce(
            new ApiError({ code: "admission.token_expired", message: "expired" }),
        );
        const wrapper = mount(FreshmanCameraFlow, {
            props: {
                maxMaterialBytes: 1024,
                schools: [
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        enabled: true,
                    },
                ],
            },
        });
        await flushPromises();
        await wrapper
            .find("[data-freshman-school-select]")
            .setValue("4111010006");
        await wrapper
            .find("[data-freshman-applicant-name-input]")
            .setValue("张三");
        await wrapper
            .find("[data-freshman-mobile-handoff-button]")
            .trigger("click");
        await flushPromises();

        expect(wrapper.emitted("expired")).toHaveLength(1);
        expect(wrapper.text()).not.toContain("手机拍照链接生成失败");
    });

    it("blocks desktop submission after mobile upload while continuation is undecided", async () => {
        const originalEventSource = window.EventSource;
        const mediaDevicesDescriptor = Object.getOwnPropertyDescriptor(
            navigator,
            "mediaDevices",
        );
        const videoWidthDescriptor = Object.getOwnPropertyDescriptor(
            HTMLVideoElement.prototype,
            "videoWidth",
        );
        const videoHeightDescriptor = Object.getOwnPropertyDescriptor(
            HTMLVideoElement.prototype,
            "videoHeight",
        );
        class FakeEventSource {
            readonly listeners = new Map<
                string,
                Array<(event: MessageEvent) => void>
            >();

            constructor() {
                sources.push(this);
            }

            addEventListener(
                type: string,
                listener: (event: MessageEvent) => void,
            ) {
                const listeners = this.listeners.get(type) ?? [];
                listeners.push(listener);
                this.listeners.set(type, listeners);
            }

            close = vi.fn();

            emit(type: string, data: unknown) {
                const event = new MessageEvent(type, {
                    data: JSON.stringify(data),
                });
                for (const listener of this.listeners.get(type) ?? []) {
                    listener(event);
                }
            }
        }
        const sources: FakeEventSource[] = [];
        const stopTrack = vi.fn();
        Object.defineProperty(navigator, "mediaDevices", {
            configurable: true,
            value: {
                getUserMedia: vi.fn().mockResolvedValue({
                    getTracks: () => [{ stop: stopTrack }],
                }),
            },
        });
        Object.defineProperty(HTMLVideoElement.prototype, "videoWidth", {
            configurable: true,
            get: () => 2,
        });
        Object.defineProperty(HTMLVideoElement.prototype, "videoHeight", {
            configurable: true,
            get: () => 2,
        });
        Object.defineProperty(window, "EventSource", {
            configurable: true,
            value: FakeEventSource as unknown as typeof EventSource,
        });
        const play = vi
            .spyOn(HTMLMediaElement.prototype, "play")
            .mockResolvedValue(undefined);
        const getContext = vi
            .spyOn(HTMLCanvasElement.prototype, "getContext")
            .mockImplementation(((contextID: string) => {
                if (contextID !== "2d") return null;
                return {
                    drawImage: vi.fn(),
                } as unknown as CanvasRenderingContext2D;
            }) as HTMLCanvasElement["getContext"]);
        const toDataURL = vi
            .spyOn(HTMLCanvasElement.prototype, "toDataURL")
            .mockReturnValue("data:image/jpeg;base64,QUJDRA==");

        try {
            mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(
                freshmanApplication,
            );
            mockAdmissionApi.createFreshmanCameraHandoff.mockResolvedValue(
                freshmanHandoff,
            );
            const wrapper = mount(FreshmanCameraFlow, {
                props: {
                    maxMaterialBytes: 1024,
                    schools: [
                        {
                            schoolID: 4111010006,
                            schoolCode: "4111010006",
                            schoolName: "北京航空航天大学",
                            verificationMethod: "manual",
                            enabled: true,
                        },
                    ],
                },
            });
            await flushPromises();
            await wrapper
                .find("[data-freshman-school-select]")
                .setValue("4111010006");
            await wrapper
                .find("[data-freshman-applicant-name-input]")
                .setValue("张三");
            await wrapper.get("button.secondary-button").trigger("click");
            await flushPromises();
            await wrapper.get("button.secondary-button").trigger("click");
            await flushPromises();

            expect(play).toHaveBeenCalled();
            expect(getContext).toHaveBeenCalled();
            expect(toDataURL).toHaveBeenCalled();
            expect(wrapper.find('[alt="录取材料预览"]').exists()).toBe(true);

            await wrapper
                .find("[data-freshman-mobile-handoff-button]")
                .trigger("click");
            await flushPromises();
            sources[0]?.emit("handoff", {
                ...freshmanHandoff,
                status: "uploaded",
                uploadedAt: "2026-06-01T10:05:00Z",
            });
            await flushPromises();

            expect(wrapper.text()).toContain(
                "手机已上传材料，请在手机上选择回到电脑端继续或在手机端继续。",
            );
            expect(
                wrapper.find<HTMLButtonElement>("[data-freshman-submit-button]")
                    .element.disabled,
            ).toBe(true);
            expect(
                wrapper.find<HTMLButtonElement>(
                    "[data-freshman-mobile-handoff-button]",
                ).element.disabled,
            ).toBe(true);

            await wrapper.find("form").trigger("submit");
            await flushPromises();

            expect(mockAdmissionApi.uploadCameraCapture).not.toHaveBeenCalled();
            expect(wrapper.emitted("submitted")).toBeUndefined();
            wrapper.unmount();
            expect(stopTrack).toHaveBeenCalled();
        } finally {
            Object.defineProperty(window, "EventSource", {
                configurable: true,
                value: originalEventSource,
            });
            if (mediaDevicesDescriptor) {
                Object.defineProperty(
                    navigator,
                    "mediaDevices",
                    mediaDevicesDescriptor,
                );
            } else {
                Object.defineProperty(navigator, "mediaDevices", {
                    configurable: true,
                    value: undefined,
                });
            }
            if (videoWidthDescriptor) {
                Object.defineProperty(
                    HTMLVideoElement.prototype,
                    "videoWidth",
                    videoWidthDescriptor,
                );
            } else {
                Reflect.deleteProperty(
                    HTMLVideoElement.prototype,
                    "videoWidth",
                );
            }
            if (videoHeightDescriptor) {
                Object.defineProperty(
                    HTMLVideoElement.prototype,
                    "videoHeight",
                    videoHeightDescriptor,
                );
            } else {
                Reflect.deleteProperty(
                    HTMLVideoElement.prototype,
                    "videoHeight",
                );
            }
            play.mockRestore();
            getContext.mockRestore();
            toDataURL.mockRestore();
        }
    });

    it("refreshes handoff state when desktop submission races with mobile upload", async () => {
        const originalEventSource = window.EventSource;
        const mediaDevicesDescriptor = Object.getOwnPropertyDescriptor(
            navigator,
            "mediaDevices",
        );
        const videoWidthDescriptor = Object.getOwnPropertyDescriptor(
            HTMLVideoElement.prototype,
            "videoWidth",
        );
        const videoHeightDescriptor = Object.getOwnPropertyDescriptor(
            HTMLVideoElement.prototype,
            "videoHeight",
        );
        class FakeEventSource {
            close = vi.fn();
            constructor() {}
            addEventListener() {}
        }
        const stopTrack = vi.fn();
        Object.defineProperty(navigator, "mediaDevices", {
            configurable: true,
            value: {
                getUserMedia: vi.fn().mockResolvedValue({
                    getTracks: () => [{ stop: stopTrack }],
                }),
            },
        });
        Object.defineProperty(HTMLVideoElement.prototype, "videoWidth", {
            configurable: true,
            get: () => 2,
        });
        Object.defineProperty(HTMLVideoElement.prototype, "videoHeight", {
            configurable: true,
            get: () => 2,
        });
        Object.defineProperty(window, "EventSource", {
            configurable: true,
            value: FakeEventSource as unknown as typeof EventSource,
        });
        const play = vi
            .spyOn(HTMLMediaElement.prototype, "play")
            .mockResolvedValue(undefined);
        const getContext = vi
            .spyOn(HTMLCanvasElement.prototype, "getContext")
            .mockImplementation(((contextID: string) => {
                if (contextID !== "2d") return null;
                return {
                    drawImage: vi.fn(),
                } as unknown as CanvasRenderingContext2D;
            }) as HTMLCanvasElement["getContext"]);
        const toDataURL = vi
            .spyOn(HTMLCanvasElement.prototype, "toDataURL")
            .mockReturnValue("data:image/jpeg;base64,QUJDRA==");

        try {
            mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(
                freshmanApplication,
            );
            mockAdmissionApi.createFreshmanCameraHandoff.mockResolvedValue(
                freshmanHandoff,
            );
            mockAdmissionApi.uploadCameraCapture.mockRejectedValueOnce(
                new ApiError({
                    code: "A0000409",
                    message: "admission camera handoff locked",
                }),
            );
            mockAdmissionApi.getFreshmanCameraHandoff.mockResolvedValue({
                ...freshmanHandoff,
                status: "uploaded",
                uploadedAt: "2026-06-01T10:05:00Z",
            });
            const wrapper = mount(FreshmanCameraFlow, {
                props: {
                    maxMaterialBytes: 1024,
                    schools: [
                        {
                            schoolID: 4111010006,
                            schoolCode: "4111010006",
                            schoolName: "北京航空航天大学",
                            verificationMethod: "manual",
                            enabled: true,
                        },
                    ],
                },
            });
            await flushPromises();
            await wrapper
                .find("[data-freshman-school-select]")
                .setValue("4111010006");
            await wrapper
                .find("[data-freshman-applicant-name-input]")
                .setValue("张三");
            await wrapper.get("button.secondary-button").trigger("click");
            await flushPromises();
            await wrapper.get("button.secondary-button").trigger("click");
            await flushPromises();
            await wrapper
                .find("[data-freshman-mobile-handoff-button]")
                .trigger("click");
            await flushPromises();

            await wrapper.find("form").trigger("submit");
            await flushPromises();

            expect(mockAdmissionApi.uploadCameraCapture).toHaveBeenCalledTimes(
                1,
            );
            expect(
                mockAdmissionApi.getFreshmanCameraHandoff,
            ).toHaveBeenCalledWith("handoff-1");
            expect(wrapper.text()).toContain("手机端已上传材料");
            expect(wrapper.text()).not.toContain("材料提交失败");
            expect(wrapper.emitted("submitted")).toBeUndefined();
            wrapper.unmount();
            expect(stopTrack).toHaveBeenCalled();
        } finally {
            Object.defineProperty(window, "EventSource", {
                configurable: true,
                value: originalEventSource,
            });
            if (mediaDevicesDescriptor) {
                Object.defineProperty(
                    navigator,
                    "mediaDevices",
                    mediaDevicesDescriptor,
                );
            } else {
                Object.defineProperty(navigator, "mediaDevices", {
                    configurable: true,
                    value: undefined,
                });
            }
            if (videoWidthDescriptor) {
                Object.defineProperty(
                    HTMLVideoElement.prototype,
                    "videoWidth",
                    videoWidthDescriptor,
                );
            } else {
                Reflect.deleteProperty(
                    HTMLVideoElement.prototype,
                    "videoWidth",
                );
            }
            if (videoHeightDescriptor) {
                Object.defineProperty(
                    HTMLVideoElement.prototype,
                    "videoHeight",
                    videoHeightDescriptor,
                );
            } else {
                Reflect.deleteProperty(
                    HTMLVideoElement.prototype,
                    "videoHeight",
                );
            }
            play.mockRestore();
            getContext.mockRestore();
            toDataURL.mockRestore();
        }
    });

    it("moves to pending review when an existing application already has submitted material", async () => {
        mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(
            freshmanApplication,
        );
        mockAdmissionApi.createFreshmanCameraHandoff.mockRejectedValueOnce(
            new ApiError({
                code: "A0000409",
                message: "admission camera handoff locked",
            }),
        );
        const wrapper = mount(FreshmanCameraFlow, {
            props: {
                admissionSessionId: "session-1",
                maxMaterialBytes: 1024,
                schools: [
                    {
                        schoolID: 4111010006,
                        schoolCode: "4111010006",
                        schoolName: "北京航空航天大学",
                        verificationMethod: "manual",
                        enabled: true,
                    },
                ],
            },
        });
        await flushPromises();
        await wrapper
            .find("[data-freshman-school-select]")
            .setValue("4111010006");
        await wrapper
            .find("[data-freshman-applicant-name-input]")
            .setValue("张三");
        await wrapper
            .find("[data-freshman-mobile-handoff-button]")
            .trigger("click");
        await flushPromises();

        expect(mockAdmissionApi.submitFreshmanApplication).toHaveBeenCalledWith({
            schoolCode: "4111010006",
            admissionSessionID: "session-1",
            applicantName: "张三",
            departmentOrMajor: undefined,
            materialType: "admission_notice",
        });
        expect(mockAdmissionApi.createFreshmanCameraHandoff).toHaveBeenCalledWith(
            "application-1",
        );
        expect(wrapper.emitted("submitted")?.[0]).toEqual([
            freshmanApplication,
        ]);
        expect(wrapper.text()).not.toContain("手机拍照链接生成失败");
    });

    it("locks desktop controls when mobile continuation is chosen", async () => {
        const originalEventSource = window.EventSource;
        class FakeEventSource {
            readonly url: string;
            readonly listeners = new Map<
                string,
                Array<(event: MessageEvent) => void>
            >();

            constructor(url: string) {
                this.url = url;
                sources.push(this);
            }

            addEventListener(
                type: string,
                listener: (event: MessageEvent) => void,
            ) {
                const listeners = this.listeners.get(type) ?? [];
                listeners.push(listener);
                this.listeners.set(type, listeners);
            }

            close = vi.fn();

            emit(type: string, data: unknown) {
                const event = new MessageEvent(type, {
                    data: JSON.stringify(data),
                });
                for (const listener of this.listeners.get(type) ?? []) {
                    listener(event);
                }
            }
        }
        const sources: FakeEventSource[] = [];
        Object.defineProperty(window, "EventSource", {
            configurable: true,
            value: FakeEventSource as unknown as typeof EventSource,
        });

        try {
            mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(
                freshmanApplication,
            );
            mockAdmissionApi.createFreshmanCameraHandoff.mockResolvedValue(
                freshmanHandoff,
            );
            const wrapper = mount(FreshmanCameraFlow, {
                props: {
                    maxMaterialBytes: 1024,
                    schools: [
                        {
                            schoolID: 4111010006,
                            schoolCode: "4111010006",
                            schoolName: "北京航空航天大学",
                            verificationMethod: "manual",
                            enabled: true,
                        },
                    ],
                },
            });
            await flushPromises();
            await wrapper
                .find("[data-freshman-school-select]")
                .setValue("4111010006");
            await wrapper
                .find("[data-freshman-applicant-name-input]")
                .setValue("张三");
            await wrapper
                .find("[data-freshman-mobile-handoff-button]")
                .trigger("click");
            await flushPromises();

            sources[0]?.emit("handoff", {
                ...freshmanHandoff,
                status: "locked",
                continueOn: "mobile",
                uploadedAt: "2026-06-01T10:05:00Z",
                chosenAt: "2026-06-01T10:06:00Z",
            });
            await flushPromises();

            expect(wrapper.text()).toContain("电脑端已锁定");
            expect(
                wrapper.find<HTMLButtonElement>(
                    "[data-freshman-mobile-handoff-button]",
                ).element.disabled,
            ).toBe(true);
            expect(wrapper.emitted("submitted")?.[0]).toEqual([
                freshmanApplication,
            ]);
        } finally {
            Object.defineProperty(window, "EventSource", {
                configurable: true,
                value: originalEventSource,
            });
        }
    });

    it("falls back to short polling when the SSE handoff stream fails", async () => {
        vi.useFakeTimers();
        const originalEventSource = window.EventSource;
        class FakeEventSource {
            onerror: (() => void) | null = null;
            close = vi.fn();

            constructor() {
                sources.push(this);
            }

            addEventListener() {}

            fail() {
                this.onerror?.();
            }
        }
        const sources: FakeEventSource[] = [];
        Object.defineProperty(window, "EventSource", {
            configurable: true,
            value: FakeEventSource as unknown as typeof EventSource,
        });

        try {
            mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(
                freshmanApplication,
            );
            mockAdmissionApi.createFreshmanCameraHandoff.mockResolvedValue(
                freshmanHandoff,
            );
            mockAdmissionApi.getFreshmanCameraHandoff.mockResolvedValue({
                ...freshmanHandoff,
                status: "uploaded",
                uploadedAt: "2026-06-01T10:05:00Z",
            });
            const wrapper = mount(FreshmanCameraFlow, {
                props: {
                    maxMaterialBytes: 1024,
                    schools: [
                        {
                            schoolID: 4111010006,
                            schoolCode: "4111010006",
                            schoolName: "北京航空航天大学",
                            verificationMethod: "manual",
                            enabled: true,
                        },
                    ],
                },
            });
            await flushPromises();
            await wrapper
                .find("[data-freshman-school-select]")
                .setValue("4111010006");
            await wrapper
                .find("[data-freshman-applicant-name-input]")
                .setValue("张三");
            await wrapper
                .find("[data-freshman-mobile-handoff-button]")
                .trigger("click");
            await flushPromises();

            expect(sources).toHaveLength(1);
            expect(
                mockAdmissionApi.getFreshmanCameraHandoff,
            ).not.toHaveBeenCalled();

            sources[0]?.fail();
            expect(sources[0]?.close).toHaveBeenCalledTimes(1);

            await vi.advanceTimersByTimeAsync(1500);
            await flushPromises();

            expect(
                mockAdmissionApi.getFreshmanCameraHandoff,
            ).toHaveBeenCalledWith("handoff-1");
            expect(wrapper.text()).toContain("手机已上传材料");

            wrapper.unmount();
        } finally {
            vi.useRealTimers();
            Object.defineProperty(window, "EventSource", {
                configurable: true,
                value: originalEventSource,
            });
        }
    });
});
