# Sessions and Security

The security layer covers Cookie sessions, token blacklist, PII encryption, and audit logging.

## Sensitive Data Protection

| Data | Storage Method |
| --- | --- |
| Government ID number | AES-256-GCM ciphertext in `user_identities.doc_number_enc` |
| ID-derived identifier | HMAC-SHA256 in `user_identities.person_uid` |
| Student ID | `user_profiles.student_ids`, `active_student_id` |
| Phone number | `user_profiles.phone` |

## PII Encryption Format

Government ID numbers use a versioned envelope format:

```text
version(1 byte) | keyID(1 byte) | nonce(12 bytes) | ciphertext+tag
```

Runtime key loading via environment variables:

```bash
DOC_AES_ACTIVE_KEY_ID=1
DOC_AES_KEYS=1:<64-char-hex-key>
HMAC_SECRET=<at-least-32-characters>
```

## Session Architecture

| Component | Implementation |
| --- | --- |
| Access Token | HttpOnly Cookie, default 15 minutes |
| Refresh Token | HttpOnly Cookie, default 7 days |
| CSRF Protection | `csrf_token` Cookie + request header |
| Session Revocation | Redis blacklist and user token tracking |

## Key Endpoints

| Endpoint | Purpose |
| --- | --- |
| `POST /api/v1/auth/logout` | Clear current device cookies and add to blacklist |
| `POST /api/v1/auth/logout-all` | Revoke all tracked tokens for the user |
| `POST /api/v1/auth/refresh` | Rotate refresh token and refresh access token |

## Token Refresh Flow

```text
1. Access token expires (15 min)
2. Frontend receives 401
3. Frontend calls POST /api/v1/auth/refresh
4. Backend validates refresh token
5. Backend issues new access token + rotates refresh token
6. Backend writes new cookies
7. Frontend retries original request
```

If refresh fails (token expired or revoked), the frontend clears local session and redirects to login.

## Audit Events

`server/internal/pkg/audit` records authentication and critical business events:

| Event | Description |
| --- | --- |
| `user.login` | Successful login |
| `user.login_failed` | Failed login attempt |
| `user.logout` | Single device logout |
| `user.logout_all` | All device logout |
| `token.refresh` | Token refresh |

## Security Headers

The `security_headers.go` middleware sets:

**All environments:**

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `X-Permitted-Cross-Domain-Policies: none`
- `Content-Security-Policy` (strict policy for API paths)

**Production only:**

- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `Cross-Origin-Resource-Policy: same-origin`
- `Cross-Origin-Opener-Policy: same-origin`
