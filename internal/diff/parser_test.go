package diff

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *DiffResult
	}{
		{
			name:  "empty diff",
			input: "",
			expected: &DiffResult{
				Files: nil,
			},
		},
		{
			name: "simple file modification",
			input: `diff --git a/hello.go b/hello.go
index 1234567..abcdef0 100644
--- a/hello.go
+++ b/hello.go
@@ -1,4 +1,5 @@
 package main
 
 func main() {
-	fmt.Println("hello")
+	fmt.Println("hello, world")
+	fmt.Println("goodbye")
 }
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "hello.go",
						NewName: "hello.go",
						Status:  "modified",
						Hunks: []Hunk{
							{
								OldStart: 1,
								OldLines: 4,
								NewStart: 1,
								NewLines: 5,
								Header:   "@@ -1,4 +1,5 @@",
								Lines: []Line{
									{Type: "context", Content: "package main", OldNum: 1, NewNum: 1},
									{Type: "context", Content: "", OldNum: 2, NewNum: 2},
									{Type: "context", Content: "func main() {", OldNum: 3, NewNum: 3},
									{Type: "delete", Content: "\tfmt.Println(\"hello\")", OldNum: 4},
									{Type: "add", Content: "\tfmt.Println(\"hello, world\")", NewNum: 4},
									{Type: "add", Content: "\tfmt.Println(\"goodbye\")", NewNum: 5},
									{Type: "context", Content: "}", OldNum: 5, NewNum: 6},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "new file",
			input: `diff --git a/new.txt b/new.txt
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/new.txt
@@ -0,0 +1,3 @@
+line one
+line two
+line three
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "/dev/null",
						NewName: "new.txt",
						Status:  "added",
						Hunks: []Hunk{
							{
								OldStart: 0,
								OldLines: 0,
								NewStart: 1,
								NewLines: 3,
								Header:   "@@ -0,0 +1,3 @@",
								Lines: []Line{
									{Type: "add", Content: "line one", NewNum: 1},
									{Type: "add", Content: "line two", NewNum: 2},
									{Type: "add", Content: "line three", NewNum: 3},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "deleted file",
			input: `diff --git a/old.txt b/old.txt
deleted file mode 100644
index 1234567..0000000
--- a/old.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-goodbye
-world
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "old.txt",
						NewName: "/dev/null",
						Status:  "deleted",
						Hunks: []Hunk{
							{
								OldStart: 1,
								OldLines: 2,
								NewStart: 0,
								NewLines: 0,
								Header:   "@@ -1,2 +0,0 @@",
								Lines: []Line{
									{Type: "delete", Content: "goodbye", OldNum: 1},
									{Type: "delete", Content: "world", OldNum: 2},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "renamed file",
			input: `diff --git a/old_name.go b/new_name.go
similarity index 100%
rename from old_name.go
rename to new_name.go
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "old_name.go",
						NewName: "new_name.go",
						Status:  "renamed",
					},
				},
			},
		},
		{
			name: "renamed file with changes",
			input: `diff --git a/old_name.go b/new_name.go
similarity index 80%
rename from old_name.go
rename to new_name.go
index 1234567..abcdef0 100644
--- a/old_name.go
+++ b/new_name.go
@@ -1,3 +1,3 @@
 package main
 
-var x = 1
+var x = 2
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "old_name.go",
						NewName: "new_name.go",
						Status:  "renamed",
						Hunks: []Hunk{
							{
								OldStart: 1,
								OldLines: 3,
								NewStart: 1,
								NewLines: 3,
								Header:   "@@ -1,3 +1,3 @@",
								Lines: []Line{
									{Type: "context", Content: "package main", OldNum: 1, NewNum: 1},
									{Type: "context", Content: "", OldNum: 2, NewNum: 2},
									{Type: "delete", Content: "var x = 1", OldNum: 3},
									{Type: "add", Content: "var x = 2", NewNum: 3},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple files",
			input: `diff --git a/a.txt b/a.txt
index 1234567..abcdef0 100644
--- a/a.txt
+++ b/a.txt
@@ -1,2 +1,2 @@
 first
-second
+SECOND
diff --git a/b.txt b/b.txt
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/b.txt
@@ -0,0 +1 @@
+new file content
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "a.txt",
						NewName: "a.txt",
						Status:  "modified",
						Hunks: []Hunk{
							{
								OldStart: 1,
								OldLines: 2,
								NewStart: 1,
								NewLines: 2,
								Header:   "@@ -1,2 +1,2 @@",
								Lines: []Line{
									{Type: "context", Content: "first", OldNum: 1, NewNum: 1},
									{Type: "delete", Content: "second", OldNum: 2},
									{Type: "add", Content: "SECOND", NewNum: 2},
								},
							},
						},
					},
					{
						OldName: "/dev/null",
						NewName: "b.txt",
						Status:  "added",
						Hunks: []Hunk{
							{
								OldStart: 0,
								OldLines: 0,
								NewStart: 1,
								NewLines: 1,
								Header:   "@@ -0,0 +1 @@",
								Lines: []Line{
									{Type: "add", Content: "new file content", NewNum: 1},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "binary file",
			input: `diff --git a/image.png b/image.png
new file mode 100644
index 0000000..1234567
Binary files /dev/null and b/image.png differ
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName:  "/dev/null",
						NewName:  "image.png",
						Status:   "added",
						IsBinary: true,
					},
				},
			},
		},
		{
			name: "binary file modification",
			input: `diff --git a/image.png b/image.png
index 1234567..abcdef0 100644
Binary files a/image.png and b/image.png differ
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName:  "image.png",
						NewName:  "image.png",
						Status:   "modified",
						IsBinary: true,
					},
				},
			},
		},
		{
			name: "hunk header with function context",
			input: `diff --git a/main.go b/main.go
index 1234567..abcdef0 100644
--- a/main.go
+++ b/main.go
@@ -10,6 +10,8 @@ func main() {
 	existing1()
 	existing2()
 	existing3()
+	newCall1()
+	newCall2()
 	existing4()
 	existing5()
 	existing6()
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "main.go",
						NewName: "main.go",
						Status:  "modified",
						Hunks: []Hunk{
							{
								OldStart: 10,
								OldLines: 6,
								NewStart: 10,
								NewLines: 8,
								Header:   "@@ -10,6 +10,8 @@ func main() {",
								Lines: []Line{
									{Type: "context", Content: "\texisting1()", OldNum: 10, NewNum: 10},
									{Type: "context", Content: "\texisting2()", OldNum: 11, NewNum: 11},
									{Type: "context", Content: "\texisting3()", OldNum: 12, NewNum: 12},
									{Type: "add", Content: "\tnewCall1()", NewNum: 13},
									{Type: "add", Content: "\tnewCall2()", NewNum: 14},
									{Type: "context", Content: "\texisting4()", OldNum: 13, NewNum: 15},
									{Type: "context", Content: "\texisting5()", OldNum: 14, NewNum: 16},
									{Type: "context", Content: "\texisting6()", OldNum: 15, NewNum: 17},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no newline at end of file",
			input: `diff --git a/hello.txt b/hello.txt
index 1234567..abcdef0 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1,2 +1,2 @@
 hello
-world
\ No newline at end of file
+world!
\ No newline at end of file
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "hello.txt",
						NewName: "hello.txt",
						Status:  "modified",
						Hunks: []Hunk{
							{
								OldStart: 1,
								OldLines: 2,
								NewStart: 1,
								NewLines: 2,
								Header:   "@@ -1,2 +1,2 @@",
								Lines: []Line{
									{Type: "context", Content: "hello", OldNum: 1, NewNum: 1},
									{Type: "delete", Content: "world", OldNum: 2},
									{Type: "add", Content: "world!", NewNum: 2},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple hunks in one file",
			input: `diff --git a/main.go b/main.go
index 1234567..abcdef0 100644
--- a/main.go
+++ b/main.go
@@ -1,4 +1,4 @@
 package main
 
-import "fmt"
+import "log"
 
@@ -10,4 +10,4 @@ func main() {
 	x := 1
 	y := 2
-	fmt.Println(x + y)
+	log.Println(x + y)
 }
`,
			expected: &DiffResult{
				Files: []FileDiff{
					{
						OldName: "main.go",
						NewName: "main.go",
						Status:  "modified",
						Hunks: []Hunk{
							{
								OldStart: 1,
								OldLines: 4,
								NewStart: 1,
								NewLines: 4,
								Header:   "@@ -1,4 +1,4 @@",
								Lines: []Line{
									{Type: "context", Content: "package main", OldNum: 1, NewNum: 1},
									{Type: "context", Content: "", OldNum: 2, NewNum: 2},
									{Type: "delete", Content: "import \"fmt\"", OldNum: 3},
									{Type: "add", Content: "import \"log\"", NewNum: 3},
									{Type: "context", Content: "", OldNum: 4, NewNum: 4},
								},
							},
							{
								OldStart: 10,
								OldLines: 4,
								NewStart: 10,
								NewLines: 4,
								Header:   "@@ -10,4 +10,4 @@ func main() {",
								Lines: []Line{
									{Type: "context", Content: "\tx := 1", OldNum: 10, NewNum: 10},
									{Type: "context", Content: "\ty := 2", OldNum: 11, NewNum: 11},
									{Type: "delete", Content: "\tfmt.Println(x + y)", OldNum: 12},
									{Type: "add", Content: "\tlog.Println(x + y)", NewNum: 12},
									{Type: "context", Content: "}", OldNum: 13, NewNum: 13},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() returned error: %v", err)
			}

			// Compare file count
			if len(result.Files) != len(tt.expected.Files) {
				t.Fatalf("got %d files, want %d files", len(result.Files), len(tt.expected.Files))
			}

			for i, gotFile := range result.Files {
				wantFile := tt.expected.Files[i]
				if gotFile.OldName != wantFile.OldName {
					t.Errorf("file[%d].OldName = %q, want %q", i, gotFile.OldName, wantFile.OldName)
				}
				if gotFile.NewName != wantFile.NewName {
					t.Errorf("file[%d].NewName = %q, want %q", i, gotFile.NewName, wantFile.NewName)
				}
				if gotFile.Status != wantFile.Status {
					t.Errorf("file[%d].Status = %q, want %q", i, gotFile.Status, wantFile.Status)
				}
				if gotFile.IsBinary != wantFile.IsBinary {
					t.Errorf("file[%d].IsBinary = %v, want %v", i, gotFile.IsBinary, wantFile.IsBinary)
				}

				// Compare hunks
				if len(gotFile.Hunks) != len(wantFile.Hunks) {
					t.Fatalf("file[%d] got %d hunks, want %d hunks", i, len(gotFile.Hunks), len(wantFile.Hunks))
				}

				for j, gotHunk := range gotFile.Hunks {
					wantHunk := wantFile.Hunks[j]
					if gotHunk.OldStart != wantHunk.OldStart {
						t.Errorf("file[%d].hunk[%d].OldStart = %d, want %d", i, j, gotHunk.OldStart, wantHunk.OldStart)
					}
					if gotHunk.OldLines != wantHunk.OldLines {
						t.Errorf("file[%d].hunk[%d].OldLines = %d, want %d", i, j, gotHunk.OldLines, wantHunk.OldLines)
					}
					if gotHunk.NewStart != wantHunk.NewStart {
						t.Errorf("file[%d].hunk[%d].NewStart = %d, want %d", i, j, gotHunk.NewStart, wantHunk.NewStart)
					}
					if gotHunk.NewLines != wantHunk.NewLines {
						t.Errorf("file[%d].hunk[%d].NewLines = %d, want %d", i, j, gotHunk.NewLines, wantHunk.NewLines)
					}
					if gotHunk.Header != wantHunk.Header {
						t.Errorf("file[%d].hunk[%d].Header = %q, want %q", i, j, gotHunk.Header, wantHunk.Header)
					}

					// Compare lines
					if len(gotHunk.Lines) != len(wantHunk.Lines) {
						t.Fatalf("file[%d].hunk[%d] got %d lines, want %d lines", i, j, len(gotHunk.Lines), len(wantHunk.Lines))
					}

					for k, gotLine := range gotHunk.Lines {
						wantLine := wantHunk.Lines[k]
						if gotLine.Type != wantLine.Type {
							t.Errorf("file[%d].hunk[%d].line[%d].Type = %q, want %q", i, j, k, gotLine.Type, wantLine.Type)
						}
						if gotLine.Content != wantLine.Content {
							t.Errorf("file[%d].hunk[%d].line[%d].Content = %q, want %q", i, j, k, gotLine.Content, wantLine.Content)
						}
						if gotLine.OldNum != wantLine.OldNum {
							t.Errorf("file[%d].hunk[%d].line[%d].OldNum = %d, want %d", i, j, k, gotLine.OldNum, wantLine.OldNum)
						}
						if gotLine.NewNum != wantLine.NewNum {
							t.Errorf("file[%d].hunk[%d].line[%d].NewNum = %d, want %d", i, j, k, gotLine.NewNum, wantLine.NewNum)
						}
					}
				}
			}
		})
	}
}
