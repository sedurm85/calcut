//go:build js && wasm

package main

import (
	"regexp"
	"strconv"
	"strings"
	"syscall/js"
)

type Event struct {
	Text    string
	Summary string
}

type ParsedCalendar struct {
	HeaderLines []string
	Timezones   []string
	Events      []Event
}

func parseIcal(content string) ParsedCalendar {
	lines := strings.Split(content, "\n")

	var headerLines []string
	var timezones []string
	var events []Event

	var currentBlock []string
	blockType := ""
	nesting := 0

	for _, line := range lines {
		stripped := strings.TrimSpace(line)

		if stripped == "BEGIN:VCALENDAR" || stripped == "END:VCALENDAR" {
			continue
		}

		if strings.HasPrefix(stripped, "BEGIN:") && blockType == "" {
			blockType = strings.SplitN(stripped, ":", 2)[1]
			currentBlock = []string{line}
			nesting = 1
			continue
		}

		if blockType != "" {
			currentBlock = append(currentBlock, line)

			if strings.HasPrefix(stripped, "BEGIN:") {
				nesting++
			} else if strings.HasPrefix(stripped, "END:") {
				nesting--
			}

			if nesting == 0 {
				blockText := strings.Join(currentBlock, "\n")

				switch blockType {
				case "VTIMEZONE":
					timezones = append(timezones, blockText)
				case "VEVENT":
					events = append(events, Event{
						Text:    blockText,
						Summary: extractProperty(blockText, "SUMMARY"),
					})
				}

				blockType = ""
				currentBlock = nil
			}
			continue
		}

		if stripped != "" {
			headerLines = append(headerLines, line)
		}
	}

	return ParsedCalendar{
		HeaderLines: headerLines,
		Timezones:   timezones,
		Events:      events,
	}
}

func extractProperty(block, propName string) string {
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, propName+":") || strings.HasPrefix(line, propName+";") {
			idx := strings.Index(line, ":")
			if idx >= 0 {
				return strings.TrimSpace(line[idx+1:])
			}
		}
	}
	return ""
}

var unsafeChars = regexp.MustCompile(`[<>:"/\\|?*]`)
var multiUnderscore = regexp.MustCompile(`_+`)

func sanitizeFilename(name string) string {
	s := unsafeChars.ReplaceAllString(name, "")
	s = strings.ReplaceAll(s, " ", "_")
	s = multiUnderscore.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "untitled"
	}
	return s
}

func buildICS(headerLines, timezones []string, eventTexts []string) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\n")
	for _, h := range headerLines {
		b.WriteString(h)
		b.WriteByte('\n')
	}
	for _, tz := range timezones {
		b.WriteString(tz)
		b.WriteByte('\n')
	}
	for _, ev := range eventTexts {
		b.WriteString(ev)
		b.WriteByte('\n')
	}
	b.WriteString("END:VCALENDAR\n")
	return b.String()
}

func skeletonSize(headerLines, timezones []string) int {
	s := buildICS(headerLines, timezones, nil)
	return len(s)
}

func parseSize(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	suffixes := map[string]int64{
		"GB": 1024 * 1024 * 1024,
		"MB": 1024 * 1024,
		"KB": 1024,
		"G":  1024 * 1024 * 1024,
		"M":  1024 * 1024,
		"K":  1024,
	}
	for _, suffix := range []string{"GB", "MB", "KB", "G", "M", "K"} {
		if strings.HasSuffix(s, suffix) {
			numStr := s[:len(s)-len(suffix)]
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0
			}
			return int64(num * float64(suffixes[suffix]))
		}
	}
	val, _ := strconv.ParseInt(s, 10, 64)
	return val
}

type SplitResult struct {
	Filename string
	Content  string
	Events   int
	Size     int
}

