package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantDir  string
		wantErr  bool
	}{
		{"normal date file", "2025-03-20.md", "2025/03", false},
		{"another date file", "2024-12-01.md", "2024/12", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := searchDir(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("searchDir(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.wantDir {
				t.Errorf("searchDir(%q) = %q, want %q", tt.input, got, tt.wantDir)
			}
		})
	}
}

func TestDefaultMemoDir(t *testing.T) {
	origMemoDir := os.Getenv("MEMO_DIR")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("MEMO_DIR", origMemoDir)
		os.Setenv("HOME", origHome)
	}()

	t.Run("default when MEMO_DIR is unset", func(t *testing.T) {
		os.Unsetenv("MEMO_DIR")
		os.Setenv("HOME", "/home/testuser")
		got := defaultMemoDir()
		want := "/home/testuser/Documents/memo"
		if got != want {
			t.Errorf("defaultMemoDir() = %q, want %q", got, want)
		}
	})

	t.Run("MEMO_DIR set to absolute path", func(t *testing.T) {
		os.Setenv("MEMO_DIR", "/tmp/mymemo")
		got := defaultMemoDir()
		want := "/tmp/mymemo"
		if got != want {
			t.Errorf("defaultMemoDir() = %q, want %q", got, want)
		}
	})

	t.Run("MEMO_DIR with tilde", func(t *testing.T) {
		os.Setenv("MEMO_DIR", "~/memos")
		os.Setenv("HOME", "/home/testuser")
		got := defaultMemoDir()
		want := "/home/testuser/memos"
		if got != want {
			t.Errorf("defaultMemoDir() = %q, want %q", got, want)
		}
	})

	t.Run("MEMO_DIR empty string falls back to default", func(t *testing.T) {
		os.Setenv("MEMO_DIR", "")
		os.Setenv("HOME", "/home/testuser")
		got := defaultMemoDir()
		want := "/home/testuser/Documents/memo"
		if got != want {
			t.Errorf("defaultMemoDir() = %q, want %q", got, want)
		}
	})
}

func TestFindPrevMemo(t *testing.T) {
	tmp := t.TempDir()

	// Create directory structure: tmp/2025/03/
	dir := filepath.Join(tmp, "2025", "03")
	os.MkdirAll(dir, 0755)

	os.WriteFile(filepath.Join(dir, "2025-03-18.md"), []byte("memo1"), 0644)
	os.WriteFile(filepath.Join(dir, "2025-03-19.md"), []byte("memo2"), 0644)

	t.Run("finds previous memo excluding today", func(t *testing.T) {
		prev, prevFile := findPrevMemo(tmp, "2025-03-19")
		if prev != "2025-03-18" {
			t.Errorf("prev = %q, want %q", prev, "2025-03-18")
		}
		wantFile := filepath.Join(dir, "2025-03-18.md")
		if prevFile != wantFile {
			t.Errorf("prevFile = %q, want %q", prevFile, wantFile)
		}
	})

	t.Run("returns latest when multiple exist", func(t *testing.T) {
		prev, _ := findPrevMemo(tmp, "2025-03-20")
		if prev != "2025-03-19" {
			t.Errorf("prev = %q, want %q", prev, "2025-03-19")
		}
	})

	t.Run("returns empty when no candidates", func(t *testing.T) {
		emptyDir := t.TempDir()
		prev, prevFile := findPrevMemo(emptyDir, "2025-03-20")
		if prev != "" || prevFile != "" {
			t.Errorf("expected empty, got prev=%q, prevFile=%q", prev, prevFile)
		}
	})
}

