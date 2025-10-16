package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runCheckSchemas() error {
    var issues []string

    err := filepath.Walk("schema", func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
            return nil
        }
        content, readErr := os.ReadFile(path)
        if readErr != nil {
            return readErr
        }

        text := string(content)
        if !strings.Contains(text, "\"$schema\"") {
            issues = append(issues, fmt.Sprintf("missing $schema in %s", path))
        }
        if !strings.Contains(text, "\"$id\"") {
            issues = append(issues, fmt.Sprintf("missing $id in %s", path))
        }
        return nil
    })

    if err != nil {
        return err
    }

    if len(issues) > 0 {
        return errors.New(strings.Join(issues, "\n"))
    }

    fmt.Println("All schema files contain $schema and $id")
    return nil
}


