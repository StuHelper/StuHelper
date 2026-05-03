# Admission Frontend And Admin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the user admission page, camera-only freshman submission, old-student email/SSO flows, and admin review/policy screens.

**Architecture:** The public web app owns `auth.stuhelper.com/admission/a/:code`; it previews token state before login, redirects unauthenticated users through existing SSO login/signup, then links the token after authentication. Vben Admin consumes generated shared API clients for review and policy operations.

**Tech Stack:** Vue 3, TypeScript, Pinia, Vite, Element Plus/Vben Admin, generated `@stuhelper/shared` OpenAPI clients, Vitest.

---

## File Structure

- Create `clients/web/src/modules/admission/{api,admissionToken,cameraCapture}.ts`, `clients/web/src/modules/admission/views/AdmissionPage.vue`, and tests under `clients/web/src/modules/admission/__tests__/`.
- Modify `clients/web/src/router/index.ts`, `clients/web/src/stores/auth.ts`, and `clients/web/src/i18n/index.ts` if route titles need translation.
- Create `clients/admin/apps/web-ele/src/api/admin/admission.ts`, `clients/admin/apps/web-ele/src/views/users/freshman-verification/index.vue`, `clients/admin/apps/web-ele/src/views/users/admission-policy/index.vue`, and tests next to routes/API files.
- Modify `clients/admin/apps/web-ele/src/router/routes/modules/user-system.ts`, `clients/admin/apps/web-ele/src/locales/langs/zh-CN/admin.json`, and `clients/admin/apps/web-ele/src/locales/langs/en-US/admin.json`.

## Task 1: Web Admission Token Page

**Files:** `clients/web/src/router/index.ts`, `clients/web/src/modules/admission/api.ts`, `clients/web/src/modules/admission/admissionToken.ts`, `clients/web/src/modules/admission/views/AdmissionPage.vue`, `clients/web/src/modules/admission/__tests__/admissionToken.test.ts`

- [ ] **Step 1: Write failing token tests**

Test `buildAdmissionReturnURL('/admission/a/ABCD?qq=123')` returns same-origin absolute URL, rejects `//evil.example`, rejects non-admission paths, and preserves `qq`.

- [ ] **Step 2: Run failing tests**

Run: `cd clients/web && pnpm exec vitest run src/modules/admission/__tests__/admissionToken.test.ts`
Expected: FAIL because the module does not exist.

- [ ] **Step 3: Implement API wrapper**

Add methods `getAdmissionSession(token, qq)`, `linkAdmissionSession(token, qq)`, `getAdmissionMe()`, `submitFreshmanApplication(payload)`, `uploadCameraCapture(applicationId, payload)`, `requestSchoolEmailOTP(payload)`, and `verifySchoolEmailOTP(payload)` using generated shared clients.

- [ ] **Step 4: Implement route and page**

Add route `/admission/a/:code` with `layout: "none"`. Page states are `loading`, `needsLogin`, `ready`, `linked`, `submitted`, `approved`, `expired`, and `error`. For `needsLogin`, call `auth.login(currentAdmissionURL)` and `auth.signup(currentAdmissionURL)`.

- [ ] **Step 5: Run tests and commit**

Run: `cd clients/web && pnpm exec vitest run src/modules/admission/__tests__/admissionToken.test.ts src/router/__tests__/auth-guard-decision.test.ts && pnpm type-check`
Expected: PASS.
Commit: `git add clients/web/src && git commit -m "feat: add admission landing page"`

## Task 2: Camera-Only Freshman Flow

**Files:** `clients/web/src/modules/admission/cameraCapture.ts`, `clients/web/src/modules/admission/views/AdmissionPage.vue`, `clients/web/src/modules/admission/__tests__/cameraCapture.test.ts`

- [ ] **Step 1: Write failing camera tests**

Cover `supportsCameraCapture()` returning false without `navigator.mediaDevices.getUserMedia`, `buildCameraConstraints()` requesting an environment-facing camera, and `AdmissionPage` rendering no `input[type="file"]` in the freshman material flow.

- [ ] **Step 2: Run failing test**

Run: `cd clients/web && pnpm exec vitest run src/modules/admission/__tests__/cameraCapture.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement camera helper**

Expose `supportsCameraCapture`, `buildCameraConstraints`, `startCameraStream`, `captureFrameAsBase64`, `stopCameraStream`, and `MAX_CAMERA_IMAGE_BYTES = 10 * 1024 * 1024`. Capture from a live video stream into canvas; do not expose file-input helpers or accept user-selected files.

- [ ] **Step 4: Wire page submission**

Render old-student and freshman tabs after link success. Freshman tab shows school, name, major, material type, live camera preview, capture button, preview, retake, and submit. If camera access is unavailable or denied, show a fixed prompt to reopen the link on a phone browser; it must not render ordinary upload, file input, drag-drop, PDF, gallery, or album language.

- [ ] **Step 5: Run tests and commit**

Run: `cd clients/web && pnpm exec vitest run src/modules/admission/__tests__ && pnpm type-check`
Expected: PASS.
Commit: `git add clients/web/src/modules/admission && git commit -m "feat: add freshman camera submission"`

## Task 3: Old-Student Email And School SSO UI

**Files:** `clients/web/src/modules/admission/views/AdmissionPage.vue`, `clients/web/src/modules/admission/__tests__/oldStudentFlow.test.ts`

- [ ] **Step 1: Write failing UI state tests**

Mount `AdmissionPage` with mocked API; assert old-student path exposes school official SSO button when `schoolSsoEnabled=true`, exposes email OTP form when `emailOtpEnabled=true`, and hides freshman submission after formal credential exists.

- [ ] **Step 2: Run failing test**

Run: `cd clients/web && pnpm exec vitest run src/modules/admission/__tests__/oldStudentFlow.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement old-student actions**

