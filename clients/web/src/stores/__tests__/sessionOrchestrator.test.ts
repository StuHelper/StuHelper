import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

import {
    clearSessionResetHandlers,
    registerSessionResetHandler,
    runSessionReset,
} from "@/stores/sessionOrchestrator";

describe("sessionOrchestrator", () => {
    beforeEach(() => {
        const pinia = createPinia();
        setActivePinia(pinia);
        clearSessionResetHandlers(pinia);
    });

    it("runs registered handlers once per key", () => {
        const first = vi.fn();
        const second = vi.fn();

        registerSessionResetHandler("user", first);
        registerSessionResetHandler("notification", second);
        registerSessionResetHandler("user", first);

        runSessionReset();

        expect(first).toHaveBeenCalledTimes(1);
        expect(second).toHaveBeenCalledTimes(1);
    });

    it("isolates handler registries per pinia instance", () => {
        const piniaA = createPinia();
        const piniaB = createPinia();
        const resetA = vi.fn();
        const resetB = vi.fn();

        registerSessionResetHandler("user", resetA, piniaA);
        registerSessionResetHandler("user", resetB, piniaB);

        runSessionReset(piniaA);
        expect(resetA).toHaveBeenCalledTimes(1);
        expect(resetB).not.toHaveBeenCalled();

        runSessionReset(piniaB);
        expect(resetB).toHaveBeenCalledTimes(1);
    });
});