func TestCreateTodayMemo(t *testing.T) {
	tmp := t.TempDir()

	templatePath := filepath.Join(tmp, "template.md")
	os.WriteFile(templatePath, []byte("<[[]]  [[]]>\n# Daily Memo\n"), 0644)

	todayFile := filepath.Join(tmp, "2025-03-20.md")

	t.Run("creates memo from template with prev link", func(t *testing.T) {
		err := createTodayMemo(templatePath, todayFile, "2025-03-19")
		if err != nil {
			t.Fatalf("createTodayMemo() error = %v", err)
		}
		b, _ := os.ReadFile(todayFile)
		content := string(b)
		if want := "<[[2025-03-19]]  [[]]>\n# Daily Memo\n"; content != want {
			t.Errorf("content = %q, want %q", content, want)
		}
	})

	t.Run("creates memo with empty prev", func(t *testing.T) {
		todayFile2 := filepath.Join(tmp, "2025-03-21.md")
		err := createTodayMemo(templatePath, todayFile2, "")
		if err != nil {
			t.Fatalf("createTodayMemo() error = %v", err)
		}
		b, _ := os.ReadFile(todayFile2)
		content := string(b)
		if want := "<[[]]  [[]]>\n# Daily Memo\n"; content != want {
			t.Errorf("content = %q, want %q", content, want)
		}
	})

	t.Run("error when template missing", func(t *testing.T) {
		err := createTodayMemo("/nonexistent/template.md", filepath.Join(tmp, "x.md"), "")
		if err == nil {
			t.Error("expected error for missing template")
		}
	})
}

func TestUpdatePrevMemo(t *testing.T) {
	t.Run("replaces [[]]> placeholder", func(t *testing.T) {
		tmp := t.TempDir()
		prevFile := filepath.Join(tmp, "prev.md")
		os.WriteFile(prevFile, []byte("<[[2025-03-18]]  [[]]>\n# Memo\n"), 0644)

		err := updatePrevMemo(prevFile, "2025-03-20")
		if err != nil {
			t.Fatalf("updatePrevMemo() error = %v", err)
		}
		b, _ := os.ReadFile(prevFile)
		content := string(b)
		want := "<[[2025-03-18]]  [[2025-03-20]]>\n# Memo\n"
		if content != want {
			t.Errorf("content = %q, want %q", content, want)
		}
	})

	t.Run("appends link when no placeholder and no existing link", func(t *testing.T) {
		tmp := t.TempDir()
		prevFile := filepath.Join(tmp, "prev.md")
		os.WriteFile(prevFile, []byte("# Memo\n"), 0644)

		err := updatePrevMemo(prevFile, "2025-03-20")
		if err != nil {
			t.Fatalf("updatePrevMemo() error = %v", err)
		}
		b, _ := os.ReadFile(prevFile)
		content := string(b)
		want := "# Memo\n[[2025-03-20]]>\n"
		if content != want {
			t.Errorf("content = %q, want %q", content, want)
		}
	})

	t.Run("does not duplicate existing link", func(t *testing.T) {
		tmp := t.TempDir()
		prevFile := filepath.Join(tmp, "prev.md")
		os.WriteFile(prevFile, []byte("[[2025-03-20]]>\n"), 0644)

		err := updatePrevMemo(prevFile, "2025-03-20")
		if err != nil {
			t.Fatalf("updatePrevMemo() error = %v", err)
		}
		b, _ := os.ReadFile(prevFile)
		content := string(b)
		want := "[[2025-03-20]]>\n"
		if content != want {
			t.Errorf("content = %q, want %q", content, want)
		}
	})

	t.Run("error when file missing", func(t *testing.T) {
		err := updatePrevMemo("/nonexistent/file.md", "2025-03-20")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

func TestMemoFilePattern(t *testing.T) {
	tests := []struct {
		path  string
		match bool
	}{
		{"/memo/2025/03/2025-03-20.md", true},
		{"/memo/2024/12/2024-12-01.md", true},
		{"/memo/template.md", false},
		{"/memo/2025/03/notes.md", false},
		{"/memo/2025/03/2025-03-20.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := memoFilePattern.MatchString(tt.path)
			if got != tt.match {
				t.Errorf("memoFilePattern.MatchString(%q) = %v, want %v", tt.path, got, tt.match)
			}
		})
	}
}
