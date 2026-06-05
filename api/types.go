package api

import "encoding/json"

// Issue represents a YouTrack issue.
type Issue struct {
	ID          string   `json:"id,omitempty"`
	IDReadable  string   `json:"idReadable,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Project     *Project `json:"project,omitempty"`
	Reporter    *User    `json:"reporter,omitempty"`
	Resolved    *int64   `json:"resolved,omitempty"`
	Created     int64    `json:"created,omitempty"`

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
	ID      string `json:"id,omitempty"`
	Text    string `json:"text,omitempty"`
	Author  *User  `json:"author,omitempty"`
	Created int64  `json:"created,omitempty"`
}

// Project is a YouTrack project.
type Project struct {
	ID        string `json:"id,omitempty"`
	ShortName string `json:"shortName,omitempty"`
	Name      string `json:"name,omitempty"`
	Leader    *User  `json:"leader,omitempty"`
}

// User is a YouTrack user.
type User struct {
	ID       string    `json:"id,omitempty"`
	Login    string    `json:"login,omitempty"`
	FullName string    `json:"fullName,omitempty"`
	Email    string    `json:"email,omitempty"`
	Guest    bool      `json:"guest,omitempty"`
	Online   bool      `json:"online,omitempty"`
	UserType *UserType `json:"userType,omitempty"`
}

// UserType represents a YouTrack user type (STANDARD_USER, AGENT, REPORTER).
type UserType struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// LinkType describes a type of issue link (e.g. "depends on").
type LinkType struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	SourceToTarget string `json:"sourceToTarget,omitempty"`
	TargetToSource string `json:"targetToSource,omitempty"`
}

type Article struct {
	ID         string   `json:"id,omitempty"`
	IDReadable string   `json:"idReadable,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Content    string   `json:"content,omitempty"`
	Reporter   *User    `json:"reporter,omitempty"`
	Project    *Project `json:"project,omitempty"`
	Created    int64    `json:"created,omitempty"`
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
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
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
	Type    string   `json:"$type,omitempty"`
	ID      string   `json:"id,omitempty"`
	Project *Project `json:"project,omitempty"`
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
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	AutoAttach bool   `json:"autoAttach,omitempty"`
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
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
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

type WorkTimeSettings struct {
	ID                      string `json:"id,omitempty"`
	DaysAWeek               int    `json:"daysAWeek,omitempty"`
	MinutesADay             int    `json:"minutesADay,omitempty"`
	MinutesADayPresentation string `json:"minutesADayPresentation,omitempty"`
	FirstDayOfWeek          int    `json:"firstDayOfWeek,omitempty"`
}

type ArticleDraft struct {
	ID      string `json:"id,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type IssueDraft struct {
	ID      string `json:"id,omitempty"`
	Summary string `json:"summary,omitempty"`
	Created int64  `json:"created,omitempty"`
}

type GlobalSettings struct {
	ID                string `json:"id,omitempty"`
	BaseURL           string `json:"baseUrl,omitempty"`
	MaxUploadFileSize int64  `json:"maxUploadFileSize,omitempty"`
}

type VCSServer struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}

type VCSProcessor struct {
	ID      string   `json:"id,omitempty"`
	Path    string   `json:"path,omitempty"`
	Enabled bool     `json:"enabled,omitempty"`
	Project *Project `json:"project,omitempty"`
}

type MailboxServer struct {
	ID       string `json:"id,omitempty"`
	Login    string `json:"login,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

type Tag struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Owner *User  `json:"owner,omitempty"`
}

type SavedSearch struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Query string `json:"query,omitempty"`
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

type RemovedDocument struct {
	ID         string `json:"id,omitempty"`
	IDReadable string `json:"idReadable,omitempty"`
	Summary    string `json:"summary,omitempty"`
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
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
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
