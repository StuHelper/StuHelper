interface ClipboardWriter {
  writeText(text: string): Promise<unknown> | unknown
}

interface TextareaCopyTarget {
  value: string
  select(): void
  remove(): void
}

export interface CopyTextEnvironment {
  clipboard?: ClipboardWriter | null
  createTextarea?: () => TextareaCopyTarget | null
  execCopy?: () => boolean
}

export async function copyTextToClipboard(text: string, environment: CopyTextEnvironment = {}): Promise<boolean> {
  const clipboard = environment.clipboard ?? browserClipboard()
  if (clipboard?.writeText) {
    try {
      await clipboard.writeText(text)
      return true
    } catch {
      // Continue to the textarea fallback below.
    }
  }

  return copyTextWithTextarea(text, environment)
}

export function copyTextWithTextarea(
  text: string,
  environment: Pick<CopyTextEnvironment, 'createTextarea' | 'execCopy'> = {},
): boolean {
  let target: TextareaCopyTarget | null = null

  try {
    target = (environment.createTextarea ?? createBrowserTextarea)()
    if (!target) return false
    target.value = text
    target.select()
    return (environment.execCopy ?? browserExecCopy)()
  } catch {
    return false
  } finally {
    target?.remove()
  }
}

function browserClipboard(): ClipboardWriter | null {
  if (typeof navigator === 'undefined') return null
  return navigator.clipboard ?? null
}

function browserExecCopy(): boolean {
  if (typeof document === 'undefined' || typeof document.execCommand !== 'function') return false
  return document.execCommand('copy')
}

function createBrowserTextarea(): TextareaCopyTarget | null {
  if (typeof document === 'undefined' || !document.body) return null
  const textarea = document.createElement('textarea')
  document.body.appendChild(textarea)

  return {
    get value() {
      return textarea.value
    },
    set value(next: string) {
      textarea.value = next
    },
    select() {
      textarea.select()
    },
    remove() {
      textarea.parentNode?.removeChild(textarea)
    },
  }
}
