package users

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/midtone/api-server/internal/db"
	mw "github.com/midtone/api-server/internal/middleware"
	"github.com/midtone/api-server/internal/storage"
)

type Handler struct {
	q               *db.Queries
	storage         *storage.Client
	embedServiceURL string // apps/embed-service (potion-base-8M), for precomputing the profile embedding on save
}

func New(q *db.Queries, s *storage.Client, embedServiceURL string) *Handler {
	return &Handler{q: q, storage: s, embedServiceURL: embedServiceURL}
}

var allowedLogoTypes = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}

const maxLogoSize = 5 << 20 // 5MB

type onboardingResponse struct {
	AccountType           string    `json:"accountType"`
	OrgName               string    `json:"orgName"`
	BusinessName          *string   `json:"businessName"`
	BusinessDescription   *string   `json:"businessDescription"`
	Industry              string    `json:"industry"`
	LogoURL               *string   `json:"logoUrl"`
	CompanySize           *string   `json:"companySize"`
	GeographicCoverage    *string   `json:"geographicCoverage"`
	PreferredContractSize *string   `json:"preferredContractSize"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

func toResponse(u db.UserOnboarding) onboardingResponse {
	return onboardingResponse{
		AccountType: u.AccountType, OrgName: u.OrgName, BusinessName: u.BusinessName,
		BusinessDescription: u.BusinessDescription,
		Industry:            u.Industry, LogoURL: u.LogoUrl, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		CompanySize:           u.CompanySize,
		GeographicCoverage:    u.GeographicCoverage,
		PreferredContractSize: u.PreferredContractSize,
	}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/onboarding", h.get)
	rg.POST("/onboarding", h.upsert)
	rg.POST("/onboarding/logo", h.uploadLogo)
}

// @Summary  Get the caller's onboarding profile
// @Tags     Users
// @Produce  json
// @Success  200  {object}  onboardingResponse
// @Failure  404  {object}  map[string]string
// @Router   /api/users/onboarding [get]
func (h *Handler) get(c *gin.Context) {
	user, ok := mw.RequireUser(c)
	if !ok {
		return
	}
	row, err := h.q.GetUserOnboarding(c, user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not onboarded"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toResponse(row))
}

type onboardingBody struct {
	AccountType           string  `json:"accountType" binding:"required,oneof=buyer supplier both"`
	OrgName               string  `json:"orgName"     binding:"required"`
	BusinessName          *string `json:"businessName"`
	BusinessDescription   *string `json:"businessDescription"`
	Industry              string  `json:"industry"    binding:"required"`
	CompanySize           *string `json:"companySize"`
	GeographicCoverage    *string `json:"geographicCoverage"`
	PreferredContractSize *string `json:"preferredContractSize"`
}

// @Summary  Create or update the caller's onboarding profile
// @Tags     Users
// @Accept   json
// @Produce  json
// @Param    body  body      onboardingBody  true  "Onboarding profile"
// @Success  200   {object}  onboardingResponse
// @Failure  401   {object}  map[string]string
// @Router   /api/users/onboarding [post]
func (h *Handler) upsert(c *gin.Context) {
	user, ok := mw.RequireUser(c)
	if !ok {
		return
	}
	var body onboardingBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row, err := h.q.UpsertUserOnboarding(c, db.UpsertUserOnboardingParams{
		UserID: user.ID, AccountType: body.AccountType, OrgName: body.OrgName,
		BusinessName: body.BusinessName, BusinessDescription: body.BusinessDescription,
		Industry:              body.Industry,
		CompanySize:           body.CompanySize,
		GeographicCoverage:    body.GeographicCoverage,
		PreferredContractSize: body.PreferredContractSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Precompute the relevance embedding so the tenders list never has to embed
	// at request time. Industry weighted ×3 to outweigh the free-text
	// description — must match the read-side weighting in the tenders handler.
	// ponytail: best-effort embed; on failure the column stays NULL and the
	// tenders list self-heals it on the next relevance read.
	if vec, err := db.Embed(c, h.embedServiceURL, "", strings.Repeat(body.Industry+" ", 3)+deref(body.BusinessDescription)); err != nil {
		log.Printf("users: onboarding embed: %v", err)
	} else if err := h.q.SetUserOnboardingEmbedding(c, user.ID, pgvector.NewVector(vec)); err != nil {
		log.Printf("users: store onboarding embed: %v", err)
	}

	c.JSON(http.StatusOK, toResponse(row))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// @Summary  Upload the caller's business logo
// @Tags     Users
// @Accept   multipart/form-data
// @Produce  json
// @Param    file  formData  file  true  "Logo image"
// @Success  200   {object}  onboardingResponse
// @Failure  400   {object}  map[string]string
// @Router   /api/users/onboarding/logo [post]
func (h *Handler) uploadLogo(c *gin.Context) {
	user, ok := mw.RequireUser(c)
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	if fileHeader.Size > maxLogoSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file exceeds 5MB limit"})
		return
	}
	contentType := fileHeader.Header.Get("Content-Type")
	ext, ok := allowedLogoTypes[contentType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file must be image/jpeg, image/png, or image/webp"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	path := user.ID + "/logo/" + uuid.NewString() + ext
	url, err := h.storage.Upload(path, contentType, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	row, err := h.q.SetUserOnboardingLogo(c, user.ID, &url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toResponse(row))
}
