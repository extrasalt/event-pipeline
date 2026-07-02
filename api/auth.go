package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	errInvalidToken = errors.New("invalid token")
	errExpiredToken = errors.New("expired token")
)

var jwtSecret = []byte("api-demo-secret-change-in-production")

type User struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"-"`
}

type Claims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Exp   int64  `json:"exp"`
}

type userStore struct {
	mu    sync.RWMutex
	users map[string]User
}

var apiUsers = userStore{users: make(map[string]User)}

func createToken(email, name string) (string, error) {
	header := `{"alg":"HS256","typ":"JWT"}`
	claims := Claims{Email: email, Name: name, Exp: time.Now().Add(24 * time.Hour).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	b64header := base64.RawURLEncoding.EncodeToString([]byte(header))
	b64payload := base64.RawURLEncoding.EncodeToString(payload)
	sigInput := b64header + "." + b64payload
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(sigInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return sigInput + "." + sig, nil
}

func validateToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errInvalidToken
	}
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return nil, errInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errExpiredToken
	}
	return &claims, nil
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			return
		}
		claims, err := validateToken(cookie)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Next()
	}
}

func handleSignup(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, email, and password are required"})
		return
	}

	apiUsers.mu.Lock()
	defer apiUsers.mu.Unlock()
	if _, exists := apiUsers.users[req.Email]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	apiUsers.users[req.Email] = User{Email: req.Email, Name: req.Name, Password: req.Password}

	token, err := createToken(req.Email, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}
	setAPICookie(c, token)
	c.JSON(http.StatusCreated, gin.H{"user": gin.H{"email": req.Email, "name": req.Name}})
}

func handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	apiUsers.mu.RLock()
	user, exists := apiUsers.users[req.Email]
	apiUsers.mu.RUnlock()
	if !exists || user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	token, err := createToken(user.Email, user.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}
	setAPICookie(c, token)
	c.JSON(http.StatusOK, gin.H{"user": gin.H{"email": user.Email, "name": user.Name}})
}

func handleLogout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func handleMe(c *gin.Context) {
	email, _ := c.Get("email")
	name, _ := c.Get("name")
	c.JSON(http.StatusOK, gin.H{"user": gin.H{"email": email, "name": name}})
}

func setAPICookie(c *gin.Context, token string) {
	c.SetCookie("token", token, 86400, "/", "", true, true)
}
