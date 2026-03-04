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
  A0000401: 'Invalid parameter format',
  A0000402: 'Missing required parameter',
  A0000403: 'Parameter out of range',
  A0000404: 'The requested resource was not found',
  A0000405: 'Method not allowed',
  A0000409: 'Resource conflict',
  A0000413: 'Payload too large',
  A0000415: 'Unsupported media type',
  A0000422: 'Validation failed',
  A0000429: 'Too many requests, please try again later',

  // A001xxxx - Auth and authorization
  A0010001: 'Session expired, please log in again',
  A0010002: 'Invalid session, please log in again',
  A0010003: 'Please log in first',
  A0010004: 'Token has been revoked',
  A0010005: 'Refresh token expired',
  A0010006: 'Invalid refresh token',
  A0010100: 'Please log in first',
  A0010101: 'Login failed',
  A0010102: 'Account disabled',
  A0010103: 'Account locked',
  A0010200: 'You do not have permission to perform this action',
  A0010201: 'Access denied',
  A0010202: 'CSRF validation failed, please refresh the page',
  A0010203: 'CSRF token missing, please refresh the page',

  // A002xxxx - User related
  A0020001: 'User not found',
  A0020002: 'User already exists',
  A0020003: 'Username already taken',
  A0020004: 'Email already taken',
  A0020005: 'Password too weak',
  A0020006: 'Password mismatch',

  // A010xxxx - Course module
  A0100001: 'Course not found',
  A0100002: 'Department not found',
  A0100003: 'Teacher not found',
  A0100004: 'Term not found',

  // A011xxxx - Review module
  A0110001: 'Review not found',
  A0110002: 'You have already reviewed this course',
  A0110003: 'Review content too short',
  A0110004: 'Review content too long',
  A0110005: 'Reply not found',
  A0110006: 'Draft not found',
  A0110007: 'Report not found',
  A0110008: 'Sensitive word not found',
  A0110009: 'Content is empty',
  A0110010: 'You do not own this review',
  A0110011: 'You do not own this reply',
  A0110100: 'You have already voted on this review',
  A0110101: 'Invalid vote type',
  A0110102: 'You cannot vote on your own review',
  A0110103: 'You have already reported this content',
  A0110104: 'Invalid vote action',
  A0110200: 'Invalid rating',
  A0110201: 'Missing required rating dimension',
  A0110300: 'Content contains dangerous elements',
  A0110301: 'Content contains sensitive words, please revise',
  A0110302: 'Invalid status transition',

  // B000xxxx - System errors
  B0000001: 'Server error, please try again later',
  B0000002: 'Database error',
  B0000003: 'Cache error',
  B0000004: 'Service temporarily unavailable',
  B0000005: 'Service overloaded',
  B0000006: 'Request timed out, please try again later',
  B0000007: 'Configuration error',

  // C000xxxx - Third-party service errors
  C0000001: 'Upstream service error',
  C0000002: 'Upstream service timeout',
  C0000003: 'Upstream service unavailable',

  // C001xxxx - SSO service errors
  C0010001: 'SSO service error',
  C0010002: 'SSO service timeout',
  C0010003: 'SSO service unavailable',

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
