<template>
    <div class="max-w-[600px] mx-auto p-6 animate-fade-in max-sm:p-4">
        <!-- Back button + title -->
        <header class="flex items-center gap-3 mb-6">
            <button
                class="p-2 bg-transparent rounded-lg text-text-muted cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
                :aria-label="t('common.actions.back')"
                @click="goBack"
            >
                <ArrowLeft class="size-5" />
            </button>
            <h1
                class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0"
            >
                {{ t("user.verification.identity.title") }}
            </h1>
        </header>

        <!-- Loading state -->
        <div v-if="storeLoading && !identity" class="flex justify-center py-12">
            <div
                class="size-8 border-2 border-border border-t-primary rounded-full animate-spin"
                role="status"
                aria-busy="true"
            >
                <span class="sr-only">{{ t("common.actions.loading") }}</span>
            </div>
        </div>

        <!-- Verified status card -->
        <div
            v-else-if="identity?.verified && !showForm"
            class="bg-bg-card border border-green-500/30 rounded-xl p-5 shadow-card"
        >
            <div class="flex items-center gap-3 mb-4">
                <div class="p-2 bg-green-500/10 rounded-lg">
                    <ShieldCheck class="size-6 text-green-500" />
                </div>
                <div>
                    <h2 class="text-base font-bold text-text-primary m-0">
                        {{ t("user.verification.identity.verified") }}
                    </h2>
                    <p class="text-sm text-text-muted m-0 mt-1">
                        {{
                            identity.verifyMethod === "academic_db_match" ||
                            identity.verifyMethod === "tencent_cloud"
                                ? t("user.verification.identity.successAuto")
                                : t("user.verification.identity.successManual")
                        }}
                    </p>
                </div>
            </div>
            <div class="space-y-2 text-sm text-text-secondary">
                <div class="flex justify-between">
                    <span class="text-text-muted">{{
                        t("user.verification.identity.docType")
                    }}</span>
                    <span>{{ docTypeLabel(identity.docType) }}</span>
                </div>
                <div class="flex justify-between">
                    <span class="text-text-muted">{{
                        t("user.verification.identity.realName")
                    }}</span>
                    <span>{{ identity.realName }}</span>
                </div>
                <div v-if="identity.verifiedAt" class="flex justify-between">
                    <span class="text-text-muted">{{
                        t("user.verification.identity.verifiedAt")
                    }}</span>
                    <span>{{
                        new Date(identity.verifiedAt).toLocaleDateString()
                    }}</span>
                </div>
            </div>
        </div>

        <!-- Pending status card -->
        <div
            v-else-if="
                identity &&
                !identity.verified &&
                !identity.reviewedAt &&
                !showForm
            "
            class="bg-bg-card border border-yellow-500/30 rounded-xl p-5 shadow-card"
        >
            <div class="flex items-center gap-3 mb-4">
                <div class="p-2 bg-yellow-500/10 rounded-lg">
                    <Clock class="size-6 text-yellow-500" />
                </div>
                <div>
                    <h2 class="text-base font-bold text-text-primary m-0">
                        {{ t("user.verification.identity.pending") }}
                    </h2>
                    <p class="text-sm text-text-muted m-0 mt-1">
                        {{ t("user.verification.identity.desc") }}
                    </p>
                </div>
            </div>
            <div class="space-y-2 text-sm text-text-secondary">
                <div class="flex justify-between">
                    <span class="text-text-muted">{{
                        t("user.verification.identity.docType")
                    }}</span>
                    <span>{{ docTypeLabel(identity.docType) }}</span>
                </div>
                <div class="flex justify-between">
                    <span class="text-text-muted">{{
                        t("user.verification.identity.realName")
                    }}</span>
                    <span>{{ identity.realName }}</span>
                </div>
            </div>
        </div>

        <!-- Rejected status card -->
        <div
            v-else-if="identity && identity.reviewedAt && !showForm"
            class="bg-bg-card border border-red-500/30 rounded-xl p-5 shadow-card"
        >
            <div class="flex items-center gap-3 mb-4">
                <div class="p-2 bg-red-500/10 rounded-lg">
                    <XCircle class="size-6 text-red-500" />
                </div>
                <div>
                    <h2 class="text-base font-bold text-text-primary m-0">
                        {{ t("user.verification.identity.rejected") }}
                    </h2>
                    <p class="text-sm text-text-muted m-0 mt-1">
                        {{ t("user.verification.identity.rejectionReason") }}:
                        {{ identity.rejectionReason }}
                    </p>
                </div>
            </div>
            <button
                class="mt-4 w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white border-0"
                @click="showForm = true"
            >
                {{ t("user.verification.identity.resubmit") }}
            </button>
        </div>

        <!-- Verification form -->
        <div v-else class="bg-bg-card rounded-xl p-5 shadow-card">
            <p class="text-sm text-text-muted mb-5 m-0">
                {{ t("user.verification.identity.desc") }}
            </p>

            <!-- Step 1: Document type -->
            <fieldset class="mb-5 border-0 p-0 m-0">
                <legend class="text-sm font-semibold text-text-primary mb-3">
                    {{ t("user.verification.identity.docType") }}
                </legend>
                <div class="grid grid-cols-2 gap-2 max-sm:grid-cols-1">
                    <label
                        v-for="dt in docTypes"
                        :key="dt.value"
                        class="flex items-center gap-2 p-3 border rounded-lg cursor-pointer transition-all duration-fast text-sm"
                        :class="
                            form.docType === dt.value
                                ? 'border-primary bg-primary/5 text-text-primary'
                                : 'border-border text-text-secondary hover:border-text-muted'
                        "
                    >
                        <input
                            type="radio"
                            :value="dt.value"
                            v-model="form.docType"
                            class="sr-only"
                        />
                        <span>{{ dt.label }}</span>
                    </label>
                </div>
            </fieldset>

            <!-- Step 2: Real name -->
            <div class="mb-5">
                <label
                    class="block text-sm font-semibold text-text-primary mb-2"
                    for="identity-real-name"
                >
                    {{ t("user.verification.identity.realName") }}
                </label>
                <input
                    id="identity-real-name"
                    data-identity-real-name-input
                    v-model="form.realName"
                    type="text"
                    class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                    :placeholder="t('user.verification.identity.realName')"
                />
            </div>

            <!-- Step 3: Document number -->
            <div class="mb-5">
                <label
                    class="block text-sm font-semibold text-text-primary mb-2"
                    for="identity-doc-number"
                >
                    {{ t("user.verification.identity.docNumber") }}
                </label>
                <input
                    id="identity-doc-number"
                    data-identity-doc-number-input
                    v-model="form.docNumber"
                    type="text"
                    class="w-full px-3 py-2.5 bg-transparent rounded-lg text-sm text-text-primary placeholder-text-muted outline-none transition-all duration-fast focus:border-primary"
                    :class="docNumberError ? 'border-red-500/60' : ''"
                    :placeholder="t('user.verification.identity.docNumber')"
                />
                <p
                    v-if="docNumberError"
                    data-identity-doc-number-error
                    class="text-xs text-red-500 mt-2 mb-0"
                >
                    {{ docNumberError }}
                </p>
            </div>

            <!-- Step 4: Photo upload (non-mainland ID only) -->
            <div v-if="form.docType !== 'MAINLAND_ID'" class="mb-5 space-y-4">
                <!-- Front photo -->
                <div>
                    <label
                        class="block text-sm font-semibold text-text-primary mb-2"
                    >
                        {{ t("user.verification.identity.photoFront") }}
                        <span class="text-red-500 ml-1">*</span>
                    </label>
                    <div
                        v-if="previews.front"
                        class="relative w-full max-w-[240px] aspect-[3/2] rounded-lg overflow-hidden"
                    >
                        <img
                            :src="previews.front"
                            alt=""
                            class="w-full h-full object-cover"
                        />
                        <button
                            class="absolute top-1 right-1 size-6 bg-black/50 border-0 rounded-full flex items-center justify-center cursor-pointer text-white"
                            :aria-label="t('common.actions.delete')"
                            @click="clearPhoto('front')"
                        >
                            <XCircle class="size-4" />
                        </button>
                    </div>
                    <label
                        v-else
                        class="flex flex-col items-center justify-center w-full max-w-[240px] aspect-[3/2] border-2 border-dashed border-border rounded-lg cursor-pointer transition-all duration-fast hover:border-primary text-text-muted"
                    >
                        <Upload class="size-6 mb-1" />
                        <span class="text-xs">{{
                            t("user.verification.identity.photoRequired")
                        }}</span>
                        <input
                            type="file"
                            accept="image/*"
                            class="sr-only"
                            @change="handlePhotoChange($event, 'front')"
                        />
                    </label>
                </div>

                <!-- Back photo -->
                <div>
                    <label
                        class="block text-sm font-semibold text-text-primary mb-2"
                    >
                        {{ t("user.verification.identity.photoBack") }}
                    </label>
                    <div
                        v-if="previews.back"
                        class="relative w-full max-w-[240px] aspect-[3/2] rounded-lg overflow-hidden"
                    >
                        <img
                            :src="previews.back"
                            alt=""
                            class="w-full h-full object-cover"
                        />
                        <button
                            class="absolute top-1 right-1 size-6 bg-black/50 border-0 rounded-full flex items-center justify-center cursor-pointer text-white"
                            :aria-label="t('common.actions.delete')"
                            @click="clearPhoto('back')"
                        >
                            <XCircle class="size-4" />
                        </button>
                    </div>
                    <label
                        v-else
                        class="flex flex-col items-center justify-center w-full max-w-[240px] aspect-[3/2] border-2 border-dashed border-border rounded-lg cursor-pointer transition-all duration-fast hover:border-primary text-text-muted"
                    >
                        <Upload class="size-6 mb-1" />
                        <span class="text-xs">{{
                            t("user.verification.identity.photoBack")
                        }}</span>
                        <input
                            type="file"
                            accept="image/*"
                            class="sr-only"
                            @change="handlePhotoChange($event, 'back')"
                        />
                    </label>
                </div>

                <!-- Selfie -->
                <div>
                    <label
                        class="block text-sm font-semibold text-text-primary mb-2"
                    >
                        {{ t("user.verification.identity.photoSelfie") }}
                        <span class="text-red-500 ml-1">*</span>
                    </label>
                    <div
                        v-if="previews.selfie"
                        class="relative w-full max-w-[240px] aspect-[3/2] rounded-lg overflow-hidden"
                    >
                        <img
                            :src="previews.selfie"
                            alt=""
                            class="w-full h-full object-cover"
                        />
                        <button
                            class="absolute top-1 right-1 size-6 bg-black/50 border-0 rounded-full flex items-center justify-center cursor-pointer text-white"
                            :aria-label="t('common.actions.delete')"
                            @click="clearPhoto('selfie')"
                        >
                            <XCircle class="size-4" />
                        </button>
                    </div>
                    <label
                        v-else
                        class="flex flex-col items-center justify-center w-full max-w-[240px] aspect-[3/2] border-2 border-dashed border-border rounded-lg cursor-pointer transition-all duration-fast hover:border-primary text-text-muted"
                    >
                        <Camera class="size-6 mb-1" />
                        <span class="text-xs">{{
                            t("user.verification.identity.photoRequired")
                        }}</span>
                        <input
                            type="file"
                            accept="image/*"
                            class="sr-only"
                            @change="handlePhotoChange($event, 'selfie')"
                        />
                    </label>
                </div>
            </div>

            <!-- Submit button -->
            <button
                data-identity-submit
                class="w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white border-0 disabled:opacity-50 disabled:cursor-not-allowed"
                :disabled="!canSubmit || submitting"
                @click="handleSubmit"
            >
                <span v-if="submitting" class="inline-flex items-center gap-2">
                    <span
                        class="size-4 border-2 border-bg-base border-t-transparent rounded-full animate-spin"
                    />
                    {{ t("common.actions.loading") }}
                </span>
                <span v-else>
                    {{ t("user.verification.identity.submit") }}
                </span>
            </button>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
    ShieldCheck,
    Clock,
    XCircle,
    ArrowLeft,
    Upload,
    Camera,
} from "lucide-vue-next";
import { useVerificationStore } from "@/stores/verification";
import { useToast } from "@/composables/useToast";
import {
    isValidMainlandIDNumber,
    normalizeMainlandIDNumber,
} from "@/modules/user/utils/mainlandID";

