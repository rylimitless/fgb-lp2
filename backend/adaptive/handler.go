package adaptive

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const learningRate = 0.1

type Handler struct {
	Queries *database.Queries
}

func NewHandler(queries *database.Queries) *Handler {
	return &Handler{Queries: queries}
}

// ---- IRT Math Engine ----

func probability(theta, beta, alpha float64) float64 {
	exponent := -alpha * (theta - beta)
	if exponent > 20 {
		return 0.0
	}
	if exponent < -20 {
		return 1.0
	}
	return 1.0 / (1.0 + math.Exp(exponent))
}

func updateTheta(theta, alpha float64, outcome int, p float64) float64 {
	return theta + learningRate*alpha*(float64(outcome)-p)
}

type itemIRT struct {
	Item  gin.H
	Beta  float64
	Alpha float64
}

func selectNextItem(theta float64, pool []itemIRT) *gin.H {
	if len(pool) == 0 {
		return nil
	}
	bestIdx := 0
	bestDist := math.MaxFloat64
	for i, it := range pool {
		p := probability(theta, it.Beta, it.Alpha)
		dist := math.Abs(p - 0.5)
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}
	return &pool[bestIdx].Item
}

func extractIRTParams(data json.RawMessage) (beta, alpha float64) {
	beta, alpha = 0.0, 1.0
	if len(data) == 0 {
		return
	}
	var m map[string]interface{}
	if json.Unmarshal(data, &m) != nil {
		return
	}
	if v, ok := m["irt_beta"].(float64); ok {
		beta = clamp(v, -3.0, 3.0)
	}
	if v, ok := m["irt_alpha"].(float64); ok {
		alpha = clamp(v, 0.5, 2.5)
	}
	return
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---- Handlers ----

func (h *Handler) StartSession(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Query("course_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id required"})
		return
	}

	items, _ := h.Queries.GetCourseItemsByCourse(c.Request.Context(), courseID)
	if len(items) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No items in this course"})
		return
	}

	// Get or init student profile
	pref, _ := h.Queries.GetLearningPreference(c.Request.Context(), 1)
	theta := pref.Theta

	pool := make([]itemIRT, 0)
	for _, item := range items {
		if item.ItemType == "content" {
			continue
		}
		beta, alpha := extractIRTParams(json.RawMessage(item.Data))
		pool = append(pool, itemIRT{
			Item: gin.H{
				"id":        item.ID,
				"item_type": item.ItemType,
				"data":      json.RawMessage(item.Data),
				"beta":      beta,
				"alpha":     alpha,
			},
			Beta:  beta,
			Alpha: alpha,
		})
	}

	if len(pool) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No assessable items in this course"})
		return
	}

	next := selectNextItem(theta, pool)
	if next == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No suitable item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"theta": round2(theta),
		"item":  *next,
		"total": len(pool),
	})
}

func (h *Handler) SubmitAnswer(c *gin.Context) {
	var body struct {
		CourseID int64 `json:"course_id"`
		ItemID   int64 `json:"item_id"`
		Outcome  int   `json:"outcome"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Outcome != 0 && body.Outcome != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "outcome must be 0 or 1"})
		return
	}

	// Get student theta
	pref, _ := h.Queries.GetLearningPreference(c.Request.Context(), 1)
	theta := pref.Theta

	items, _ := h.Queries.GetCourseItemsByCourse(c.Request.Context(), body.CourseID)
	pool := make([]itemIRT, 0)
	var submittedBeta, submittedAlpha float64
	for _, item := range items {
		if item.ItemType == "content" {
			continue
		}
		beta, alpha := extractIRTParams(json.RawMessage(item.Data))
		it := itemIRT{
			Item: gin.H{
				"id":        item.ID,
				"item_type": item.ItemType,
				"data":      json.RawMessage(item.Data),
				"beta":      beta,
				"alpha":     alpha,
			},
			Beta:  beta,
			Alpha: alpha,
		}
		if item.ID == body.ItemID {
			submittedBeta, submittedAlpha = beta, alpha
		}
		pool = append(pool, it)
	}

	// Update theta
	p := probability(theta, submittedBeta, submittedAlpha)
	newTheta := updateTheta(theta, submittedAlpha, body.Outcome, p)
	newTheta = clamp(newTheta, -3.0, 3.0)

	// Save theta
	h.Queries.UpsertLearningPreference(c.Request.Context(), database.UpsertLearningPreferenceParams{
		UserID: 1,
		Theta:  newTheta,
	})

	// Select next item (exclude the one just answered)
	filteredPool := make([]itemIRT, 0)
	for _, it := range pool {
		if id, _ := it.Item["id"].(int64); id != body.ItemID {
			filteredPool = append(filteredPool, it)
		}
	}

	next := selectNextItem(newTheta, filteredPool)

	c.JSON(http.StatusOK, gin.H{
		"theta":       round2(newTheta),
		"delta":       round2(newTheta - theta),
		"probability": round2(p),
		"next_item":   next,
		"remaining":   len(filteredPool),
	})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/adaptive/start", h.StartSession)
	r.POST("/adaptive/submit", h.SubmitAnswer)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
