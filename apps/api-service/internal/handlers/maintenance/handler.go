// Package maintenance exposes per-app maintenance flags keyed by a "scope"
// (e.g. "beta" for web-beta, "portfolio" for web-chad). The read endpoint is
// public; the write endpoint is admin-only (the global "admin" role, looked up
// in the database since the service's JWT does not carry it).
package maintenance

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/midtone/api-server/internal/db"
	mw "github.com/midtone/api-server/internal/middleware"
)

type Handler struct{ q *db.Queries }

func New(q *db.Queries) *Handler { return &Handler{q: q} }

func (h *Handler) Register(rg *gin.RouterGroup) {
	// Per-app scoped flags, e.g. /api/maintenance/beta, /api/maintenance/portfolio.
	rg.GET("/:scope", h.getScoped)
	rg.PUT("/:scope", h.setScoped)
}

type maintenanceResponse struct {
	Enabled   bool   `json:"enabled"`
	Message   string `json:"message"`
	UpdatedAt any    `json:"updatedAt"`
}

type setMaintenanceRequest struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

// @Summary  Get a scoped maintenance flag
// @Tags     Maintenance
// @Produce  json
// @Param    scope  path      string  true  "Maintenance scope (e.g. portfolio)"
// @Success  200    {object}  maintenanceResponse
// @Router   /api/maintenance/{scope} [get]
func (h *Handler) getScoped(c *gin.Context) {
	row, err := h.q.GetMaintenanceScope(c, c.Param("scope"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Scope not yet created — report the safe default of disabled.
			c.JSON(http.StatusOK, maintenanceResponse{Enabled: false, Message: "", UpdatedAt: nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, maintenanceResponse{Enabled: row.Enabled, Message: row.Message, UpdatedAt: row.UpdatedAt})
}

// @Summary  Update a scoped maintenance flag (admin only)
// @Tags     Maintenance
// @Accept   json
// @Produce  json
// @Param    scope  path      string                 true  "Maintenance scope (e.g. portfolio)"
// @Param    body   body      setMaintenanceRequest  true  "Maintenance state"
// @Success  200    {object}  maintenanceResponse
// @Failure  401    {object}  map[string]string
// @Failure  403    {object}  map[string]string
// @Router   /api/maintenance/{scope} [put]
func (h *Handler) setScoped(c *gin.Context) {
	if _, ok := mw.RequireAdmin(c); !ok {
		return
	}

	var body setMaintenanceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row, err := h.q.UpsertMaintenanceScope(c, c.Param("scope"), body.Enabled, body.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, maintenanceResponse{Enabled: row.Enabled, Message: row.Message, UpdatedAt: row.UpdatedAt})
}
