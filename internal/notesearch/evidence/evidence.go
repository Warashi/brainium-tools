// Package evidence extracts literal, source-addressable passages from an Org document.
package evidence

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Warashi/brainium-tools/internal/notesearch/tokenize"
)

const (
	maxSnippetLines = 12
)

// Metadata is file-level metadata. Missing values remain empty.
type Metadata struct {
	ID         string
	Title      string
	Status     string
	Verified   string
	StaleAfter string
}

// Snippet is an exact range of source lines and its Org heading ancestry.
type Snippet struct {
	StartLine int
	EndLine   int
	Headings  []string
	Text      string
	Kind      string
}

// Document contains file metadata and selected source evidence.
type Document struct {
	Metadata Metadata
	Snippets []Snippet
}

type sourceLine struct {
	text       string
	start, end int
	number     int
	headings   []string
	selectable bool
	inBlock    bool
}

type passage struct {
	first, last int
	headings    []string
	score       relevanceScore
}

type relevanceScore struct {
	full     int
	coverage int
	rarity   int
}

type queryComponent struct {
	terms  []string
	maxLen int
}

// Extract parses only lightweight Org structure, then ranks paragraphs by overlap with
// the shared notesearch tokenizer. Relevance first compares how many lexical query
// segments are fully represented, then their normalized token coverage; a long
// word therefore cannot count as many independent facts. Body and heading tokens form one
// evidence set rather than receiving separate rewards. Ties prefer components that occur
// in fewer document locations, then source order. Extract returns the best match followed by up to
// 12 lines of complete opening paragraphs, or by the second match when that context
// overlaps the best. With no body match (including filename-only upstream matches), the
// opening is explicitly marked as context. Text is sliced from content, never reconstructed.
func Extract(content, query string) Document {
	if content == "" {
		return Document{}
	}

	lines := splitLines(content)
	doc := Document{Metadata: extractMetadata(lines)}
	parseStructure(lines)
	passages := makePassages(lines)
	if len(passages) == 0 {
		return doc
	}

	components := makeQueryComponents(query)
	coverages := make([][]int, len(passages))
	fullFreq := make([]int, len(components))
	for i := range passages {
		p := &passages[i]
		body := sourceRange(content, lines[p.first], lines[p.last])
		available := uniqueTokens(body)
		for token := range uniqueTokens(strings.Join(p.headings, "\n")) {
			available[token] = struct{}{}
		}
		coverages[i] = componentCoverages(components, available)
		for component, coverage := range coverages[i] {
			if coverage == 1000 {
				fullFreq[component]++
			}
		}
	}
	titleCoverage := componentCoverages(components, uniqueTokens(doc.Metadata.Title))
	for component, coverage := range titleCoverage {
		if coverage == 1000 {
			fullFreq[component]++
		}
	}
	for i := range passages {
		passages[i].score = scoreCoverage(coverages[i], fullFreq)
	}

	ranked := slices.Clone(passages)
	slices.SortFunc(ranked, func(a, b passage) int {
		if c := compareScore(b.score, a.score); c != 0 {
			return c
		}
		return cmp.Compare(a.first, b.first)
	})

	opening := openingPassage(lines, passages)
	type selection struct {
		passage
		kind string
	}
	var selected []selection
	if ranked[0].score.coverage == 0 {
		selected = append(selected, selection{passage: opening, kind: "context"})
	} else {
		best := ranked[0]
		selected = append(selected, selection{passage: best, kind: "match"})
		if !overlaps(best, opening) {
			selected = append(selected, selection{passage: opening, kind: "context"})
		} else {
			for _, candidate := range ranked[1:] {
				if candidate.score.coverage == 0 {
					break
				}
				if !overlaps(candidate, best) &&
					sourceRange(content, lines[candidate.first], lines[candidate.last]) !=
						sourceRange(content, lines[best.first], lines[best.last]) {
					selected = append(selected, selection{passage: candidate, kind: "match"})
					break
				}
			}
		}
	}

	for _, selected := range selected {
		p := selected.passage
		doc.Snippets = append(doc.Snippets, Snippet{
			StartLine: lines[p.first].number,
			EndLine:   lines[p.last].number,
			Headings:  slices.Clone(p.headings),
			Text:      sourceRange(content, lines[p.first], lines[p.last]),
			Kind:      selected.kind,
		})
	}

	return doc
}

// ParseMetadata returns only the file-level metadata, without tokenizing the
// body as Extract does.
func ParseMetadata(content string) Metadata {
	return extractMetadata(splitLines(content))
}