const router = useRouter();
const { t } = useI18n();
const store = useVerificationStore();
const toast = useToast();

const identity = computed(() => store.identity);
const storeLoading = computed(() => store.loading);
const showForm = ref(false);
const submitting = ref(false);

type DocType = "MAINLAND_ID" | "HK_MACAU" | "TW" | "PASSPORT";

const docTypes = computed(() => [
    {
        value: "MAINLAND_ID" as const,
        label: t("user.verification.identity.mainlandId"),
    },
    {
        value: "HK_MACAU" as const,
        label: t("user.verification.identity.hkMacau"),
    },
    { value: "TW" as const, label: t("user.verification.identity.twPass") },
    {
        value: "PASSPORT" as const,
        label: t("user.verification.identity.passport"),
    },
]);

const form = reactive({
    docType: "MAINLAND_ID" as DocType,
    realName: "",
    docNumber: "",
});

const photos = reactive<{
    front: IdentityPhotoFile | null;
    back: IdentityPhotoFile | null;
    selfie: IdentityPhotoFile | null;
}>({
    front: null,
    back: null,
    selfie: null,
});

const previews = reactive<{
    front: string | null;
    back: string | null;
    selfie: string | null;
}>({
    front: null,
    back: null,
    selfie: null,
});

function goBack() {
    void router.push("/identity");
}

