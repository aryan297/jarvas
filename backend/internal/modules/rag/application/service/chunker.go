package service

import "strings"

// Chunk is a piece of text ready for embedding.
type Chunk struct {
	Content    string
	ChunkIndex int
	TokenCount int // rough estimate: len(content)/4
}

// ChunkText splits text into fixed-size overlapping chunks.
// chunkSize and overlap are in approximate token counts (1 token ≈ 4 chars).
func ChunkText(text string, chunkSize, overlap int) []Chunk {
	// Normalise whitespace
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	charChunk := chunkSize * 4
	charOverlap := overlap * 4

	// Split on sentence boundaries first for cleaner chunks.
	sentences := splitSentences(text)

	var chunks []Chunk
	var buf strings.Builder
	idx := 0

	for _, sent := range sentences {
		if buf.Len()+len(sent) > charChunk && buf.Len() > 0 {
			content := strings.TrimSpace(buf.String())
			if content != "" {
				chunks = append(chunks, Chunk{
					Content:    content,
					ChunkIndex: idx,
					TokenCount: len(content) / 4,
				})
				idx++
			}

			// Keep overlap: trim from the front
			overlap_text := buf.String()
			if len(overlap_text) > charOverlap {
				overlap_text = overlap_text[len(overlap_text)-charOverlap:]
			}
			buf.Reset()
			buf.WriteString(overlap_text)
		}
		buf.WriteString(sent)
		buf.WriteString(" ")
	}

	// Flush remaining
	if content := strings.TrimSpace(buf.String()); content != "" {
		chunks = append(chunks, Chunk{
			Content:    content,
			ChunkIndex: idx,
			TokenCount: len(content) / 4,
		})
	}

	return chunks
}

// splitSentences breaks text on common sentence-ending punctuation and double newlines.
func splitSentences(text string) []string {
	var sentences []string
	var buf strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		buf.WriteRune(r)

		nextIsSpace := i+1 < len(runes) && runes[i+1] == ' '
		nextIsNewline := i+1 < len(runes) && runes[i+1] == '\n'

		if (r == '.' || r == '!' || r == '?') && (nextIsSpace || nextIsNewline) {
			sentences = append(sentences, buf.String())
			buf.Reset()
			continue
		}
		if r == '\n' {
			// double newline = paragraph break
			if i+1 < len(runes) && runes[i+1] == '\n' {
				if s := strings.TrimSpace(buf.String()); s != "" {
					sentences = append(sentences, s)
				}
				buf.Reset()
			}
		}
	}
	if s := strings.TrimSpace(buf.String()); s != "" {
		sentences = append(sentences, s)
	}
	return sentences
}
