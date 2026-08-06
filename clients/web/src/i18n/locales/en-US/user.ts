/**
 * User center text - English
 */
export default {
    center: "User Center",
    myReviews: "My Reviews",
    myVotes: "My Votes",
    myFavorites: "My Favorites",
    myAuthorizedApps: "Authorized Apps",
    identityHome: {
        eyebrow: "Account Center",
        title: "Account Center",
        subtitle:
            "Manage your StuHelper profile, verification status, third-party authorizations, and developer apps.",
        sectionsLabel: "Account center sections",
        open: "Open",
        accountProfile: {
            title: "Profile",
            description:
                "Review account display details, contact fields, verification facts, and fields available for authorized disclosure.",
        },
        accountSecurity: "Account Security",
        authorizedApps: {
            title: "Authorized Apps",
            description:
                "Review third-party app scopes, recent disclosures, and revoke authorizations.",
        },
        identityVerification: {
            title: "Identity Verification",
            description:
                "Maintain real-name verification for sensitive features that require it.",
        },
        studentVerification: {
            title: "Student Verification",
            description:
                "Connect your school identity and academic status for campus and course-review capabilities.",
        },
        phoneBinding: {
            title: "Phone Binding",
            description:
                "Maintain phone binding for account security and authorized profile disclosure.",
        },
        qqBinding: {
            title: "QQ Binding",
            description:
                "Connect bot-side account recognition and group admission verification.",
        },
        academicInfo: {
            title: "Academic Info",
            description:
                "Review verified school, department, major, and academic-status details.",
        },
        connect: {
            title: "StuHelper Connect",
            description:
                "Review the public OIDC / OAuth issuer, discovery, UserInfo, JWKS, and token lifecycle endpoints.",
        },
        developerApps: {
            title: "Developer Apps",
            description:
                "Register apps, request scopes, maintain redirect URIs, and review app audit summaries.",
        },
    },
    accountProfile: {
        eyebrow: "Account Center",
        title: "Profile",
        subtitle:
            "Review the account details, contact fields, verification status, and third-party disclosure fields for your current StuHelper account.",
        backToIdentity: "Back to Account Center",
        missing: {
            value: "Not available",
            email: "No email",
            phone: "No phone binding",
            qq: "No QQ binding",
        },
        fields: {
            displayName: "Display Name",
            username: "Username",
            accountId: "Account ID",
            email: "Email",
            phone: "Phone",
            qq: "QQ",
        },
        status: {
            available: "Available",
            missing: "Missing",
            verified: "Verified",
            unverified: "Unverified",
            bound: "Bound",
            unbound: "Unbound",
        },
        contact: {
            title: "Contact Fields",
            description:
                "These fields are disclosed only after app approval and your authorization.",
        },
        verification: {
            actionsLabel: "Verification actions",
            identity: {
                title: "Identity Verification",
                description:
                    "Maintain real-name verification for sensitive features that require it.",
            },
            student: {
                title: "Student Verification",
                description:
                    "Connect your school identity and academic status for campus and course-review capabilities.",
            },
            phone: {
                title: "Phone Binding",
                description:
                    "Maintain phone binding for account security and authorized profile disclosure.",
            },
        },
        disclosure: {
            title: "Authorized Disclosure Fields",
            description:
                "Third-party apps can read each field only after scope approval and your authorization.",
            profile: {
                title: "Basic Profile",
                description: "Display name, username, and avatar.",
            },
            email: {
                title: "Email",
                description: "Account email address.",
            },
            phone: {
                title: "Phone",
                description: "Verified phone number.",
            },
            identity: {
                title: "Identity Status",
                description: "Real-name verification status and identity type.",
            },
            student: {
                title: "Student Status",
                description:
                    "Student verification status and school information.",
            },
        },
    },
    accountSecurity: {
        eyebrow: "Account Center",
        title: "Account Security",
        subtitle:
            "Manage the current browser session, authorized apps, and in-app identity security entries.",
        backToIdentity: "Back to Account Center",
        actionsLabel: "Account security actions",
        emailMissing: "No email",
        accountSummary:
            "This page only shows the current sign-in state and in-app security entries. It does not display internal account IDs, usernames, or upstream account security details.",
        fields: {
            accountId: "Account ID",
            username: "Username",
        },
        session: {
            title: "Current Browser Session",
            description:
                "This device is signed in with StuHelper unified sign-in.",
            active: "Session active",
            logout: "Sign out current session",
            loggingOut: "Signing out...",
        },
        provider: {
            title: "Password and Upstream Account",
            description:
                "Open the upstream account settings for password and security profile changes.",
            open: "Open account settings",
        },
        phone: {
            title: "Phone Binding",
            description:
                "Maintain phone binding for account security and authorized profile disclosure.",
        },
        authorizedApps: {
            title: "Authorized Apps",
            description:
                "Review third-party app scopes, recent disclosures, and revoke authorizations.",
        },
        identity: {
            title: "Identity Verification",
            description:
                "Maintain real-name verification for sensitive features that require it.",
        },
        student: {
            title: "Student Verification",
            description:
                "Connect your school identity and academic status for campus and course-review capabilities.",
        },
        qq: {
            title: "QQ Binding",
            description:
                "Connect bot-side account recognition and group admission verification.",
        },
        academic: {
            title: "Academic Info",
            description:
                "Review verified school, department, major, and academic-status details.",
        },
    },
    reviews: {
        empty: "No reviews yet",
        emptyDesc: "Post your first course review",
        browseCourses: "Browse Courses",
    },
    votes: {
        empty: "No votes yet",
        emptyDesc: "Like reviews you find helpful",
    },
    favorites: {
        empty: "No favorites yet",
        emptyDesc: "Favorite courses for quick access",
        browseCourses: "Browse Courses",
    },
    authorizedApps: {
        pageTitle: "Authorized Apps",
        pageSubtitle:
            "Review third-party app scopes, recent disclosures, and revoke access by app or individual scope.",
        backToIdentity: "Back to Account Center",
        empty: "No authorized apps",
        emptyDesc: "Third-party app authorizations will appear here",
        homepage: "Homepage",
        privacy: "Privacy Policy",
        scopeCount: "{count} permissions",
        reason: "Purpose: {reason}",
        grantedAt: "Granted at: {time}",
        lastUsedAt: "Last used: {time}",
        neverUsed: "Last used: no successful disclosure yet",
        activityTitle: "Authorization Activity",
        activityEmpty: "No authorization activity yet",
        activityLoadFailed:
            "Failed to load authorization activity. Please retry",
        activityScopes: "Scopes: {scopes}",
        activityEndpoint: "Endpoint: {endpoint}",
        activityResult: "Result: {result}",
        unknownApp: "Unknown app",
        activityTypes: {
            granted: "Authorization granted",
            denied: "Authorization denied",
            revoked: "Authorization revoked",
            disclosureGranted: "Information disclosed",
            disclosureDenied: "Information disclosure denied",
            replayDetected: "Repeated request recorded",
        },
        revokeApp: "Revoke all",
        revokeScope: "Revoke",
        revoking: "Revoking...",
        revokeSuccess: "Authorization revoked",
        revokeFailed: "Failed to revoke authorization. Please retry",
        revokeAppDialogTitle: "Revoke application authorization",
        revokeAppDialogDescription:
            "This revokes all authorizations for {app}. The application must start authorization again before it can access your information.",
        revokeScopeDialogTitle: "Revoke {scope}",
        revokeScopeDialogDescription:
            'This revokes the "{scope}" permission for {app}. The application must obtain your authorization again before using it.',
        confirmRevoke: "Confirm revoke",
        revokeScopeLabel: "Revoke {scope}",
    },
    notification: {
        title: "Notifications",
        markAllRead: "Mark All Read",
        filterAll: "All",
        filterUnread: "Unread",
        filterRead: "Read",
        empty: "No notifications",
        emptyDesc: "New messages will appear here",
        bell: "Notifications",
        bellMarkAllRead: "Mark All Read",
        viewAll: "View All",
        unreadCount: "{count} unread notifications",
    },
    verification: {
        identity: {
            title: "Identity Verification",
            verified: "Verified",
            unverified: "Not Verified",
            pending: "Under Review",
            rejected: "Rejected",
            submit: "Submit",
            resubmit: "Resubmit",
            mainlandId: "Mainland China ID",
            hkMacau: "HK/Macau Travel Permit",
            twPass: "Taiwan Travel Permit",
            passport: "Passport",
            docType: "Document Type",
            docNumber: "Document Number",
            realName: "Real Name",
            photoFront: "Front Photo",
            photoBack: "Back Photo",
            photoSelfie: "Selfie with Document",
            photoRequired: "Upload clear verification evidence",
            manualEvidencePrompt:
                "We could not verify this identity from your linked academic record. Upload the document front and a selfie holding the document for manual review.",
            uploadFailed: "Failed to upload identity photo. Please try again.",
            photoTooLarge: "Image size must not exceed {max}",
            photoInvalidType: "Only JPG, PNG, or WebP images are supported",
            invalidMainlandID: "Enter a valid Mainland China ID number",
            successAuto: "Verified (auto-matched with academic records)",
            successManual: "Verified (manual review)",
            verifiedAt: "Verified At",
            rejectionReason: "Rejection Reason",
            desc: "Used for sensitive features that require real-name identity. Student verification can be completed separately.",
        },
        student: {
            title: "Student Verification",
            verified: "Verified",
            unverified: "Not Verified",
            pending: "Under Review",
            rejected: "Rejected",
            selectSchool: "Select School",
            studentId: "Student ID",
            password: "Unified Auth Password",
            consent: "I have read and agree to",
            consentPlain:
                "I confirm the submitted information is accurate and agree to use it for student verification",
            consentText: "the Risk Disclosure",
            verify: "Verify",
            identityRequired:
                "This capability requires identity verification first",
            verifySuccess: "Student verification successful",
            verifyPending: "Submitted, pending admin review",
            verifyFailed:
                "Verification failed, please check your student ID and password",
            manualHint:
                "Fill in the required information and submit it for manual review",
            desc: "Student verification unlocks full review access",
            boundSchool: "Bound School",
            activeStudentId: "Current Student ID",
            studentIds: "Bound Student IDs",
            method: "Verification Method",
            verifiedAt: "Verified At",
            academicEmail: {
                studentName: "Name on Academic Record",
                studentNamePlaceholder:
                    "Enter the name in your academic record",
                schoolEmail: "Student Email",
                schoolEmailPlaceholder:
                    "Filled after your student ID and name match",
                emailCode: "Email Verification Code",
                emailCodePlaceholder: "Enter the verification code",
                sendCode: "Send Code",
                codeSent: "Verification code sent",
                sendFailed:
                    "Failed to send the verification code. Please try again.",
                enterIdentity: "Enter your student ID and name first.",
                checking: "Matching your student ID and name...",
                matched: "Your student ID and name match.",
                mismatch:
                    "Your student ID and name do not match. Check them before requesting a code.",
                unavailable:
                    "Academic record matching is temporarily unavailable. Please try again later.",
                waitForMatch:
                    "Wait for student ID and name matching to finish.",
                matchRequired: "Complete student ID and name matching first.",
            },
            methods: {
                ldap: "Unified School Login",
                manual: "Manual Review",
                school_email_otp: "School Email Code",
                school_sso: "School SSO",
            },
            platform: {
                eyebrow: "Trusted student identity",
                title: "Student verification",
                subtitle:
                    "Choose your school and a verification method that works for you. Student credentials remain independent from QQ binding, group admission, and phone binding.",
                verify: "Verify",
                processing: "Processing…",
                cancel: "Cancel this application",
                cancelConfirm:
                    "Cancel this verification application? Unsubmitted material and temporary challenges will also expire.",
                school: {
                    title: "Choose a school",
                    description:
                        "The school directory is not an allowlist. Only schools with validated, currently available verification methods appear here.",
                    search: "Search schools",
                    placeholder: "School name, location, or school code",
                    notFound: "No available school found",
                    suggest: "Suggest a school",
                    schoolName: "School name",
                    location: "Location (optional)",
                    submitSuggestion: "Submit suggestion",
                    suggestionSent: "School suggestion submitted",
                    change: "Change school",
                },
                method: {
                    title: "Choose a verification method",
                    description:
                        "Methods are configured per school. Completing any one method can create a student credential.",
                    available: "Available",
                    unavailable: "Unavailable",
                    change: "Choose another method",
                },
                methods: {
                    realName: "Real-name information check",
                    sso: "Unified school account check",
                    emailReceive: "Receive a code at student email",
                    emailSend: "Send a verification email",
                    manual: "Manual material review",
                    realNameDescription:
                        "Enter your student ID, name, and Mainland China ID number for a one-time check. Sensitive values are not stored as readable profile fields.",
                    ssoDescription:
                        "This check simulates signing in to your school's unified account. StuHelper does not store, cache, or log your school password.",
                    emailReceiveDescription:
                        "After checking the student ID and name, a code is sent only to the canonical student-ID email address, never an alias.",
                    emailSendDescription:
                        "Send a one-time challenge from the canonical student-ID mailbox, useful when outbound code delivery is busy.",
                    manualDescription:
                        "For international students, students from other schools, or cases that automatic checks cannot complete. Schools reuse a secure form and camera workflow.",
                },
                fields: {
                    studentID: "Student ID",
                    name: "Name",
                    documentNumber: "Mainland China ID number",
                    documentHint:
                        "Used only for this real-name information check. Internal data sources and matching details are not exposed here.",
                    password: "Unified school account password",
                    passwordHint:
                        "Used only in controlled memory for this encrypted request. It is never written to databases, caches, queues, or logs.",
                },
                privacy: {
                    dataUsed: "Data used for this check",
                    learnMore: "Privacy and rights details",
                    consent:
                        "I have read this notice, confirm that I am submitting my own information, and consent to the listed data processing for this student verification.",
                },
                email: {
                    code: "Email verification code",
                    sendCode: "Send code",
                    sent: "A code was sent to {email}",
                    expiry: "Enter it before {time}.",
                    verifyCode: "Verify code",
                },
                inbound: {
                    title: "Send a verification email",
                    create: "Create email challenge",
                    from: "Send from the canonical mailbox represented by {email}.",
                    to: "Send to {email}.",
                    subject: "Use this subject:",
                    body: "Include this in the body:",
                    waiting: "This page checks the result automatically. Do not create duplicate challenges.",
                },
                manual: {
                    materialType: "Material type",
                    campusCard: "Campus card",
                    studentCard: "Student ID card",
                    admissionNotice: "Admission notice",
                    otherApproved: "Other school-approved material",
                    createDraft: "Save and capture material",
                    updateDraft: "Update review details",
                    emailTitle: "Contact email check",
                    emailDescription:
                        "The email code is supporting review evidence and never grants student identity by itself.",
                    emailVerified: "Contact email verified",
                    sendEmailCode: "Send email code",
                    codeSentTo: "A code was sent to {email}",
                    cameraTitle: "Capture material live",
                    cameraDescription:
                        "Photograph the original document with a live camera. The browser requests a rear camera, but an ordinary web page cannot attest the hardware source.",
                    openCamera: "Open camera",
                    cameraUnsupported: "Camera unavailable here",
                    capture: "Capture and upload",
                    uploading: "Uploading…",
                    capturePreview: "Captured verification material preview",
                    usePhone: "Scan with phone",
                    qrAlt: "One-time phone camera QR code",
                    openPhoneLink: "Open camera link on this device",
                    scanTitle: "Waiting for phone scan",
                    scanDescription:
                        "This is a short-lived one-time link and contains no name, student ID, or material. This page refreshes after upload.",
                    uploadedTitle: "Material uploaded from phone",
                    chooseDeviceDescription:
                        "Confirm on the phone that you want to return to the desktop, preventing duplicate submissions.",
                    returnedTitle: "Material synchronized",
                    returnedDescription: "The phone capture is now attached to this review application.",
                    expiredTitle: "Camera link expired",
                    expiredDescription: "Create a new one-time QR code.",
                    retentionConfirmation:
                        "By submitting, you confirm the material is used only for manual student verification and retained according to the privacy notice above.",
                    submit: "Submit for manual review",
                    submitting: "Submitting…",
                    pendingTitle: "Waiting for manual review",
                    pendingDescription:
                        "Your material is in the school's authorized review queue. The result will appear in the verification center.",
                    approvedTitle: "Manual review approved",
                    approvedDescription:
                        "A student credential is active and authorized campus services can read the current minimum eligibility result.",
                    rejectedTitle: "This review was not approved",
                    closedTitle: "This application has ended",
                    closedDescription: "Follow the guidance to apply again or use another verification method.",
                },
                credentials: {
                    activeTitle: "Student credential active",
                    activeDescription:
                        "Eligibility is derived from current credentials rather than a permanent verified boolean.",
                    expires: "Valid until {date}",
                    noExpiry: "No fixed expiry; school status changes or revocation can still invalidate it",
                    revoke: "Revoke",
                    revokeConfirm:
                        "Revoking immediately invalidates eligibility that depends on this credential. Continue?",
                    revoked: "Student credential revoked",
                },
                result: {
                    approved: "Student verification complete",
                    approvedDescription:
                        "An auditable and revocable student credential is active. Phone and QQ bindings remain separate workflows.",
                    pending: "Material submitted",
                    pendingDescription:
                        "You do not need to submit again while review is pending. It does not affect credentials for other schools.",
                    done: "Back to account center",
                    addAnother: "Verify another school",
                },
                recovery: {
                    manual: "Can't complete automatically? Use manual review",
                },
                trust: {
                    title: "Verification progress",
                    school: "Confirm school",
                    schoolPending: "Choose your school",
                    method: "Complete identity check",
                    methodPending: "Choose an available method",
                    credential: "Create student credential",
                    credentialPending: "Eligibility is derived after approval",
                    credentialDone: "Credential activated",
                    separation:
                        "Student credentials contain no QQ number, group ID, or phone. Admission reads only minimum eligibility.",
                    phoneLink: "Manage account phone separately",
                },
                mobileCamera: {
                    title: "Capture verification material",
                    subtitle:
                        "A one-time secure handoff for this manual-review application only.",
                    privacy:
                        "This link shows no name, student ID, or form details. The photo goes directly to protected review-material storage.",
                    previewAlt: "Captured verification material preview",
                    openCamera: "Open camera",
                    unsupported: "Camera is not supported in this browser",
                    capture: "Capture and upload",
                    uploading: "Uploading…",
                    uploadedTitle: "Material uploaded",
                    uploadedDescription:
                        "Choose which device to continue on. The other side is locked after confirmation to prevent duplicates.",
                    continueMobile: "Continue on phone",
                    returnDesktop: "Return to desktop",
                    desktopTitle: "Continue on your desktop",
                    desktopDescription:
                        "The material is synchronized and the desktop page will refresh automatically.",
                    expiredTitle: "Link expired",
                    expiredDescription:
                        "Return to the desktop and create a new one-time QR code.",
                    errorTitle: "Unable to open camera link",
                    errorDescription:
                        "The link is invalid or the service is temporarily unavailable.",
                    cameraError:
                        "Unable to open the camera. Check browser permissions.",
                    uploadError: "Material upload failed. Capture it again.",
                    choiceError:
                        "Unable to confirm the continuation device. Try again.",
                },
                errors: {
                    loadTitle: "Verification is temporarily unavailable",
                    load: "Unable to load verification capabilities. Try again later.",
                    createApplication: "Unable to create a verification application. Try again later.",
                    cannotComplete:
                        "This method could not complete the check. Review your input, choose another method, or request manual review.",
                    refresh: "Unable to refresh status. Try again later.",
                    cancel: "Unable to cancel the application. Refresh and try again.",
                    revoke: "Unable to revoke this credential. Try again later.",
                    suggestion: "Unable to submit the school suggestion. Try again later.",
                    saveDraft: "Unable to save manual-review details. Check the form.",
                    emailCode: "Unable to send the email code. Try again later.",
                    invalidCode: "The code is invalid or expired. Request a new one.",
                    camera: "Unable to capture or upload material. Try again.",
                    handoff: "Phone camera handoff is temporarily unavailable. Try again.",
                    submitReview: "Unable to submit manual review. Check the material and email status.",
                },
            },
        },
        phone: {
            title: "Phone Binding",
            bindTitle: "Bind phone number",
            updateTitle: "Update phone number",
            bind: "Bind",
            update: "Update phone number",
            phoneNumber: "Phone Number",
            verifyCode: "Verification Code",
            sendCode: "Send Code",
            verified: "Verified",
            unverified: "Not verified",
            bound: "Bound",
            unbound: "Not Bound",
            sending: "Sending...",
            resend: "Resend in {seconds}s",
            codeSent: "Verification code sent",
            bindSuccess: "Phone number bound successfully",
            invalidPhone: "Please enter a valid phone number",
            invalidCode: "Invalid verification code",
            alreadyBound: "Phone number already bound",
            tooManyRequests: "Too many requests, please try again later",
            tooManyAttempts:
                "Too many failed attempts, please request a new code",
            serviceUnavailable:
                "Phone binding is not configured, so verification codes cannot be sent or checked right now",
            ssoEquivalent:
                "Binding or updating a phone number here verifies the SMS code and syncs the change to the unified sign-in account. StuHelper only keeps the synchronized masked phone and verification state.",
            updateSuccess: "Phone number synchronized",
            platform: {
                eyebrow: "Account security",
                title: "Phone number",
                subtitle:
                    "Your phone supports publishing-account checks and security contact. It remains independent from student verification.",
                success: "Phone verified and synchronized",
                status: {
                    verified: "Phone verified",
                    verifiedDescription:
                        "The current number is in your unified sign-in account and has a separate phone-verification credential.",
                    syncing: "Synchronizing data",
                    syncingDescription:
                        "The number is being updated and read back from the unified sign-in account. Publishing remains closed until this completes.",
                    reviewRequired: "Phone status needs review",
                    reviewDescription:
                        "The unified account and local read-only projection do not currently agree. Retry later or contact support.",
                    unbound: "No phone bound",
                    unboundDescription:
                        "Student verification does not require a phone. Bind one here when publishing content or managing account security.",
                    method: "Confirmation method",
                    verifiedAt: "Confirmed at",
                    schoolConfirmed: "School account information confirmation",
                    smsConfirmed: "SMS code confirmation",
                    verifiedMethod: "Verified",
                },
                form: {
                    bindTitle: "Bind a phone number",
                    changeTitle: "Change phone number",
                    description:
                        "Enter the Mainland China phone number you currently use. The service selects the shortest secure verification path.",
                    phone: "Phone number",
                    authority:
                        "The canonical number in the Casdoor sign-in account is the only writable source of truth. StuHelper keeps only a protected, rebuildable business projection and verification credential.",
                    continue: "Continue",
                },
                sms: {
                    title: "Enter the SMS code",
                    sent: "A code was sent to {phone}",
                    code: "SMS verification code",
                    verify: "Confirm and bind",
                    resend: "Resend code",
                    resendIn: "Resend in {seconds}s",
                    changeNumber: "Use another number",
                },
                syncing: {
                    title: "Synchronizing securely",
                    description:
                        "This usually takes only a few seconds. An idempotent background task continues reconciliation if you close the page.",
                },
                manage: {
                    title: "Manage current number",
                    description:
                        "Changing or removing the number affects publishing requirements, account recovery, and security contact, and requires high-risk confirmation.",
                    change: "Change number",
                    unbind: "Remove phone",
                    unbindConfirm:
                        "Removing the phone temporarily blocks features that require a phone credential and may affect account recovery. Continue?",
                },
                why: {
                    title: "Why is a phone needed?",
                    description:
                        "StuHelper uses a verified phone as a product risk-control requirement for publishing and for necessary security contact. Regulations permit multiple identity-information methods; we do not claim that only phones are legally allowed.",
                    notStudentEvidence:
                        "A phone is not student evidence and never grants student identity alone or in combination.",
                },
                errors: {
                    load: "Unable to load phone status. Try again later.",
                    create: "Unable to create the phone operation. Check the number or try later.",
                    sms: "Unable to send the SMS code. Try again later.",
                    invalidCode: "The code is invalid or expired. Request a new one.",
                    sync: "Phone synchronization has not completed. Refresh later.",
                    unbind: "The phone removal did not complete. Try again later.",
                },
            },
        },
        qq: {
            title: "QQ Binding",
            bound: "Bound",
            unbound: "Not Bound",
            desc: "Bind your QQ account so the bot can map your StuHelper identity and apply group verification rules.",
            createCode: "Create Binding Code",
            regenerateCode: "Regenerate Binding Code",
            creating: "Creating...",
            codeCreated: "Binding code created",
            codeExpired: "The binding code has expired. Create a new one.",
            refreshStatus: "Refresh Status",
            checkingStatus: "Checking...",
            statusUpdated: "QQ binding status updated",
            notYetBound: "No QQ binding result detected yet",
            alreadyBound: "This account is already bound to a QQ number",
            instruction:
                "Send the following command to the bot in a private chat",
            instructionWithoutEntry:
                "The bot entry is not configured. This code can still be used with a deployed bot; ask an administrator to publish the entry.",
            botEntryUnavailable: "Bot entry not configured",
            botEntryMissingHint:
                "This site has no reachable QQ bot entry configured. After creating a code, binding still requires a deployed bot.",
            copyCommand: "Copy binding command",
            commandCopied: "Binding command copied",
            copyCommandFailed: "Copy failed, please copy manually",
            expiresAt: "Expires At",
            qqNumber: "QQ Number",
            boundAt: "Bound At",
            verifiedHint:
                "This account has completed student verification and will be auto-approved in protected groups.",
            pendingHint:
                "After binding QQ, you still need to complete student verification in StuHelper.",
        },
        academic: {
            title: "Academic Info",
            studentId: "Student ID",
            name: "Name",
            college: "College",
            major: "Major",
            class: "Class",
            currentGrade: "Current Grade",
            enrollGrade: "Enrollment Grade",
            eduLevel: "Education Level",
            phone: "Phone",
            email: "Email",
            noData: "No academic data available",
            needVerification: "Please complete student verification first",
            goVerify: "Verify Now",
        },
        profile: {
            title: "Profile",
            identityStatus: "Identity Status",
            studentStatus: "Student Status",
            school: "School",
            studentId: "Student ID",
            goVerify: "Verify Now",
            viewDetail: "View Details",
        },
    },
};
