package worker

import (
	"bytes"
	"context"
	database "fgb-lp/database/queries"
	"fgb-lp/embeddings"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ledongthuc/pdf"
	"github.com/pgvector/pgvector-go"
)

const (
	chunkSize    = 800 // characters per chunk
	chunkOverlap = 100 // overlap between chunks
	batchSize    = 20  // number of chunks per embedding API call
	pollInterval = 2 * time.Second
	// maxDocProcessingTime caps how long a single document may spend in the
	// worker (extraction + embedding). Without it a stuck embedding service or
	// a pathological PDF can hold the worker indefinitely, starving every
	// other document behind it (addresses audit item H11's timeout portion).
	maxDocProcessingTime = 15 * time.Minute
)

type Worker struct {
	queries   *database.Queries
	embClient *embeddings.Client
	uploadDir string
}

func New(queries *database.Queries, uploadDir string) *Worker {
	return &Worker{
		queries:   queries,
		embClient: embeddings.NewClient(),
		uploadDir: uploadDir,
	}
}

// Start begins the worker loop, polling for pending documents.
func (w *Worker) Start(ctx context.Context) {
	log.Println("[worker] started, polling for pending documents")
	// Immediate first poll, then every pollInterval
	w.poll(ctx)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] shutting down")
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *Worker) poll(ctx context.Context) {
	pending, err := w.queries.GetPendingDocuments(ctx)
	if err != nil {
		log.Printf("[worker] error fetching pending documents: %v", err)
		return
	}

	for _, doc := range pending {
		claimed, err := w.queries.ClaimDocument(ctx, doc.ID)
		if err != nil {
			log.Printf("[worker] could not claim doc %d: %v", doc.ID, err)
			continue
		}
		log.Printf("[worker] processing document %d: %s", claimed.ID, claimed.Title)

		// Per-document deadline so one bad upload can't block the queue forever.
		docCtx, cancel := context.WithTimeout(ctx, maxDocProcessingTime)
		if err := w.processDocument(docCtx, claimed); err != nil {
			errMsg := err.Error()
			log.Printf("[worker] failed to process doc %d: %v", claimed.ID, err)
			w.queries.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
				ID:           claimed.ID,
				Status:       "failed",
				ErrorMessage: pgtype.Text{String: truncateStr(errMsg, 500), Valid: true},
			})
		}
		cancel()
	}
}

func (w *Worker) processDocument(ctx context.Context, doc database.Document) error {
	// Extract text from the uploaded document (PDF, TXT, or Markdown).
	text, err := ExtractDocumentText(filepath.Join(w.uploadDir, doc.FilePath))
	if err != nil {
		return fmt.Errorf("extract document text: %w", err)
	}

	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("no extractable text found in PDF")
	}

	// Sanitize the full extracted text before chunking
	text = sanitizeText(text)

	// Chunk the text
	chunks := chunkText(text, chunkSize, chunkOverlap)
	totalChunks := int32(len(chunks))
	log.Printf("[worker] doc %d: extracted %d chars, split into %d chunks", doc.ID, len(text), totalChunks)

	// Write total_chunks so frontend can show a progress bar
	if _, err := w.queries.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
		ID:          doc.ID,
		Status:      "processing",
		TotalChunks: pgtype.Int4{Int32: totalChunks, Valid: true},
	}); err != nil {
		log.Printf("[worker] doc %d: failed to set total_chunks: %v", doc.ID, err)
	}

	// Generate embeddings in batches
	var chunksDone int32 = 0
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[i:end]

		vectors, err := w.embClient.EmbedCtx(ctx, batch)
		if err != nil {
			return fmt.Errorf("embed batch starting at chunk %d: %w", i, err)
		}

		// Insert chunks with embeddings
		for j, vec := range vectors {
			chunkIdx := i + j
			content := sanitizeText(chunks[chunkIdx])
			_, err := w.queries.InsertDocumentChunk(ctx, database.InsertDocumentChunkParams{
				DocumentID: doc.ID,
				ChunkIndex: int32(chunkIdx),
				Content:    content,
				Embedding:  pgvector.NewVector(float64ToFloat32(vec)),
			})
			if err != nil {
				return fmt.Errorf("insert chunk %d: %w", chunkIdx, err)
			}
		}

		chunksDone = int32(end)
		w.queries.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
			ID:         doc.ID,
			Status:     "processing",
			ChunksDone: pgtype.Int4{Int32: chunksDone, Valid: true},
		})
		log.Printf("[worker] doc %d: embedded chunks %d-%d/%d", doc.ID, i, end-1, totalChunks)
	}

	// Mark as ready (auto-approve)
	_, err = w.queries.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
		ID:     doc.ID,
		Status: "ready",
		// Approved:     pgtype.Bool{Bool: false, Valid: true},
		ReviewStatus: pgtype.Text{String: "pending", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("mark ready: %w", err)
	}

	log.Printf("[worker] doc %d: completed successfully", doc.ID)
	return nil
}

