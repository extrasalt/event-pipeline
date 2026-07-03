package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func resetUsers() {
	users = userStore{users: make(map[string]User)}
}

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func hmacSHA256(secret []byte, data string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func ginTest() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestCreateAndValidateToken(t *testing.T) {
	token, err := createToken("a@b.com", "Alice")
	if err != nil {
		t.Fatalf("createToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}

	claims, err := validateToken(token)
	if err != nil {
		t.Fatalf("validateToken: %v", err)
	}
	if claims.Email != "a@b.com" || claims.Name != "Alice" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.Exp <= time.Now().Unix() {
		t.Fatal("exp should be in the future")
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	if _, err := validateToken("not-a-jwt"); err == nil {
		t.Fatal("expected error for invalid format")
	}
	if _, err := validateToken("a.b"); err == nil {
		t.Fatal("expected error for 2-part token")
	}
	if _, err := validateToken("a.b.c.d"); err == nil {
		t.Fatal("expected error for 4-part token")
	}
}

func TestValidateToken_BadSignature(t *testing.T) {
	token, _ := createToken("a@b.com", "Alice")
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + parts[1] + ".invalidsig"
	if _, err := validateToken(tampered); err == nil {
		t.Fatal("expected error for bad signature")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	old := jwtSecret
	jwtSecret = []byte("test")
	header := `{"alg":"HS256","typ":"JWT"}`
	claims := Claims{Email: "a@b.com", Name: "Alice", Exp: time.Now().Add(-1 * time.Hour).Unix()}
	payload, _ := json.Marshal(claims)
	b64h := base64URLEncode([]byte(header))
	b64p := base64URLEncode(payload)
	mac := hmacSHA256([]byte("test"), b64h+"."+b64p)
	token := b64h + "." + b64p + "." + mac
	if _, err := validateToken(token); err == nil {
		t.Fatal("expected error for expired token")
	}
	jwtSecret = old
}

func TestHandleSignup(t *testing.T) {
	resetUsers()
	c, w := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`{"name":"Alice","email":"a@b.com","password":"pass123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handleSignup(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	user := body["user"].(map[string]any)
	if user["email"] != "a@b.com" || user["name"] != "Alice" {
		t.Fatalf("unexpected user: %+v", user)
	}
	cookies := w.Result().Cookies()
	var tokenFound bool
	for _, ck := range cookies {
		if ck.Name == "token" {
			tokenFound = true
			if ck.Value == "" {
				t.Fatal("token cookie should not be empty")
			}
			break
		}
	}
	if !tokenFound {
		t.Fatal("expected token cookie")
	}
}

func TestHandleSignup_MissingFields(t *testing.T) {
	resetUsers()
	c, w := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`{"name":"","email":"","password":""}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handleSignup(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleSignup_InvalidBody(t *testing.T) {
	resetUsers()
	c, w := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`not json`))
	c.Request.Header.Set("Content-Type", "application/json")
	handleSignup(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleSignup_DuplicateEmail(t *testing.T) {
	resetUsers()
	c, _ := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`{"name":"Alice","email":"a@b.com","password":"pass123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handleSignup(c)

	c2, w2 := ginTest()
	c2.Request = httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`{"name":"Bob","email":"a@b.com","password":"pass456"}`))
	c2.Request.Header.Set("Content-Type", "application/json")
	handleSignup(c2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandleLogin(t *testing.T) {
	resetUsers()
	users.users["a@b.com"] = User{Email: "a@b.com", Name: "Alice", Password: "pass123"}
	c, w := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"a@b.com","password":"pass123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handleLogin(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	resetUsers()
	users.users["a@b.com"] = User{Email: "a@b.com", Name: "Alice", Password: "pass123"}
	c, w := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"a@b.com","password":"wrong"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handleLogin(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleLogin_UnknownUser(t *testing.T) {
	resetUsers()
	c, w := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"nobody@b.com","password":"pass"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	handleLogin(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleLogin_InvalidBody(t *testing.T) {
	resetUsers()
	c, w := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`not json`))
	c.Request.Header.Set("Content-Type", "application/json")
	handleLogin(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleLogout(t *testing.T) {
	resetUsers()
	c, w := ginTest()
	c.Request = httptest.NewRequest("POST", "/api/auth/logout", nil)
	handleLogout(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["ok"] != true {
		t.Fatal("expected ok: true")
	}
	cookies := w.Result().Cookies()
	for _, ck := range cookies {
		if ck.Name == "token" && ck.MaxAge >= 0 {
			t.Fatal("logout cookie should have negative max-age")
		}
	}
}

func TestAuthMiddleware_NoCookie(t *testing.T) {
	resetUsers()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/api/auth/me", authMiddleware, handleMe)
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_ValidCookie(t *testing.T) {
	resetUsers()
	token, _ := createToken("a@b.com", "Alice")
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/api/auth/me", authMiddleware, handleMe)
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	user := body["user"].(map[string]any)
	if user["email"] != "a@b.com" || user["name"] != "Alice" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestAuthMiddleware_ExpiredCookie(t *testing.T) {
	resetUsers()
	old := jwtSecret
	jwtSecret = []byte("test")
	header := `{"alg":"HS256","typ":"JWT"}`
	claims := Claims{Email: "a@b.com", Name: "Alice", Exp: time.Now().Add(-1 * time.Hour).Unix()}
	payload, _ := json.Marshal(claims)
	b64h := base64URLEncode([]byte(header))
	b64p := base64URLEncode(payload)
	mac := hmacSHA256([]byte("test"), b64h+"."+b64p)
	token := b64h + "." + b64p + "." + mac
	jwtSecret = old

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/api/auth/me", authMiddleware, handleMe)
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleMe(t *testing.T) {
	resetUsers()
	c, w := ginTest()
	c.Set("email", "a@b.com")
	c.Set("name", "Alice")
	c.Request = httptest.NewRequest("GET", "/api/auth/me", nil)
	handleMe(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	user := body["user"].(map[string]any)
	if user["email"] != "a@b.com" || user["name"] != "Alice" {
		t.Fatalf("unexpected user: %+v", user)
	}
}
