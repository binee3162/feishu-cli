package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// imagePathPattern matches local file paths that look like images.
// Supports absolute paths, relative paths, and home-dir paths (~/).
var imagePathPattern = regexp.MustCompile(`(?i)(?:^|["\s:])(/[\w./-]+\.(?:png|jpe?g|gif|bmp|tiff?|webp))\b`)

// expandHomePath expands ~ or ~/ prefix to the user's home directory.
func expandHomePath(s string) string {
	if s == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return s
		}
		return home
	}
	if strings.HasPrefix(s, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return s
		}
		return filepath.Join(home, s[2:])
	}
	return s
}

// isLocalImagePath checks if a string looks like a local image file path
// and the file actually exists on disk. Supports ~ home directory expansion.
func isLocalImagePath(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Expand ~ to home directory
	expanded := expandHomePath(s)

	// Must have an image extension
	ext := strings.ToLower(filepath.Ext(expanded))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tiff", ".tif", ".webp":
		// ok
	default:
		return false
	}

	// Must exist on disk
	info, err := os.Stat(expanded)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// resolveImagePath resolves a potentially tilde-prefixed path to an absolute path.
// This is used when passing the path to UploadIMImage.
func resolveImagePath(s string) string {
	return expandHomePath(strings.TrimSpace(s))
}

// ReplaceLocalImagesInPost scans a post (rich text) message content JSON string,
// finds img tags with local file paths in the image_key field, uploads them,
// and replaces with the real image_key.
//
// Post format: {"zh_cn":{"title":"...","content":[[{"tag":"img","image_key":"/path/to/img.png",...}]]}}
func ReplaceLocalImagesInPost(contentJSON string) (string, error) {
	var postMsg map[string]interface{}
	if err := json.Unmarshal([]byte(contentJSON), &postMsg); err != nil {
		return contentJSON, nil // not valid JSON, return as-is
	}

	modified := false
	// Iterate over all language variants (zh_cn, en_us, etc.)
	for _, langVal := range postMsg {
		langMap, ok := langVal.(map[string]interface{})
		if !ok {
			continue
		}
		contentArr, ok := langMap["content"].([]interface{})
		if !ok {
			continue
		}
		for _, paragraph := range contentArr {
			elements, ok := paragraph.([]interface{})
			if !ok {
				continue
			}
			for _, elem := range elements {
				elemMap, ok := elem.(map[string]interface{})
				if !ok {
					continue
				}
				tag, _ := elemMap["tag"].(string)
				if tag != "img" {
					continue
				}
				imageKey, _ := elemMap["image_key"].(string)
				if imageKey == "" || !isLocalImagePath(imageKey) {
					continue
				}
				// Upload the local image (resolve ~ path)
				resolvedPath := resolveImagePath(imageKey)
				fmt.Fprintf(os.Stderr, "正在上传富文本图片: %s\n", filepath.Base(resolvedPath))
				newKey, err := UploadIMImage(resolvedPath)
				if err != nil {
					return "", fmt.Errorf("上传图片 %s 失败: %w", imageKey, err)
				}
				elemMap["image_key"] = newKey
				modified = true
			}
		}
	}

	if !modified {
		return contentJSON, nil
	}

	result, err := json.Marshal(postMsg)
	if err != nil {
		return "", fmt.Errorf("序列化消息内容失败: %w", err)
	}
	return string(result), nil
}

// ReplaceLocalImagesInCard scans an interactive (card) message content JSON string,
// finds img elements with local file paths in the img_key field, uploads them,
// and replaces with the real image_key.
//
// Card elements may contain:
// - {"tag":"img","img_key":"/path/to/img.png","alt":{...}}
// - Nested in elements arrays (including column_set → columns → elements)
func ReplaceLocalImagesInCard(contentJSON string) (string, error) {
	var cardMsg map[string]interface{}
	if err := json.Unmarshal([]byte(contentJSON), &cardMsg); err != nil {
		return contentJSON, nil // not valid JSON, return as-is
	}

	// Check if it's a template or card_id reference — no image replacement needed
	if t, ok := cardMsg["type"].(string); ok && (t == "template" || t == "card") {
		return contentJSON, nil
	}

	modified := false

	// Process v1 format: {"header":{...},"elements":[...]}
	if elements, ok := cardMsg["elements"].([]interface{}); ok {
		m, err := replaceImagesInElements(elements)
		if err != nil {
			return "", err
		}
		modified = modified || m
	}

	// Process v2 format: {"schema":"2.0","header":{...},"body":{"elements":[...]}}
	if body, ok := cardMsg["body"].(map[string]interface{}); ok {
		if elements, ok := body["elements"].([]interface{}); ok {
			m, err := replaceImagesInElements(elements)
			if err != nil {
				return "", err
			}
			modified = modified || m
		}
	}

	if !modified {
		return contentJSON, nil
	}

	result, err := json.Marshal(cardMsg)
	if err != nil {
		return "", fmt.Errorf("序列化卡片内容失败: %w", err)
	}
	return string(result), nil
}

// replaceImagesInElements recursively scans card elements for img tags
// with local file paths and uploads them.
func replaceImagesInElements(elements []interface{}) (bool, error) {
	modified := false
	for _, elem := range elements {
		elemMap, ok := elem.(map[string]interface{})
		if !ok {
			continue
		}
		tag, _ := elemMap["tag"].(string)

		// Handle img elements
		if tag == "img" {
			imgKey, _ := elemMap["img_key"].(string)
			if imgKey != "" && isLocalImagePath(imgKey) {
				resolvedPath := resolveImagePath(imgKey)
				fmt.Fprintf(os.Stderr, "正在上传卡片图片: %s\n", filepath.Base(resolvedPath))
				newKey, err := UploadIMImage(resolvedPath)
				if err != nil {
					return false, fmt.Errorf("上传图片 %s 失败: %w", imgKey, err)
				}
				elemMap["img_key"] = newKey
				modified = true
			}
		}

		// Recurse into nested elements
		if subElements, ok := elemMap["elements"].([]interface{}); ok {
			m, err := replaceImagesInElements(subElements)
			if err != nil {
				return false, err
			}
			modified = modified || m
		}

		// Recurse into column_set → columns → elements
		if columns, ok := elemMap["columns"].([]interface{}); ok {
			for _, col := range columns {
				colMap, ok := col.(map[string]interface{})
				if !ok {
					continue
				}
				if colElements, ok := colMap["elements"].([]interface{}); ok {
					m, err := replaceImagesInElements(colElements)
					if err != nil {
						return false, err
					}
					modified = modified || m
				}
			}
		}

		// Recurse into div → extra (extra can be an img)
		if extra, ok := elemMap["extra"].(map[string]interface{}); ok {
			extraTag, _ := extra["tag"].(string)
			if extraTag == "img" {
				imgKey, _ := extra["img_key"].(string)
				if imgKey != "" && isLocalImagePath(imgKey) {
					resolvedPath := resolveImagePath(imgKey)
					fmt.Fprintf(os.Stderr, "正在上传卡片图片: %s\n", filepath.Base(resolvedPath))
					newKey, err := UploadIMImage(resolvedPath)
					if err != nil {
						return false, fmt.Errorf("上传图片 %s 失败: %w", imgKey, err)
					}
					extra["img_key"] = newKey
					modified = true
				}
			}
		}

		// Recurse into actions → actions array
		if actions, ok := elemMap["actions"].([]interface{}); ok {
			m, err := replaceImagesInElements(actions)
			if err != nil {
				return false, err
			}
			modified = modified || m
		}
	}
	return modified, nil
}
