package service

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

//go:embed pinyin_data_generated.txt
var pinyinData string

var (
	pinyinMapOnce sync.Once
	pinyinMapData map[rune]string
	pinyinMapErr  error
)

func getPinyinMap() (map[rune]string, error) {
	pinyinMapOnce.Do(func() {
		pinyinMapData = make(map[rune]string, 20378)
		scanner := bufio.NewScanner(strings.NewReader(pinyinData))
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) != 2 {
				pinyinMapErr = fmt.Errorf("invalid pinyin data at line %d", lineNo)
				return
			}

			codepoint, err := strconv.ParseInt(fields[0], 16, 32)
			if err != nil {
				pinyinMapErr = fmt.Errorf("parse pinyin codepoint at line %d: %w", lineNo, err)
				return
			}
			pinyinMapData[rune(codepoint)] = fields[1]
		}
		if err := scanner.Err(); err != nil {
			pinyinMapErr = fmt.Errorf("scan pinyin data: %w", err)
		}
	})

	return pinyinMapData, pinyinMapErr
}

func convertNameToPinyin(name string) (string, error) {
	pinyinMap, err := getPinyinMap()
	if err != nil {
		return "", err
	}

	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", fmt.Errorf("name is empty")
	}

	var builder strings.Builder
	for _, r := range normalized {
		switch {
		case unicode.Is(unicode.Han, r):
			pinyin, ok := pinyinMap[r]
			if !ok || pinyin == "" {
				return "", fmt.Errorf("no pinyin mapping for %q", r)
			}
			builder.WriteString(pinyin)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
		case unicode.IsSpace(r):
			continue
		case isSkippableNameSeparator(r):
			continue
		default:
			return "", fmt.Errorf("unsupported character %q", r)
		}
	}

	result := builder.String()
	if result == "" {
		return "", fmt.Errorf("empty pinyin result")
	}
	return result, nil
}

func isSkippableNameSeparator(r rune) bool {
	switch r {
	case '·', '•', '・', '-', '_', '.', '．':
		return true
	default:
		return false
	}
}
