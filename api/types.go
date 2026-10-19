package api

import "encoding/json"

// Issue represents a YouTrack issue.
type Issue struct {
	ID                  string            `json:"id,omitempty"`
	IDReadable          string            `json:"idReadable,omitempty"`
	Summary             string            `json:"summary,omitempty"`
	Description         string            `json:"description,omitempty"`
	WikifiedDescription string            `json:"wikifiedDescription,omitempty"`
	Project             *Project          `json:"project,omitempty"`
	Reporter            *User             `json:"reporter,omitempty"`
	Updater             *User             `json:"updater,omitempty"`
	DraftOwner          *User             `json:"draftOwner,omitempty"`
	Resolved            *int64            `json:"resolved,omitempty"`
	Created             int64             `json:"created,omitempty"`
	Updated             int64             `json:"updated,omitempty"`
	NumberInProject     int64             `json:"numberInProject,omitempty"`
	CommentsCount       int               `json:"commentsCount,omitempty"`
	Votes               int               `json:"votes,omitempty"`
	IsDraft             bool              `json:"isDraft,omitempty"`
	Parent              *IssueLink        `json:"parent,omitempty"`
	Subtasks            *IssueLink        `json:"subtasks,omitempty"`
	Links               []IssueLink       `json:"links,omitempty"`
	Attachments         []IssueAttachment `json:"attachments,omitempty"`
	Comments            []Comment         `json:"comments,omitempty"`
	PinnedComments      []Comment         `json:"pinnedComments,omitempty"`
	Tags                []Tag             `json:"tags,omitempty"`
	CustomFields        []IssueField      `json:"customFields,omitempty"`
	Voters              *IssueVoters      `json:"voters,omitempty"`
	Watchers            *IssueWatchers    `json:"watchers,omitempty"`
	Visibility          *Visibility       `json:"visibility,omitempty"`
	ExternalIssue       *ExternalIssue    `json:"externalIssue,omitempty"`

	// raw preserves the original API JSON so --fields projection can access
	// any field returned by YouTrack, not just the ones declared above.
	raw json.RawMessage
}

func (i *Issue) UnmarshalJSON(data []byte) error {
	type issueAlias Issue
	if err := json.Unmarshal(data, (*issueAlias)(i)); err != nil {
		return err
	}
	i.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (i *Issue) MarshalJSON() ([]byte, error) {
	if i.raw != nil {
		return i.raw, nil
	}
	type issueAlias Issue
	return json.Marshal((*issueAlias)(i))
}

// Comment on an issue or article.
type Comment struct {
	ID         string      `json:"id,omitempty"`
	Text       string      `json:"text,omitempty"`
	Author     *User       `json:"author,omitempty"`
	Created    int64       `json:"created,omitempty"`
	Updated    int64       `json:"updated,omitempty"`
	Deleted    bool        `json:"deleted"`
	Visibility *Visibility `json:"visibility,omitempty"`

	// raw preserves fields requested through --fields, including fields the
	// server returns for deleted comments whose text may still be present.
	raw json.RawMessage
}

func (c *Comment) UnmarshalJSON(data []byte) error {
	type commentAlias Comment
	if err := json.Unmarshal(data, (*commentAlias)(c)); err != nil {
		return err
	}
	c.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (c *Comment) MarshalJSON() ([]byte, error) {
	if c.raw != nil {
		return c.raw, nil
	}
	type commentAlias Comment
	return json.Marshal((*commentAlias)(c))
}

// Project is a YouTrack project.
type Project struct {
	ID          string `json:"id,omitempty"`
	ShortName   string `json:"shortName,omitempty"`
	Name        string `json:"name,omitempty"`
	Leader      *User  `json:"leader,omitempty"`
	Description string `json:"description,omitempty"`
	Archived    bool   `json:"archived,omitempty"`
}

// User is a YouTrack user.
type User struct {
	ID           string        `json:"id,omitempty"`
	Login        string        `json:"login,omitempty"`
	FullName     string        `json:"fullName,omitempty"`
	Email        string        `json:"email,omitempty"`
	Guest        bool          `json:"guest,omitempty"`
	Online       bool          `json:"online,omitempty"`
	Banned       bool          `json:"banned,omitempty"`
	AvatarURL    string        `json:"avatarUrl,omitempty"`
	RingID       string        `json:"ringId,omitempty"`
	UserType     *UserType     `json:"userType,omitempty"`
	Tags         []Tag         `json:"tags,omitempty"`
	SavedQueries []SavedSearch `json:"savedQueries,omitempty"`
}

// UserType represents a YouTrack user type (STANDARD_USER, AGENT, REPORTER).
type UserType struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// LinkType describes a type of issue link (e.g. "depends on").
type LinkType = IssueLinkType

// IssueLinkType describes a type of issue link (e.g. "depends on").
type IssueLinkType struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	SourceToTarget string `json:"sourceToTarget,omitempty"`
	TargetToSource string `json:"targetToSource,omitempty"`
	Directed       bool   `json:"directed,omitempty"`
}

type Article struct {
	ID            string    `json:"id,omitempty"`
	IDReadable    string    `json:"idReadable,omitempty"`
	Summary       string    `json:"summary,omitempty"`
	Content       string    `json:"content,omitempty"`
	Reporter      *User     `json:"reporter,omitempty"`
	Project       *Project  `json:"project,omitempty"`
	Created       int64     `json:"created,omitempty"`
	Updated       int64     `json:"updated,omitempty"`
	UpdatedBy     *User     `json:"updatedBy,omitempty"`
	ParentArticle *Article  `json:"parentArticle,omitempty"`
	Tags          []Tag     `json:"tags,omitempty"`
	ChildArticles []Article `json:"childArticles,omitempty"`
}

type ArticleAttachment struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

type Group struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Permission struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Key         string `json:"key,omitempty"`
	Description string `json:"description,omitempty"`
}

