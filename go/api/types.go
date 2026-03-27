package api

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
	ID       string `json:"id,omitempty"`
	Login    string `json:"login,omitempty"`
	FullName string `json:"fullName,omitempty"`
	Email    string `json:"email,omitempty"`
	Guest    bool   `json:"guest,omitempty"`
	Online   bool   `json:"online,omitempty"`
}

// LinkType describes a type of issue link (e.g. "depends on").
type LinkType struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	SourceToTarget string `json:"sourceToTarget,omitempty"`
	TargetToSource string `json:"targetToSource,omitempty"`
}
