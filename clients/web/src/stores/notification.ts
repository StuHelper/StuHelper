/**
 * 通知状态管理
 */
import { computed, onScopeDispose, ref, watch } from "vue";
import { defineStore, getActivePinia } from "pinia";
import type { Notification as AppNotification } from "@stuhelper/shared/notification";
import { api, NOTIFICATION_STREAM_PATH } from "@/api";
import { registerSessionResetHandler } from "@/stores/sessionOrchestrator";

const PAGE_DEFAULT_SIZE = 20;
const BELL_PREVIEW_SIZE = 5;
const POLL_INTERVAL_MS = 30_000;
const MAX_POLL_FAILURES = 5;
const SSE_INITIAL_RECONNECT_MS = 1_000;
const SSE_MAX_RECONNECT_MS = 30_000;

export interface SSENotificationEvent {
    seq: number;
    type:
        | "notification"
        | "notification_read"
        | "notification_read_all"
        | "notification_deleted"
        | "unread_count";
    data: unknown;
}

function mergeUnique<T extends { id: string }>(primary: T[], secondary: T[]) {
    const seen = new Set(primary.map((item) => item.id));
    return [...primary, ...secondary.filter((item) => !seen.has(item.id))];
}

function applyItemUpdate<T extends { id: string }>(
    items: T[],
    id: string,
    update: (item: T) => T,
) {
    let changed = false;
    const next = items.map((item) => {
        if (item.id !== id) {
            return item;
        }
        changed = true;
        return update(item);
    });
    return { changed, next };
}

function isUnreadNotification(notification: AppNotification | undefined) {
    return notification && !notification.isRead;
}

