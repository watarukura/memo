package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		help()
		os.Exit(1)
	}
}

func run() error {
	memoDir := defaultMemoDir()

	args := os.Args[1:]

	if len(args) == 1 {
		cmd := args[0]
		switch cmd {
		case "list":
			listMemo(memoDir)
			return nil
		case "cd":
			// Launch a subshell with cwd set to todoDir (chezmoi-like behavior).
			runSubshell(memoDir)
			return nil
		case "help":
		case "-h":
		case "-help":
		case "--help":
			help()
			return nil
		default:
			// ファイルが存在するか確認
			fileDir, err := searchDir(cmd + ".md")
			if err != nil {
				log.Fatalf("Can not searchDir %v\n", err)
			}
			filePath := filepath.Join(memoDir, fileDir, cmd+".md")
			if _, err := os.Stat(filePath); err == nil {
				return openEditor(filePath)
			}
		}
	} else if len(args) > 1 {
		help()
		log.Fatalf("Too many args")
	}

	templatePath := filepath.Join(memoDir, "template.md")
	now := time.Now()
	today := now.Format("2006-01-02")
	todayDir, _ := searchDir(today)
	todayFile := filepath.Join(memoDir, todayDir, today+".md")

	prev, prevFile := findPrevMemo(memoDir, today)

	if _, err := os.Stat(todayFile); os.IsNotExist(err) {
		if err := createTodayMemo(templatePath, todayFile, prev); err != nil {
			return err
		}
	}

	if prevFile != "" {
		if err := updatePrevMemo(prevFile, today); err != nil {
			return err
		}
	}

	return openEditor(todayFile)
}

func searchDir(memoFile string) (string, error) {
	splited := strings.Split(memoFile, "-")
	yyyy := splited[0]
	mm := splited[1]
	return yyyy + "/" + mm, nil
}

var memoFilePattern = regexp.MustCompile(`/\d{4}/\d{2}/\d{4}-\d{2}-\d{2}\.md$`)

func findPrevMemo(memoDir, today string) (string, string) {
	var candidates []string

	_ = filepath.Walk(memoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !memoFilePattern.MatchString(filepath.ToSlash(path)) {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), ".md")
		if base == today {
			return nil
		}
		candidates = append(candidates, path)
		return nil
	})

	if len(candidates) == 0 {
		return "", ""
	}

	sort.Strings(candidates)
	prevFile := candidates[len(candidates)-1]
	prev := strings.TrimSuffix(filepath.Base(prevFile), ".md")
	return prev, prevFile
}

func createTodayMemo(templatePath, todayFile, prev string) error {
	b, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	content := string(b)
	content = strings.ReplaceAll(content, "<[]()", "<["+prev+"]("+prev+")")

	if err := os.WriteFile(todayFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write today's memo: %w", err)
	}

	return nil
}

func updatePrevMemo(prevFile, today string) error {
	b, err := os.ReadFile(prevFile)
	if err != nil {
		return fmt.Errorf("failed to read previous memo: %w", err)
	}

	content := string(b)
	newLink := "[" + today + "](" + today + ")>"

	if strings.Contains(content, "[]()>") {
		content = strings.Replace(content, "[]()>", newLink, 1)
	} else if !strings.Contains(content, newLink) {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += newLink + "\n"
	}

	if err := os.WriteFile(prevFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to update previous memo: %w", err)
	}

	return nil
}

func defaultMemoDir() string {
	if envDir, ok := os.LookupEnv("MEMO_DIR"); ok && envDir != "" {
		if strings.HasPrefix(envDir, "~") {
			home := os.Getenv("HOME")
			if home != "" {
				return filepath.Clean(filepath.Join(home, strings.TrimPrefix(envDir, "~")))
			}
		}
		return filepath.Clean(envDir)
	}
	return filepath.Join(os.Getenv("HOME"), "Documents/memo")
}

func help() {
	fmt.Println("Usage: memo [command] [args...]")
	fmt.Println("Commands:")
	fmt.Println("  list    List all memo")
	fmt.Println("  cd      Launch a subshell with cwd set to memoDir")
	fmt.Println("  [YYYY-mm-dd] Open memo")
	fmt.Println("Env:")
	fmt.Println("  MEMO_DIR    Override the memo directory (default: ~/Documents/memo)")
}

func listMemo(memoDir string) {
	files, err := filepath.Glob(filepath.Join(memoDir, "2*", "*", "*.md"))
	if err != nil {
		log.Fatalf("Error listing files: %v\n", err)
	}

	if len(files) == 0 {
		return
	}

	var filenames []string
	for _, f := range files {
		dir, _ := searchDir(filepath.Base(f))
		filenames = append(filenames, filepath.Join(dir, filepath.Base(f)))
	}

	selected := runFzf(strings.Join(filenames, "\n"), memoDir)
	if selected != "" {
		if err := openEditor(filepath.Join(memoDir, selected)); err != nil {
			log.Fatalf("Error opening editor: %v\n", err)
		}
	}
}

func runFzf(input string, memoDir string) string {
	cmd := exec.Command("fzf", "--preview", "fzf-preview.sh {}")
	cmd.Dir = memoDir
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func runSubshell(dir string) {
	// Spawn the user's shell as a subshell in the target directory.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.Command(shell)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
