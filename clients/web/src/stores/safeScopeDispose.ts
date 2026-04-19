import { getCurrentScope, onScopeDispose } from "vue";

// safeOnScopeDispose 只在存在活动 effect scope 时注册清理逻辑，
// 避免 store 在测试环境初始化时触发 Vue 告警。
export function safeOnScopeDispose(cleanup: () => void): void {
    if (getCurrentScope()) {
        onScopeDispose(cleanup);
    }
}
