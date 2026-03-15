import { describe, expect, it } from "vitest";
import {
    isManualSubmitReady,
    normalizeManualFields,
    normalizeManualFormData,
} from "../utils/manualVerification";

describe("manualVerification helpers", () => {
    it("returns manual field descriptors as-is when provided", () => {
        const fields = normalizeManualFields([
            { key: "studentID", label: "学号", type: "text", required: true },
        ]);

        expect(fields).toHaveLength(1);
        expect(fields[0]?.key).toBe("studentID");
    });

    it("normalizes manual form data and removes empty entries", () => {
        expect(
            normalizeManualFormData({
                studentID: " 20240001 ",
                department: "   ",
                "  ": "ignored",
            }),
        ).toEqual({
            studentID: "20240001",
        });
    });

    it("checks only required manual fields before enabling submit", () => {
        const fields = [
            { key: "studentID", label: "学号", type: "text", required: true },
            { key: "department", label: "院系", type: "text", required: false },
        ] as const;

        expect(
            isManualSubmitReady(fields, {
                studentID: "",
                department: "计算机学院",
            }),
        ).toBe(false);
        expect(
            isManualSubmitReady(fields, {
                studentID: "20240001",
                department: "",
            }),
        ).toBe(true);
    });
});