function docTypeLabel(docType: string): string {
    const found = docTypes.value.find((dt) => dt.value === docType);
    return found ? found.label : docType;
}

const canSubmit = computed(() => {
    if (!form.docType || !form.realName.trim() || !form.docNumber.trim())
        return false;
    if (docNumberError.value) return false;
    if (form.docType !== "MAINLAND_ID") {
        if (!photos.front || !photos.selfie) return false;
    }
    return true;
});

const docNumberError = computed(() => {
    if (form.docType !== "MAINLAND_ID" || !form.docNumber.trim()) {
        return "";
    }
    return isValidMainlandIDNumber(form.docNumber)
        ? ""
        : t("user.verification.identity.invalidMainlandID");
});

const MAX_PHOTO_SIZE = 5 * 1024 * 1024; // 5MB
const ALLOWED_PHOTO_TYPES = ["image/jpeg", "image/png", "image/webp"] as const;
type AllowedPhotoType = (typeof ALLOWED_PHOTO_TYPES)[number];
type IdentityPhotoFile = File & { type: AllowedPhotoType };

function isAllowedPhotoType(
    contentType: string,
): contentType is AllowedPhotoType {
    return (ALLOWED_PHOTO_TYPES as readonly string[]).includes(contentType);
}

function handlePhotoChange(event: Event, key: "front" | "back" | "selfie") {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    if (file.size > MAX_PHOTO_SIZE) {
        toast.error(
            t("user.verification.identity.photoTooLarge", { max: "5MB" }),
        );
        input.value = "";
        return;
    }
    if (!isAllowedPhotoType(file.type)) {
        toast.error(t("user.verification.identity.photoInvalidType"));
        input.value = "";
        return;
    }

    const reader = new FileReader();
    reader.onload = () => {
        const result = reader.result as string;
        photos[key] = file as IdentityPhotoFile;
        previews[key] = result;
    };
    reader.readAsDataURL(file);

    // Reset input so the same file can be re-selected
    input.value = "";
}

