package internal

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type Session struct {
	AK     string `json:"ak"`
	RK     string `json:"rk"`
	Expire int    `json:"ak_expire"`
	Email  string `json:"email"`
}

func checkRefreshGHStatus(r *http.Request) (*Session, error) {
	session := getCookieSession(r)
	if session == nil {
		return nil, fmt.Errorf("browser session is nil")
	}

	if time.Now().Unix() > int64(session.Expire) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func getCookieSession(r *http.Request) *Session {
	s, _ := r.Cookie("s")
	if s == nil || s.Value == "" {
		return nil
	}

	session, err := decryptSession(s.Value)
	if err != nil {
		log.Infof("decrypt session error: %v", err)
		return nil
	}

	return &session
}

func setCookieSession(w http.ResponseWriter, name string, session Session) {
	encryptSess, err := encryptSession(session)
	if err != nil {
		log.Infof("encrypt session error: %v", err)
		return
	}

	cookie := makeCookie(name, encryptSess, 24*60*60*7)

	log.Infof("set cookie: %s, session: %+v\n", cookie.String(), session)
	http.SetCookie(w, cookie)
}

func makeCookie(name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		HttpOnly: true,
		Secure:   strings.HasPrefix(SiteURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	}
}

func newOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func getGithubAccessToken(code, rk, clientID, clientSecret string) (string, string, int) {
	params := map[string]string{"client_id": clientID, "client_secret": clientSecret}
	if rk != "" {
		params["refresh_token"] = rk
		params["grant_type"] = "refresh_token"
	} else {
		params["code"] = code
	}

	rbody, _ := json.Marshal(params)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(rbody))
	if err != nil {
		log.Infof("Error: %s\n", err)
		return "", "", 0
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		log.Infof("Error: %s\n", resperr)
		return "", "", 0
	}

	type githubAKResp struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}

	var ghresp githubAKResp

	err = json.NewDecoder(resp.Body).Decode(&ghresp)
	if err != nil {
		log.Infof("Error: %s\n", err)
		return "", "", 0
	}

	return ghresp.AccessToken, ghresp.RefreshToken, ghresp.ExpiresIn
}

func getGithubData(accessToken string) (string, string) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", ""
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", ""
	}

	type githubDataResp struct {
		Login string `json:"login"`
		Email string `json:"email"`
	}

	var ghresp githubDataResp

	err = json.NewDecoder(resp.Body).Decode(&ghresp)
	if err != nil {
		return "", ""
	}

	// GitHub 用户未公开邮箱时，用 login@users.noreply.github.com 作为稳定身份
	if ghresp.Email == "" {
		ghresp.Email = githubIdentityEmail(ghresp.Login)
	}

	log.Infof("github data: %+v", ghresp)
	return ghresp.Login, ghresp.Email
}

// githubIdentityEmail 把 GitHub 用户名转成 RSSy 内部使用的邮箱身份。
func githubIdentityEmail(login string) string {
	if login == "" {
		return ""
	}
	if strings.Contains(login, "@") {
		return login
	}
	return login + "@users.noreply.github.com"
}

func encryptSession(session Session) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("could not marshal: %v", err)
	}

	return encryptData(data)
}

func decryptSession(str string) (Session, error) {
	var session Session

	data, err := decryptStr(str)
	if err != nil {
		return session, fmt.Errorf("could not decrypt: %v", err)
	}

	err = json.Unmarshal(data, &session)
	if err != nil {
		return session, fmt.Errorf("could not unmarshal: %v", err)
	}

	return session, nil
}
