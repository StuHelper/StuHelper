/**
 * 通知状态管理
 */
import { computed, ref, watch } from "vue";
import { defineStore, getActivePinia } from "pinia";
import type {
    Notification as AppNotification,
    NotificationType,
} from "@stuhelper/shared/notification";
import { api, NOTIFICATION_STREAM_PATH } from "@/api";
import { safeOnScopeDispose } from "@/stores/safeScopeDispose";
import { registerSessionResetHandler } from "@/stores/sessionOrchestrator";

const PAGE_DEFAULT_SIZE = 20;
const BELL_PREVIEW_SIZE = 5;
const POLL_INTERVAL_MS = 30_000;
const MAX_POLL_FAILURES = 5;
const SSE_INITIAL_RECONNECT_MS = 1_000;
const SSE_MAX_RECONNECT_MS = 30_000;
const NOTIFICATION_TYPES = new Set<NotificationType>([
    "reply",
    "like",
    "vote",
    "review_hidden",
    "review_restored",
    "report_resolved",
    "identity_approved",
    "identity_rejected",
    "student_approved",
    "student_rejected",
    "freshman_approved",
    "freshman_rejected",
    "freshman_near_expiry",
    "system",
]);

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

function isRecord(value: unknown): value is Record<string, unknown> {
    return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function readString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string {
    const value = record[key];
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readOptionalString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readOptionalInteger(
    record: Record<string, unknown>,
    key: string,
    message: string,
): number | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (typeof value !== "number" || !Number.isInteger(value)) {
        throw new Error(message);
    }
    return value;
}

function readBoolean(
    record: Record<string, unknown>,
    key: string,
    message: string,
): boolean {
    const value = record[key];
    if (typeof value !== "boolean") {
        throw new Error(message);
    }
    return value;
}

function readNotificationType(
    record: Record<string, unknown>,
    key: string,
    message: string,
): NotificationType {
    const value = readString(record, key, message);
    if (!NOTIFICATION_TYPES.has(value as NotificationType)) {
        throw new Error(message);
    }
    return value as NotificationType;
}

function readOptionalPayload(
    record: Record<string, unknown>,
    key: string,
    message: string,
): AppNotification["payload"] | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (!isRecord(value)) {
        throw new Error(message);
    }
    return { ...value };
}

function readNotificationPayload(
    payload: unknown,
    message = "Invalid notification response",
): AppNotification {
    if (!isRecord(payload)) {
        throw new Error(message);
    }

    return {
        id: readString(payload, "id", message),
        type: readNotificationType(payload, "type", message),
        title: readString(payload, "title", message),
        content: readOptionalString(payload, "content", message),
        payload: readOptionalPayload(payload, "payload", message),
        sourceModule: readOptionalString(payload, "sourceModule", message),
        sourceId: readOptionalString(payload, "sourceId", message),
        sourceUrl: readOptionalString(payload, "sourceUrl", message),
        courseID: readOptionalInteger(payload, "courseID", message),
        isRead: readBoolean(payload, "isRead", message),
        createdAt: readString(payload, "createdAt", message),
    };
}

function readNotificationIDPayload(
    payload: unknown,
    message = "Invalid notification SSE event",
): { id: string } {
    if (!isRecord(payload)) {
        throw new Error(message);
    }
    return { id: readString(payload, "id", message) };
}

function readUnreadCountPayload(
    payload: unknown,
    message = "Invalid unread count response",
): { count: number } {
    if (!isRecord(payload)) {
        throw new Error(message);
    }

    const { count } = payload;
    if (typeof count !== "number" || !Number.isInteger(count) || count < 0) {
        throw new Error(message);
    }
    return { count };
}

function readNotificationPage(payload: unknown): {
    list: AppNotification[];
    total: number;
} {
    if (!isRecord(payload)) {
        throw new Error("Invalid notification page response");
    }

    const { list, total } = payload;
    if (
        !Array.isArray(list) ||
        typeof total !== "number" ||
        !Number.isInteger(total) ||
        total < 0
    ) {
        throw new Error("Invalid notification page response");
    }

    return {
        list: list.map((item) =>
            readNotificationPayload(
                item,
                "Invalid notification page response",
            ),
        ),
        total,
    };
}

function readUnreadCount(payload: unknown): number {
    return readUnreadCountPayload(payload).count;
}

