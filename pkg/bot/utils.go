package bot

import (
	"strings"
	"unicode/utf16"
)

type Annotation struct {
	Tag     string
	Start   int
	End     int
	Length  int
	UOffset int
	Ulength int
}

type tagInfo struct {
	tagType  string
	startPos int
}

type TelegramMessage struct {
	Text        string
	Annotations []Annotation
}

// order matters
var allowedPrefixes = []string{
	"```",
	"**",
	"__",
	"~~",
	"`",
}

var llmSupportedPrefixes = map[string]string{
	"```": "pre",
	"**":  "bold",
	"__":  "italic",
	"~~":  "strike",
	"`":   "code",
}

func findPrefix(input string) (string, bool) {
	for _, v := range allowedPrefixes {
		if strings.HasPrefix(input, v) {
			return v, true
		}
	}
	return "", false
}

func insertUnparsed(input []byte, tag tagInfo) []byte {
	var b []byte
	var inserted bool
	for i := 0; i < len(input); i++ {
		if i == tag.startPos {
			inserted = true
			b = append(b, []byte(tag.tagType)...)
		}
		b = append(b, input[i])
	}
	if !inserted {
		b = append(b, []byte(tag.tagType)...)
	}
	return b
}

func parseMarkupInternal(input string) (string, []Annotation) {
	var annotations []Annotation
	var stack []tagInfo
	var plainText []byte
	i := 0
	for i < len(input) {
		val, ok := findPrefix(input[i:])
		if !ok {
			plainText = append(plainText, input[i])
			i++
			continue
		}

		if len(stack) == 0 {
			stack = append(stack, tagInfo{val, len(plainText)})
			i += len(val)
			continue
		}

		var littleStack []tagInfo
		var startPos int
		var found bool
		for j := len(stack); j > 0; j-- {
			if stack[j-1].tagType == val {
				found = true
				littleStack = stack[j:]
				startPos = stack[j-1].startPos
				stack = stack[:j-1]
				break
			}
		}

		for _, b := range littleStack {
			plainText = insertUnparsed(plainText, b)
		}

		if !found {
			stack = append(stack, tagInfo{val, len(plainText)})
			i += len(val)
			continue
		}

		utfOffset := len(utf16.Encode([]rune(string(plainText[:startPos]))))
		utfPlainTextLength := len(utf16.Encode([]rune(string(plainText))))
		annotations = append(annotations, Annotation{
			Tag:     val,
			Start:   startPos,
			End:     len(plainText),
			Length:  len(plainText) - startPos,
			UOffset: utfOffset,
			Ulength: utfPlainTextLength - utfOffset,
		})
		i += len(val)
	}

	return string(plainText), annotations
}

func prepareChunkEntities(chunk string, annotations []Annotation, start int) []Annotation {
	var res []Annotation
	chunkStart := start
	chunkEnd := len(chunk) + start

	var annotation Annotation
	for k := range annotations {
		annotation = annotations[k]
		if annotation.Start > chunkEnd {
			continue
		}
		if annotation.End < chunkStart {
			continue
		}

		if annotation.End > chunkEnd {
			annotation.End = chunkEnd
		}
		if annotation.Start < chunkStart {
			annotation.Start = chunkStart
		}
		annotation.Start -= chunkStart
		annotation.End -= chunkStart
		annotation.Length = annotation.End - annotation.Start

		utfOffset := len(utf16.Encode([]rune((chunk[:annotation.Start]))))
		utfPlainTextLength := len(utf16.Encode([]rune(chunk[:annotation.End])))

		annotation.UOffset = utfOffset
		annotation.Ulength = utfPlainTextLength - utfOffset
		res = append(res, annotation)
	}
	return res
}

func splitTextByNewline(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	var chunks []string
	startIndex := 0

	for startIndex < len(text) {
		endIndex := startIndex + maxSize
		if endIndex >= len(text) {
			chunks = append(chunks, text[startIndex:])
			break
		}
		searchSlice := text[startIndex:endIndex]
		lastNewlineIndex := strings.LastIndex(searchSlice, "\n")

		if lastNewlineIndex == -1 {
			nextNewlineIndex := strings.Index(text[endIndex:], "\n")
			if nextNewlineIndex == -1 {
				endIndex = len(text)
			} else {
				endIndex = endIndex + nextNewlineIndex + 1
			}
		} else {
			endIndex = startIndex + lastNewlineIndex + 1
		}

		chunks = append(chunks, text[startIndex:endIndex])
		startIndex = endIndex
	}

	return chunks
}

func prepareTelegramMessages(text string, annotations []Annotation) ([]TelegramMessage, error) {
	var messages []TelegramMessage
	start := 0
	chunks := splitTextByNewline(text, 3500)
	for _, chunk := range chunks {
		chunkAnnotation := prepareChunkEntities(chunk, annotations, start)
		messages = append(messages, TelegramMessage{
			Text:        chunk,
			Annotations: chunkAnnotation,
		})
		start += len(chunk)
	}

	return messages, nil
}
