import { describe, expect, it } from "vitest";
import {
    extractOptionalResultData,
    extractResultData,
    extractResultErrorCode,
    extractResultErrorMessage,
    extractResultList,
    isResultFailure,
    readResultStatus,
} from "../api/result";

describe("shared api result helpers", () => {
    it("reads status and structured error code from result envelopes", () => {
        expect(readResultStatus({ response: { status: 401 } })).toBe(401);
        expect(
            extractResultErrorCode({ data: { error: { code: "A0010100" } } }),
        ).toBe("A0010100");
        expect(
            extractResultErrorCode({ error: { error: { code: "B0000001" } } }),
        ).toBe("B0000001");
    });

    it("extracts backend error messages from data or transport error envelopes", () => {
        expect(
            extractResultErrorMessage(
                {
                    data: {
                        error: {
                            code: "A0040999",
                            message: "学生认证已被其他管理员处理",
                        },
                    },
                },
                "fallback",
            ),
        ).toBe("学生认证已被其他管理员处理");
        expect(
            extractResultErrorMessage(
                {
                    error: {
                        error: "admission session is already cancelled",
                    },
                },
                "fallback",
            ),
        ).toBe("admission session is already cancelled");
        expect(extractResultErrorMessage({}, "fallback")).toBe("fallback");
    });

    it("extracts payload and list data without platform wrappers", () => {
        expect(extractResultData({ data: { data: { id: 1 } } })).toEqual({
            id: 1,
        });
        expect(extractOptionalResultData({ data: { data: null } })).toBeNull();
        expect(
            extractResultList({
                data: { data: { list: [{ id: 1 }], total: 1 } },
            }),
        ).toEqual({
            list: [{ id: 1 }],
            total: 1,
        });
        expect(
            extractResultList({
                data: { data: { items: [{ id: 2 }], total: 1 } },
            }),
        ).toEqual({
            list: [{ id: 2 }],
            total: 1,
        });
    });

    it("marks failure on http errors, transport errors, or explicit success=false", () => {
        expect(isResultFailure({ response: { status: 500 } })).toBe(true);
        expect(isResultFailure({ error: new Error("network") })).toBe(true);
        expect(isResultFailure({ data: { success: false } })).toBe(true);
        expect(
            isResultFailure({
                data: { success: true, data: { ok: true } },
                response: { status: 200 },
            }),
        ).toBe(false);
    });
});
