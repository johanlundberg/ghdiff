package diff

// DiffResult contains all file diffs parsed from a unified diff.
type DiffResult struct { //nolint:revive // renaming would break public API
	Files []FileDiff `json:"files"`
}

// FileDiff represents the diff for a single file.
type FileDiff struct {
	OldName  string `json:"oldName"`
	NewName  string `json:"newName"`
	Status   string `json:"status"` // "added", "deleted", "modified", "renamed"
	IsBinary bool   `json:"isBinary"`
	Hunks    []Hunk `json:"hunks"`
}

// Hunk represents a contiguous block of changes within a file diff.
type Hunk struct {
	OldStart int    `json:"oldStart"`
	OldLines int    `json:"oldLines"`
	NewStart int    `json:"newStart"`
	NewLines int    `json:"newLines"`
	Header   string `json:"header"`
	Lines    []Line `json:"lines"`
}

// Line represents a single line within a hunk.
type Line struct {
	Type    string `json:"type"` // "add", "delete", "context"
	Content string `json:"content"`
	OldNum  int    `json:"oldNum,omitempty"`
	NewNum  int    `json:"newNum,omitempty"`
}