func splitLines(content string) []sourceLine {
	lines := make([]sourceLine, 0, strings.Count(content, "\n")+1)
	for start, number := 0, 1; start < len(content); number++ {
		newline := strings.IndexByte(content[start:], '\n')
		next := len(content)
		end := next
		if newline >= 0 {
			next = start + newline + 1
			end = next - 1
			if end > start && content[end-1] == '\r' {
				end--
			}
		}
		lines = append(lines, sourceLine{
			text: content[start:end], start: start, end: end, number: number,
		})
		start = next
	}
	return lines
}

func extractMetadata(lines []sourceLine) Metadata {
	metadata := leadingProperties(lines)
	var blockType string
	inDrawer := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line.text)
		if _, _, ok := heading(line.text); ok {
			break
		}
		if blockType != "" {
			if endType, ok := blockEndType(trimmed); ok && endType == blockType {
				blockType = ""
			}
			continue
		}
		if beginType, ok := blockBeginType(trimmed); ok {
			blockType = beginType
			continue
		}
		if inDrawer {
			if strings.EqualFold(trimmed, ":END:") {
				inDrawer = false
			}
			continue
		}
		if isDrawerBegin(trimmed) {
			inDrawer = true
			continue
		}
		if key, value, ok := keyword(trimmed); ok && strings.EqualFold(key, "title") {
			metadata.Title = value
			break
		}
	}
	return metadata
}

func leadingProperties(lines []sourceLine) Metadata {
	var pending Metadata
	first := 0
	for first < len(lines) && strings.TrimSpace(lines[first].text) == "" {
		first++
	}
	if first >= len(lines) || !strings.EqualFold(strings.TrimSpace(lines[first].text), ":PROPERTIES:") {
		return Metadata{}
	}
	for i := first + 1; i < len(lines); i++ {
		text := strings.TrimSpace(lines[i].text)
		if strings.EqualFold(text, ":END:") {
			return pending
		}
		if _, _, ok := heading(lines[i].text); ok {
			return Metadata{}
		}
		key, value, ok := property(text)
		if !ok {
			continue
		}
		switch strings.ToUpper(key) {
		case "ID":
			pending.ID = value
		case "STATUS":
			pending.Status = value
		case "VERIFIED":
			pending.Verified = value
		case "STALE_AFTER":
			pending.StaleAfter = value
		}
	}
	return Metadata{}
}

func parseStructure(lines []sourceLine) {
	var headings []string
	var blockType string
	inDrawer := false
	for i := range lines {
		line := &lines[i]
		trimmed := strings.TrimSpace(line.text)

		// Org blocks cannot span a headline, even when an end marker follows it.
		if level, title, ok := heading(line.text); ok {
			blockType, inDrawer = "", false
			if level <= len(headings) {
				headings = headings[:level-1]
			}
			for len(headings) < level-1 {
				headings = append(headings, "")
			}
			headings = append(headings, title)
			continue
		}
		if blockType != "" {
			if endType, ok := blockEndType(trimmed); ok && endType == blockType {
				blockType = ""
				continue
			}
			line.headings = slices.Clone(headings)
			line.selectable = trimmed != ""
			line.inBlock = true
			continue
		}
		if beginType, ok := blockBeginType(trimmed); ok {
			blockType = beginType
			continue
		}
		if inDrawer {
			if strings.EqualFold(trimmed, ":END:") {
				inDrawer = false
			}
			continue
		}
		if isDrawerBegin(trimmed) {
			inDrawer = true
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#+") {
			continue
		}
		line.headings = slices.Clone(headings)
		line.selectable = true
	}
}

func makePassages(lines []sourceLine) []passage {
	var out []passage
	for i := 0; i < len(lines); {
		if !lines[i].selectable {
			i++
			continue
		}
		start := i
		headings := lines[i].headings
		for i+1 < len(lines) && lines[i+1].selectable &&
			slices.Equal(lines[i+1].headings, headings) {
			i++
		}
		end := i
		for chunk := start; chunk <= end; chunk += maxSnippetLines {
			last := min(chunk+maxSnippetLines-1, end)
			out = append(out, passage{first: chunk, last: last, headings: slices.Clone(headings)})
		}
		i++
	}
	return out
}