func splitBySize(parsed ParsedCalendar, prefix string, maxBytes int64) []SplitResult {
	skelSize := int64(skeletonSize(parsed.HeaderLines, parsed.Timezones))
	var results []SplitResult
	var currentEvents []string
	currentSize := skelSize
	chunkIdx := 1
	tag := prefix
	if tag == "" {
		tag = "part"
	}

	flush := func() {
		filename := tag + "_" + padNumber(chunkIdx) + ".ics"
		content := buildICS(parsed.HeaderLines, parsed.Timezones, currentEvents)
		results = append(results, SplitResult{
			Filename: filename,
			Content:  content,
			Events:   len(currentEvents),
			Size:     len(content),
		})
		chunkIdx++
		currentEvents = nil
		currentSize = skelSize
	}

	for _, event := range parsed.Events {
		eventBytes := int64(len(event.Text)) + 1
		projected := currentSize + eventBytes

		if eventBytes+skelSize > maxBytes {
			if len(currentEvents) > 0 {
				flush()
			}
			currentEvents = []string{event.Text}
			flush()
			continue
		}

		if projected > maxBytes && len(currentEvents) > 0 {
			flush()
		}

		currentEvents = append(currentEvents, event.Text)
		currentSize += eventBytes
	}

	if len(currentEvents) > 0 {
		flush()
	}

	return results
}

func splitPerEvent(parsed ParsedCalendar, prefix string) []SplitResult {
	var results []SplitResult

	for i, event := range parsed.Events {
		idx := i + 1
		summaryPart := "event"
		if event.Summary != "" {
			summaryPart = sanitizeFilename(event.Summary)
		}

		var filename string
		if prefix != "" {
			filename = prefix + "_" + padNumber(idx) + "_" + summaryPart + ".ics"
		} else {
			filename = padNumber(idx) + "_" + summaryPart + ".ics"
		}

		content := buildICS(parsed.HeaderLines, parsed.Timezones, []string{event.Text})
		results = append(results, SplitResult{
			Filename: filename,
			Content:  content,
			Events:   1,
			Size:     len(content),
		})
	}
	return results
}

func padNumber(n int) string {
	return strings.Repeat("0", 3-len(strconv.Itoa(n))) + strconv.Itoa(n)
}

func splitIcalJS(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return js.ValueOf(map[string]interface{}{
			"error": "인자가 부족합니다 (content, options)",
		})
	}

	content := args[0].String()
	options := args[1]

	maxSize := options.Get("maxSize").String()
	prefix := options.Get("prefix").String()
	mode := options.Get("mode").String()

	parsed := parseIcal(content)

	if len(parsed.Events) == 0 {
		return js.ValueOf(map[string]interface{}{
			"error":  "이벤트가 없습니다",
			"events": 0,
		})
	}

	var results []SplitResult

	if mode == "size" && maxSize != "" {
		maxBytes := parseSize(maxSize)
		if maxBytes <= 0 {
			return js.ValueOf(map[string]interface{}{
				"error": "잘못된 크기 형식입니다",
			})
		}
		results = splitBySize(parsed, prefix, maxBytes)
	} else {
		results = splitPerEvent(parsed, prefix)
	}

	jsResults := make([]interface{}, len(results))
	for i, r := range results {
		jsResults[i] = map[string]interface{}{
			"filename": r.Filename,
			"content":  r.Content,
			"events":   r.Events,
			"size":     r.Size,
		}
	}

	return js.ValueOf(map[string]interface{}{
		"success":     true,
		"totalEvents": len(parsed.Events),
		"files":       jsResults,
	})
}

func getInfoJS(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.ValueOf(map[string]interface{}{
			"error": "파일 내용이 필요합니다",
		})
	}

	content := args[0].String()
	parsed := parseIcal(content)

	return js.ValueOf(map[string]interface{}{
		"events": len(parsed.Events),
		"size":   len(content),
	})
}

func main() {
	js.Global().Set("calcut", js.ValueOf(map[string]interface{}{
		"split":   js.FuncOf(splitIcalJS),
		"getInfo": js.FuncOf(getInfoJS),
		"version": "1.0.0",
		"name":    "CalCut",
	}))

	select {}
}
