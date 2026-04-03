package skills

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/snana7mi/conchtalk-dlc/relay"
)

const SkillsDir = ".conchtalk/skills"

// Load scans ~/.conchtalk/skills/ for .md files and parses frontmatter
func Load() []relay.SkillDefinition {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	dir := filepath.Join(home, SkillsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // directory doesn't exist, fine
	}

	var skills []relay.SkillDefinition
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, e.Name())
		skill, err := parseSkillFile(path)
		if err != nil {
			log.Printf("[skills] skipping %s: %v", e.Name(), err)
			continue
		}
		skills = append(skills, skill)
	}

	log.Printf("[skills] loaded %d skills from %s", len(skills), dir)
	return skills
}

func parseSkillFile(path string) (relay.SkillDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return relay.SkillDefinition{}, err
	}

	content := string(data)
	skill := relay.SkillDefinition{}

	// Parse YAML frontmatter between --- delimiters
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content[3:], "---", 2)
		if len(parts) == 2 {
			frontmatter := parts[0]
			skill.Content = strings.TrimSpace(parts[1])

			for _, line := range strings.Split(frontmatter, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "name:") {
					skill.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				} else if strings.HasPrefix(line, "displayName:") {
					skill.DisplayName = strings.TrimSpace(strings.TrimPrefix(line, "displayName:"))
				} else if strings.HasPrefix(line, "description:") {
					skill.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				}
			}
		}
	}

	if skill.Name == "" {
		skill.Name = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	if skill.Content == "" {
		skill.Content = content
	}

	return skill, nil
}
