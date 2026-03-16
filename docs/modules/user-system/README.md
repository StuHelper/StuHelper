# User System

The user system covers identity verification, student verification, school configuration, system configuration, and academic information.

## Code Scope

| Code Location | Purpose |
| --- | --- |
| `server/internal/modules/user` | Identity verification, student verification, school config, system config |
| `server/internal/modules/ldap` | LDAP login validation and user info query |
| `server/internal/pkg/crypto/pii` | Government ID encryption |

## Subdomains

### Identity Verification

Users submit government ID information for real-name verification. The process:

1. User submits ID type and ID number via `POST /api/v1/user/identity`
2. ID number is encrypted with AES-256-GCM and stored as `doc_number_enc`
3. A stable identifier (`person_uid`) is derived via HMAC-SHA256 for matching
4. Admin reviews and approves/rejects via `PUT /api/v1/admin/identities/{userID}`

### Student Verification

Students verify their enrollment status. Two methods are supported:

- **LDAP verification**: Automatic validation against school LDAP directory
- **Manual review**: Admin reviews submitted documents

The verification method is configured per school in `school_configs`.

### School Configuration

Per-school settings that control verification behavior:

| Field | Description |
| --- | --- |
| `verificationMethod` | `ldap` or `manual` |
| `ldapConfig` | LDAP connection settings (when method is `ldap`) |
| `manualFormFields` | Required form fields for manual verification |
| `consentText` | Agreement text shown to users |

### System Configuration

Global configuration items maintained through the admin console.

## Business Rules

- Identity verification status participates in the student verification flow
- Government ID numbers are stored as ciphertext; `person_uid` is used for stable matching
- School configuration updates use merge-write semantics
- Admin review endpoints require appropriate capabilities

## API Endpoints

### User-Facing

| Endpoint | Method | Description |
| --- | --- | --- |
| `/api/v1/user/identity` | GET | Check identity verification status |
| `/api/v1/user/identity` | POST | Submit identity verification |
| `/api/v1/user/profile` | GET | Get student verification profile |
| `/api/v1/user/profile/verify` | POST | Submit student verification |
| `/api/v1/user/profile/bind-phone` | POST | Bind phone number |
| `/api/v1/user/profile/academic-info` | GET | Get academic information |
| `/api/v1/user/schools` | GET | List schools |

### Admin

| Endpoint | Method | Description |
| --- | --- | --- |
| `/api/v1/admin/identities` | GET | List identity verification requests |
| `/api/v1/admin/identities/{userID}` | PUT | Review identity verification |
| `/api/v1/admin/student-verifications` | GET | List student verification requests |
| `/api/v1/admin/student-verifications/{userID}` | PUT | Review student verification |
| `/api/v1/admin/school-configs` | GET | List school configurations |
| `/api/v1/admin/school-configs/{schoolID}` | PUT | Update school configuration |
| `/api/v1/admin/system-configs` | GET | List system configurations |
| `/api/v1/admin/system-configs/{key}` | PUT | Update system configuration |

## Database Tables

| Table | Purpose |
| --- | --- |
| `users` | Local user records synced from Casdoor |
| `user_identities` | Identity verification data (encrypted ID numbers) |
| `user_profiles` | Student verification data, school, phone |
| `school_configs` | Per-school verification settings |
| `system_configs` | Global system settings |

## Related Documentation

- [API Overview](../../reference/api-overview.md)
- [Database Design](../../reference/database.md)
- [Identity and Authorization](../../architecture/ecosystem-identity-and-authorization.md)
- [LDAP Verification](../auth/02-ldap.md)
