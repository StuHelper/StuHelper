# Manual Student Verification Flow

> Executable contract for the `POST /api/v1/user/profile/verify` flow when a school uses `verificationMethod = manual`.

---

## 1. Scope / Trigger

Use this guide when changing any of the following:

- `server/api/components/schemas/user-system.yaml`
- `server/api/paths/user-identity.yaml`
- `server/api/paths/admin-user-system.yaml`
- `server/internal/modules/user/handler.go`
- `server/internal/modules/user/service.go`
- `server/internal/modules/user/repository.go`
- `server/internal/modules/user/models.go`
- `server/scripts/init.sql`
- `clients/web/src/stores/verification.ts`
- `clients/web/src/modules/user/views/StudentVerificationPage.vue`
- `clients/web/src/api/index.ts`
- `clients/admin/src/views/user-system/IdentityReview.vue`
- `clients/admin/src/views/user-system/StudentVerificationReview.vue`
- `clients/shared/src/types/api.gen.ts`
- `clients/shared/src/api/user-admin.ts`

This flow is cross-layer and must stay synchronized across:

- OpenAPI source spec
- Generated Go / TypeScript types
- Handler request binding
- Service validation rules
- Database persistence
- Frontend typed payloads
- Frontend form rendering
- Admin review visibility

---

## 2. HTTP Contract

### 2.1 Endpoint

```http
POST /api/v1/user/profile/verify
```

### 2.2 Request body

```json
{
  "schoolID": "10006",
  "studentID": "21370001",
  "password": "secret",
  "manualFormData": {
    "admissionTicket": "A-2026-001"
  },
  "consent": true
}
```

Field contract:

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `schoolID` | `string` | yes | Always required |
| `studentID` | `string` | conditional | Required for `ldap`; optional for `manual` |
| `password` | `string` | conditional | Required for `ldap`; ignored for `manual` |
| `manualFormData` | `object \| null` | conditional | Used only for `manual`; keys are defined by the selected school's `manualFormFields` |
| `consent` | `boolean` | yes | Always required when `consentText` is configured |

### 2.3 Response body

Success response stays aligned with `UserProfile`:

```json
{
  "success": true,
  "data": {
    "userID": 1,
    "schoolID": "10006",
    "studentIDs": ["21370001"],
    "activeStudentID": "21370001",
    "verificationStatus": "pending",
    "verificationMethod": "manual",
    "phone": null,
    "phoneVerified": false,
    "consentGivenAt": "2026-03-15T09:00:00Z",
    "verifiedAt": null,
    "createdAt": "2026-03-15T09:00:00Z",
    "updatedAt": "2026-03-15T09:00:00Z"
  }
}
```

`manualFormData` is **not** returned in the public user profile response. If admins need to review it, expose it through the admin student verification list/detail contract, not the public profile schema.

---

## 3. School Config Contract

Public school list:

```http
GET /api/v1/user/schools
```

When `verificationMethod = manual`, the public `SchoolConfig` payload must include `manualFormFields`.

Recommended field shape:

```json
[
  {
    "key": "studentID",
    "label": "学号",
    "type": "text",
    "required": true,
    "placeholder": "请输入学号"
  }
]
```

Current project rule:

- `manualFormFields` must be treated as a structured array of field descriptors, not an opaque bag.
- Frontend rendering logic must not guess the shape.
- Backend decode errors must be surfaced, not swallowed.

---

## 4. Service Validation Rules

Validation lives in `Handler -> Service -> Repository` order, but **mode-specific required rules belong only in the service layer**.

### 4.1 Handler rules

`server/internal/modules/user/handler.go`

- `ShouldBindJSON` validates request shape only
- `schoolID` remains `binding:"required"`
- `studentID` and `password` must not use unconditional `required`
- handler must not query school config to decide LDAP vs manual

### 4.2 Service rules

`server/internal/modules/user/service.go`

After loading `school_configs` and checking consent:

| School method | Required fields | Persistence rule |
| --- | --- | --- |
| `ldap` | `studentID`, `password` | LDAP login succeeds, then set verified profile data |
| `manual` | `manualFormFields.required` fields only | Create or update profile with `verificationStatus = pending` |

Additional rules:

