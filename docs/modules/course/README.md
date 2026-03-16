# Course Review Community

The course review community revolves around course entities, teacher profiles, review content, notifications, and admin moderation. This is the most complete business domain in the system.

## Code Scope

| Location | Purpose |
| --- | --- |
| `server/internal/modules/course` | Departments, terms, categories, courses, portal statistics |
| `server/internal/modules/course/review` | Review read/write, replies, drafts, favorites, notifications, reports, admin moderation |
| `server/api/paths/course.yaml` | Course entity contracts |
| `server/api/paths/review-*.yaml` | Review subdomain contracts |

## Capabilities

### Course Entities

- Course search and listing
- Course details
- Departments, terms, and categories
- Portal statistics

### Review Content

- Post, edit, delete reviews
- Rating dimensions and rating statistics
- Teacher statistics
- Review visibility based on verification status

### User Interactions

- Upvote and downvote reviews
- Favorite courses
- Save and load drafts
- Reply to reviews
- View my reviews and votes

### Notifications

- Notification list
- Unread count
- Mark as read
- Mark all as read

### Content Moderation

- Report reviews
- Hide and restore reviews
- Batch review operations
- Export reviews
- Operation logs
- Teacher management
- Sensitive word management

## Access Rules

| Scenario | Rule |
| --- | --- |
| Public browsing | Returns publicly visible content |
| Full content viewing | Determined by access facts (student verification status) |
| Post review | Requires student verification and identity verification |
| Admin management | Requires appropriate capabilities |

## API Endpoints

### Public Endpoints

```text
GET  /api/v1/course/departments
GET  /api/v1/course/terms
GET  /api/v1/course/categories
GET  /api/v1/course/courses
GET  /api/v1/course/courses/search
GET  /api/v1/course/courses/{courseID}
GET  /api/v1/course/stats

GET  /api/v1/course/review/rating-dimensions
GET  /api/v1/course/review/courses/{courseID}/rating-stats
GET  /api/v1/course/review/courses/{courseID}/rating-trend
GET  /api/v1/course/review/courses/{courseID}/teachers
GET  /api/v1/course/review/courses/{courseID}/reviews
GET  /api/v1/course/review/reviews/latest
GET  /api/v1/course/review/reviews/batch
GET  /api/v1/course/review/reviews/{reviewID}/replies
GET  /api/v1/course/review/stats
GET  /api/v1/course/review/rankings/hot
GET  /api/v1/course/review/teachers/{teacherID}/stats
```

### Authenticated Endpoints

```text
POST   /api/v1/course/review/reviews
PUT    /api/v1/course/review/reviews/{reviewID}
DELETE /api/v1/course/review/reviews/{reviewID}
POST   /api/v1/course/review/reviews/{reviewID}/votes
POST   /api/v1/course/review/reviews/{reviewID}/reports
POST   /api/v1/course/review/reviews/{reviewID}/replies
DELETE /api/v1/course/review/replies/{replyID}
POST   /api/v1/course/review/content/check
POST   /api/v1/course/review/courses/{courseID}/favorites
DELETE /api/v1/course/review/courses/{courseID}/favorites
POST   /api/v1/course/review/drafts
GET    /api/v1/course/review/drafts/{courseID}
DELETE /api/v1/course/review/drafts/{courseID}

GET  /api/v1/course/review/user/reviews
GET  /api/v1/course/review/user/votes
GET  /api/v1/course/review/user/favorites
GET  /api/v1/course/review/user/notifications
GET  /api/v1/course/review/user/notifications/unread-count
PUT  /api/v1/course/review/user/notifications/{notificationID}/read
PUT  /api/v1/course/review/user/notifications/read-all
```

### Admin Endpoints

```text
GET  /api/v1/course/review/admin/reports
PUT  /api/v1/course/review/admin/reports/{reportID}
GET  /api/v1/course/review/admin/reviews
PUT  /api/v1/course/review/admin/reviews/{reviewID}
POST /api/v1/course/review/admin/reviews/batch
POST /api/v1/course/review/admin/reviews/{reviewID}/edit
GET  /api/v1/course/review/admin/stats
GET  /api/v1/course/review/admin/logs
GET  /api/v1/course/review/admin/export
GET  /api/v1/course/review/admin/teachers
POST /api/v1/course/review/admin/teachers
PUT  /api/v1/course/review/admin/teachers/{teacherID}
DELETE /api/v1/course/review/admin/teachers/{teacherID}
GET  /api/v1/course/review/admin/sensitive-words
POST /api/v1/course/review/admin/sensitive-words
PUT  /api/v1/course/review/admin/sensitive-words/{sensitiveWordID}
DELETE /api/v1/course/review/admin/sensitive-words/{sensitiveWordID}
```

## Database Tables

| Table | Purpose |
| --- | --- |
| `departments` | Academic departments |
| `courses` | Course records |
| `teachers` | Teacher profiles |
| `rating_dimensions` | Configurable rating dimensions |
| `reviews` | Course reviews |
| `review_votes` | Upvotes and downvotes |
| `review_reports` | User reports |
| `review_replies` | Review replies |
| `course_favorites` | User course favorites |
| `review_drafts` | Unsaved review drafts |
| `notifications` | User notifications |
| `admin_operation_logs` | Admin operation audit |

## Related Documentation

- [API Overview](../../reference/api-overview.md)
- [Database Design](../../reference/database.md)
- [Security Design](06-security.md)
- [Rating Dimensions](07-rating-dimensions.md)
