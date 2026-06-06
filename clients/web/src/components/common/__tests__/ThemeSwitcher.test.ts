// @vitest-environment jsdom

import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ThemeSwitcher from "../ThemeSwitcher.vue";

vi.mock("vue-i18n", () => ({
    useI18n: () => ({
        t: (key: string) => key,
    }),
}));

function stubMatchMedia(matches = false) {
    Object.defineProperty(window, "matchMedia", {
        configurable: true,
        writable: true,
        value: vi.fn().mockImplementation((query: string) => ({
            matches,
            media: query,
            onchange: null,
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
            addListener: vi.fn(),
            removeListener: vi.fn(),
            dispatchEvent: vi.fn(),
        })),
    });
}

describe("ThemeSwitcher", () => {
    beforeEach(() => {
        stubMatchMedia(false);
        localStorage.clear();
        document.documentElement.removeAttribute("data-theme");
        document.documentElement.classList.remove("dark");
        setActivePinia(createPinia());
    });

    afterEach(() => {
        vi.restoreAllMocks();
        localStorage.clear();
        document.documentElement.removeAttribute("data-theme");
        document.documentElement.classList.remove("dark");
    });

    it("follows system theme by default and persists manual toggles", async () => {
        const wrapper = mount(ThemeSwitcher);
        const button = wrapper.get("button");

        expect(document.documentElement.getAttribute("data-theme")).toBeNull();
        expect(button.attributes("aria-label")).toBe(
            "common.theme.switchToDark",
        );

        await button.trigger("click");

        expect(localStorage.getItem("theme-mode")).toBe("dark");
        expect(document.documentElement.getAttribute("data-theme")).toBe(
            "dark",
        );
        expect(document.documentElement.classList.contains("dark")).toBe(true);
        expect(button.attributes("aria-label")).toBe(
            "common.theme.switchToLight",
        );

        await button.trigger("click");

        expect(localStorage.getItem("theme-mode")).toBe("light");
        expect(document.documentElement.getAttribute("data-theme")).toBe(
            "light",
        );
        expect(document.documentElement.classList.contains("dark")).toBe(false);
        expect(button.attributes("aria-label")).toBe(
            "common.theme.switchToDark",
        );
    });
});