- Empty `studentID` must **not** produce `studentIDs = [""]`
- Empty `studentID` must **not** produce `activeStudentID = ""`
- `manualFormData` should only be persisted when it contains at least one key
- Enabled `ldap` schools must have a valid `academic_db_table` and a valid `ldap_config`
- Missing or invalid `academic_db_table` must fail closed, never fall back to a hardcoded table
- Missing or invalid `ldap_config` must fail closed, never fall back to a global default

### 4.3 Sentinel errors

Service-level errors should include at least:

- `ErrStudentIDRequired`
- `ErrPasswordRequired`
- `ErrConsentRequired`
- validation error for missing required manual fields

Handler mapping:

| Error | HTTP status |
| --- | --- |
| `ErrStudentIDRequired` | `400 Bad Request` |
| `ErrPasswordRequired` | `400 Bad Request` |
| missing required manual field | `400 Bad Request` |
| `ErrSchoolNotFound` | `404 Not Found` |
| `ErrSchoolDisabled` | `400 Bad Request` |
| `ErrLDAPFailed` | `400 Bad Request` |
| `ErrAcademicTableNotConfigured` | `400 Bad Request` |
| `ErrInvalidAcademicDBTable` | `400 Bad Request` |
| `ErrSchoolLDAPConfigMissing` | `400 Bad Request` |
| `ErrLDAPConfigInvalid` | `400 Bad Request` |

---

## 5. Database Contract

Table:

```sql
user_profiles
```

If manual dynamic fields are persisted, the column contract is:

```sql
manual_form_data JSONB NULL
```

Rules:

- Update `server/scripts/init.sql`
- Update repository `SELECT / INSERT / UPDATE / Scan` logic
- Do not reference a non-existent `student_profiles` table
- Keep the column nullable so LDAP profiles do not need placeholder data
- `academic_db_table` must use `schema.table` format and must be validated before enabling an LDAP school
- `user_profiles` review audit fields are part of the contract:

```sql
verification_status VARCHAR(20) NOT NULL DEFAULT 'unverified',
rejection_reason TEXT NULL,
reviewed_at TIMESTAMPTZ NULL,
verified_at TIMESTAMPTZ NULL
```

- Rejecting a student verification must set `verification_status='rejected'`, clear `verified_at`, and update `reviewed_at`
- Approving a student verification must set `verification_status='verified'`, set `verified_at`, clear any old `rejection_reason`, and update `reviewed_at`

### 5.1 School config validation contract

`server/internal/modules/user/service_admin.go`

When updating `school_configs`:

- `verification_method='ldap'` + `enabled=true` requires non-empty `academic_db_table`
- `academic_db_table` must pass identifier validation and repository existence validation
- `ldap_config` must decode into the typed school LDAP settings struct
- enabling an LDAP school must validate the parsed LDAP config with `ldap.NewClient(...)`
- admin responses must never echo the stored system bind password; the read contract exposes only `hasSystemBindPassword`

Good / Base / Bad:

| Case | Input | Expected Result |
| --- | --- | --- |
| Good | enabled LDAP school with valid `academic_db_table` and valid `ldap_config` | `200 OK` |
| Base | manual school with no `ldap_config` | `200 OK` |
| Bad | enabled LDAP school with empty `academic_db_table` | `400 Bad Request` + `ErrAcademicTableNotConfigured` |
| Bad | enabled LDAP school with invalid `academic_db_table` | `400 Bad Request` + `ErrInvalidAcademicDBTable` |
| Bad | enabled LDAP school with missing `ldap_config` | `400 Bad Request` + `ErrSchoolLDAPConfigMissing` |
| Bad | enabled LDAP school with invalid `ldap_config` | `400 Bad Request` + `ErrLDAPConfigInvalid` |

---

## 6. Frontend Contract

### 6.1 Store + API

Current direction:

- Prefer the shared typed client (`clients/shared/src/api/identity.ts`)
- Avoid growing the legacy hand-written `api.verification` wrapper

Frontend state must include:

- `SchoolConfig.manualFormFields`
- optional `studentID`
- optional `password`
- optional `manualFormData`

### 6.2 Page behavior

`clients/web/src/modules/user/views/StudentVerificationPage.vue`

Rules:

