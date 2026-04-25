package academics

type Source struct {
	ID       int64  `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Config   []byte `json:"-"`
	Enabled  bool   `json:"enabled"`
}

type ImportJob struct {
	ID                int64          `json:"id"`
	SourceKey         string         `json:"sourceKey"`
	SourceName        string         `json:"sourceName"`
	Provider          string         `json:"provider"`
	Status            string         `json:"status"`
	TriggerMode       string         `json:"triggerMode"`
	RequestedByUserID *string        `json:"requestedByUserID,omitempty"`
	Stats             map[string]any `json:"stats"`
	ErrorMessage      *string        `json:"errorMessage,omitempty"`
	StartedAt         *string        `json:"startedAt,omitempty"`
	FinishedAt        *string        `json:"finishedAt,omitempty"`
	CreatedAt         string         `json:"createdAt"`
}

type Term struct {
	ID        int64   `json:"id"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	StartDate *string `json:"startDate,omitempty"`
	EndDate   *string `json:"endDate,omitempty"`
	IsCurrent bool    `json:"isCurrent"`
}

type Offering struct {
	ID             int64          `json:"id"`
	TermCode       string         `json:"termCode"`
	TermName       string         `json:"termName"`
	CourseCode     string         `json:"courseCode"`
	CourseName     string         `json:"courseName"`
	SectionCode    string         `json:"sectionCode"`
	SchoolName     *string        `json:"schoolName,omitempty"`
	DepartmentName *string        `json:"departmentName,omitempty"`
	Campus         *string        `json:"campus,omitempty"`
	TeacherNames   []string       `json:"teacherNames"`
	Schedules      []ScheduleSlot `json:"schedules,omitempty"`
}

type ScheduleSlot struct {
	Weekday     int16   `json:"weekday"`
	StartPeriod int16   `json:"startPeriod"`
	EndPeriod   int16   `json:"endPeriod"`
	Location    string  `json:"location"`
	Building    *string `json:"building,omitempty"`
	WeeksText   string  `json:"weeksText"`
}

type OfferingFilters struct {
	TermCode       string
	SchoolName     string
	DepartmentName string
	CourseQuery    string
	TeacherQuery   string
	Page           int
	PageSize       int
}

type ImportRequest struct {
	SourceKey string `json:"sourceKey"`
}

type ImportTerm struct {
	ExternalID string  `json:"externalId"`
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	StartDate  *string `json:"startDate,omitempty"`
	EndDate    *string `json:"endDate,omitempty"`
	IsCurrent  bool    `json:"isCurrent"`
}

type ImportCourse struct {
	ExternalID     string   `json:"externalId"`
	Code           string   `json:"code"`
	Name           string   `json:"name"`
	DepartmentCode *string  `json:"departmentCode,omitempty"`
	DepartmentName *string  `json:"departmentName,omitempty"`
	Credit         *float64 `json:"credit,omitempty"`
}

type ImportTeacher struct {
	ExternalID     string  `json:"externalId"`
	Name           string  `json:"name"`
	DepartmentName *string `json:"departmentName,omitempty"`
}

type ImportOffering struct {
	ExternalID         string         `json:"externalId"`
	TermExternalID     string         `json:"termExternalId"`
	CourseExternalID   string         `json:"courseExternalId"`
	SectionCode        string         `json:"sectionCode"`
	SchoolName         *string        `json:"schoolName,omitempty"`
	DepartmentName     *string        `json:"departmentName,omitempty"`
	Campus             *string        `json:"campus,omitempty"`
	EnrollmentLimit    *int           `json:"enrollmentLimit,omitempty"`
	TeacherExternalIDs []string       `json:"teacherExternalIds"`
	Schedules          []ScheduleSlot `json:"schedules"`
}

type ImportMembership struct {
	OfferingExternalID string  `json:"offeringExternalId"`
	ExternalUserID     string  `json:"externalUserId"`
	StudentID          *string `json:"studentId,omitempty"`
	Role               string  `json:"role"`
}

type Snapshot struct {
	Terms       []ImportTerm       `json:"terms"`
	Courses     []ImportCourse     `json:"courses"`
	Teachers    []ImportTeacher    `json:"teachers"`
	Offerings   []ImportOffering   `json:"offerings"`
	Memberships []ImportMembership `json:"memberships"`
}

func (s Snapshot) Stats() map[string]any {
	return map[string]any{
		"terms":       len(s.Terms),
		"courses":     len(s.Courses),
		"teachers":    len(s.Teachers),
		"offerings":   len(s.Offerings),
		"memberships": len(s.Memberships),
	}
}
