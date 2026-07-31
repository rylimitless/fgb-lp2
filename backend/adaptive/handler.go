package adaptive

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

const learningRate = 0.1

// minTypeSamples is how many attempts a learner must have on a question type
// before its observed accuracy is trusted over "no preference". Below this we
// don't yet know whether the learner responds better to that type.
const minTypeSamples = 2

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

// itemIRT is a single assessable item with its IRT parameters.
type itemIRT struct {
	Item  gin.H
	Beta  float64
	Alpha float64
	Type  string
	ID    int64 // denormalized from Item so selection never type-asserts
}

// conceptGroup bundles the type-variants of one assessment concept (items that
// share a question_group_id). The adaptive engine serves the learner ONE
// variant per concept — the format that matches their preferred/best type — so
// they don't have to answer the same idea repeatedly in different formats.
// Items with no group (legacy/standalone) are treated as single-item groups.
type conceptGroup struct {
	key      string // question_group_id, or the item id for standalone items
	variants []itemIRT
}

// effectivePreferredType resolves which question type to favor for a learner:
// an explicit preferred_type override wins; otherwise we infer the type they
// respond to best from observed accuracy (needs >= minTypeSamples attempts).
// Returns "" when there's no usable signal.
func effectivePreferredType(explicit pgtype.Text, typeStats []byte) string {
	if explicit.Valid && explicit.String != "" {
		return explicit.String
	}
	stats := parseTypeStats(typeStats)
	if len(stats) == 0 {
		return ""
	}
	bestType := ""
	bestAcc := -1.0
	for t, s := range stats {
		if s.Answered < minTypeSamples {
			continue
		}
		acc := float64(s.Correct) / float64(s.Answered)
		// Prefer higher accuracy; tie-break on more attempts via the >= so the
		// most-practiced type wins ties.
		if acc > bestAcc || (acc == bestAcc && s.Answered > stats[bestType].Answered) {
			bestAcc = acc
			bestType = t
		}
	}
	return bestType
}

type typeStat struct {
	Answered int
	Correct  int
}

func parseTypeStats(raw []byte) map[string]typeStat {
	out := map[string]typeStat{}
	if len(raw) == 0 {
		return out
	}
	var m map[string]map[string]int
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	for t, v := range m {
		out[t] = typeStat{Answered: v["answered"], Correct: v["correct"]}
	}
	return out
}

// pickVariant chooses which type-variant of a concept to serve a learner: their
// preferred type if the group has one, otherwise the variant whose difficulty
// best targets their current theta. Falls back to the first variant.
func pickVariant(theta float64, group conceptGroup, preferredType string) itemIRT {
	if preferredType != "" {
		for _, v := range group.variants {
			if v.Type == preferredType {
				return v
			}
		}
	}
	best := group.variants[0]
	bestDist := math.Abs(probability(theta, best.Beta, best.Alpha) - 0.5)
	for _, v := range group.variants[1:] {
		dist := math.Abs(probability(theta, v.Beta, v.Alpha) - 0.5)
		if dist < bestDist {
			bestDist = dist
			best = v
		}
	}
	return best
}

// selectNextConcept picks the next concept to serve by IRT targeting (minimize
// |p - 0.5| on the variant we'd actually serve), then returns that variant.
// Groups whose representative item is in answeredSet are considered done.
func selectNextConcept(theta float64, groups []conceptGroup, answeredSet map[int64]struct{}, preferredType string) *itemIRT {
	bestScore := math.MaxFloat64
	var best *itemIRT
	for gi := range groups {
		g := groups[gi]
		// A concept is consumed once any of its variants has been answered.
		consumed := false
		for _, v := range g.variants {
			if _, ok := answeredSet[v.ID]; ok {
				consumed = true
				break
			}
		}
		if consumed || len(g.variants) == 0 {
			continue
		}
		chosen := pickVariant(theta, g, preferredType)
		p := probability(theta, chosen.Beta, chosen.Alpha)
		score := math.Abs(p - 0.5)
		if score < bestScore {
			bestScore = score
			chosenCopy := chosen
			best = &chosenCopy
		}
	}
	return best
}

