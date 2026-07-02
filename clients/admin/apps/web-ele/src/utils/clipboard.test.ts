import { describe, expect, it, vi } from 'vitest';

import { copyTextToClipboard, copyTextWithTextarea } from './clipboard';

describe('copyTextToClipboard', () => {
  it('uses the async clipboard API when available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);

    const copied = await copyTextToClipboard('token-123', {
      clipboard: { writeText },
    });

    expect(copied).toBe(true);
    expect(writeText).toHaveBeenCalledWith('token-123');
  });

  it('falls back to the textarea copy when the clipboard API rejects', async () => {
    const target = {
      value: '',
      remove: vi.fn(),
      select: vi.fn(),
    };

    const copied = await copyTextToClipboard('token-123', {
      clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
      createTextarea: () => target,
      execCopy: () => true,
    });

    expect(copied).toBe(true);
    expect(target.value).toBe('token-123');
    expect(target.select).toHaveBeenCalledOnce();
    expect(target.remove).toHaveBeenCalledOnce();
  });

  it('reports failure when no copy mechanism is available', async () => {
    const copied = await copyTextToClipboard('token-123', {
      clipboard: null,
      createTextarea: () => null,
    });

    expect(copied).toBe(false);
  });
});

describe('copyTextWithTextarea', () => {
  it('removes the textarea even when execCopy throws', () => {
    const target = {
      value: '',
      remove: vi.fn(),
      select: vi.fn(),
    };

    const copied = copyTextWithTextarea('token-123', {
      createTextarea: () => target,
      execCopy: () => {
        throw new Error('execCommand unavailable');
      },
    });

    expect(copied).toBe(false);
    expect(target.remove).toHaveBeenCalledOnce();
  });
});
