package handler

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func filterSkillPaths(skills []string) []string {
	if len(skills) == 0 {
		return skills
	}

	cwd, _ := os.Getwd()
	forbidden := buildForbiddenSkillPaths(cwd)

	filtered := make([]string, 0, len(skills))
	for _, skillPath := range skills {
		if skillPath == "" {
			continue
		}
		if isSkillPathForbidden(skillPath, cwd, forbidden) {
			log.Printf("⚠️  Skills path blocked to protect secrets: %s", skillPath)
			continue
		}
		filtered = append(filtered, skillPath)
	}

	return filtered
}

func buildForbiddenSkillPaths(cwd string) []string {
	if cwd == "" {
		return nil
	}
	return []string{
		filepath.Join(cwd, "configs.json"),
		filepath.Join(cwd, "configs", "configs.json"),
	}
}

func isSkillPathForbidden(skillPath string, cwd string, forbidden []string) bool {
	if len(forbidden) == 0 {
		return false
	}

	cleaned := filepath.Clean(skillPath)
	absPath := cleaned
	if !filepath.IsAbs(cleaned) && cwd != "" {
		absPath = filepath.Join(cwd, cleaned)
	}
	absPath = filepath.Clean(absPath)

	info, err := os.Stat(absPath)
	isDir := err == nil && info.IsDir()

	for _, blocked := range forbidden {
		if absPath == blocked {
			return true
		}
		if isDir && isPathUnderDir(absPath, blocked) {
			return true
		}
	}

	return false
}

func isPathUnderDir(dir string, filePath string) bool {
	rel, err := filepath.Rel(dir, filePath)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
