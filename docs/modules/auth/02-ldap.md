# LDAP Validation

The repository provides an LDAP client via `server/internal/modules/ldap/client.go`. This is used in the student verification flow for LDAP-based validation.

## Client Methods

| Method | Purpose |
| --- | --- |
| `NewClient` | Validate configuration and create client |
| `Login` | Perform LDAP bind with `uid + password` |
| `QueryUserByUID` | Query user profile using system account |

## Integration Point

The student verification service selects the verification method based on school configuration. When `verificationMethod` is `ldap`, `user.Service.VerifyStudent` performs:

1. Validate `schoolID`, `studentID`, `password`, and consent status
2. Call `ldap.Client.Login` to verify credentials
3. Call `ldap.Client.QueryUserByUID` to fetch user profile
4. Update phone number, student ID list, active student ID, and verification status

## Configuration

The LDAP client is initialized in `server/cmd/stuhelper/main.go` with these environment variables:

| Variable | Description |
| --- | --- |
| `LDAP_URL` | LDAP server URL |
| `LDAP_BASE_DN` | Base DN for searches |
| `LDAP_SYSTEM_BIND_DN` | System account bind DN |
| `LDAP_SYSTEM_BIND_PASSWORD` | System account password |
| `LDAP_USE_TLS` | Enable TLS connection |
| `LDAP_INSECURE_SKIP_VERIFY` | Skip TLS certificate verification (development only) |

## Data Flow

School configuration API returns `verificationMethod` and `ldapConfig`. The main app student verification page displays either an LDAP login form or a manual submission form based on the school configuration.

```text
School Config (verificationMethod: "ldap")
    ↓
Frontend: Show LDAP login form
    ↓
POST /api/v1/user/profile/verify
    ↓
Backend: user.Service.VerifyStudent
    ↓
    ├─→ ldap.Client.Login (verify credentials)
    └─→ ldap.Client.QueryUserByUID (fetch profile)
    ↓
Update user_profiles (student_ids, active_student_id, student_verified)
```
