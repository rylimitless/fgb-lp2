package worker

import (
	"context"
	database "fgb-lp/database/queries"
	"fgb-lp/embeddings"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ledongthuc/pdf"
	"github.com/pgvector/pgvector-go"
)

const (
	chunkSize    = 800 // characters per chunk
	chunkOverlap = 100 // overlap between chunks
	batchSize    = 20  // number of chunks per embedding API call
	pollInterval = 2 * time.Second
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

		if err := w.processDocument(ctx, claimed); err != nil {
			log.Printf("[worker] failed to process doc %d: %v", claimed.ID, err)
			w.queries.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
				ID:     claimed.ID,
				Status: "failed",
			})
		}
	}
}

func (w *Worker) processDocument(ctx context.Context, doc database.Document) error {
	// Extract text from PDF
	text, err := extractPDFText(filepath.Join(w.uploadDir, doc.FilePath))
	if err != nil {
		return fmt.Errorf("extract pdf text: %w", err)
	}

	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("no extractable text found in PDF")
	}

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

		vectors, err := w.embClient.Embed(batch)
		if err != nil {
			return fmt.Errorf("embed batch starting at chunk %d: %w", i, err)
		}

		// Insert chunks with embeddings
		for j, vec := range vectors {
			chunkIdx := i + j
			_, err := w.queries.InsertDocumentChunk(ctx, database.InsertDocumentChunkParams{
				DocumentID: doc.ID,
				ChunkIndex: int32(chunkIdx),
				Content:    chunks[chunkIdx],
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

	// Mark as ready
	_, err = w.queries.UpdateDocumentStatus(ctx, database.UpdateDocumentStatusParams{
		ID:     doc.ID,
		Status: "ready",
	})
	if err != nil {
		return fmt.Errorf("mark ready: %w", err)
	}

	log.Printf("[worker] doc %d: completed successfully", doc.ID)
	return nil
}

// extractPDFText reads a PDF file and returns all extractable text.
func extractPDFText(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	totalPage := r.NumPage()

	for pageNum := 1; pageNum <= totalPage; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n")
	}

	return buf.String(), nil
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

// GenerateFilePath creates a unique filename for an uploaded PDF.
// Returns just the filename (not the full path) — the caller joins with uploadDir.
func GenerateFilePath(originalName string) string {
	ext := filepath.Ext(originalName)
	if !strings.EqualFold(ext, ".pdf") {
		ext = ".pdf"
	}
	clean := strings.TrimSuffix(originalName, ext)
	// Sanitize: replace spaces and special chars
	clean = strings.Map(func(r rune) rune {
		if r == ' ' || r == '(' || r == ')' || r == '[' || r == ']' {
			return '_'
		}
		return r
	}, clean)
	return fmt.Sprintf("%d_%s.pdf", time.Now().UnixNano(), clean)
}
