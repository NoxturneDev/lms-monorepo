package web

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/noxturnedev/lms-monorepo/gateway/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Print("AUTH MIDDLEWARE")
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format. Use: Bearer <token>"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := utils.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_type", claims.UserType)

		c.Next()
	}
}

func TeacherOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists || userType != "teacher" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Teacher access only"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func StudentOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists || userType != "student" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Student access only"})
			c.Abort()
			return
		}
		c.Next()
	}
}
