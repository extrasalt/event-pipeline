package api

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

func resetAPIUsers() {
	apiUsers = userStore{users: make(map[string]User)}
}

func TestAPICreateAndValidateToken(t *testing.T) {
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

func TestAPIValidateToken_InvalidFormat(t *testing.T) {
	if _, err := validateToken("not-a-jwt"); err == nil {
		t.Fatal("expected error for invalid format")
	}
	if _, err := validateToken("a.b"); err == nil {
		t.Fatal("expected error for 2-part token")
	}
}

func TestAPIValidateToken_BadSignature(t *testing.T) {
	token, _ := createToken("a@b.com", "Alice")
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + parts[1] + ".invalidsig"
	if _, err := validateToken(tampered); err == nil {
		t.Fatal("expected error for bad signature")
	}
}

func base64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func hmacSHA(secret []byte, data string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestAPIValidateToken_Expired(t *testing.T) {
	old := jwtSecret
	jwtSecret = []byte("test")
	header := `{"alg":"HS256","typ":"JWT"}`
	claims := Claims{Email: "a@b.com", Name: "Alice", Exp: time.Now().Add(-1 * time.Hour).Unix()}
	payload, _ := json.Marshal(claims)
	b64h := base64URL([]byte(header))
	b64p := base64URL(payload)
	sig := hmacSHA([]byte("test"), b64h+"."+b64p)
	token := b64h + "." + b64p + "." + sig
	jwtSecret = old

	if _, err := validateToken(token); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func apiTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := r.Group("/api/auth")
	auth.POST("/signup", handleSignup)
	auth.POST("/login", handleLogin)
	auth.POST("/logout", handleLogout)
	auth.GET("/me", authMiddleware(), handleMe)
	return r
}

func TestAPIHandleSignup(t *testing.T) {
	resetAPIUsers()
	r := apiTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`{"name":"Alice","email":"a@b.com","password":"pass123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	user := body["user"].(map[string]any)
	if user["email"] != "a@b.com" || user["name"] != "Alice" {
		t.Fatalf("unexpected user: %+v", user)
	}
	var tokenFound bool
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "token" && ck.Value != "" {
			tokenFound = true
			break
		}
	}
	if !tokenFound {
		t.Fatal("expected token cookie")
	}
}

func TestAPIHandleSignup_MissingFields(t *testing.T) {
	resetAPIUsers()
	r := apiTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`{"name":"","email":"","password":""}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAPIHandleSignup_DuplicateEmail(t *testing.T) {
	resetAPIUsers()
	r := apiTestRouter()

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`{"name":"Alice","email":"a@b.com","password":"pass123"}`))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/auth/signup", strings.NewReader(`{"name":"Bob","email":"a@b.com","password":"pass456"}`))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestAPIHandleLogin(t *testing.T) {
	resetAPIUsers()
	apiUsers.users["a@b.com"] = User{Email: "a@b.com", Name: "Alice", Password: "pass123"}
	r := apiTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"a@b.com","password":"pass123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIHandleLogin_WrongPassword(t *testing.T) {
	resetAPIUsers()
	apiUsers.users["a@b.com"] = User{Email: "a@b.com", Name: "Alice", Password: "pass123"}
	r := apiTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"a@b.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAPIHandleLogin_UnknownUser(t *testing.T) {
	resetAPIUsers()
	r := apiTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"email":"nobody@b.com","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAPIHandleLogout(t *testing.T) {
	resetAPIUsers()
	r := apiTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["ok"] != true {
		t.Fatal("expected ok: true")
	}
}

func TestAPIAuthMiddleware_NoCookie(t *testing.T) {
	resetAPIUsers()
	r := apiTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIAuthMiddleware_ValidCookie(t *testing.T) {
	resetAPIUsers()
	token, _ := createToken("a@b.com", "Alice")
	r := apiTestRouter()
	w := httptest.NewRecorder()
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

func TestAPIAuthMiddleware_ExpiredCookie(t *testing.T) {
	resetAPIUsers()
	old := jwtSecret
	jwtSecret = []byte("test")
	header := `{"alg":"HS256","typ":"JWT"}`
	claims := Claims{Email: "a@b.com", Name: "Alice", Exp: time.Now().Add(-1 * time.Hour).Unix()}
	payload, _ := json.Marshal(claims)
	b64h := base64URL([]byte(header))
	b64p := base64URL(payload)
	sig := hmacSHA([]byte("test"), b64h+"."+b64p)
	token := b64h + "." + b64p + "." + sig
	jwtSecret = old

	r := apiTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
