import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api } from '@/api';
import { assertMutationSuccess, unwrapData, unwrapOptionalData } from '@/api/result';
import { translate } from '@/i18n';
const BOOTSTRAP_STALE_MS = 60000;
function buildCurrentRouteRedirect() {
    try {
        const pages = (typeof getCurrentPages === 'function' ? getCurrentPages() : []);
        const currentPage = pages[pages.length - 1];
        if (!currentPage?.route) {
            return '/pages/user/index';
        }
        const query = currentPage.options
            ? Object.entries(currentPage.options)
                .filter(([, value]) => typeof value === 'string' && value.length > 0)
                .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
            : [];
        return `/${currentPage.route}${query.length > 0 ? `?${query.join('&')}` : ''}`;
    }
    catch {
        return '/pages/user/index';
    }
}
export const useAuthStore = defineStore('auth', () => {
    const user = ref(null);
    const loading = ref(false);
    const initialized = ref(false);
    const lastBootstrapAt = ref(0);
    const isAuthenticated = computed(() => !!user.value);
    const displayName = computed(() => user.value?.displayName || user.value?.name || translate('common.notLoggedIn'));
    function setUser(nextUser) {
        user.value = nextUser;
        lastBootstrapAt.value = nextUser ? Date.now() : 0;
    }
    function clearSession() {
        user.value = null;
        lastBootstrapAt.value = 0;
    }
    async function bootstrapSession(force = false) {
        const now = Date.now();
        if (!force && initialized.value && now - lastBootstrapAt.value < BOOTSTRAP_STALE_MS) {
            return;
        }
        if (loading.value)
            return;
        loading.value = true;
        try {
            const result = await api.auth.me();
            user.value = unwrapOptionalData(result);
            lastBootstrapAt.value = Date.now();
            initialized.value = true;
        }
        catch (error) {
            const status = error?.status
                ?? error?.response?.status;
            if (status === 401 || status === 403) {
                user.value = null;
                lastBootstrapAt.value = Date.now();
                initialized.value = true;
            }
            // 网络错误 / 超时 / 5xx：不更新 lastBootstrapAt 和 initialized，允许后续重试
        }
        finally {
            loading.value = false;
        }
    }
    async function requestPhoneOTP(phone) {
        const result = await api.auth.requestPhoneOTP(phone);
        return unwrapData(result);
    }
    async function verifyPhoneOTP(phone, code) {
        const result = await api.auth.verifyPhoneOTP(phone, code);
        const data = unwrapData(result);
        setUser(data.user);
        initialized.value = true;
        return data;
    }
    async function logout() {
        assertMutationSuccess(await api.auth.logout());
        clearSession();
    }
    async function requireAuth(message = translate('auth.requireLogin')) {
        if (isAuthenticated.value)
            return true;
        await bootstrapSession();
        if (isAuthenticated.value)
            return true;
        uni.showToast({ title: message, icon: 'none' });
        uni.navigateTo({ url: `/pages/auth/login?redirect=${encodeURIComponent(buildCurrentRouteRedirect())}` });
        return false;
    }
    return {
        user,
        loading,
        initialized,
        isAuthenticated,
        displayName,
        setUser,
        clearSession,
        bootstrapSession,
        requestPhoneOTP,
        verifyPhoneOTP,
        logout,
        requireAuth,
    };
});
