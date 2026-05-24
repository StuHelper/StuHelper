import { readFileSync } from 'node:fs';

import { OPEN_PLATFORM_AUDIT_EVENT_TYPES } from '@stuhelper/shared/constants';
import { describe, expect, it } from 'vitest';

import {
  eventTagType,
  knownOpenPlatformAuditEventTypes,
  openPlatformAuditEventTypeLabelKeys,
} from './auditEvents';

const zhCNAdmin = readLocale('../../../locales/langs/zh-CN/admin.json');
const enUSAdmin = readLocale('../../../locales/langs/en-US/admin.json');

describe('admin Open Platform audit event taxonomy', () => {
  it('keeps filter options aligned with the shared audit event list', () => {
    expect(knownOpenPlatformAuditEventTypes).toEqual(
      OPEN_PLATFORM_AUDIT_EVENT_TYPES,
    );
  });

  it('provides a localization key for every known audit event type', () => {
    for (const eventType of knownOpenPlatformAuditEventTypes) {
      expect(openPlatformAuditEventTypeLabelKeys[eventType], eventType).toMatch(
        /^admin\.openPlatform\.audit\.event\./,
      );
    }
  });

  it('keeps admin audit event labels present in Chinese and English locales', () => {
    for (const key of Object.values(openPlatformAuditEventTypeLabelKeys)) {
      expect(resolveLocaleKey(zhCNAdmin, key), `zh-CN ${key}`).toEqual(
        expect.any(String),
      );
      expect(resolveLocaleKey(enUSAdmin, key), `en-US ${key}`).toEqual(
        expect.any(String),
      );
    }
  });

  it('classifies lifecycle and resource event tags', () => {
    expect(eventTagType('open_platform.app.resumed')).toBe('success');
    expect(eventTagType('open_platform.app.approved_app_ensured')).toBe(
      'success',
    );
    expect(
      eventTagType('open_platform.app.identity_public_smoke_bootstrapped'),
    ).toBe('info');
    expect(eventTagType('open_platform.resource_access.granted')).toBe(
      'success',
    );
    expect(eventTagType('open_platform.resource_access.revoked')).toBe(
      'danger',
    );
  });
});

function readLocale(relativePath: string): unknown {
  return JSON.parse(
    readFileSync(new URL(relativePath, import.meta.url), 'utf8'),
  );
}

function resolveLocaleKey(locale: unknown, key: string): unknown {
  const normalizedKey = key.startsWith('admin.') ? key.slice('admin.'.length) : key;
  return normalizedKey.split('.').reduce<unknown>((current, segment) => {
    if (typeof current !== 'object' || current === null) {
      return undefined;
    }
    return (current as Record<string, unknown>)[segment];
  }, locale);
}
