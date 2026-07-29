import { listIcons } from '@vben/icons';

import { describe, expect, it } from 'vitest';

const requiredIcons = [
  'carbon:workspace',
  'ep:expand',
  'ep:fold',
  'fluent-mdl2:world-clock',
  'lucide:activity',
  'lucide:app-window',
  'lucide:area-chart',
  'lucide:badge-check',
  'lucide:calendar-plus',
  'lucide:copy',
  'lucide:eye-off',
  'lucide:file-check-2',
  'lucide:file-text',
  'lucide:flag',
  'lucide:graduation-cap',
  'lucide:id-card',
  'lucide:key-round',
  'lucide:layout-dashboard',
  'lucide:list-checks',
  'lucide:message-square',
  'lucide:message-square-text',
  'lucide:school',
  'lucide:scroll-text',
  'lucide:settings',
  'lucide:shield-alert',
  'lucide:shield-check',
  'lucide:user',
  'lucide:user-check',
  'lucide:user-x',
  'lucide:users',
  'mdi:home-outline',
  'mdi:keyboard-esc',
];

describe('local admin icon registry', () => {
  it('registers every icon used by production navigation and core actions', () => {
    const registeredIcons = new Set(
      requiredIcons.flatMap((icon) => {
        const [prefix] = icon.split(':');
        return listIcons('', prefix);
      }),
    );

    expect(requiredIcons.filter((icon) => !registeredIcons.has(icon))).toEqual(
      [],
    );
  });
});
