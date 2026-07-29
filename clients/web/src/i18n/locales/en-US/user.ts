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
            methods: {
                ldap: "Unified School Login",
                manual: "Manual Review",
                school_email_otp: "School Email Code",
                school_sso: "School SSO",
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
