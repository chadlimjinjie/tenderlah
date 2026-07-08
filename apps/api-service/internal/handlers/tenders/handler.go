package tenders

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/midtone/api-server/internal/auth"
	"github.com/midtone/api-server/internal/db"
	"github.com/midtone/api-server/internal/gebizdetail"
	"github.com/midtone/api-server/internal/llm"
	mw "github.com/midtone/api-server/internal/middleware"
	"github.com/pgvector/pgvector-go"
)

// decodeTenderID undoes the URL-safe base64url encoding the frontend applies
// to ids (every real id is source:source_id, and source_id can contain '/').
// A literal id always contains ':' (not in the base64url alphabet) so it
// fails decode and is used as-is for direct callers.
func decodeTenderID(id string) string {
	if b, err := base64.RawURLEncoding.DecodeString(id); err == nil {
		return string(b)
	}
	return id
}

type Handler struct {
	q                *db.Queries
	llm              *llm.Client
	embedServiceURL  string   // apps/embed-service (potion-base-8M) — distinct from llm's cloud chat endpoint, which serves no embedding models
	inFlight         sync.Map // tender id -> struct{}, guards fetchDetails from concurrent runs
	briefInFlight    sync.Map // tender id -> struct{}, guards generateBrief from concurrent runs
	timelineInFlight sync.Map // tender id -> struct{}, guards generateDeliveryTimeline from concurrent runs
	matchInFlight    sync.Map // "userID:tenderID" -> struct{}, guards generateMatch from concurrent runs
}

func New(q *db.Queries, llmClient *llm.Client, embedServiceURL string) *Handler {
	return &Handler{q: q, llm: llmClient, embedServiceURL: embedServiceURL}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.list)
	rg.GET("/filters", h.filters)
	rg.GET("/:id", h.get)
	rg.POST("/:id/favourite", h.addFavourite)
	rg.DELETE("/:id/favourite", h.removeFavourite)
	rg.POST("/:id/brief", h.generateBrief)
	rg.POST("/:id/delivery-timeline", h.generateDeliveryTimeline)
	rg.POST("/:id/match", h.generateMatch)
}

type tenderRow struct {
	db.Tender
	IsFavourite bool         `json:"isFavourite"`
	Match       *TenderMatch `json:"match,omitempty"`
}

