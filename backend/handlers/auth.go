package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	vqauth "violin-quest-api/auth"
	"violin-quest-api/db"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login validates credentials and returns a JWT + refresh token.
// POST /api/auth/login
func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var adminID int64
	var hash string
	if err := db.DB.QueryRow(
		`SELECT id, password_hash FROM admins WHERE username = ?`, req.Username,
	).Scan(&adminID, &hash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ongeldige gebruikersnaam of wachtwoord"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ongeldige gebruikersnaam of wachtwoord"})
		return
	}

	tokenStr, err := vqauth.Sign(adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generatie mislukt"})
		return
	}

	plain, hashed, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refresh token generatie mislukt"})
		return
	}

	expiresAt := time.Now().AddDate(0, 0, 30).Format(time.RFC3339)
	if _, err := db.DB.Exec(
		`INSERT INTO refresh_tokens (admin_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		adminID, hashed, expiresAt,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenStr, "refresh_token": plain})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken exchanges a valid refresh token for a new JWT.
// POST /api/auth/refresh
func RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashed := hashToken(req.RefreshToken)

	var adminID int64
	var expiresAt string
	var revoked int
	if err := db.DB.QueryRow(
		`SELECT admin_id, expires_at, revoked FROM refresh_tokens WHERE token_hash = ?`, hashed,
	).Scan(&adminID, &expiresAt, &revoked); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ongeldig refresh token"})
		return
	}
	if revoked == 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token is ingetrokken"})
		return
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().After(expiry) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token is verlopen"})
		return
	}

	tokenStr, err := vqauth.Sign(adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generatie mislukt"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func generateRefreshToken() (plain, hashed string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	plain = hex.EncodeToString(b)
	hashed = hashToken(plain)
	return
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
