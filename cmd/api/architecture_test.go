package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestPlatformImportsStayInCompositionRoots(t *testing.T) {
	const platform = "github.com/emount4/udav-back/internal/platform"

	project := filepath.Join("..", "..")
	platformDir := filepath.Join(project, "internal", "platform")
	fileSet := token.NewFileSet()
	for _, root := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(project, root), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == platformDir {
				return filepath.SkipDir
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}

			file, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			compositionRoot := root == "cmd" && entry.Name() == "main.go"
			for _, spec := range file.Imports {
				name, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return err
				}
				platformImport := name == platform || strings.HasPrefix(name, platform+"/")
				driverImport := name == "log/slog" || strings.HasPrefix(name, "github.com/jackc/pgx/")
				if (platformImport && !compositionRoot) || driverImport {
					t.Errorf("%s imports concrete dependency %s", path, name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