type tenderPage struct {
	Items    []tenderRow `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// @Summary  List GeBIZ tenders (paginated, searchable, filterable, sortable)
// @Tags     Tenders
// @Produce  json
// @Param    page      query     int     false  "1-based page number"
// @Param    pageSize  query     int     false  "items per page (max 200)"
// @Param    q         query     string  false  "search title/agency/referenceNo"
// @Param    status    query     string  false  "exact status filter"
// @Param    openState query     string  false  "open|closed (by closing date; default all)"
// @Param    source    query     string  false  "exact source filter"
// @Param    agency    query     string  false  "exact agency filter"
// @Param    sortBy    query     string  false  "closingDate|publishedDate|title|relevance (default publishedDate)"
// @Param    sortDir   query     string  false  "asc|desc (default desc)"
// @Param    favouritesOnly  query  bool  false  "only the signed-in user's starred tenders"
// @Success  200  {object}  tenderPage
// @Failure  401  {object}  map[string]string
// @Router   /api/tenders [get]
func (h *Handler) list(c *gin.Context) {
	page := max(atoiDefault(c.Query("page"), 1), 1)
	pageSize := atoiDefault(c.Query("pageSize"), 50)
	if pageSize < 1 {
		pageSize = 50
	}
	pageSize = min(pageSize, 200)

	favouritesOnly := c.Query("favouritesOnly") == "true"
	var userID string
	if favouritesOnly {
		user, ok := mw.RequireUser(c)
		if !ok {
			return
		}
		userID = user.ID
	}

	// The signed-in user's precomputed onboarding-profile embedding (written on
	// profile save) ranks tenders by cosine distance: it's the primary order for
	// "relevance" sort and a secondary tie-breaker under every other sort, so we
	// load it for any signed-in user. Absent (anonymous visitors, unonboarded
	// users, profiles not yet embedded) it falls back to recency in
	// tenderSortClause.
	var relevanceEmbedding *pgvector.Vector
	if v, ok := c.Get(mw.UserKey); ok {
		if user, ok := v.(*auth.User); ok {
			if stored, err := h.q.GetUserOnboardingEmbedding(c, user.ID); err != nil {
				log.Printf("tenders: read profile embed: %v", err)
			} else if stored != nil {
				relevanceEmbedding = stored
			}
		}
	}

	params := db.ListTendersSearchParams{
		Q:                  c.Query("q"),
		Status:             c.Query("status"),
		OpenState:          c.Query("openState"),
		Source:             c.Query("source"),
		Agency:             c.Query("agency"),
		SortBy:             c.Query("sortBy"),
		SortDir:            c.Query("sortDir"),
		FavouritesOnly:     favouritesOnly,
		UserID:             userID,
		RelevanceEmbedding: relevanceEmbedding,
		Limit:              int32(pageSize),
		Offset:             int32((page - 1) * pageSize),
	}

	total, err := h.q.CountTendersSearch(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, err := h.q.ListTendersSearch(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	favourites := h.favouriteSet(c)
	items := make([]tenderRow, len(rows))
	for i, row := range rows {
		items[i] = tenderRow{Tender: row, IsFavourite: favourites[row.ID]}
	}
	c.JSON(http.StatusOK, tenderPage{Items: items, Total: total, Page: page, PageSize: pageSize})
}

// favouriteSet returns the signed-in user's favourited tender ids as a set,
// or an empty set if the request is unauthenticated (list stays public).
func (h *Handler) favouriteSet(c *gin.Context) map[string]bool {
	v, ok := c.Get(mw.UserKey)
	if !ok {
		return nil
	}
	user, ok := v.(*auth.User)
	if !ok {
		return nil
	}
	ids, err := h.q.ListFavouriteTenderIDs(c, user.ID)
	if err != nil {
		return nil
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

type tenderFilters struct {
	Statuses []string `json:"statuses"`
	Sources  []string `json:"sources"`
	Agencies []string `json:"agencies"`
}

// @Summary  List distinct tender statuses/sources/agencies for filter dropdowns
// @Tags     Tenders
// @Produce  json
// @Param    source  query  string  false  "pipe-separated source filter; scopes returned agencies to those sources"
// @Success  200  {object}  tenderFilters
// @Router   /api/tenders/filters [get]
func (h *Handler) filters(c *gin.Context) {
	statuses, err := h.q.ListDistinctTenderStatuses(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sources, err := h.q.ListDistinctTenderSources(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agencies, err := h.q.ListDistinctTenderAgencies(c, c.Query("source"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tenderFilters{Statuses: statuses, Sources: sources, Agencies: agencies})
}

// toTenderMatch converts the DB row (jsonb columns as json.RawMessage) into the
// API response shape, tolerating malformed JSON by falling back to an empty slice.
func toTenderMatch(m db.TenderMatch) *TenderMatch {
	out := &TenderMatch{Score: int(m.Score), Priority: m.Priority}
	_ = json.Unmarshal(m.FitReasons, &out.FitReasons)
	_ = json.Unmarshal(m.MissingRequirements, &out.MissingRequirements)
	_ = json.Unmarshal(m.Risks, &out.Risks)
	return out
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// @Summary  Get a tender by id
// @Tags     Tenders
// @Produce  json
// @Param    id  path      string  true  "Tender ID (source:source_id)"
// @Success  200  {object}  tenderRow
// @Failure  404  {object}  map[string]string
// @Router   /api/tenders/{id} [get]
func (h *Handler) get(c *gin.Context) {
	id := decodeTenderID(c.Param("id"))
	row, err := h.q.GetTender(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tender not found"})
		return
	}
	// GeBIZ is hidden from the listing pending DSTA permission (see COMPLIANCE.md); block
	// stale deep-links too. Flip GEBIZ_LISTING_ENABLED=true to surface GeBIZ again.
	if row.Source == "gebiz" && os.Getenv("GEBIZ_LISTING_ENABLED") != "true" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tender not found"})
		return
	}

	_, authed := c.Get(mw.UserKey)

	// GeBIZ detail isn't in the RSS feed, so on first view of a tender it can be slow.
	// Don't block the response on it — return the cached row immediately and let the
	// frontend poll until it lands. AI brief/timeline generation is triggered separately
	// via POST /:id/brief and /:id/delivery-timeline, not from this GET.
	// ponytail: flag defaults off pending DSTA written permission to reproduce GeBIZ
	// opportunity-page content; flip GEBIZ_DETAIL_SCRAPE_ENABLED=true once granted.
	needsScrape := row.Source == "gebiz" && row.DetailsFetchedAt == nil && row.Link != "" &&
		os.Getenv("GEBIZ_DETAIL_SCRAPE_ENABLED") == "true"
	if needsScrape {
		if _, alreadyRunning := h.inFlight.LoadOrStore(row.ID, struct{}{}); !alreadyRunning {
			go h.fetchDetails(row)
		}
	}

	if !authed {
		row.AiBrief = nil
		row.AiBriefAt = nil
		row.AiDeliveryTimeline = nil
		row.AiDeliveryTimelineAt = nil
	}

	c.JSON(http.StatusOK, tenderRow{Tender: row, IsFavourite: h.favouriteSet(c)[row.ID]})
}

// fetchDetails scrapes the GeBIZ opportunity page and caches the result. Runs detached
// from the request context (which is cancelled once the response is written).
// ponytail: in-process guard only (sync.Map on Handler); fine for a single
// api-server-go instance. If this ever runs with multiple replicas, swap for a
// DB-level advisory lock (pg_try_advisory_lock on the tender id) instead.
func (h *Handler) fetchDetails(row db.Tender) {
	defer h.inFlight.Delete(row.ID)
	ctx := context.Background()

	if d, err := gebizdetail.Fetch(ctx, row.Link); err != nil {
		log.Printf("tenders: scrape %s: %v", row.ID, err)
	} else if b, err := json.Marshal(d); err == nil {
		if _, err := h.q.UpdateTenderDetails(ctx, row.ID, string(b)); err != nil {
			log.Printf("tenders: store details %s: %v", row.ID, err)
		}
	}
}

// @Summary  Generate (or return cached) AI bid brief for a tender
// @Tags     Tenders
// @Produce  json
// @Param    id  path      string  true  "Tender ID (base64url or literal)"
// @Success  200  {object}  BidBrief
// @Failure  401  {object}  map[string]string
// @Failure  404  {object}  map[string]string
// @Failure  409  {object}  map[string]string
// @Router   /api/tenders/{id}/brief [post]
func (h *Handler) generateBrief(c *gin.Context) {
	if _, ok := mw.RequireUser(c); !ok {
		return
	}
	if h.llm == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI brief generation unavailable"})
		return
	}

	id := decodeTenderID(c.Param("id"))
	row, err := h.q.GetTender(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tender not found"})
		return
	}
	if row.AiBriefAt != nil {
		c.JSON(http.StatusOK, row.AiBrief)
		return
	}
	if row.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tender has no title to generate a brief from"})
		return
	}

	if _, alreadyRunning := h.briefInFlight.LoadOrStore(row.ID, struct{}{}); alreadyRunning {
		c.JSON(http.StatusConflict, gin.H{"error": "Brief generation already in progress"})
		return
	}
	defer h.briefInFlight.Delete(row.ID)

	_, brief, err := generateBrief(c, h.llm, row)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.q.UpdateTenderBrief(c, row.ID, brief)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated.AiBrief)
}

// @Summary  Generate (or return cached) AI delivery timeline for a tender
// @Tags     Tenders
// @Produce  json
// @Param    id  path      string  true  "Tender ID (base64url or literal)"
// @Success  200  {object}  DeliveryTimeline
// @Failure  401  {object}  map[string]string
// @Failure  404  {object}  map[string]string
// @Failure  409  {object}  map[string]string
// @Router   /api/tenders/{id}/delivery-timeline [post]
func (h *Handler) generateDeliveryTimeline(c *gin.Context) {
	if _, ok := mw.RequireUser(c); !ok {
		return
	}
	if h.llm == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI delivery timeline generation unavailable"})
		return
	}

	id := decodeTenderID(c.Param("id"))
	row, err := h.q.GetTender(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tender not found"})
		return
	}
	if row.AiDeliveryTimelineAt != nil {
		c.JSON(http.StatusOK, row.AiDeliveryTimeline)
		return
	}
	if row.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tender has no title to generate a delivery timeline from"})
		return
	}

	if _, alreadyRunning := h.timelineInFlight.LoadOrStore(row.ID, struct{}{}); alreadyRunning {
		c.JSON(http.StatusConflict, gin.H{"error": "Delivery timeline generation already in progress"})
		return
	}
	defer h.timelineInFlight.Delete(row.ID)

	_, timeline, err := generateTimeline(c, h.llm, row)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.q.UpdateTenderDeliveryTimeline(c, row.ID, timeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated.AiDeliveryTimeline)
}

// @Summary  Generate (or return cached) AI match score for a tender, for the signed-in user's company profile
// @Tags     Tenders
// @Produce  json
// @Param    id  path      string  true  "Tender ID (base64url or literal)"
// @Success  200  {object}  TenderMatch
// @Failure  400  {object}  map[string]string
// @Failure  401  {object}  map[string]string
// @Failure  404  {object}  map[string]string
// @Failure  409  {object}  map[string]string
// @Router   /api/tenders/{id}/match [post]
func (h *Handler) generateMatch(c *gin.Context) {
	user, ok := mw.RequireUser(c)
	if !ok {
		return
	}
	if h.llm == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI match generation unavailable"})
		return
	}

	id := decodeTenderID(c.Param("id"))
	row, err := h.q.GetTender(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tender not found"})
		return
	}
	if cached, cerr := h.q.GetTenderMatch(c, user.ID, row.ID); cerr == nil {
		c.JSON(http.StatusOK, toTenderMatch(cached))
		return
	}
	if row.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tender has no title to generate a match from"})
		return
	}

	key := user.ID + ":" + row.ID
	if _, alreadyRunning := h.matchInFlight.LoadOrStore(key, struct{}{}); alreadyRunning {
		c.JSON(http.StatusConflict, gin.H{"error": "Match generation already in progress"})
		return
	}
	defer h.matchInFlight.Delete(key)

	onboarding, oerr := h.q.GetUserOnboarding(c, user.ID)
	if oerr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Complete your business profile first"})
		return
	}
	match, fitReasonsJSON, missingJSON, risksJSON, err := generateMatchJSON(c, h.llm, row, onboarding)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.q.UpsertTenderMatch(c, db.UpsertTenderMatchParams{
		UserID: user.ID, TenderID: row.ID, Score: int32(match.Score), Priority: match.Priority,
		FitReasons: fitReasonsJSON, MissingRequirements: missingJSON, Risks: risksJSON,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, match)
}

// @Summary  Favourite a tender
// @Tags     Tenders
// @Param    id  path  string  true  "Tender ID (base64url or literal)"
// @Success  204
// @Failure  401  {object}  map[string]string
// @Router   /api/tenders/{id}/favourite [post]
func (h *Handler) addFavourite(c *gin.Context) {
	user, ok := mw.RequireUser(c)
	if !ok {
		return
	}
	if err := h.q.CreateTenderFavourite(c, user.ID, decodeTenderID(c.Param("id"))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary  Unfavourite a tender
// @Tags     Tenders
// @Param    id  path  string  true  "Tender ID (base64url or literal)"
// @Success  204
// @Failure  401  {object}  map[string]string
// @Router   /api/tenders/{id}/favourite [delete]
func (h *Handler) removeFavourite(c *gin.Context) {
	user, ok := mw.RequireUser(c)
	if !ok {
		return
	}
	if err := h.q.DeleteTenderFavourite(c, user.ID, decodeTenderID(c.Param("id"))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
