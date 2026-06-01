/**
 * Common text - English
 */
export default {
  actions: {
    confirm: 'Confirm',
    cancel: 'Cancel',
    save: 'Save',
    saving: 'Saving...',
    delete: 'Delete',
    deleting: 'Deleting...',
    edit: 'Edit',
    search: 'Search',
    loading: 'Loading...',
    loadMore: 'Load More',
    retry: 'Retry',
    submit: 'Submit',
    send: 'Send',
    sending: 'Sending...',
    reply: 'Reply',
    back: 'Back',
    more: 'More',
    close: 'Close',
    clear: 'Clear',
    operationFailed: 'Operation failed, please retry',
    refresh: 'Refresh',
    learnMore: 'Learn More'
  },
  status: {
    available: 'Available',
    coming: 'Coming Soon',
    success: 'Success',
    failed: 'Failed',
    pending: 'Pending'
  },
  pagination: {
    total: 'Total {total}',
    page: 'Page {page}',
    pageSize: '{size} per page',
    perPage: 'Per page',
    items: 'items',
    pageLabel: 'Page {page}',
    prevPage: 'Previous page',
    nextPage: 'Next page',
    nav: 'Pagination',
    pageSizeLabel: 'Items per page'
  },
  empty: {
    data: 'No data',
    result: 'No results found'
  },
  loadFailed: 'Load failed',
  noMore: 'No more data',
  time: {
    justNow: 'Just now',
    minutesAgo: '{n} minutes ago',
    hoursAgo: '{n} hours ago',
    daysAgo: '{n} days ago'
  },
  login: {
    title: 'StuHelper Sign-in',
    subtitle: 'Account sign-in, verification, and Open Platform access',
    identityLogin: 'Continue with unified sign-in',
    signup: 'Create account',
    redirecting: 'Redirecting...',
    hint: 'Sign in with unified authentication',
    loginFailed: 'Login failed',
    signupFailed: 'Signup failed',
    networkError: 'Network connection failed',
    loginUrlFailed: 'Failed to get login URL',
    signupUrlFailed: 'Failed to get signup URL',
    invalidState: 'Invalid authentication state',
    callbackFailed: 'Authentication callback failed',
    fetchUserFailed: 'Failed to fetch user info',
    phoneLogin: 'Phone Login',
    phonePlaceholder: 'Enter phone number',
    codePlaceholder: 'Enter verification code',
    sendCode: 'Send Code',
    resendCode: 'Resend in {n}s',
    verifyAndLogin: 'Verify & Login',
    verifying: 'Verifying...',
    otpSendFailed: 'Failed to send verification code',
    otpVerifyFailed: 'Verification failed',
    otpSent: 'Verification code sent',
    invalidPhone: 'Please enter a valid phone number',
    phoneHint: 'Quick login with phone number',
    identityHint: 'Use StuHelper for account sign-in, student verification, and third-party app authorization'
  },
  openPlatformConsent: {
    loading: 'Loading StuHelper Connect authorization request...',
    loadFailed: 'Failed to load authorization request',
    invalidToken: 'The authorization request is expired or missing a token. Return to the account center or start again from the requesting app.',
    submitFailedTitle: 'Authorization failed',
    submitFailed: 'Authorization failed. Please retry',
    connectEyebrow: 'StuHelper Connect',
    openIdentityHome: 'Back to Account Center',
    title: '{app} wants to access your StuHelper account through StuHelper Connect',
    identityLine: 'Current StuHelper account: {user}',
    appInfo: 'Connected app',
    appName: 'App',
    description: 'Description',
    redirect: 'Redirect host',
    links: 'Links',
    homepage: 'Homepage',
    privacy: 'Privacy Policy',
    permissions: 'Information disclosed through Connect',
    reason: 'Purpose: {reason}',
    expiresAt: 'Valid until {time}',
    accept: 'Allow',
    accepting: 'Allowing...',
    deny: 'Deny',
    denying: 'Denying...',
    sensitivity: {
      low: 'Low',
      medium: 'Medium',
      high: 'High',
      very_high: 'Very high'
    }
  },
  openPlatformProfileCompletion: {
    loading: 'Checking StuHelper account profile...',
    loadFailed: 'Failed to load profile completion request',
    invalidToken: 'The profile completion request is expired or missing a token. Return to the account center or start again from the requesting app.',
    submitFailedTitle: 'Failed to continue authorization',
    submitFailed: 'Failed to continue authorization. Please retry',
    connectEyebrow: 'StuHelper Connect',
    openIdentityHome: 'Back to Account Center',
    title: 'Complete your StuHelper account profile before continuing to {app}',
    identityLine: 'Current StuHelper account: {user}',
    requiredFields: 'Required fields',
    requestedPermissions: 'Information requested through Connect',
    reason: 'Purpose: {reason}',
    openAction: 'Complete',
    continue: 'I have completed this',
    continuing: 'Continuing...',
    refresh: 'Check again',
    noMissingFields: 'Your StuHelper account profile now satisfies this authorization request',
    expiresAt: 'Valid until {time}'
  },
  meta: {
    description: 'StuHelper Course Review Community - Authentic course reviews and teaching feedback to help students make informed choices',
    ogTitle: 'Course Review - StuHelper'
  },
  search: {
    placeholder: 'Search...',
    label: 'Search',
    clear: 'Clear search'
  },
  infoPages: {
    actions: {
      home: 'Back to Home',
      learningCenter: 'Enter Learning Center'
    },
    about: {
      eyebrow: 'About',
      title: 'About StuHelper',
      intro: 'StuHelper is a campus-focused platform for course reviews and academic information sharing, designed to make course selection, teacher discovery, and study planning more direct.',
      sections: {
        currentCapabilities: {
          heading: 'Current Capabilities',
          body: 'Course reviews, teacher details, the notification center, and a basic moderation backend are available today, with more modules rolling out in priority order.'
        },
        contentPrinciples: {
          heading: 'Content Principles',
          body: 'We encourage rational expression based on real experiences, with an emphasis on information that is verifiable, traceable, and helpful to others.'
        },
        roadmap: {
          heading: 'Product Direction',
          body: 'The project uses an OpenAPI 3 spec-first workflow, shared multi-client APIs, and StuHelper unified sign-in, and we will continue expanding the teaching-related product surface.'
        }
      }
    },
    privacy: {
      eyebrow: 'Privacy',
      title: 'Privacy Policy',
      intro: 'StuHelper only collects data necessary to keep the service running and avoids directly processing personally identifiable information whenever possible.',
      sections: {
        collection: {
          heading: 'What We Collect',
          body: 'Login state, essential access logs, course interaction data, and cookies required for security checks.'
        },
        usage: {
          heading: 'How We Use It',
          body: 'For authentication, risk control, content moderation, feature analytics, and continuous improvement of the course review experience.'
        },
        control: {
          heading: 'Your Control',
          body: 'You can sign out at any time and delete or edit the content you posted. If you need additional data assistance, please contact the maintainers.'
        }
      }
    },
    terms: {
      eyebrow: 'Terms',
      title: 'Terms of Service',
      intro: 'By using StuHelper, you agree to follow the basic rules of the campus community and not publish illegal, abusive, deceptive, or malicious content.',
      sections: {
        responsibility: {
          heading: 'Content Responsibility',
          body: 'You are responsible for the reviews, replies, and reports you publish. The platform may hide, delete, or restrict content that violates the rules.'
        },
        access: {
          heading: 'Accounts & Access',
          body: 'Login depends on the unified identity system. Abnormal access, bulk scraping, or attempts to bypass security controls may be restricted.'
        },
        changes: {
          heading: 'Service Changes',
          body: 'The platform will continue iterating on features and policies, and important changes will be announced through in-app notices or documentation updates.'
        }
      }
    }
  },
  locale: {
    switchToEnglish: 'Switch to English',
    switchToChinese: 'Switch to Chinese',
    englishShort: 'EN',
    chineseShort: '中'
  },
  bootstrap: {
    failed: 'App startup failed. Please refresh the page and try again.',
    contactSupport: 'If the issue persists, contact the maintainer with the console error details.'
  },
  loading: {
    label: 'Loading'
  }
}