export const useNotificationStore = defineStore("notification", () => {
    const pinia = getActivePinia();
    if (!pinia) {
        throw new Error("notification store requires an active Pinia instance");
    }

    const pageNotifications = ref<AppNotification[]>([]);
    const pageTotal = ref(0);
    const pageLoading = ref(false);
    const pageHasMore = ref(false);
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
                let notification: AppNotification;
                try {
                    notification = readNotificationPayload(
                        event.data,
                        "Invalid notification SSE event",
                    );
                } catch (err) {
                    setStreamError("invalid notification SSE event", err);
                    break;
                }
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
                let id: string;
                try {
                    id = readNotificationIDPayload(event.data).id;
                } catch (err) {
                    setStreamError("invalid notification_read SSE event", err);
                    break;
                }
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
                let id: string;
                try {
                    id = readNotificationIDPayload(event.data).id;
                } catch (err) {
                    setStreamError("invalid notification_deleted SSE event", err);
                    break;
                }
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
                try {
                    unreadCount.value = readUnreadCountPayload(
                        event.data,
                        "Invalid unread_count SSE event",
                    ).count;
                } catch (err) {
                    setStreamError("invalid unread_count SSE event", err);
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
            const { list, total } = readNotificationPage(res.data?.data);

            pageNotifications.value =
                normalizedPage === 1
                    ? list
                    : mergeUnique(pageNotifications.value, list);

            pageTotal.value = total;
            pageHasMore.value =
                list.length === normalizedPageSize &&
                pageNotifications.value.length < total;
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
            const { list } = readNotificationPage(res.data?.data);

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
            unreadCount.value = readUnreadCount(res.data?.data);
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

    const handleSSEPayloadError = (
        source: EventSource,
        message: string,
        err: unknown,
    ) => {
        consecutiveFailures++;
        setStreamError(message, err);
        if (consecutiveFailures >= MAX_POLL_FAILURES) {
            closeEventSource(source);
            startPollingFallback();
            scheduleReconnect();
        }
    };

    const readSSEPayload = <T>(
        event: MessageEvent,
        source: EventSource,
        reader: (payload: unknown) => T,
        message: string,
    ): T | null => {
        try {
            const data = reader(JSON.parse(event.data));
            consecutiveFailures = 0;
            clearStreamError();
            return data;
        } catch (err) {
            handleSSEPayloadError(source, message, err);
            return null;
        }
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

        // 通知实时流是浏览器原生 SSE 长连接，不能通过 shared API client 表达。
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
            const data = readSSEPayload(
                event,
                source,
                (payload) =>
                    readUnreadCountPayload(
                        payload,
                        "Invalid unread_count SSE payload",
                    ),
                "failed to parse unread_count SSE payload",
            );
            if (data) {
                unreadCount.value = data.count;
            }
        });

        source.addEventListener("notification", (event) => {
            const notification = readSSEPayload(
                event,
                source,
                (payload) =>
                    readNotificationPayload(
                        payload,
                        "Invalid notification SSE payload",
                    ),
                "failed to parse notification SSE payload",
            );
            if (notification) {
                publishSSEEvent("notification", notification);
            }
            void fetchUnreadCount();
        });

        source.addEventListener("notification_read", (event) => {
            const data = readSSEPayload(
                event,
                source,
                readNotificationIDPayload,
                "failed to parse notification_read SSE payload",
            );
            if (data) {
                publishSSEEvent("notification_read", data);
            }
            void fetchUnreadCount();
        });

        source.addEventListener("notification_read_all", (event) => {
            const data = readSSEPayload(
                event,
                source,
                (payload) => {
                    if (!isRecord(payload)) {
                        throw new Error("Invalid notification_read_all SSE payload");
                    }
                    return { ...payload };
                },
                "failed to parse notification_read_all SSE payload",
            );
            if (data) {
                publishSSEEvent("notification_read_all", data);
            }
            void fetchUnreadCount();
        });

        source.addEventListener("notification_deleted", (event) => {
            const data = readSSEPayload(
                event,
                source,
                readNotificationIDPayload,
                "failed to parse notification_deleted SSE payload",
            );
            if (data) {
                publishSSEEvent("notification_deleted", data);
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

    safeOnScopeDispose(stopPolling);

    const reset = () => {
        stopPolling();
        pageNotifications.value = [];
        pageTotal.value = 0;
        pageLoading.value = false;
        pageHasMore.value = false;
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
    safeOnScopeDispose(unregisterSessionReset);

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
