<template>
    <div>
        <div class="mb-4 flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900">学校配置</h2>
        </div>

        <!-- Table -->
        <el-table v-loading="loading" :data="list" border stripe>
            <el-table-column prop="schoolID" label="学校ID" width="140" />
            <el-table-column
                prop="schoolName"
                label="学校名称"
                min-width="160"
            />
            <el-table-column label="认证方式" width="130">
                <template #default="{ row }">
                    {{ verificationMethodLabel(row.verificationMethod) }}
                </template>
            </el-table-column>
            <el-table-column label="启用状态" width="110">
                <template #default="{ row }">
                    <el-switch
                        v-model="row.enabled"
                        :loading="row._toggling"
                        @change="handleToggleEnabled(row)"
                    />
                </template>
            </el-table-column>
            <el-table-column label="学籍表" width="160">
                <template #default="{ row }">
                    <span class="text-sm text-gray-700">
                        {{ row.academicDbTable || "未配置" }}
                    </span>
                </template>
            </el-table-column>
            <el-table-column label="LDAP 配置" width="140">
                <template #default="{ row }">
                    <el-tag
                        :type="row.ldapConfig ? 'success' : 'info'"
                        size="small"
                    >
                        {{ row.ldapConfig ? "已配置" : "未配置" }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
                <template #default="{ row }">
                    <el-button size="small" @click="openEditDialog(row)"
                        >编辑</el-button
                    >
                </template>
            </el-table-column>
        </el-table>

        <!-- Edit Dialog -->
        <el-dialog
            v-model="editDialogVisible"
            title="编辑学校配置"
            width="680px"
            @closed="resetForm"
        >
            <el-form
                ref="formRef"
                :model="form"
                :rules="formRules"
                label-width="100px"
            >
                <el-form-item label="学校名称" prop="schoolName">
                    <el-input
                        v-model="form.schoolName"
                        placeholder="请输入学校名称"
                    />
                </el-form-item>
                <el-form-item label="认证方式" prop="verificationMethod">
                    <el-select
                        v-model="form.verificationMethod"
                        style="width: 100%"
                    >
                        <el-option label="统一身份认证" value="ldap" />
                        <el-option label="人工审核" value="manual" />
                    </el-select>
                </el-form-item>
                <el-form-item label="学籍表" prop="academicDbTable">
                    <el-input
                        v-model="form.academicDbTable"
                        placeholder="academic.students"
                    />
                    <div class="mt-1 text-xs text-gray-500">
                        统一使用 schema.table 格式。启用统一身份认证时必须配置。
                    </div>
                </el-form-item>
                <template v-if="showLdapSection">
                    <el-divider content-position="left"
                        >统一身份认证配置</el-divider
                    >
                    <el-form-item label="LDAP 地址" prop="ldapConfig.url">
                        <el-input
                            v-model="form.ldapConfig.url"
                            placeholder="ldaps://ldap.example.edu:636"
                        />
                    </el-form-item>
                    <el-form-item label="Base DN" prop="ldapConfig.baseDN">
                        <el-input
                            v-model="form.ldapConfig.baseDN"
                            placeholder="ou=users,dc=example,dc=edu"
                        />
                    </el-form-item>
                    <el-form-item
                        label="系统绑定 DN"
                        prop="ldapConfig.systemBindDN"
                    >
                        <el-input
                            v-model="form.ldapConfig.systemBindDN"
                            placeholder="cn=system,dc=example,dc=edu"
                        />
                    </el-form-item>
                    <el-form-item
                        label="绑定密码"
                        prop="ldapConfig.systemBindPassword"
                    >
                        <el-input
                            v-model="form.ldapConfig.systemBindPassword"
                            type="password"
                            show-password
                            :placeholder="
                                form.ldapConfig.hasSystemBindPassword
                                    ? '留空则保留当前密码'
                                    : '请输入系统绑定密码'
                            "
                        />
                        <div class="mt-1 text-xs text-gray-500">
                            {{
                                form.ldapConfig.hasSystemBindPassword
                                    ? "当前已保存系统绑定密码，留空不会覆盖。"
                                    : "首次启用统一身份认证时需要填写系统绑定密码。"
                            }}
                        </div>
                    </el-form-item>
                    <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                        <el-form-item label="StartTLS" prop="ldapConfig.useTLS">
                            <el-switch v-model="form.ldapConfig.useTLS" />
                        </el-form-item>
                        <el-form-item
                            label="跳过证书校验"
                            prop="ldapConfig.insecureSkipVerify"
                        >
                            <el-switch
                                v-model="form.ldapConfig.insecureSkipVerify"
                            />
                        </el-form-item>
                    </div>
                </template>
                <el-form-item label="同意条款" prop="consentText">
                    <el-input
                        v-model="form.consentText"
                        type="textarea"
                        :rows="4"
                        placeholder="用户同意条款文本（可选）"
                    />
                </el-form-item>
                <el-form-item label="启用状态">
                    <el-switch v-model="form.enabled" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="editDialogVisible = false">取消</el-button>
                <el-button
                    type="primary"
                    :loading="submitting"
                    @click="handleSave"
                    >保存</el-button
                >
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { components } from "@stuhelper/shared";
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { api } from "@/api";

type SchoolConfigRow = components["schemas"]["AdminSchoolConfig"] & {
    _toggling?: boolean;
};
type VerificationMethod =
    components["schemas"]["AdminSchoolConfig"]["verificationMethod"];
type SchoolLDAPConfigView = components["schemas"]["SchoolLDAPConfigView"];
type SchoolLDAPConfigInput = components["schemas"]["SchoolLDAPConfigInput"];

type EditableSchoolLDAPConfig = {
    url: string;
    baseDN: string;
    systemBindDN: string;
    systemBindPassword: string;
    useTLS: boolean;
    insecureSkipVerify: boolean;
    hasSystemBindPassword: boolean;
};

type SchoolConfigForm = {
    schoolName: string;
    verificationMethod: VerificationMethod;
    consentText: string;
    enabled: boolean;
    academicDbTable: string;
    ldapConfig: EditableSchoolLDAPConfig;
};

const loading = ref(false);
const submitting = ref(false);
const list = ref<SchoolConfigRow[]>([]);

const editDialogVisible = ref(false);
const editingSchoolID = ref("");
const formRef = ref<FormInstance>();
const form = ref<SchoolConfigForm>({
    schoolName: "",
    verificationMethod: "manual",
    consentText: "",
    enabled: true,
    academicDbTable: "",
    ldapConfig: createEmptyLdapForm(),
});

const showLdapSection = computed(
    () =>
        form.value.verificationMethod === "ldap" ||
        form.value.ldapConfig.url !== "" ||
        form.value.ldapConfig.baseDN !== "" ||
        form.value.ldapConfig.systemBindDN !== "" ||
        form.value.ldapConfig.hasSystemBindPassword,
);

const formRules: FormRules = {
    schoolName: [
        { required: true, message: "请输入学校名称", trigger: "blur" },
    ],
    verificationMethod: [
        { required: true, message: "请选择认证方式", trigger: "change" },
    ],
    academicDbTable: [
        {
            validator: validateAcademicDbTable,
            trigger: "blur",
        },
    ],
    "ldapConfig.url": [
        { validator: validateRequiredLdapField("LDAP 地址"), trigger: "blur" },
    ],
    "ldapConfig.baseDN": [
        { validator: validateRequiredLdapField("Base DN"), trigger: "blur" },
    ],
    "ldapConfig.systemBindDN": [
        {
            validator: validateRequiredLdapField("系统绑定 DN"),
            trigger: "blur",
        },
    ],
    "ldapConfig.systemBindPassword": [
        { validator: validateSystemBindPassword, trigger: "blur" },
    ],
};

function verificationMethodLabel(method: string): string {
    const map: Record<string, string> = {
        ldap: "统一身份认证",
        manual: "人工审核",
    };
    return map[method] ?? method;
}

function createEmptyLdapForm(): EditableSchoolLDAPConfig {
    return {
        url: "",
        baseDN: "",
        systemBindDN: "",
        systemBindPassword: "",
        useTLS: false,
        insecureSkipVerify: false,
        hasSystemBindPassword: false,
    };
}

function toEditableLdapForm(
    ldapConfig?: SchoolLDAPConfigView,
): EditableSchoolLDAPConfig {
    return {
        url: ldapConfig?.url ?? "",
        baseDN: ldapConfig?.baseDN ?? "",
        systemBindDN: ldapConfig?.systemBindDN ?? "",
        systemBindPassword: "",
        useTLS: ldapConfig?.useTLS ?? false,
        insecureSkipVerify: ldapConfig?.insecureSkipVerify ?? false,
        hasSystemBindPassword: ldapConfig?.hasSystemBindPassword ?? false,
    };
}

function trimToUndefined(value: string): string | undefined {
    const trimmed = value.trim();
    return trimmed ? trimmed : undefined;
}

function buildLdapPayload(): SchoolLDAPConfigInput | undefined {
    const payload: SchoolLDAPConfigInput = {
        url: form.value.ldapConfig.url.trim(),
        baseDN: form.value.ldapConfig.baseDN.trim(),
        systemBindDN: form.value.ldapConfig.systemBindDN.trim(),
        useTLS: form.value.ldapConfig.useTLS,
        insecureSkipVerify: form.value.ldapConfig.insecureSkipVerify,
    };

    const password = trimToUndefined(form.value.ldapConfig.systemBindPassword);
    if (password !== undefined) {
        payload.systemBindPassword = password;
    }

    const hasVisibleConfig =
        payload.url !== "" ||
        payload.baseDN !== "" ||
        payload.systemBindDN !== "" ||
        payload.useTLS ||
        payload.insecureSkipVerify ||
        password !== undefined ||
        form.value.ldapConfig.hasSystemBindPassword;

    if (!hasVisibleConfig) {
        return undefined;
    }

    return payload;
}

async function fetchList() {
    loading.value = true;
    try {
        const res = await api.userAdmin.listSchoolConfigs();
        list.value = (res.data?.data ?? []).map((item) => ({
            ...item,
            _toggling: false,
        }));
    } catch (err) {
        ElMessage.error(getApiErrorMessage(err, "获取数据失败"));
    } finally {
        loading.value = false;
    }
}

async function handleToggleEnabled(row: SchoolConfigRow) {
    row._toggling = true;
    try {
        await api.userAdmin.updateSchoolConfig(row.schoolID, {
            enabled: row.enabled,
        });
        ElMessage.success(row.enabled ? "已启用" : "已禁用");
        await fetchList();
    } catch (err) {
        row.enabled = !row.enabled;
        ElMessage.error(getApiErrorMessage(err, "操作失败"));
    } finally {
        row._toggling = false;
    }
}

function openEditDialog(row: SchoolConfigRow) {
    editingSchoolID.value = row.schoolID;
    form.value = {
        schoolName: row.schoolName,
        verificationMethod: row.verificationMethod,
        consentText: row.consentText ?? "",
        enabled: row.enabled,
        academicDbTable: row.academicDbTable ?? "",
        ldapConfig: toEditableLdapForm(row.ldapConfig),
    };
    editDialogVisible.value = true;
}

function resetForm() {
    editingSchoolID.value = "";
    form.value = {
        schoolName: "",
        verificationMethod: "manual",
        consentText: "",
        enabled: true,
        academicDbTable: "",
        ldapConfig: createEmptyLdapForm(),
    };
    formRef.value?.clearValidate();
}

async function handleSave() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    submitting.value = true;
    try {
        const payload: components["schemas"]["UpdateSchoolConfigRequest"] = {
            schoolName: form.value.schoolName.trim(),
            verificationMethod: form.value.verificationMethod,
            consentText: trimToUndefined(form.value.consentText),
            enabled: form.value.enabled,
            academicDbTable: trimToUndefined(form.value.academicDbTable),
        };

        const ldapConfig = buildLdapPayload();
        if (ldapConfig !== undefined) {
            payload.ldapConfig = ldapConfig;
        }

        await api.userAdmin.updateSchoolConfig(editingSchoolID.value, {
            ...payload,
        });
        ElMessage.success("保存成功");
        editDialogVisible.value = false;
        await fetchList();
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

function validateAcademicDbTable(
    _rule: unknown,
    value: string,
    callback: (err?: Error) => void,
) {
    const trimmed = value.trim();
    if (
        form.value.verificationMethod === "ldap" &&
        form.value.enabled &&
        trimmed === ""
    ) {
        callback(new Error("启用统一身份认证时必须填写学籍表"));
        return;
    }
    if (
        trimmed !== "" &&
        !/^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*$/.test(trimmed)
    ) {
        callback(new Error("学籍表名必须是 schema.table 格式"));
        return;
    }
    callback();
}

function validateRequiredLdapField(label: string) {
    return (_rule: unknown, value: string, callback: (err?: Error) => void) => {
        if (form.value.verificationMethod !== "ldap" || !form.value.enabled) {
            callback();
            return;
        }
        if (!value.trim()) {
            callback(new Error(`${label}不能为空`));
            return;
        }
        callback();
    };
}

function validateSystemBindPassword(
    _rule: unknown,
    value: string,
    callback: (err?: Error) => void,
) {
    if (form.value.verificationMethod !== "ldap" || !form.value.enabled) {
        callback();
        return;
    }
    if (!form.value.ldapConfig.hasSystemBindPassword && !value.trim()) {
        callback(new Error("请输入系统绑定密码"));
        return;
    }
    callback();
}

onMounted(fetchList);
</script>
