package middleware

import (
	"GoNetDisk/internal/model"
	"GoNetDisk/internal/repository"
	"GoNetDisk/internal/util"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtManager *util.JWTManager, userRepo *repository.UserRepo) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ""

		authHeader := ctx.GetHeader("Authorization")
		if authHeader != "" {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenString == "" {
			tokenString = ctx.Query("token")
		}

		if tokenString == "" || tokenString == authHeader {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证token"})
			ctx.Abort()
			return
		}

		claims, err := jwtManager.VerifyToken(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			ctx.Abort()
			return
		}

		userID, err := strconv.ParseUint(claims.RegisteredClaims.Subject, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			ctx.Abort()
			return
		}

		user, err := userRepo.GetUserByID(userID)
		if err != nil || user.Status != model.UserStatusNormal {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "用户已被禁用"})
			ctx.Abort()
			return
		}

		ctx.Set("userID", userID)
		ctx.Set("username", user.Username)
		ctx.Set("email", user.Email)

		ctx.Next()
	}
}

func GetUsername(ctx *gin.Context) (string, bool) {
	username, exists := ctx.Get("username")
	if !exists {
		return "", false
	}
	return username.(string), true
}

func GetEmail(ctx *gin.Context) (string, bool) {
	email, exists := ctx.Get("email")
	if !exists {
		return "", false
	}
	v, ok := email.(string)
	return v, ok
}

func GetUserID(ctx *gin.Context) (uint64, bool) {
	userID, exists := ctx.Get("userID")
	if !exists {
		return 0, false
	}
	v, ok := userID.(uint64)
	return v, ok
}