func extractIRTParams(raw json.RawMessage) (beta, alpha float64) {
	beta, alpha = 0.0, 1.0
	if len(raw) == 0 {
		return
	}
	var m map[string]interface{}
	if json.Unmarshal(raw, &m) != nil {
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

// buildGroups turns the flat item list into concept groups. Items sharing a
// question_group_id cluster together; standalone items become singleton
// groups keyed by their own id. Content items are skipped.
func buildGroups(items []database.CourseItemWithGroup) ([]conceptGroup, int) {
	groupOrder := make([]string, 0, 8)
	byKey := make(map[string]*conceptGroup, 8)
	assessable := 0
	irtParamCount := 0
	for i := range items {
		item := items[i]
		if item.ItemType == "content" {
			continue
		}
		assessable++
		beta, alpha := extractIRTParams(json.RawMessage(item.Data))
		if beta != 0.0 || alpha != 1.0 {
			irtParamCount++
		}
		key := item.QuestionGroupID.String
		if !item.QuestionGroupID.Valid || key == "" {
			// Standalone item: its own group so it's still served once.
			key = "single-" + strconv.FormatInt(item.ID, 10)
		}
		irt := itemIRT{
			Item: gin.H{
				"id":        item.ID,
				"item_type": item.ItemType,
				"data":      json.RawMessage(item.Data),
				"beta":      beta,
				"alpha":     alpha,
			},
			Beta:  beta,
			Alpha: alpha,
			Type:  item.ItemType,
			ID:    item.ID,
		}
		if g, ok := byKey[key]; ok {
			g.variants = append(g.variants, irt)
		} else {
			byKey[key] = &conceptGroup{key: key, variants: []itemIRT{irt}}
			groupOrder = append(groupOrder, key)
		}
	}
	groups := make([]conceptGroup, 0, len(groupOrder))
	for _, k := range groupOrder {
		groups = append(groups, *byKey[k])
	}
	log.Printf("[adaptive] built pool: assessable=%d irt_calibrated=%d/%d groups=%d",
		assessable, irtParamCount, assessable, len(groups))
	return groups, assessable
}

// ---- Handlers ----

func (h *Handler) StartSession(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Query("course_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_id required"})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	items, _ := h.Queries.GetCourseItemsByCourseWithGroup(c.Request.Context(), courseID)
	if len(items) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No items in this course"})
		return
	}

	pref, _ := h.Queries.GetLearningPreferenceTypes(c.Request.Context(), userID)
	theta := pref.Theta
	preferredType := effectivePreferredType(pref.PreferredType, pref.TypeStats)

	groups, totalAssessable := buildGroups(items)
	if totalAssessable == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No assessable items in this course"})
		return
	}

	log.Printf("[adaptive] session start: user=%d course=%d theta=%.2f groups=%d preferred_type=%q",
		userID, courseID, theta, len(groups), preferredType)

	next := selectNextConcept(theta, groups, map[int64]struct{}{}, preferredType)
	if next == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No suitable item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"theta":          round2(theta),
		"item":           next.Item,
		"total":          totalAssessable,
		"preferred_type": preferredType,
	})
}

