/**
 * Shared clipboard helpers for admin views.
 *
 * The API mirrors the koishi WebUI `client/utils/clipboard.ts` implementation:
 * a pure `copyTextToClipboard` that reports success via its return value and
 * leaves user feedback (toasts) to the caller.
 */

interface ClipboardWriter {
  writeText(text: string): Promise<unknown> | unknown;
}

interface TextareaCopyTarget {
  value: string;
  remove(): void;
  select(): void;
}

export interface CopyTextEnvironment {
  clipboard?: ClipboardWriter | null;
  createTextarea?: () => null | TextareaCopyTarget;
  execCopy?: () => boolean;
}

export async function copyTextToClipboard(
  text: string,
  environment: CopyTextEnvironment = {},
): Promise<boolean> {
  const clipboard =
    environment.clipboard === undefined
      ? browserClipboard()
      : environment.clipboard;
  if (clipboard?.writeText) {
    try {
      await clipboard.writeText(text);
      return true;
    } catch {
      // Continue to the textarea fallback below.
    }
  }

  return copyTextWithTextarea(text, environment);
}

export function copyTextWithTextarea(
  text: string,
  environment: Pick<CopyTextEnvironment, 'createTextarea' | 'execCopy'> = {},
): boolean {
  let target: null | TextareaCopyTarget = null;

  try {
    target = (environment.createTextarea ?? createBrowserTextarea)();
    if (!target) return false;
    target.value = text;
    target.select();
    return (environment.execCopy ?? browserExecCopy)();
  } catch {
    return false;
  } finally {
    target?.remove();
  }
}

function browserClipboard(): ClipboardWriter | null {
  if (typeof navigator === 'undefined') return null;
  return navigator.clipboard ?? null;
}

function browserExecCopy(): boolean {
  if (
    typeof document === 'undefined' ||
    typeof document.execCommand !== 'function'
  ) {
    return false;
  }
  return document.execCommand('copy');
}

function createBrowserTextarea(): null | TextareaCopyTarget {
  if (typeof document === 'undefined' || !document.body) return null;
  const textarea = document.createElement('textarea');
  document.body.append(textarea);

  return {
    get value() {
      return textarea.value;
    },
    set value(next: string) {
      textarea.value = next;
    },
    select() {
      textarea.select();
    },
    remove() {
      textarea.remove();
    },
  };
}
