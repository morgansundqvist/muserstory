package domain

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type MarkdownFile struct {
	Metadata map[string]interface{} `yaml:"metadata"`
	Summary  string
	Stories  []UserStory
}

func ParseMarkdownFileContent(content string) (*MarkdownFile, error) {
	var metadata map[string]interface{}
	var summaryBuilder strings.Builder
	var stories []UserStory
	var storyContentLines []string

	scanner := bufio.NewScanner(strings.NewReader(content))
	isReadingSummary := false
	isReadingMetadata := false
	var metadataBuffer bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "---" {
			if !isReadingMetadata {
				isReadingMetadata = true
				continue
			} else {
				isReadingMetadata = false
				if err := yaml.Unmarshal(metadataBuffer.Bytes(), &metadata); err != nil {
					return nil, fmt.Errorf("error parsing YAML metadata: %w", err)
				}
				continue
			}
		}

		if isReadingMetadata {
			metadataBuffer.WriteString(line + "\n")
			continue
		}

		if strings.HasPrefix(trimmedLine, "# Summary") {
			isReadingSummary = true
			continue
		}

		if isReadingSummary {
			if strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "**") {
				isReadingSummary = false
				storyContentLines = append(storyContentLines, line)
			} else {
				if summaryBuilder.Len() > 0 {
					summaryBuilder.WriteString("\n")
				}
				summaryBuilder.WriteString(line)
			}
		} else {
			storyContentLines = append(storyContentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	lineNumber := 0
	for _, line := range storyContentLines {
		lineNumber++
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "- ") {
			content := strings.TrimPrefix(trimmedLine, "- ")
			description := content
			category := "Uncategorized"

			catStartIndex := strings.LastIndex(content, "[Category: ")
			catEndIndex := strings.LastIndex(content, "]")

			if catStartIndex != -1 && catEndIndex != -1 && catEndIndex > catStartIndex && catEndIndex == len(content)-1 {
				description = strings.TrimSpace(content[:catStartIndex])
				category = content[catStartIndex+len("[Category: ") : catEndIndex]
			}

			stories = append(stories, UserStory{
				ID:          uuid.NewString(),
				Description: description,
				Category:    category,
			})
		}
	}

	return &MarkdownFile{
		Metadata: metadata,
		Summary:  strings.TrimSpace(summaryBuilder.String()),
		Stories:  stories,
	}, nil
}

func (m *MarkdownFile) WriteToFile(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening/creating file %s for writing: %w", filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	if len(m.Metadata) > 0 {
		if _, err := writer.WriteString("---\n"); err != nil {
			return fmt.Errorf("error writing metadata start: %w", err)
		}
		metadataBytes, err := yaml.Marshal(m.Metadata)
		if err != nil {
			return fmt.Errorf("error marshaling metadata: %w", err)
		}
		if _, err := writer.Write(metadataBytes); err != nil {
			return fmt.Errorf("error writing metadata: %w", err)
		}
		if _, err := writer.WriteString("---\n\n"); err != nil {
			return fmt.Errorf("error writing metadata end: %w", err)
		}
	}

	if m.Summary != "" {
		if _, err := writer.WriteString("# Summary\n"); err != nil {
			return fmt.Errorf("error writing summary header: %w", err)
		}
		if _, err := writer.WriteString(m.Summary + "\n\n"); err != nil {
			return fmt.Errorf("error writing summary content: %w", err)
		}
	}

	if len(m.Stories) > 0 {
		storiesByCategory := make(map[string][]UserStory)
		var categoryOrder []string

		for _, story := range m.Stories {
			cat := story.Category
			if cat == "" {
				cat = "Uncategorized"
			}
			if _, exists := storiesByCategory[cat]; !exists {
				categoryOrder = append(categoryOrder, cat)
			}
			storiesByCategory[cat] = append(storiesByCategory[cat], story)
		}

		for _, category := range categoryOrder {
			if _, err := writer.WriteString(fmt.Sprintf("**%s**\n", category)); err != nil {
				return fmt.Errorf("error writing category header: %w", err)
			}
			for _, story := range storiesByCategory[category] {
				line := fmt.Sprintf("- %s [Category: %s]\n", story.Description, story.Category)
				if _, err := writer.WriteString(line); err != nil {
					return fmt.Errorf("error writing story: %w", err)
				}
			}
			if _, err := writer.WriteString("\n"); err != nil {
				return fmt.Errorf("error writing newline: %w", err)
			}
		}
	}

	return writer.Flush()
}
