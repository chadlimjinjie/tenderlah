package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/midtone/api-server/internal/auth"
)

const UserKey = "user"

// Authenticate resolves the JWT from Authorization header and sets c.Keys[UserKey].
// Does not abort — routes that require auth must call RequireUser.
func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := auth.ParseBearer(c.GetHeader("Authorization"))
		if ok {
			user, err := auth.VerifyToken(token)
			if err == nil {
				c.Set(UserKey, user)
			}
		}
		c.Next()
	}
}

// RequireUser aborts with 401 if no authenticated user is present.
func RequireUser(c *gin.Context) (*auth.User, bool) {
	v, exists := c.Get(UserKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return nil, false
	}
	user, ok := v.(*auth.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return nil, false
	}
	return user, true
}

// RequireAdmin calls RequireUser and also checks that the caller's global role
// is "admin". The role comes from the JWT's "role" claim (set by auth-service),
// so no database lookup is needed.
func RequireAdmin(c *gin.Context) (*auth.User, bool) {
	user, ok := RequireUser(c)
	if !ok {
		return nil, false
	}
	if user.Role == nil || *user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return nil, false
	}
	return user, true
}

// RequireOrg calls RequireUser and also checks that ActiveOrganizationID is set.
func RequireOrg(c *gin.Context) (*auth.User, string, bool) {
	user, ok := RequireUser(c)
	if !ok {
		return nil, "", false
	}
	if user.ActiveOrganizationID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active organisation"})
		return nil, "", false
	}
	return user, *user.ActiveOrganizationID, true
}
