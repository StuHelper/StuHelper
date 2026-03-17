<template>
    <div>
        <div class="mb-4 flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900">系统配置</h2>
        </div>

        <!-- Table -->
        <el-table v-loading="loading" :data="list" border stripe>
            <el-table-column prop="key" label="配置键" width="220" />
            <el-table-column label="配置值" min-width="200">
                <template #default="{ row }">
                    <span v-if="editingKey !== row.key">{{ row.value }}</span>
                    <el-input
                        v-else
                        v-model="editingValue"
                        size="small"
                        style="width: 100%"
                        @keyup.enter="handleSaveInline(row)"
                        @keyup.escape="cancelEdit"
                    />
                </template>
            </el-table-column>
            <el-table-column
                prop="description"
                label="描述"
                min-width="200"
                show-overflow-tooltip
            />
            <el-table-column label="更新时间" width="180">
                <template #default="{ row }">
                    {{ formatDate(row.updatedAt) }}
                </template>
            </el-table-column>
            <el-table-column label="操作" width="160" fixed="right">
                <template #default="{ row }">
                    <template v-if="editingKey !== row.key">
                        <el-button size="small" @click="startEdit(row)"
                            >编辑</el-button
                        >
                    </template>
                    <template v-else>
                        <el-button
                            size="small"
                            type="primary"
                            :loading="submitting"
                            @click="handleSaveInline(row)"
                            >保存</el-button
                        >
                        <el-button size="small" @click="cancelEdit"
                            >取消</el-button
                        >
                    </template>
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import type { components } from "@stuhelper/shared";
import { ElMessage } from "element-plus";
import { api } from "@/api";

type SystemConfig = components["schemas"]["SystemConfig"];

const loading = ref(false);
const submitting = ref(false);
const list = ref<SystemConfig[]>([]);

const editingKey = ref("");
const editingValue = ref("");

function formatDate(dateStr?: string): string {
    if (!dateStr) return "—";
    const d = new Date(dateStr);
    return d.toLocaleString("zh-CN", { hour12: false });
}

async function fetchList() {
    loading.value = true;
    try {
        const res = await api.userAdmin.listSystemConfigs();
        list.value = res.data?.data ?? [];
    } catch (err) {
        ElMessage.error(getApiErrorMessage(err, "获取数据失败"));
    } finally {
        loading.value = false;
    }
}

function startEdit(row: SystemConfig) {
    editingKey.value = row.key;
    editingValue.value = row.value;
}

function cancelEdit() {
    editingKey.value = "";
    editingValue.value = "";
}

async function handleSaveInline(row: SystemConfig) {
    submitting.value = true;
    try {
        await api.userAdmin.updateSystemConfig(row.key, {
            value: editingValue.value,
        });
        row.value = editingValue.value;
        row.updatedAt = new Date().toISOString();
        ElMessage.success("保存成功");
        cancelEdit();
    } catch (err) {
        ElMessage.error(getApiErrorMessage(err, "操作失败"));
    } finally {
        submitting.value = false;
    }
}

function getApiErrorMessage(error: unknown, fallback: string): string {
    if (error && typeof error === "object") {
        const err = error as {
            response?: {
                data?: { error?: { message?: string } };
                error?: { message?: string };
            };
            error?: { message?: string };
            message?: string;
        };
        const body =
            err.response?.data?.error ?? err.response?.error ?? err.error;
        if (body?.message) {
            return body.message;
        }
        if (err.message) {
            return err.message;
        }
    }
    return fallback;
}

onMounted(fetchList);
</script>