- `ldap` schools render student ID + password inputs
- `manual` schools render fields from `manualFormFields`
- if a manual school has no dynamic fields, render a pending/manual hint only
- `canSubmit` must validate only the fields required by the active verification mode
- submit payload must omit empty `studentID` / `password`

Forbidden behavior:

- `manual` mode returning `true` from `canSubmit` without checking required manual fields
- hard-coding `studentID` + `password` as always required in the store or API wrapper

---

## 7. Good / Base / Bad Cases

### Good

- Manual school exposes descriptor-based `manualFormFields`
- User fills required manual fields
- Frontend submits `schoolID + consent + manualFormData`
- Backend stores `verificationStatus = pending`
- `studentIDs` and `activeStudentID` stay empty when no student ID is provided

### Base

- Manual school defines no dynamic fields
- Frontend shows the pending/manual hint
- User can submit with `schoolID + consent`
- Backend creates a pending profile without student identifiers

### Bad

- Manual school hides student ID/password inputs but handler still requires both fields
- Manual submit writes `studentIDs = [""]`
- Frontend assumes `manualFormFields` is an array while the backend still returns an object
- Admin review cannot see persisted manual data

---

## 8. Required Tests

### Backend

`server/internal/modules/user/handler_test.go`

- manual mode with empty `studentID/password` returns `200`
- LDAP mode with empty `studentID` returns `400`
- LDAP mode with empty `password` returns `400`
- manual mode required dynamic field missing returns `400`

`server/internal/modules/user/service_test.go`

- manual mode without student ID does not persist empty identifiers
- manual mode with `manualFormData` persists structured data
- LDAP mode still requires `studentID/password`

### Frontend

`clients/web/src/modules/user/__tests__/...`

- manual form fields render from `manualFormFields`
- required manual fields gate `canSubmit`
- submit payload omits empty `studentID/password`
- manual payload includes `manualFormData` only when present

### Codegen / Contract

Run:

```bash
cd server && make lint-spec && make generate
cd clients && pnpm type-check && pnpm test:web
```

Assertion points:

- `SubmitStudentVerificationRequest` generated types show `studentID?` / `password?`
- generated types include `manualFormData?`
- public `SchoolConfig` generated types include `manualFormFields`

---

## 9. Wrong vs Correct

### Wrong

```go
type verifyStudentHTTPRequest struct {
    SchoolID  string `json:"schoolID" binding:"required"`
    StudentID string `json:"studentID" binding:"required"`
    Password  string `json:"password" binding:"required"`
}
```

```ts
if (selectedSchool.verificationMethod === 'manual') {
  return true
}
```

### Correct

```go
type verifyStudentHTTPRequest struct {
    SchoolID       string         `json:"schoolID" binding:"required,max=10"`
    StudentID      string         `json:"studentID" binding:"max=50"`
    Password       string         `json:"password" binding:"max=200"`
    ManualFormData map[string]any `json:"manualFormData"`
    Consent        bool           `json:"consent"`
}
```

```go
if school.VerificationMethod == "ldap" {
    if strings.TrimSpace(req.StudentID) == "" {
        return nil, ErrStudentIDRequired
    }
    if req.Password == "" {
        return nil, ErrPasswordRequired
    }
}
```

```ts
if (selectedSchool.value?.verificationMethod === 'manual') {
  return manualFields.value
    .filter((field) => field.required)
    .every((field) => String(form.manualFormData[field.key] ?? '').trim() !== '')
}
```

---

## 10. Admin 审核列表筛选契约

### 实名认证列表 (`user_identities` 表)

数据库没有独立 status 列，状态由 `verified` + `reviewed_at` 派生：

| 状态 | 数据库条件 | 前端派生逻辑 |
| --- | --- | --- |
| `pending` | `verified=false AND reviewed_at IS NULL` | `!row.verified && !row.reviewedAt` |
| `rejected` | `verified=false AND reviewed_at IS NOT NULL` | `!row.verified && !!row.reviewedAt` |
| `verified` | `verified=true` | `row.verified` |

### 学生认证列表 (`user_profiles` 表)

数据库有 `verification_status VARCHAR(20)` 列，直接比较：

```sql
WHERE verification_status = $1
```

审核展示还要带出：

