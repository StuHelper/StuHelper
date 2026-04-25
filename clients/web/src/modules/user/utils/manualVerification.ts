import type { components } from "@stuhelper/shared/types";

export type ManualFieldDescriptor =
    components["schemas"]["ManualFieldDescriptor"];

export function normalizeManualFields(
    fields: components["schemas"]["SchoolConfig"]["manualFormFields"],
): ManualFieldDescriptor[] {
    return fields ?? [];
}

export function normalizeManualFormData(
    data: Record<string, string>,
): Record<string, string> | undefined {
    const entries = Object.entries(data)
        .map(([key, value]) => [key.trim(), value.trim()] as const)
        .filter(([key, value]) => key !== "" && value !== "");

    if (entries.length === 0) {
        return undefined;
    }

    return Object.fromEntries(entries);
}

export function isManualSubmitReady(
    fields: readonly ManualFieldDescriptor[],
    data: Record<string, string>,
): boolean {
    return fields
        .filter((field) => field.required)
        .every((field) => {
            const value = data[field.key];
            return value !== undefined && value.trim() !== "";
        });
}
