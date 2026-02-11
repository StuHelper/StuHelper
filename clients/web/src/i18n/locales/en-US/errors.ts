/**
 * Error messages - English
 * Corresponds to backend error codes, see docs/api/error-codes.md
 */
export default {
  // Network errors
  NETWORK_ERROR: 'Network connection failed, please check your network',
  TIMEOUT: 'Request timed out, please try again',

  // Auth errors
  UNAUTHORIZED: 'Please log in first',
  TOKEN_EXPIRED: 'Session expired, please log in again',
  INVALID_TOKEN: 'Invalid session, please log in again',

  // Permission errors
  FORBIDDEN: 'You do not have permission to perform this action',

  // Client errors
  BAD_REQUEST: 'Invalid request parameters',
  NOT_FOUND: 'The requested resource was not found',
  VALIDATION_ERROR: 'Validation failed',
  CONFLICT: 'Resource conflict',
  RATE_LIMIT_EXCEEDED: 'Too many requests, please try again later',

  // Server errors
  INTERNAL_ERROR: 'Server error, please try again later',
  SERVICE_UNAVAILABLE: 'Service temporarily unavailable',

  // Business errors
  BUSINESS_ERROR: 'Operation failed',

  // Unknown errors
  UNKNOWN: 'An unknown error occurred',

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

  // Auth callback
  authCallback: {
    loading: 'Logging in...',
    error: 'Login Failed',
    backToLogin: 'Back to Login',
    missingCode: 'Missing authorization code',
    missingState: 'Missing state parameter',
    loginFailed: 'Login failed, please retry'
  }
}
