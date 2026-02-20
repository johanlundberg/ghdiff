// Package diff parses unified diff output into structured types.
package diff

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	diffHeaderRe = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)
	renameFromRe = regexp.MustCompile(`^rename from (.+)$`)
	renameToRe   = regexp.MustCompile(`^rename to (.+)$`)
	binaryRe     = regexp.MustCompile(`^Binary files (.+) and (.+) differ$`)
)

// Parse parses a unified diff string into structured data.
func Parse(input string) (*DiffResult, error) {
	if input == "" {
		return &DiffResult{}, nil
	}

	lines := strings.Split(input, "\n")
	result := &DiffResult{}
	i := 0

	for i < len(lines) {
		// Look for diff header
		m := diffHeaderRe.FindStringSubmatch(lines[i])
		if m == nil {
			i++
			continue
		}

		file := FileDiff{
			OldName: m[1],
			NewName: m[2],
		}
		i++

		// Parse extended header lines until we hit --- or another diff header or a hunk or binary
		for i < len(lines) {
			line := lines[i]

			if strings.HasPrefix(line, "diff --git ") {
				break
			}

			if rm := renameFromRe.FindStringSubmatch(line); rm != nil {
				file.OldName = rm[1]
				file.Status = "renamed"
				i++
				continue
			}
			if rm := renameToRe.FindStringSubmatch(line); rm != nil {
				file.NewName = rm[1]
				file.Status = "renamed"
				i++
				continue
			}

			if bm := binaryRe.FindStringSubmatch(line); bm != nil {
				file.IsBinary = true
				// Extract names from "Binary files a/foo and b/bar differ"
				oldSide := bm[1]
				newSide := bm[2]
				if oldSide == "/dev/null" {
					file.OldName = "/dev/null"
					file.Status = "added"
				} else {
					file.OldName = strings.TrimPrefix(oldSide, "a/")
				}
				if newSide == "/dev/null" {
					file.NewName = "/dev/null"
					file.Status = "deleted"
				} else {
					file.NewName = strings.TrimPrefix(newSide, "b/")
				}
				if file.Status == "" {
					file.Status = "modified"
				}
				i++
				break
			}

			if strings.HasPrefix(line, "--- ") {
				file.OldName = parseFileName(line[4:])
				i++
				if i < len(lines) && strings.HasPrefix(lines[i], "+++ ") {
					file.NewName = parseFileName(lines[i][4:])
					i++
				}

				// Determine status from file names if not already set
				if file.Status == "" {
					switch {
					case file.OldName == "/dev/null":
						file.Status = "added"
					case file.NewName == "/dev/null":
						file.Status = "deleted"
					default:
						file.Status = "modified"
					}
				}
				break
			}

			if strings.HasPrefix(line, "@@ ") {
				// No --- / +++ lines, go directly to hunks
				break
			}

			i++
		}

		// Parse hunks
		for i < len(lines) {
			if strings.HasPrefix(lines[i], "diff --git ") {
				break
			}

			hm := hunkHeaderRe.FindStringSubmatch(lines[i])
			if hm == nil {
				i++
				continue
			}

			hunk, err := parseHunk(hm, lines, &i)
			if err != nil {
				return nil, err
			}
			file.Hunks = append(file.Hunks, hunk)
		}

		// Default status if not set
		if file.Status == "" {
			file.Status = "modified"
		}

		result.Files = append(result.Files, file)
	}

	return result, nil
}

// parseFileName extracts the file name from a --- or +++ line value.
// Handles "a/path", "b/path", and "/dev/null".
func parseFileName(s string) string {
	s = strings.TrimSpace(s)
	if s == "/dev/null" {
		return "/dev/null"
	}
	// Strip the a/ or b/ prefix
	if strings.HasPrefix(s, "a/") || strings.HasPrefix(s, "b/") {
		return s[2:]
	}
	return s
}

// parseHunk parses a single hunk starting at the @@ header line.
// It advances i past all lines belonging to this hunk.
func parseHunk(hm []string, lines []string, i *int) (Hunk, error) {
	oldStart, err := strconv.Atoi(hm[1])
	if err != nil {
		return Hunk{}, fmt.Errorf("invalid old start: %w", err)
	}
	oldLines := 1
	if hm[2] != "" {
		oldLines, err = strconv.Atoi(hm[2])
		if err != nil {
			return Hunk{}, fmt.Errorf("invalid old lines: %w", err)
		}
	}
	newStart, err := strconv.Atoi(hm[3])
	if err != nil {
		return Hunk{}, fmt.Errorf("invalid new start: %w", err)
	}
	newLines := 1
	if hm[4] != "" {
		newLines, err = strconv.Atoi(hm[4])
		if err != nil {
			return Hunk{}, fmt.Errorf("invalid new lines: %w", err)
		}
	}

	// Build the header string: include function context if present but trim trailing whitespace
	header := "@@ -" + hm[1]
	if hm[2] != "" {
		header += "," + hm[2]
	}
	header += " +" + hm[3]
	if hm[4] != "" {
		header += "," + hm[4]
	}
	header += " @@"
	if funcCtx := strings.TrimSpace(hm[5]); funcCtx != "" {
		header += " " + funcCtx
	}

	hunk := Hunk{
		OldStart: oldStart,
		OldLines: oldLines,
		NewStart: newStart,
		NewLines: newLines,
		Header:   header,
	}

	oldNum := oldStart
	newNum := newStart
	*i++ // advance past @@ line

loop:
	for *i < len(lines) {
		line := lines[*i]

		// Stop at next hunk or next diff
		if strings.HasPrefix(line, "@@ ") || strings.HasPrefix(line, "diff --git ") {
			break
		}

		// Skip "no newline" marker
		if strings.HasPrefix(line, `\ No newline at end of file`) {
			*i++
			continue
		}

		if len(line) == 0 {
			// Empty line in diff output -- could be end of input or a context line
			// with empty content. If we're in the middle of a hunk, treat as end.
			*i++
			break
		}

		prefix := line[0]
		content := line[1:]

		switch prefix {
		case ' ':
			hunk.Lines = append(hunk.Lines, Line{
				Type:    "context",
				Content: content,
				OldNum:  oldNum,
				NewNum:  newNum,
			})
			oldNum++
			newNum++
		case '+':
			hunk.Lines = append(hunk.Lines, Line{
				Type:    "add",
				Content: content,
				NewNum:  newNum,
			})
			newNum++
		case '-':
			hunk.Lines = append(hunk.Lines, Line{
				Type:    "delete",
				Content: content,
				OldNum:  oldNum,
			})
			oldNum++
		default:
			// Unknown prefix, likely end of hunk
			break loop
		}

		*i++
	}

	return hunk, nil
}