// extractPDFText reads a PDF file and returns all extractable text.
// Uses pdftotext (poppler-utils) as the primary extractor because it handles
// virtually all PDF encodings including CID fonts and custom CMaps that the
// ledongthuc/pdf Go library cannot decode. Falls back to the Go library if
// pdftotext is not available.
func extractPDFText(path string) (string, error) {
	text, err := extractWithPdftotext(path)
	if err == nil && strings.TrimSpace(text) != "" {
		return text, nil
	}
	if err != nil {
		log.Printf("[worker] pdftotext failed for %s: %v — falling back to Go library", path, err)
	} else {
		log.Printf("[worker] pdftotext returned empty for %s — falling back to Go library", path)
	}

	return extractWithGoPDF(path)
}

// extractWithPdftotext runs the pdftotext CLI tool to extract text.
func extractWithPdftotext(path string) (string, error) {
	// -layout: preserve physical layout as much as possible
	// -nopgbrk: don't insert page breaks (we handle those ourselves)
	cmd := exec.Command("pdftotext", "-layout", "-nopgbrk", path, "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), nil
}

// extractWithGoPDF uses the ledongthuc/pdf Go library as a fallback.
func extractWithGoPDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	totalPage := r.NumPage()
	var failedPages int

	for pageNum := 1; pageNum <= totalPage; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			failedPages++
			log.Printf("[worker] Go PDF lib: page %d failed: %v", pageNum, err)
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n")
	}

	if failedPages > 0 {
		log.Printf("[worker] Go PDF lib: %d/%d pages failed to extract", failedPages, totalPage)
	}

	return buf.String(), nil
}

// sanitizeText cleans extracted text to ensure it is safe for PostgreSQL text
// columns. It strips null bytes, other control characters (except newlines and
// tabs), replaces invalid UTF-8 sequences with the replacement character, and
// normalizes whitespace.
func sanitizeText(s string) string {
	// 1. Convert to valid UTF-8, replacing any invalid sequences
	s = strings.ToValidUTF8(s, "\ufffd")

	// 2. Strip null bytes — PostgreSQL rejects \x00 in text columns
	s = strings.ReplaceAll(s, "\x00", "")

	// 3. Filter runes: keep only printable characters, newlines, tabs, and spaces.
	//    This removes control chars like \x01–\x1F (except \t, \n), \x7F (DEL), etc.
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' || r == '\r' || (r >= ' ' && r != utf8.RuneError) {
			b.WriteRune(r)
		}
	}
	s = b.String()

	// 4. Normalize line endings: \r\n -> \n, standalone \r -> \n
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// 5. Collapse multiple blank lines into at most two
	lines := strings.Split(s, "\n")
	var out []string
	blankCount := 0
	for _, line := range lines {
		trimmed := strings.TrimRightFunc(line, unicode.IsSpace)
		if trimmed == "" {
			blankCount++
			if blankCount <= 2 {
				out = append(out, "")
			}
		} else {
			blankCount = 0
			out = append(out, trimmed)
		}
	}
	// Trim trailing blank lines
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}

	return strings.Join(out, "\n")
}

// chunkText splits text into overlapping chunks of approximately chunkSize chars.
func chunkText(text string, size, overlap int) []string {
	runes := []rune(text)
	if len(runes) <= size {
		return []string{text}
	}

	var chunks []string
	for i := 0; i < len(runes); i += size - overlap {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

func float64ToFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}

// CreateUploadDir ensures the upload directory exists.
func CreateUploadDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// SupportedExtensions lists the file extensions the ingest pipeline can parse.
// Widening the list requires adding a corresponding branch to ExtractDocumentText.
var SupportedExtensions = map[string]bool{
	".pdf": true,
	".txt": true,
	".md":  true,
}

// IsSupportedExtension reports whether the given filename has one of the
// supported document extensions (case-insensitive).
func IsSupportedExtension(name string) bool {
	return SupportedExtensions[strings.ToLower(filepath.Ext(name))]
}

// GenerateFilePath creates a unique filename for an uploaded document,
// preserving the original extension when supported. Falls back to `.pdf` for
// unrecognised extensions so callers that skip validation still land somewhere
// sensible. Returns just the filename (not the full path).
func GenerateFilePath(originalName string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	if !SupportedExtensions[ext] {
		ext = ".pdf"
	}
	clean := strings.TrimSuffix(originalName, filepath.Ext(originalName))
	clean = strings.Map(func(r rune) rune {
		if r == ' ' || r == '(' || r == ')' || r == '[' || r == ']' {
			return '_'
		}
		return r
	}, clean)
	return fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), clean, ext)
}

// ExtractDocumentText dispatches to the right extractor for the file's
// extension. PDFs go through pdftotext with a Go-library fallback; plain text
// and Markdown are read straight off disk.
func ExtractDocumentText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return extractPDFText(path)
	case ".txt", ".md":
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", ext, err)
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("unsupported extension: %s", ext)
	}
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