export const useNotificationStore = defineStore("notification", () => {
    const pinia = getActivePinia();
    if (!pinia) {
        throw new Error("notification store requires an active Pinia instance");
    }

    const pageNotifications = ref<AppNotification[]>([]);
    const pageTotal = ref(0);
    const pageLoading = ref(false);
    const pageHasMore = ref(true);
    const pageFetchError = ref<Error | null>(null);

    const bellNotifications = ref<AppNotification[]>([]);
    const bellLoading = ref(false);

    const unreadCount = ref(0);
    const streamError = ref<Error | null>(null);
    const lastSSEEvent = ref<SSENotificationEvent>({
        seq: 0,
        type: "notification",
        data: null,
    });

    let pollTimer: ReturnType<typeof setInterval> | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let eventSource: EventSource | null = null;
    let consecutiveFailures = 0;
    let monitoringActive = false;
    let pollingFallbackActive = false;
    let reconnectDelay = SSE_INITIAL_RECONNECT_MS;
    let sseSequence = 0;

    const hasUnread = computed(() => unreadCount.value > 0);

    const clearPollTimer = () => {
        if (pollTimer) {
            clearInterval(pollTimer);
            pollTimer = null;
        }
    };

    const clearReconnectTimer = () => {
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }
    };

    const closeEventSource = (source?: EventSource) => {
        const target = source ?? eventSource;
        if (!target) {
            return;
        }

        target.close();
        if (!source || eventSource === source) {
            eventSource = null;
        }
    };

    const stopPollingFallback = () => {
        pollingFallbackActive = false;
        clearPollTimer();
    };

    const publishSSEEvent = (
        type: SSENotificationEvent["type"],
        data: unknown,
    ) => {
        sseSequence += 1;
        lastSSEEvent.value = { seq: sseSequence, type, data };
    };

    const setStreamError = (message: string, err?: unknown) => {
        streamError.value =
            err instanceof Error ? err : new Error(message);
        console.warn(`[Notification] ${message}`, err);
    };

    const clearStreamError = () => {
        streamError.value = null;
    };

    const syncUnreadFromNotification = (
        notification: AppNotification | undefined,
        delta: number,
    ) => {
        if (!isUnreadNotification(notification)) {
            return;
        }
        unreadCount.value = Math.max(0, unreadCount.value + delta);
    };

    const upsertBellNotification = (notification: AppNotification) => {
        const existingIndex = bellNotifications.value.findIndex(
            (item) => item.id === notification.id,
        );
        if (existingIndex >= 0) {
            bellNotifications.value = [
                notification,
                ...bellNotifications.value.filter(
                    (item) => item.id !== notification.id,
                ),
            ].slice(0, BELL_PREVIEW_SIZE);
            return true;
        }

        bellNotifications.value = [
            notification,
            ...bellNotifications.value,
        ].slice(0, BELL_PREVIEW_SIZE);
        return true;
    };

    watch(lastSSEEvent, (event) => {
        if (!event || event.seq === 0) {
            return;
        }

        switch (event.type) {
            case "notification": {
                const notification = event.data as AppNotification;
                const existing = bellNotifications.value.find(
                    (item) => item.id === notification.id,
                );
                upsertBellNotification(notification);
                if (!existing) {
                    syncUnreadFromNotification(notification, 1);
                }
                break;
            }
            case "notification_read": {
                const { id } = event.data as { id: string };
                const previous = bellNotifications.value.find(
                    (item) => item.id === id,
                );
                const { changed, next } = applyItemUpdate(
                    bellNotifications.value,
                    id,
                    (item) => ({ ...item, isRead: true }),
                );
                if (changed) {
                    bellNotifications.value = next;
                }
                if (changed && isUnreadNotification(previous)) {
                    unreadCount.value = Math.max(0, unreadCount.value - 1);
                }
                break;
            }
            case "notification_read_all": {
                const hadUnread = bellNotifications.value.some(
                    (item) => !item.isRead,
                );
                if (bellNotifications.value.length > 0) {
                    bellNotifications.value = bellNotifications.value.map(
                        (item) => ({ ...item, isRead: true }),
                    );
                }
                if (hadUnread) {
                    unreadCount.value = 0;
                }
                break;
            }
            case "notification_deleted": {
                const { id } = event.data as { id: string };
                const previous = bellNotifications.value.find(
                    (item) => item.id === id,
                );
                const next = bellNotifications.value.filter(
                    (item) => item.id !== id,
                );
                if (next.length !== bellNotifications.value.length) {
                    bellNotifications.value = next;
                    if (isUnreadNotification(previous)) {
                        unreadCount.value = Math.max(0, unreadCount.value - 1);
                    }
                }
                break;
            }
            case "unread_count": {
                const data = event.data as { count?: number };
                if (typeof data?.count === "number") {
                    unreadCount.value = Math.max(0, data.count);
                }
                break;
            }
        }
    });

    const fetchPageNotifications = async (
        page = 1,
        pageSize = PAGE_DEFAULT_SIZE,
    ) => {
        const normalizedPage = page < 1 ? 1 : page;
        const normalizedPageSize = pageSize < 1 ? PAGE_DEFAULT_SIZE : pageSize;

        pageLoading.value = true;
        pageFetchError.value = null;
        try {
            const res = await api.notification.getNotifications(
                normalizedPage,
                normalizedPageSize,
            );
            const list = (res.data?.data?.list || []) as AppNotification[];
            const total = res.data?.data?.total || 0;

            pageNotifications.value =
                normalizedPage === 1
                    ? list
                    : mergeUnique(pageNotifications.value, list);

            pageTotal.value = total;
            pageHasMore.value = pageNotifications.value.length < total;
        } catch (err) {
            pageFetchError.value =
                err instanceof Error ? err : new Error(String(err));
            throw err;
        } finally {
            pageLoading.value = false;
        }
    };

    const fetchBellNotifications = async (
        page = 1,
        pageSize = BELL_PREVIEW_SIZE,
    ) => {
        const normalizedPage = page < 1 ? 1 : page;
        const normalizedPageSize = pageSize < 1 ? BELL_PREVIEW_SIZE : pageSize;

        bellLoading.value = true;
        try {
            const res = await api.notification.getNotifications(
                normalizedPage,
                normalizedPageSize,
            );
            const list = (res.data?.data?.list || []) as AppNotification[];

            bellNotifications.value =
                normalizedPage === 1
                    ? mergeUnique(list, bellNotifications.value)
                    : mergeUnique(bellNotifications.value, list);
        } finally {
            bellLoading.value = false;
        }
    };

    const fetchUnreadCount = async () => {
        try {
            const res = await api.notification.getUnreadCount();
            unreadCount.value = res.data?.data?.count || 0;
            consecutiveFailures = 0;
            clearStreamError();
        } catch (_error) { void _error;
            consecutiveFailures++;
            if (
                consecutiveFailures >= MAX_POLL_FAILURES &&
                pollingFallbackActive
            ) {
                setStreamError(
                    `${MAX_POLL_FAILURES} consecutive unread-count poll failures`,
                );
                stopPollingFallback();
            }
        }
    };

    const markAsRead = async (id: string) => {
        const previous =
            pageNotifications.value.find(
                (notification) => notification.id === id,
            ) ??
            bellNotifications.value.find(
                (notification) => notification.id === id,
            );

        await api.notification.markAsRead(id);

        const pageUpdate = applyItemUpdate(
            pageNotifications.value,
            id,
            (notification) => ({ ...notification, isRead: true }),
        );
        if (pageUpdate.changed) {
            pageNotifications.value = pageUpdate.next;
        }

        const bellUpdate = applyItemUpdate(
            bellNotifications.value,
            id,
            (notification) => ({ ...notification, isRead: true }),
        );
        if (bellUpdate.changed) {
            bellNotifications.value = bellUpdate.next;
        }

        if (isUnreadNotification(previous)) {
            unreadCount.value = Math.max(0, unreadCount.value - 1);
        }
    };

    const markAllAsRead = async () => {
        await api.notification.markAllAsRead();
        pageNotifications.value = pageNotifications.value.map(
            (notification) => ({ ...notification, isRead: true }),
        );
        bellNotifications.value = bellNotifications.value.map(
            (notification) => ({ ...notification, isRead: true }),
        );
        unreadCount.value = 0;
    };

    const startPollingFallback = (interval = POLL_INTERVAL_MS) => {
        stopPollingFallback();
        pollingFallbackActive = true;
        consecutiveFailures = 0;
        void fetchUnreadCount();

        pollTimer = setInterval(() => {
            if (!monitoringActive || !pollingFallbackActive) {
                stopPollingFallback();
                return;
            }
            void fetchUnreadCount();
        }, interval);
    };

    const scheduleReconnect = () => {
        if (!monitoringActive || reconnectTimer) {
            return;
        }

        const delay = reconnectDelay;
        reconnectDelay = Math.min(reconnectDelay * 2, SSE_MAX_RECONNECT_MS);

        reconnectTimer = setTimeout(() => {
            reconnectTimer = null;
            if (!monitoringActive) {
                return;
            }
            connectSSE();
        }, delay);
    };

    const startPolling = (interval = POLL_INTERVAL_MS) => {
        monitoringActive = true;
        clearReconnectTimer();
        closeEventSource();
        reconnectDelay = SSE_INITIAL_RECONNECT_MS;
        startPollingFallback(interval);
    };

    const stopPolling = () => {
        monitoringActive = false;
        stopPollingFallback();
        clearReconnectTimer();
        closeEventSource();
        reconnectDelay = SSE_INITIAL_RECONNECT_MS;
        consecutiveFailures = 0;
    };

    const connectSSE = () => {
        monitoringActive = true;
        clearReconnectTimer();

        if (typeof EventSource === "undefined") {
            startPollingFallback();
            return;
        }

        if (!pollingFallbackActive) {
            void fetchUnreadCount();
        }

        closeEventSource();

        const source = new EventSource(NOTIFICATION_STREAM_PATH, {
            withCredentials: true,
        });
        eventSource = source;

        source.onopen = () => {
            if (eventSource !== source) {
                return;
            }
            reconnectDelay = SSE_INITIAL_RECONNECT_MS;
            consecutiveFailures = 0;
            clearStreamError();
            stopPollingFallback();
        };

        source.addEventListener("unread_count", (event) => {
            try {
                const data = JSON.parse(event.data);
                if (typeof data.count === "number") {
                    unreadCount.value = data.count;
                    consecutiveFailures = 0;
                    clearStreamError();
                }
            } catch (err) {
                consecutiveFailures++;
                setStreamError("failed to parse unread_count SSE payload", err);
                if (consecutiveFailures >= MAX_POLL_FAILURES) {
                    closeEventSource(source);
                    startPollingFallback();
                    scheduleReconnect();
                }
            }
        });

        source.addEventListener("notification", (event) => {
            try {
                publishSSEEvent("notification", JSON.parse(event.data));
                consecutiveFailures = 0;
                clearStreamError();
            } catch (err) {
                consecutiveFailures++;
                setStreamError("failed to parse notification SSE payload", err);
                if (consecutiveFailures >= MAX_POLL_FAILURES) {
                    closeEventSource(source);
                    startPollingFallback();
                    scheduleReconnect();
                }
            }
            void fetchUnreadCount();
        });

        source.addEventListener("notification_read", (event) => {
            try {
                publishSSEEvent("notification_read", JSON.parse(event.data));
                consecutiveFailures = 0;
                clearStreamError();
            } catch (err) {
                consecutiveFailures++;
                setStreamError(
                    "failed to parse notification_read SSE payload",
                    err,
                );
                if (consecutiveFailures >= MAX_POLL_FAILURES) {
                    closeEventSource(source);
                    startPollingFallback();
                    scheduleReconnect();
                }
            }
            void fetchUnreadCount();
        });

        source.addEventListener("notification_read_all", (event) => {
            try {
                publishSSEEvent(
                    "notification_read_all",
                    JSON.parse(event.data),
                );
                consecutiveFailures = 0;
                clearStreamError();
            } catch (err) {
                consecutiveFailures++;
                setStreamError(
                    "failed to parse notification_read_all SSE payload",
                    err,
                );
                if (consecutiveFailures >= MAX_POLL_FAILURES) {
                    closeEventSource(source);
                    startPollingFallback();
                    scheduleReconnect();
                }
            }
            void fetchUnreadCount();
        });

        source.addEventListener("notification_deleted", (event) => {
            try {
                publishSSEEvent("notification_deleted", JSON.parse(event.data));
                consecutiveFailures = 0;
                clearStreamError();
            } catch (err) {
                consecutiveFailures++;
                setStreamError(
                    "failed to parse notification_deleted SSE payload",
                    err,
                );
                if (consecutiveFailures >= MAX_POLL_FAILURES) {
                    closeEventSource(source);
                    startPollingFallback();
                    scheduleReconnect();
                }
            }
            void fetchUnreadCount();
        });

        source.onerror = () => {
            setStreamError("notification SSE connection lost");
            closeEventSource(source);
            if (!monitoringActive) {
                return;
            }
            startPollingFallback();
            scheduleReconnect();
        };
    };

    onScopeDispose(stopPolling);

    const reset = () => {
        stopPolling();
        pageNotifications.value = [];
        pageTotal.value = 0;
        pageLoading.value = false;
        pageHasMore.value = true;
        pageFetchError.value = null;
        bellNotifications.value = [];
        bellLoading.value = false;
        unreadCount.value = 0;
        streamError.value = null;
        lastSSEEvent.value = { seq: 0, type: "notification", data: null };
    };

    const unregisterSessionReset = registerSessionResetHandler(
        "notification",
        () => {
            stopPolling();
            reset();
        },
        pinia,
    );
    onScopeDispose(unregisterSessionReset);

    return {
        pageNotifications,
        pageTotal,
        pageLoading,
        pageHasMore,
        pageFetchError,
        bellNotifications,
        bellLoading,
        unreadCount,
        hasUnread,
        streamError,
        lastSSEEvent,
        fetchPageNotifications,
        fetchBellNotifications,
        fetchUnreadCount,
        markAsRead,
        markAllAsRead,
        startPolling,
        stopPolling,
        connectSSE,
        reset,
    };
});