function clearPhoto(key: "front" | "back" | "selfie") {
    photos[key] = null;
    previews[key] = null;
}

async function handleSubmit() {
    if (!canSubmit.value || submitting.value) return;

    submitting.value = true;
    try {
        const docNumber =
            form.docType === "MAINLAND_ID"
                ? normalizeMainlandIDNumber(form.docNumber)
                : form.docNumber.trim();
        const uploaded = {
            front:
                form.docType !== "MAINLAND_ID" && photos.front
                    ? await uploadPhoto("front", photos.front)
                    : undefined,
            back:
                form.docType !== "MAINLAND_ID" && photos.back
                    ? await uploadPhoto("back", photos.back)
                    : undefined,
            selfie:
                form.docType !== "MAINLAND_ID" && photos.selfie
                    ? await uploadPhoto("selfie", photos.selfie)
                    : undefined,
        };
        await store.submitIdentity({
            docType: form.docType,
            realName: form.realName.trim(),
            docNumber,
            ...(form.docType !== "MAINLAND_ID" && {
                docPhotoFront: uploaded.front,
                docPhotoBack: uploaded.back,
                docPhotoSelfie: uploaded.selfie,
            }),
        });
        await store.fetchStatus();
        showForm.value = false;
    } catch (err) {
        toast.error(
            err instanceof Error
                ? err.message
                : t("user.verification.identity.submit"),
        );
    } finally {
        submitting.value = false;
    }
}

onMounted(() => {
    store.fetchStatus().catch(() => {
        toast.error(t("common.loadFailed"));
    });
});

async function uploadPhoto(
    slot: "front" | "back" | "selfie",
    file: IdentityPhotoFile,
): Promise<string> {
    const key = await store.uploadIdentityPhoto({
        slot,
        filename: file.name,
        contentType: file.type,
        dataBase64: await fileToBase64(file),
    });
    if (!key) {
        throw new Error(t("user.verification.identity.uploadFailed"));
    }
    return key;
}

function fileToBase64(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
            const result = String(reader.result ?? "");
            const comma = result.indexOf(",");
            resolve(comma >= 0 ? result.slice(comma + 1) : result);
        };
        reader.onerror = () =>
            reject(reader.error ?? new Error("failed to read file"));
        reader.readAsDataURL(file);
    });
}
</script>
