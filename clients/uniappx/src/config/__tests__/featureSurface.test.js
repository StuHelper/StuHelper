import { describe, expect, it } from 'vitest';
import { setLocale } from '@/i18n';
import { getHomeFeatures, getUniappxAppNotice, getUserMenuItems, } from '../featureSurface';
describe('uniappx feature surface', () => {
    it('exposes the current user center routes', () => {
        setLocale('zh-CN');
        expect(getUserMenuItems().map((item) => item.path)).toEqual([
            '/pages/user/reviews',
            '/pages/user/votes',
            '/pages/user/favorites',
            '/pages/user/notifications',
        ]);
    });
    it('does not expose teacher profile without a required id route parameter', () => {
        setLocale('zh-CN');
        expect(getHomeFeatures().some((item) => item.path === '/pages/teacher/profile')).toBe(false);
    });
    it('states that uniappx now uses the shared openapi contract', () => {
        setLocale('en-US');
        expect(getUniappxAppNotice()).toContain('OpenAPI');
    });
});
