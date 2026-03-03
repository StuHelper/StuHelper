/**
 * Error messages - English
 * Error codes correspond to backend 8-digit codes or client codes, see docs/reference/error-codes.md
 */
export default {
  // Client-only codes (backend never returns these)
  NETWORK_ERROR: 'Network connection failed, please check your network',
  OFFLINE: 'You are offline. Please check your connection and try again',
  TIMEOUT: 'Request timed out, please try again',

  // A000xxxx - General client errors
  A0000400: 'Invalid request parameters',
  A0000404: 'The requested resource was not found',
  A0000409: 'Resource conflict',
  A0000422: 'Validation failed',
  A0000429: 'Too many requests, please try again later',

  // A001xxxx - Auth and authorization
  A0010001: 'Session expired, please log in again',
  A0010002: 'Invalid session, please log in again',
  A0010003: 'Please log in first',
  A0010100: 'Please log in first',
  A0010200: 'You do not have permission to perform this action',
  A0010201: 'Access denied',
  A0010202: 'CSRF validation failed, please refresh the page',
  A0010203: 'CSRF token missing, please refresh the page',

  // A010xxxx - Course module
  A0100001: 'Course not found',
  A0100003: 'Teacher not found',

  // A011xxxx - Review module
  A0110001: 'Review not found',
  A0110002: 'You have already reviewed this course',
  A0110005: 'Reply not found',
  A0110006: 'Draft not found',
  A0110007: 'Report not found',
  A0110008: 'Sensitive word not found',
  A0110010: 'You do not own this review',
  A0110011: 'You do not own this reply',
  A0110100: 'You have already voted on this review',
  A0110102: 'You cannot vote on your own review',
  A0110103: 'You have already reported this content',
  A0110301: 'Content contains sensitive words, please revise',
  A0110302: 'Invalid status transition',

  // B000xxxx - System errors
  B0000001: 'Server error, please try again later',
  B0000004: 'Service temporarily unavailable',
  B0000006: 'Request timed out, please try again later',

  // C000xxxx - Third-party service errors
  C0000001: 'Upstream service error',

  // Error pages
  notFound: {
    title: 'Page Not Found',
    description: 'The page you visited may have been removed or the address is incorrect',
    backHome: 'Back to Home',
    goBack: 'Go Back'
  },
  loadError: {
    title: 'Load Failed',
    description: 'An error occurred while loading the page. Please refresh and try again',
    reload: 'Refresh Page'
  },

  // Error boundary
  boundary: {
    title: 'Something went wrong',
    description: 'An unexpected error occurred while rendering the page. Please try refreshing.',
    reload: 'Refresh Page'
  },

  // Auth callback
  authCallback: {
    loading: 'Logging in...',
    error: 'Login Failed',
    backToLogin: 'Back to Login',
    missingCode: 'Missing authorization code',
    missingState: 'Missing state parameter',
    loginFailed: 'Login failed, please retry',
    orgMismatch: 'Your SSO account does not have access to this system',
    orgMismatchHint: 'Please log out and sign in with the correct account',
    ssoLogout: 'Log out and sign in again'
  }
}