type Role struct {
	ID          string       `json:"id,omitempty"`
	Name        string       `json:"name,omitempty"`
	Description string       `json:"description,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}

type AssignedRole struct {
	Type   string      `json:"$type,omitempty"`
	ID     string      `json:"id,omitempty"`
	Role   *Role       `json:"role,omitempty"`
	Holder *RoleHolder `json:"holder,omitempty"`
	Scope  *RoleScope  `json:"scope,omitempty"`
}

type RoleHolder struct {
	Type  string `json:"$type,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Login string `json:"login,omitempty"`
	Email string `json:"email,omitempty"`
}

type RoleScope struct {
	Type         string        `json:"$type,omitempty"`
	ID           string        `json:"id,omitempty"`
	Project      *Project      `json:"project,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

type Workflow struct {
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Title string         `json:"title,omitempty"`
	Rules []WorkflowRule `json:"rules,omitempty"`
}

type WorkflowRule struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Title string `json:"title,omitempty"`
}

type FieldType struct {
	ID string `json:"id,omitempty"`
}

type CustomField struct {
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name,omitempty"`
	FieldType *FieldType `json:"fieldType,omitempty"`
}

type Bundle struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type WorkItemType struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	AutoAttached bool   `json:"autoAttached,omitempty"`
}

type IssueLink struct {
	ID        string    `json:"id,omitempty"`
	Direction string    `json:"direction,omitempty"`
	LinkType  *LinkType `json:"linkType,omitempty"`
	Issues    []Issue   `json:"issues,omitempty"`
}

type IssueAttachment struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Created  int64  `json:"created,omitempty"`
}

type WorkItem struct {
	ID       string        `json:"id,omitempty"`
	Text     string        `json:"text,omitempty"`
	Date     int64         `json:"date,omitempty"`
	Duration *WorkDuration `json:"duration,omitempty"`
	Author   *User         `json:"author,omitempty"`
}

type WorkDuration struct {
	Minutes int `json:"minutes,omitempty"`
}

type Board struct {
	ID       string    `json:"id,omitempty"`
	Name     string    `json:"name,omitempty"`
	Projects []Project `json:"projects,omitempty"`
	Sprints  []Sprint  `json:"sprints,omitempty"`
}

type Sprint struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Goal      string `json:"goal,omitempty"`
	Start     int64  `json:"start,omitempty"`
	Finish    int64  `json:"finish,omitempty"`
	Archived  bool   `json:"archived,omitempty"`
	IsDefault bool   `json:"isDefault,omitempty"`
}

// VCS change states, as used by the state field of VCSChange.
const (
	VCSChangeStateDefault          = 0 // attached by a VCS integration
	VCSChangeStateManuallyAttached = 1
	VCSChangeStateDetached         = 2
	VCSChangeStateLegacy           = 3
)

// VCSChange is a commit or pull request linked to an issue.
// The $type field distinguishes the two: "VcsChange" or "PullRequest".
type VCSChange struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"$type,omitempty"`
	Version string `json:"version,omitempty"`
	// State is a pointer because 0 (attached by an integration) is a meaningful
	// value that omitempty would otherwise drop from a request body.
	State   *int     `json:"state,omitempty"`
	Text    string   `json:"text,omitempty"`
	Date    int64    `json:"date,omitempty"`
	Fetched int64    `json:"fetched,omitempty"`
	Files   int      `json:"files,omitempty"`
	URLs    []string `json:"urls,omitempty"`
	Author  *User    `json:"author,omitempty"`

	// Pull request fields, empty for commits.
	IDReadable string `json:"idReadable,omitempty"`
	Title      string `json:"title,omitempty"`
}

type Dashboard struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type ProjectTemplate struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	ShortName string `json:"shortName,omitempty"`
}

type ProjectCustomField struct {
	ID    string       `json:"id,omitempty"`
	Field *CustomField `json:"field,omitempty"`
}

type Plugin struct {
	ID      string `json:"id,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
	State   string `json:"state,omitempty"`
}