func (h *Handler) SubmitAnswer(c *gin.Context) {
	var body struct {
		CourseID        int64   `json:"course_id"`
		ItemID          int64   `json:"item_id"`
		Outcome         int     `json:"outcome"`
		AnsweredItemIds []int64 `json:"answered_item_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Outcome != 0 && body.Outcome != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "outcome must be 0 or 1"})
		return
	}

	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	pref, _ := h.Queries.GetLearningPreferenceTypes(c.Request.Context(), userID)
	theta := pref.Theta
	preferredType := effectivePreferredType(pref.PreferredType, pref.TypeStats)

	// answeredSet marks every concept the learner has finished. Because a
	// concept is consumed once ANY variant is answered, we seed the set with
	// the just-answered item plus the client's reported answered_item_ids.
	answeredSet := make(map[int64]struct{}, len(body.AnsweredItemIds)+1)
	answeredSet[body.ItemID] = struct{}{}
	for _, id := range body.AnsweredItemIds {
		answeredSet[id] = struct{}{}
	}

	items, _ := h.Queries.GetCourseItemsByCourseWithGroup(c.Request.Context(), body.CourseID)
	groups, _ := buildGroups(items)

	// Find the submitted item to read its IRT params + type.
	var submittedBeta, submittedAlpha float64
	var submittedType string
	for _, item := range items {
		if item.ID == body.ItemID {
			submittedBeta, submittedAlpha = extractIRTParams(json.RawMessage(item.Data))
			submittedType = item.ItemType
			break
		}
	}

	p := probability(theta, submittedBeta, submittedAlpha)
	newTheta := updateTheta(theta, submittedAlpha, body.Outcome, p)
	newTheta = clamp(newTheta, -3.0, 3.0)

	h.Queries.UpsertLearningPreference(c.Request.Context(), database.UpsertLearningPreferenceParams{
		UserID: userID,
		Theta:  newTheta,
	})

	// Track per-type outcomes so "responds better to a certain type" becomes
	// data-driven. Skipped for untyped/content items.
	if submittedType != "" && submittedType != "content" {
		if _, err := h.Queries.RecordTypeOutcome(c.Request.Context(), userID, submittedType, body.Outcome == 1); err != nil {
			log.Printf("[adaptive] record type outcome: %v", err)
		}
		// Re-resolve the preferred type from the just-updated stats so the next
		// pick reflects the latest signal.
		if updated, err := h.Queries.GetLearningPreferenceTypes(c.Request.Context(), userID); err == nil {
			preferredType = effectivePreferredType(updated.PreferredType, updated.TypeStats)
		}
	}

	next := selectNextConcept(newTheta, groups, answeredSet, preferredType)

	// remaining = concepts not yet consumed (i.e. groups with no answered item).
	remaining := 0
	for _, g := range groups {
		consumed := false
		for _, v := range g.variants {
			if _, ok := answeredSet[v.ID]; ok {
				consumed = true
				break
			}
		}
		if !consumed {
			remaining++
		}
	}

	log.Printf("[adaptive] submit: user=%d course=%d item=%d type=%s outcome=%d theta=%.2f->%.2f p=%.2f beta=%.2f alpha=%.2f remaining_concepts=%d preferred_type=%q",
		userID, body.CourseID, body.ItemID, submittedType, body.Outcome, theta, newTheta, p, submittedBeta, submittedAlpha, remaining, preferredType)

	var nextItem any
	if next != nil {
		nextItem = next.Item
	}
	c.JSON(http.StatusOK, gin.H{
		"theta":          round2(newTheta),
		"delta":          round2(newTheta - theta),
		"probability":    round2(p),
		"next_item":      nextItem,
		"remaining":      remaining,
		"preferred_type": preferredType,
	})
}

// GetPreferences returns the learner's adaptive-by-type state: their explicit
// preferred_type override (if any) and the learned per-type accuracy that the
// engine uses to infer the best format when no override is set.
func (h *Handler) GetPreferences(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pref, _ := h.Queries.GetLearningPreferenceTypes(c.Request.Context(), userID)
	preferred := ""
	if pref.PreferredType.Valid {
		preferred = pref.PreferredType.String
	}
	c.JSON(http.StatusOK, gin.H{
		"preferred_type":     preferred,
		"inferred_best_type": effectivePreferredType(pgtype.Text{}, pref.TypeStats),
		"type_stats":         json.RawMessage(pref.TypeStats),
	})
}

// SetPreferredType lets a learner set (or clear, with empty string) their
// explicit preferred question-type override. The engine honors this over the
// learned best-performing type.
func (h *Handler) SetPreferredType(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var body struct {
		PreferredType string `json:"preferred_type"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Queries.SetPreferredType(c.Request.Context(), userID, body.PreferredType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save preference"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"preferred_type": body.PreferredType})
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/adaptive/start", h.StartSession)
	r.POST("/adaptive/submit", h.SubmitAnswer)
	r.GET("/adaptive/preferences", h.GetPreferences)
	r.PUT("/adaptive/preferences", h.SetPreferredType)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
