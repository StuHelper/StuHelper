# Casdoor SSO

The main login flow goes through Casdoor. The StuHelper backend initiates the OAuth flow, exchanges tokens, writes cookies, and returns user information to the frontend.

## API Endpoints

| Endpoint | Purpose |
| --- | --- |
| `/api/v1/auth/login` | Generate login URL and `state` |
| `/api/v1/auth/signup` | Generate signup URL and `state` |
| `/api/v1/auth/callback` | Exchange authorization code for tokens, write cookies, return `user` and `expiresIn` |
| `/api/v1/auth/me` | Return current user info and capabilities |
| `/api/v1/auth/refresh` | Refresh access token |

## Login Flow

```text
1. Frontend requests /api/v1/auth/login or /api/v1/auth/signup
2. Backend generates Casdoor authorization URL via server/internal/pkg/sso
3. Browser redirects to https://sso.stuhelper.com
4. Casdoor redirects back to frontend /auth/callback
5. Frontend calls /api/v1/auth/callback
6. Backend exchanges authorization code for access token and refresh token
7. Backend writes HttpOnly cookies and returns current user info
```

## UserInfo Response

Both `/api/v1/auth/callback` and `/api/v1/auth/me` return these fields:

| Field | Type | Description |
| --- | --- | --- |
| `id` | string | Casdoor user ID |
| `name` | string | Username |
| `displayName` | string | Display name |
| `email` | string | Email address |
| `avatar` | string | Avatar URL |
| `isPlatformAdmin` | boolean | Casdoor platform admin flag |
| `capabilities` | string[] | Application capability set |
| `canAccessAdmin` | boolean | Whether user has at least one admin capability |

## Code Paths

| Location | Purpose |
| --- | --- |
| `server/internal/modules/auth/handler_login.go` | Login URL, callback, cookie writing |
| `server/internal/modules/auth/handler_userinfo.go` | `/auth/me` and UserInfo assembly |
| `server/internal/modules/auth/user_sync.go` | Post-login sync to `users` table |
| `server/internal/pkg/sso/client.go` | OAuth URL generation, token exchange, JWT parsing |

## Platform Admin Semantics

`isPlatformAdmin` enters the application with Casdoor user profile data. Admin menus and admin endpoints are driven by `capabilities`, and both frontend and backend use the same capability constants.
