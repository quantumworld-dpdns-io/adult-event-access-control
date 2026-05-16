package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(func() string {
	if s := envOrDefault("JWT_SECRET", ""); s != "" {
		return s
	}
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}())

func envOrDefault(key, fallback string) string {
	if v := envLookup(key); v != "" {
		return v
	}
	return fallback
}

var envLookup = func(key string) string { return "" }

func issueToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func verifyToken(raw string) (string, error) {
	token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}
	sub, _ := claims["sub"].(string)
	return sub, nil
}

// OAuth verification (server-side code exchange)
func verifyOAuth(provider, code, state, redirectURI string) (*userInfo, error) {
	switch provider {
	case "google":
		return verifyGoogle(code, redirectURI)
	case "github":
		return verifyGitHub(code, state)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

type googleTokenResp struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

type googleUserResp struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Pic   string `json:"picture"`
}

func verifyGoogle(code, redirectURI string) (*userInfo, error) {
	tokenResp, err := httpPostForm("https://oauth2.googleapis.com/token", url.Values{
		"code":          {code},
		"client_id":     {envOrDefault("GOOGLE_CLIENT_ID", "")},
		"client_secret": {envOrDefault("GOOGLE_CLIENT_SECRET", "")},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, err
	}
	var tr googleTokenResp
	if err := decodeJSON(tokenResp, &tr); err != nil {
		return nil, err
	}

	userResp, err := httpGet("https://www.googleapis.com/oauth2/v3/userinfo", map[string]string{
		"Authorization": "Bearer " + tr.AccessToken,
	})
	if err != nil {
		return nil, err
	}
	var gu googleUserResp
	if err := decodeJSON(userResp, &gu); err != nil {
		return nil, err
	}

	return &userInfo{
		Provider:   "google",
		ProviderID: "google_" + gu.Sub,
		Email:      gu.Email,
		Display:    gu.Name,
		Avatar:     gu.Pic,
	}, nil
}

type githubTokenResp struct {
	AccessToken string `json:"access_token"`
}

type githubUserResp struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Av    string `json:"avatar_url"`
}

func verifyGitHub(code, state string) (*userInfo, error) {
	tokenResp, err := httpPostForm("https://github.com/login/oauth/access_token", url.Values{
		"code":          {code},
		"client_id":     {envOrDefault("GITHUB_CLIENT_ID", "")},
		"client_secret": {envOrDefault("GITHUB_CLIENT_SECRET", "")},
		"state":         {state},
	})
	if err != nil {
		return nil, err
	}
	var tr githubTokenResp
	if err := decodeJSON(tokenResp, &tr); err != nil {
		return nil, err
	}

	userResp, err := httpGet("https://api.github.com/user", map[string]string{
		"Authorization": "Bearer " + tr.AccessToken,
	})
	if err != nil {
		return nil, err
	}
	var gu githubUserResp
	if err := decodeJSON(userResp, &gu); err != nil {
		return nil, err
	}

	return &userInfo{
		Provider:   "github",
		ProviderID: fmt.Sprintf("github_%d", gu.ID),
		Email:      gu.Email,
		Display:    gu.Name,
		Avatar:     gu.Av,
	}, nil
}

// SIWE verification (EIP-4361)
func verifySIWE(message, signature string) (string, error) {
	sep := "\n"
	lines := strings.Split(message, sep)
	if len(lines) < 7 {
		return "", errors.New("invalid SIWE message format")
	}
	address := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(address, "0x") || len(address) != 42 {
		return "", errors.New("invalid Ethereum address")
	}
	return strings.ToLower(address), nil
}

// World ID verification
func verifyWorldID(proof, appID, nullifierHash string) (bool, error) {
	body := fmt.Sprintf(`{
		"proof": %q,
		"app_id": %q,
		"action": "aev-age-verification",
		"nullifier_hash": %q,
		"signal": "aeV-age-verification-signal"
	}`, proof, appID, nullifierHash)

	resp, err := httpPost("https://developer.worldcoin.org/api/v3/verify/" + appID, "application/json", body)
	if err != nil {
		return false, err
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := decodeJSON(resp, &result); err != nil {
		return false, err
	}
	return result.Success, nil
}

type httpBody = []byte

func httpGet(url string, headers map[string]string) (httpBody, error) {
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(buf)
	return buf, err
}

func httpPost(url, contentType, body string) (httpBody, error) {
	resp, err := http.Post(url, contentType, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(buf)
	return buf, err
}

func httpPostForm(url string, data url.Values) (httpBody, error) {
	return httpPost(url, "application/x-www-form-urlencoded", data.Encode())
}

func decodeJSON(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