- `rejection_reason`
- `reviewed_at`
- `verified_at`

其中 `rejection_reason` 当前允许留空。前端可以把空字符串和 `null` 都视为“无拒绝理由”，但拒绝态必须以 `reviewedAt` 为准，不能再靠 `rejectionReason` 是否为空推断。

### 后端 normalizeAdminReviewStatus

两个列表共用同一个状态归一化函数：

| 前端发送 | 归一化结果 | Repository 收到 |
| --- | --- | --- |
| `"pending"` | `"pending"` | 过滤 pending |
| `"verified"` | `"verified"` | 过滤 verified |
| `"rejected"` | `"rejected"` | 过滤 rejected |
| `"all"` | `""` | 不过滤，返回全部 |
| `""` 或省略 | `"pending"` | 默认过滤 pending |
| 其他值 | 拒绝 | `400 Bad Request` |

### 关键规则

- 前端"全部"按钮必须显式发送 `status=all`，**不能**省略参数或发送空字符串（后端默认为 `pending`）
- 前端必须使用 typed client (`api.userAdmin.*`)，不使用 raw fetchJSON
- `"unverified"` 不是合法的 admin 列表筛选值（后端返回 400）
- 实名认证审核页的状态标签应从 `verified` + `reviewedAt` 派生，不要只显示 `verified/unverified` 二态
- 拒绝操作允许 `rejectionReason = null`、空字符串或仅空白字符，后端统一按“无拒绝理由”处理，但仍必须写入 `reviewed_at`

### Repository 实现规则

`ListIdentityReviewItems` 只处理三个有效状态分支：

```go
switch status {
case StatusPending:
    qb.WriteString(` AND verified = false AND reviewed_at IS NULL`)
case StatusRejected:
    qb.WriteString(` AND verified = false AND reviewed_at IS NOT NULL`)
case StatusVerified:
    qb.WriteString(` AND verified = true`)
// 空字符串（status=""）不添加过滤条件，返回全部
}
```

**不应出现的分支**：

- `case "unverified"` — 该状态不存在于 `user_identities` 表的派生状态模型中
- `case ""` with explicit filtering — 空字符串应该跳过过滤，不应映射到任何 WHERE 条件

### Wrong

```ts
// "全部"按钮发送空字符串 → 后端默认 pending → 用户只看到待审核
<el-radio-button value="">全部</el-radio-button>
// ...
status: filterStatus.value || undefined  // "" → undefined → 省略参数
```

```go
// Repository 包含不可达的 unverified 分支
switch status {
case StatusPending:
    qb.WriteString(` AND verified = false AND reviewed_at IS NULL`)
case StatusUnverified:  // 永远不会被调用
    qb.WriteString(` AND verified = false`)
case StatusVerified:
    qb.WriteString(` AND verified = true`)
}
```

### Correct

```ts
// 显式发送 "all"
<el-radio-button value="all">全部</el-radio-button>
<el-radio-button value="pending">待审核</el-radio-button>
<el-radio-button value="verified">已认证</el-radio-button>
<el-radio-button value="rejected">已拒绝</el-radio-button>
// ...
status: filterStatus.value  // "all" → normalizeAdminReviewStatus("all") → "" → 不过滤
```

```go
// Repository 只处理三个有效状态
switch status {
case StatusPending:
    qb.WriteString(` AND verified = false AND reviewed_at IS NULL`)
case StatusRejected:
    qb.WriteString(` AND verified = false AND reviewed_at IS NOT NULL`)
case StatusVerified:
    qb.WriteString(` AND verified = true`)
// 空字符串不添加过滤条件
}
```

### Required Tests

Backend:

- `ListIdentityReviewItems` with `status=""` returns all records
- `ListIdentityReviewItems` with `status="pending"` returns only pending records
- `ListIdentityReviewItems` with `status="verified"` returns only verified records
- `ListIdentityReviewItems` with `status="rejected"` returns only rejected records
- Handler rejects `status="unverified"` with `400 Bad Request`

Frontend:

- "全部" button sends `status=all`
- Status badge renders correct label from `verified` + `reviewedAt`
- Rejecting with empty `rejectionReason` still shows the item as rejected
- Uses typed `api.userAdmin.listIdentities` instead of raw fetch
