<template>
    <div class="login-page">
        <div class="login-card">
            <h1>StuHelper</h1>
            <p class="subtitle">航小伴</p>

            <div class="login-buttons">
                <button
                    class="btn-primary"
                    @click="handleLogin"
                    :disabled="loading"
                >
                    {{ loading ? "跳转中..." : "使用 SSO 登录" }}
                </button>

                <button
                    class="btn-secondary"
                    @click="handleSignup"
                    :disabled="loading"
                >
                    注册新账号
                </button>
            </div>

            <p class="hint">使用 StuHelper SSO 统一身份认证登录</p>
        </div>
    </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { ElMessage } from "element-plus";
import { useAuthStore } from "@/stores/auth";

const authStore = useAuthStore();
const { loading } = storeToRefs(authStore);

const handleLogin = async () => {
    try {
        await authStore.login();
    } catch (e) {
        const message = e instanceof Error ? e.message : "登录失败";
        ElMessage.error(message);
    }
};

const handleSignup = async () => {
    try {
        await authStore.signup();
    } catch (e) {
        const message = e instanceof Error ? e.message : "注册失败";
        ElMessage.error(message);
    }
};
</script>

<style scoped>
.login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
    background: white;
    padding: 3rem;
    border-radius: 1rem;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
    text-align: center;
    max-width: 400px;
    width: 90%;
}

h1 {
    margin: 0 0 0.5rem;
    color: #333;
    font-size: 2rem;
}

.subtitle {
    color: #666;
    margin-bottom: 2rem;
}

.login-buttons {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.btn-primary,
.btn-secondary {
    padding: 0.875rem 1.5rem;
    border: none;
    border-radius: 0.5rem;
    font-size: 1rem;
    cursor: pointer;
    transition: all 0.2s;
}

.btn-primary {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
}

.btn-primary:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.btn-secondary {
    background: #f5f5f5;
    color: #333;
}

.btn-secondary:hover:not(:disabled) {
    background: #eee;
}

.btn-primary:disabled,
.btn-secondary:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.hint {
    margin-top: 1.5rem;
    color: #999;
    font-size: 0.875rem;
}
</style>
