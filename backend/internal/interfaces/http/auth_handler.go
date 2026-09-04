package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	tokenTTL = 24 * time.Hour
)

type AuthHandler struct {
	secret   string
	username string
	password string
}

type LoginCmd struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginDTO struct {
	Token     string `json:"token"`
	Username  string `json:"username"`
	ExpiresAt string `json:"expires_at"`
}

type authClaims struct {
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
}

func NewAuthHandler(secret, username, password string) *AuthHandler {
	return &AuthHandler{secret: secret, username: username, password: password}
}

func (h *AuthHandler) RegisterPublicRoutes(api *gin.RouterGroup) {
	api.POST("/auth/login", h.Login)
}

func (h *AuthHandler) RegisterProtectedRoutes(api *gin.RouterGroup) {
	api.GET("/auth/me", h.Me)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var cmd LoginCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		response.ErrBadRequest(c, "用户名和密码不能为空")
		return
	}

	if subtle.ConstantTimeCompare([]byte(cmd.Username), []byte(h.username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(cmd.Password), []byte(h.password)) != 1 {
		response.ErrUnauthorized(c)
		return
	}

	expiresAt := time.Now().Add(tokenTTL)
	token, err := h.issueToken(h.username, expiresAt)
	if err != nil {
		response.ErrInternal(c, err)
		return
	}

	response.OK(c, LoginDTO{
		Token:     token,
		Username:  h.username,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	username, _ := c.Get("username")
	response.OK(c, gin.H{"username": username})
}

func AuthMiddleware(handler *AuthHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.ErrUnauthorized(c)
			c.Abort()
			return
		}

		claims, err := handler.parseToken(strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")))
		if err != nil || claims.Subject != handler.username {
			response.ErrUnauthorized(c)
			c.Abort()
			return
		}
		if time.Now().Unix() > claims.ExpiresAt {
			response.ErrUnauthorized(c)
			c.Abort()
			return
		}

		c.Set("username", claims.Subject)
		c.Next()
	}
}

func (h *AuthHandler) issueToken(subject string, expiresAt time.Time) (string, error) {
	payload, err := json.Marshal(authClaims{Subject: subject, ExpiresAt: expiresAt.Unix()})
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := h.sign(encodedPayload)
	return encodedPayload + "." + signature, nil
}

func (h *AuthHandler) parseToken(token string) (*authClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}
	expected := h.sign(parts[0])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[1])) != 1 {
		return nil, fmt.Errorf("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var claims authClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

func (h *AuthHandler) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write([]byte(payload))
	mac.Write([]byte("."))
	mac.Write([]byte(strconv.Itoa(len(payload))))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
