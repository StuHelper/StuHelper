type LogExtra = Record<string, unknown>;

export const adminLogger = {
  warn(event: string, error?: unknown, extra: LogExtra = {}) {
    console.warn('[admin-auth]', event, extra, error);
  },
};
