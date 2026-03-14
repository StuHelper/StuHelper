# Casdoor SSO Login Page Customization

This directory contains custom CSS and JavaScript files for the Casdoor SSO login page, designed to match the StuHelper Glassmorphism visual style.

## Files

| File | Purpose |
|------|---------|
| `custom-login.css` | Custom CSS with Glassmorphism theme (cyan/indigo gradient, glass cards, animated orbs) |
| `custom-login.js` | Custom JS for StuHelper branding injection and entrance animations |

## Design

- **Background**: Dark gradient (#0f172a → #1e293b) with animated cyan/indigo gradient orbs
- **Login Card**: Glassmorphism with `backdrop-blur(24px)`, translucent white border, soft shadow
- **Primary Colors**: Cyan (#06b6d4) and Indigo (#4f46e5) matching stuhelper.com
- **Button**: Gradient from cyan to indigo with hover shimmer effect
- **Inputs**: Semi-transparent dark glass with cyan focus ring
- **Typography**: System UI font, gradient text for the title

## Configuration Steps

### 1. Access Casdoor Admin Console

Navigate to your Casdoor admin panel (e.g., `https://sso.stuhelper.com/.well-known/`).

### 2. Configure Application Custom CSS

1. Go to **Applications** > Select your StuHelper application
2. Find the **Custom CSS** field
3. Copy the entire contents of `custom-login.css` and paste it into the field
4. Click **Save**

### 3. Configure Application Custom JS

1. In the same application settings page
2. Find the **Custom JS** field (or **Header HTML** / **Custom Script** depending on Casdoor version)
3. Copy the contents of `custom-login.js` and wrap it in a `<script>` tag if needed:
   ```html
   <script>
   // Paste custom-login.js contents here
   </script>
   ```
4. Click **Save**

### 4. Configure Theme (Optional)

In Casdoor's global settings or application-level theme:

1. Go to **Applications** > StuHelper > **Theme**
2. Set primary color to `#06b6d4` (cyan)
3. Set the background to dark mode if available
4. This ensures consistency for any UI elements not covered by the custom CSS

### 5. Configure Login Page Logo

1. In application settings, upload the StuHelper logo
2. Recommended logo: White/light version for dark background
3. Recommended size: 48px height, SVG or PNG with transparent background

### 6. Verify

1. Open an incognito window
2. Navigate to `https://your-app.stuhelper.com/login`
3. You should be redirected to the Casdoor login page with StuHelper branding
4. Verify:
   - Dark gradient background with animated gradient orbs
   - Glass-morphism login card
   - Cyan/indigo gradient button
   - StuHelper title with gradient text
   - Smooth entrance animation
   - Mobile responsive layout

## Customization Notes

### Modifying Colors

The main color variables are used throughout the CSS:
- **Cyan**: `#06b6d4` (primary)
- **Indigo**: `#4f46e5` (secondary)
- **Dark BG**: `#0f172a` / `#1e293b`

Search and replace these values to change the color scheme.

### Casdoor Version Compatibility

This CSS targets Casdoor's default UI built with Ant Design (antd). The selectors cover:
- Ant Design components (`.ant-*`)
- Casdoor's custom components (`.login-form`, `.panel-module`)
- Generic fallback selectors

If Casdoor updates its UI structure, some selectors may need adjustment.

### Loading Transition

To achieve seamless visual transition between your app and Casdoor:

1. Add a full-screen loading overlay in your app before redirecting to Casdoor
2. The Casdoor page entrance animation (in `custom-login.js`) provides a smooth fade-in
3. On callback, show a loading state until the auth flow completes

This is already implemented in the web app's `LoginPage.vue` and `AuthCallbackPage.vue`.
