import { getActivePinia, type Pinia } from "pinia";

export type SessionResetHandler = () => void;

const registries = new WeakMap<Pinia, Map<string, SessionResetHandler>>();

function requirePinia(pinia?: Pinia): Pinia {
    const resolved = pinia ?? getActivePinia();
    if (!resolved) {
        throw new Error(
            "session orchestrator requires an active Pinia instance",
        );
    }
    return resolved;
}

function getRegistry(pinia: Pinia): Map<string, SessionResetHandler> {
    let registry = registries.get(pinia);
    if (!registry) {
        registry = new Map<string, SessionResetHandler>();
        registries.set(pinia, registry);
    }
    return registry;
}

export function registerSessionResetHandler(
    key: string,
    handler: SessionResetHandler,
    pinia?: Pinia,
): () => void {
    const resolvedPinia = requirePinia(pinia);
    const registry = getRegistry(resolvedPinia);
    registry.set(key, handler);
    return () => {
        registry.delete(key);
    };
}

export function runSessionReset(pinia?: Pinia): void {
    const registry = registries.get(requirePinia(pinia));
    if (!registry) {
        return;
    }
    for (const reset of registry.values()) {
        reset();
    }
}

export function clearSessionResetHandlers(pinia?: Pinia): void {
    registries.delete(requirePinia(pinia));
}
