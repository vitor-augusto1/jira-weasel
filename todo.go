package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/vitor-augusto1/jira-weasel/pkg/assert"
)

type Todo struct {
	Prefix     string
	Keyword    string
	Priority   PrioritiesID
	Title      string
	Body       []string
	FilePath   string
	Line       uint64
	RemoteAddr string
	ReportedID *string
	Regex      string
}

type TodoTransformer func(Todo) error

func (td *Todo) LineHasTodoPrefix(line string) *string {
	if strings.HasPrefix(line, td.Prefix) {
		lineContent := strings.TrimPrefix(line, td.Prefix)
		return &lineContent
	}
	return nil
}

func (td *Todo) CommitTodoUpdate(commitMessage string) error {
	if td.ReportedID != nil {
		addCmd := exec.Command("git", "add", td.FilePath)
		err := addCmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't add changes: '%s'.\n", err)
			return err
		}
		commitCmd := exec.Command("git", "commit", "-m", commitMessage)
		err = commitCmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't commit changes: '%s'.\n", err)
			return err
		}
		return nil
	}
	return errors.New("Error commiting changes")
}

func (td *Todo) UpdatedTodoString(defaultStr string) string {
	if td.ReportedID != nil {
		updatedTodo := fmt.Sprintf(
			"%s %s P%s (%s): %s",
			td.Prefix,
			td.Keyword,
			td.Priority,
			*td.ReportedID,
			td.Title,
		)
		fmt.Println(">>> Updated todo content: ", updatedTodo)
		return updatedTodo
	}
	return defaultStr
}

func (td *Todo) ReturnTodoFirstLine() string {
	if td.ReportedID != nil {
		return fmt.Sprintf(
			"%s%s P%s (%s): %s",
			td.Prefix,
			td.Keyword,
			td.Priority,
			*td.ReportedID,
			td.Title,
		)
	}
	return fmt.Sprintf(
		"%s%s P%s: %s",
		td.Prefix,
		td.Keyword,
		td.Priority,
		td.Title,
	)
}

func (td *Todo) StringBody() string {
	return strings.Join(td.Body, "\n")
}

// Changes the Todo status in its line
func (td *Todo) ChangeTodoStatusToReported() error {
	fmt.Println("Changing the todo status...")
	tmpFileName := "/tmp/tmp-wasel.weasel"
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Can't create the tmp file: '%s'. Skipping for now.\n",
			err,
		)
		return err
	}
	defer tmpFile.Close()
	todoFile, err := os.Open(td.FilePath)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Can't open the Todo file: '%s'. Skipping for now.\n",
			err,
		)
		return err
	}
	defer todoFile.Close()
	todoFileInfo, _ := os.Stat(td.FilePath)
	scanner := bufio.NewScanner(todoFile)
	lnn := uint64(0)
	for scanner.Scan() {
		lnContent := scanner.Text()
		if td.Line == (lnn + 1) {
			fmt.Fprintln(tmpFile, td.UpdatedTodoString(lnContent))
		} else {
			fmt.Fprintln(tmpFile, lnContent)
		}
		lnn++
	}
	err = os.Chmod(tmpFileName, todoFileInfo.Mode())
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Can't set permissions: '%s'. Skipping for now.\n",
			err,
		)
		return err
	}
	err = os.Rename(tmpFileName, td.FilePath)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Can't rename the file: '%s'. Skipping for now.\n",
			err,
		)
		return err
	}
	return nil
}

func (td *Todo) SelfPurge() error {
	fmt.Println("Purging todo: ", *td.ReportedID)
	tmpFileName := "/tmp/tmp-weasel.weasel"
	tmpFile, err := os.Create(tmpFileName)
	assert.NoError(err, "Can't create the tmp file")
	defer tmpFile.Close()
	todoFile, err := os.Open(td.FilePath)
	assert.NoError(err, "Can't open the todo file", "td.FilePath", td.FilePath)
	defer todoFile.Close()
	todoFileInfo, _ := os.Stat(td.FilePath)
	scanner := bufio.NewScanner(todoFile)
	lnn := uint64(0)
	for scanner.Scan() {
		lnContent := scanner.Text()
		if td.Line == (lnn + 1) {
			continue
		}
    lineIsPartOfTheTodoBody := td.LineHasTodoPrefix(lnContent)
		if lineIsPartOfTheTodoBody != nil && lnn > td.Line {
			continue
		}
		fmt.Fprintln(tmpFile, lnContent)
		lnn++
	}
  err = os.Chmod(tmpFileName, todoFileInfo.Mode())
  assert.NoError(err, "Can't set permissions")
  err = os.Rename(tmpFileName, td.FilePath)
  assert.NoError(err, "Can't change file name")
	return nil
}

func (td *Todo) PrintCurrentStatus() {
	if td.ReportedID != nil {
		fmt.Fprintf(
			os.Stdout,
			" [%s] %s\n [%s] [%s]\n\n",
			colors.Success(*td.ReportedID),
			strings.TrimLeft(td.ReturnTodoFirstLine(), td.Prefix),
			colors.Info(td.FilePath),
			colors.Remote(td.RemoteAddr),
		)
		return
	}
	fmt.Fprintf(
		os.Stdout,
		" [%s] %s\n [%s] [%s]\n\n",
		colors.Error("UNREPORTED"),
		strings.TrimLeft(td.ReturnTodoFirstLine(), td.Prefix),
		colors.Info(td.FilePath),
		colors.Remote(td.RemoteAddr),
	)
}