func openingPassage(lines []sourceLine, passages []passage) passage {
	firstPassage := -1
	for i, candidate := range passages {
		if !lines[candidate.first].inBlock && len(candidate.headings) == 0 {
			firstPassage = i
			break
		}
	}
	if firstPassage < 0 {
		for _, candidate := range passages {
			if !lines[candidate.first].inBlock {
				return candidate
			}
		}
		return passages[0]
	}
	opening := passages[firstPassage]
	for _, candidate := range passages[firstPassage+1:] {
		if len(candidate.headings) > 0 || lines[candidate.first].inBlock {
			break
		}
		if candidate.last-opening.first+1 > maxSnippetLines {
			break
		}
		opening.last = candidate.last
	}
	return opening
}

func sourceRange(content string, first, last sourceLine) string {
	return content[first.start:last.end]
}

func makeQueryComponents(query string) []queryComponent {
	groups := tokenize.Groups(query)
	components := make([]queryComponent, 0, len(groups))
	for _, group := range groups {
		maxLen := 0
		for _, term := range group {
			maxLen = max(maxLen, utf8.RuneCountInString(term))
		}
		if maxLen == 0 {
			continue
		}
		components = append(components, queryComponent{terms: group, maxLen: maxLen})
	}
	return components
}

func componentCoverages(components []queryComponent, available map[string]struct{}) []int {
	coverages := make([]int, len(components))
	for i, component := range components {
		matchedMax, totalMax, longestMatched := 0, 0, 0
		for _, term := range component.terms {
			n := utf8.RuneCountInString(term)
			if n == component.maxLen {
				totalMax++
			}
			if _, ok := available[term]; !ok {
				continue
			}
			if component.maxLen > 1 && n == 1 {
				continue
			}
			longestMatched = max(longestMatched, n)
			if n == component.maxLen {
				matchedMax++
			}
		}
		if matchedMax > 0 {
			coverages[i] = matchedMax * 1000 / totalMax
		} else if longestMatched > 0 {
			coverages[i] = longestMatched * 1000 / component.maxLen
		}
	}
	return coverages
}

func scoreCoverage(coverages, fullFreq []int) relevanceScore {
	var score relevanceScore
	for i, coverage := range coverages {
		score.coverage += coverage
		if coverage == 1000 {
			score.full++
			score.rarity += 1_000_000 / fullFreq[i]
		}
	}
	return score
}

func compareScore(a, b relevanceScore) int {
	if c := cmp.Compare(a.full, b.full); c != 0 {
		return c
	}
	if c := cmp.Compare(a.coverage, b.coverage); c != 0 {
		return c
	}
	return cmp.Compare(a.rarity, b.rarity)
}

func uniqueTokens(text string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, token := range tokenize.Tokens(text) {
		out[token] = struct{}{}
	}
	return out
}

func property(line string) (key, value string, ok bool) {
	if len(line) < 3 || line[0] != ':' {
		return "", "", false
	}
	end := strings.IndexByte(line[1:], ':')
	if end < 0 {
		return "", "", false
	}
	end++
	key = strings.TrimSpace(line[1:end])
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(line[end+1:]), true
}

func keyword(line string) (key, value string, ok bool) {
	if !strings.HasPrefix(line, "#+") {
		return "", "", false
	}
	end := strings.IndexByte(line[2:], ':')
	if end < 0 {
		return "", "", false
	}
	end += 2
	return strings.TrimSpace(line[2:end]), strings.TrimSpace(line[end+1:]), true
}

func heading(line string) (level int, title string, ok bool) {
	for level < len(line) && line[level] == '*' {
		level++
	}
	if level == 0 || level >= len(line) || (line[level] != ' ' && line[level] != '\t') {
		return 0, "", false
	}
	return level, strings.TrimSpace(line[level:]), true
}

func blockBeginType(line string) (string, bool) {
	const prefix = "#+begin_"
	lower := strings.ToLower(line)
	if !strings.HasPrefix(lower, prefix) {
		return "", false
	}
	name := strings.Fields(lower[len(prefix):])
	if len(name) == 0 {
		return "", false
	}
	return name[0], true
}

func blockEndType(line string) (string, bool) {
	const prefix = "#+end_"
	lower := strings.ToLower(line)
	if !strings.HasPrefix(lower, prefix) {
		return "", false
	}
	name := strings.TrimSpace(lower[len(prefix):])
	if name == "" || strings.ContainsAny(name, " \t") {
		return "", false
	}
	return name, true
}

func isDrawerBegin(line string) bool {
	if len(line) < 3 || line[0] != ':' || line[len(line)-1] != ':' ||
		strings.EqualFold(line, ":END:") {
		return false
	}
	for _, r := range line[1 : len(line)-1] {
		if !(r == '_' || r == '-' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' ||
			r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func overlaps(a, b passage) bool {
	return a.first <= b.last && b.first <= a.last
}