type Reaction struct {
	ID       string `json:"id,omitempty"`
	Reaction string `json:"reaction,omitempty"`
	Author   *User  `json:"author,omitempty"`
}

type IssueField struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Value any    `json:"value,omitempty"`
}

type IssueVoters struct {
	HasVote bool `json:"hasVote,omitempty"`
	Votes   int  `json:"votes,omitempty"`
}

type IssueWatchers struct {
	HasStar bool `json:"hasStar,omitempty"`
}

type Visibility struct {
	Type            string  `json:"$type,omitempty"`
	PermittedGroups []Group `json:"permittedGroups,omitempty"`
	PermittedUsers  []User  `json:"permittedUsers,omitempty"`
}

type ExternalIssue struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type WorkTimeSettings struct {
	ID                      string `json:"id,omitempty"`
	DaysAWeek               int    `json:"daysAWeek,omitempty"`
	MinutesADay             int    `json:"minutesADay,omitempty"`
	MinutesADayPresentation string `json:"minutesADayPresentation,omitempty"`
	FirstDayOfWeek          int    `json:"firstDayOfWeek"` // 0 is Sunday, so no omitempty
}

type ArticleDraft struct {
	ID      string `json:"id,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type IssueDraftFieldValue struct {
	Type string `json:"$type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type IssueDraftField struct {
	ID    string                `json:"id,omitempty"`
	Name  string                `json:"name,omitempty"`
	Type  string                `json:"$type,omitempty"`
	Value *IssueDraftFieldValue `json:"value,omitempty"`
}

type IssueDraft struct {
	ID          string            `json:"id,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Description string            `json:"description,omitempty"`
	Created     int64             `json:"created,omitempty"`
	Fields      []IssueDraftField `json:"fields,omitempty"`
}

type SystemSettings struct {
	ID                        string `json:"id,omitempty"`
	BaseURL                   string `json:"baseUrl,omitempty"`
	MaxUploadFileSize         int64  `json:"maxUploadFileSize,omitempty"`
	MaxExportItems            int    `json:"maxExportItems,omitempty"`
	AdministratorEmail        string `json:"administratorEmail,omitempty"`
	AllowStatisticsCollection bool   `json:"allowStatisticsCollection,omitempty"`
	IsApplicationReadOnly     bool   `json:"isApplicationReadOnly,omitempty"`
}

type GlobalSettings struct {
	ID             string          `json:"id,omitempty"`
	SystemSettings *SystemSettings `json:"systemSettings,omitempty"`
}

// InstanceVersion is the version/build info reported by a YouTrack instance's
// /api/config endpoint. Version is a release string like "2026.3"; Build is
// the numeric build within that release, e.g. "4599".
type InstanceVersion struct {
	Version string `json:"version,omitempty"`
	Build   string `json:"build,omitempty"`
}

type Tag struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Owner *User  `json:"owner,omitempty"`
}

type SharingSettings struct {
	PermittedGroups []Group `json:"permittedGroups,omitempty"`
	PermittedUsers  []User  `json:"permittedUsers,omitempty"`
}

type SavedSearch struct {
	ID                  string           `json:"id,omitempty"`
	Name                string           `json:"name,omitempty"`
	Query               string           `json:"query,omitempty"`
	ReadSharingSettings *SharingSettings `json:"readSharingSettings,omitempty"`
}

type Organization struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type App struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

type Canvas struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Owner *User  `json:"owner,omitempty"`
}

type CanvasNode struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	X     int    `json:"x,omitempty"`
	Y     int    `json:"y,omitempty"`
}

type GanttChart struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	StartDate int64  `json:"startDate,omitempty"`
}

type GanttResource struct {
	ID        string `json:"id,omitempty"`
	Issue     *Issue `json:"issue,omitempty"`
	StartDate int64  `json:"startDate,omitempty"`
}

// CanvasLinkKind enumerates the valid link types for planning canvas links.
type CanvasLinkKind string

const (
	CanvasLinkKindOneWay     CanvasLinkKind = "ONE_WAY"
	CanvasLinkKindTwoWay     CanvasLinkKind = "TWO_WAY"
	CanvasLinkKindPrototyped CanvasLinkKind = "PROTOTYPED"
)

type CanvasLink struct {
	ID         string         `json:"id,omitempty"`
	SourceCard map[string]any `json:"sourceCard,omitempty"`
	TargetCard map[string]any `json:"targetCard,omitempty"`
}