Official SSO button navigates to `/api/v1/admission/school-sso/{schoolID}/login?return=<current admission URL>`. Email form calls request OTP, then verify OTP, then refreshes admission status and shows verified state.

- [ ] **Step 4: Run tests and commit**

Run: `cd clients/web && pnpm exec vitest run src/modules/admission/__tests__ && pnpm type-check`
Expected: PASS.
Commit: `git add clients/web/src/modules/admission && git commit -m "feat: add old student admission verification"`

## Task 4: Admin API And Routes

**Files:** `clients/admin/apps/web-ele/src/api/admin/admission.ts`, `clients/admin/apps/web-ele/src/api/admin/index.ts`, `clients/admin/apps/web-ele/src/router/routes/modules/user-system.ts`, `clients/admin/apps/web-ele/src/router/routes/modules/user-system.test.ts`, locale JSON files.

- [ ] **Step 1: Write failing route/API tests**

Assert `UserSystem` route contains `FreshmanVerification` with authority `admission:freshman:review` and `AdmissionPolicy` with authority `admission:policy:update`.

- [ ] **Step 2: Run failing tests**

Run: `cd clients/admin && pnpm --filter @vben/web-ele exec vitest run apps/web-ele/src/router/routes/modules/user-system.test.ts apps/web-ele/src/api/shared-client.test.ts`
Expected: FAIL.

- [ ] **Step 3: Add API wrapper**

Wrap generated calls `listFreshmanVerifications`, `getFreshmanVerification`, `reviewFreshmanVerification`, `listAdmissionPolicies`, `updateAdmissionPolicy`, `releaseAdmissionBlacklist`, and `listAdmissionSessions`.

- [ ] **Step 4: Add routes and locales**

Add menu entries under `/users/freshman-verification` and `/users/admission-policy`, using icons `lucide:file-check-2` and `lucide:shield-check`.

- [ ] **Step 5: Run tests and commit**

Run: `cd clients/admin && pnpm --filter @vben/web-ele exec vitest run apps/web-ele/src/router/routes/modules/user-system.test.ts && pnpm type-check`
Expected: PASS.
Commit: `git add clients/admin/apps/web-ele/src && git commit -m "feat: add admission admin routes"`

## Task 5: Admin Review And Policy Views

**Files:** `clients/admin/apps/web-ele/src/views/users/freshman-verification/index.vue`, `clients/admin/apps/web-ele/src/views/users/admission-policy/index.vue`, `clients/admin/apps/web-ele/src/api/admin/admission.ts`

- [ ] **Step 1: Write failing view tests**

Test freshman list renders status, school, QQ, application time, failure count, approve, approve with days, reject, and material preview. Test policy view renders freshman channel switch, close time, provisional expiry, mute duration, token TTL, wait timeout, reminder interval, failed limit, permanent blacklist, management group, and raw material forwarding switch.

- [ ] **Step 2: Run failing tests**

Run: `cd clients/admin && pnpm --filter @vben/web-ele exec vitest run apps/web-ele/src/views/users/freshman-verification apps/web-ele/src/views/users/admission-policy`
Expected: FAIL.

- [ ] **Step 3: Implement Vben views**

Use existing Element Plus table/form patterns from `views/users/student-verification/index.vue` and `views/users/school-config/index.vue`. Show material image in a preview dialog, mask applicant display where API provides masked fields, and require rejection reason before calling reject.

- [ ] **Step 4: Run tests and commit**

Run: `cd clients/admin && pnpm --filter @vben/web-ele exec vitest run apps/web-ele/src/views/users && pnpm type-check`
Expected: PASS.
Commit: `git add clients/admin/apps/web-ele/src && git commit -m "feat: add admission admin pages"`

## Self-Review

- Spec coverage: web covers auth-domain admission URL, SSO login/signup return, token mismatch display, old-student email/SSO, camera-only freshman material, and no gallery upload. Admin covers review, per-application expiry override, policy config, and blacklist release.
- No placeholders: every task names files, commands, expected results, and commit boundary.
- Type consistency: frontend uses generated OpenAPI clients after backend Task 1 generation.
