package authflow

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	callbackHost          = "127.0.0.1"
	callbackPath          = "/callback"
	defaultLoginTimeout   = 2 * time.Minute
	callbackResponseBody  = "Authentication received. Return to the Code-Find CLI."
	maxCallbackBodyBytes  = 1 << 20
)

type BrowserOpener func(string) error

type TokenClaims struct {
	OrgID   string `json:"org_id"`
	OrgRole string `json:"org_role"`
	Exp     int64  `json:"exp"`
	Org     struct {
		ID   string `json:"id"`
		Role string `json:"rol"`
	} `json:"o"`
}

type callbackPayload struct {
	Token string `json:"token"`
}

func DefaultBrowserOpener(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func BuildSignInURL(serverURL, redirectURI string) (string, error) {
	if strings.TrimSpace(serverURL) == "" {
		return "", errors.New("server URL is required")
	}
	if strings.TrimSpace(redirectURI) == "" {
		return "", errors.New("redirect URI is required")
	}

	base, err := url.Parse(serverURL)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/auth/signin"
	query := base.Query()
	query.Set("redirect_uri", redirectURI)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func StartCallbackServer(
	ctx context.Context,
	listener net.Listener,
	allowedOrigin string,
) (redirectURI string, waitForToken func() (string, error), err error) {
	if listener == nil {
		return "", nil, errors.New("listener is required")
	}
	if strings.TrimSpace(allowedOrigin) == "" {
		return "", nil, errors.New("allowed origin is required")
	}

	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{
		Handler: callbackHandler(tokenCh, allowedOrigin),
	}

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	redirectURI = "http://" + listener.Addr().String() + callbackPath
	waitForToken = func() (string, error) {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
		}()

		select {
		case token := <-tokenCh:
			return token, nil
		case serveErr := <-errCh:
			return "", fmt.Errorf("serve callback: %w", serveErr)
		case <-ctx.Done():
			return "", fmt.Errorf("wait for auth callback: %w", ctx.Err())
		}
	}

	return redirectURI, waitForToken, nil
}

func NewLocalCallbackListener() (net.Listener, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(callbackHost, "0"))
	if err != nil {
		return nil, fmt.Errorf("listen for auth callback: %w", err)
	}
	return listener, nil
}

func LoginTimeout() time.Duration {
	return defaultLoginTimeout
}

func DecodeTokenClaims(token string) (TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenClaims{}, errors.New("token must have three JWT segments")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return TokenClaims{}, fmt.Errorf("decode token payload: %w", err)
	}

	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return TokenClaims{}, fmt.Errorf("parse token claims: %w", err)
	}
	if claims.OrgID == "" {
		claims.OrgID = claims.Org.ID
	}
	if claims.OrgRole == "" {
		claims.OrgRole = normalizeOrgRole(claims.Org.Role)
	} else {
		claims.OrgRole = normalizeOrgRole(claims.OrgRole)
	}
	return claims, nil
}

func normalizeOrgRole(role string) string {
	switch role {
	case "admin", "org:admin":
		return "org:admin"
	case "member", "org:member":
		return "org:member"
	default:
		return role
	}
}

func TokenExpiryTime(token string) (time.Time, error) {
	claims, err := DecodeTokenClaims(token)
	if err != nil {
		return time.Time{}, err
	}
	if claims.Exp == 0 {
		return time.Time{}, errors.New("token is missing exp claim")
	}
	return time.Unix(claims.Exp, 0).UTC(), nil
}

func callbackHandler(tokenCh chan<- string, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodOptions {
			if !originAllowed(r, allowedOrigin) {
				http.Error(w, "Origin not allowed.", http.StatusForbidden)
				return
			}
			setCorsHeaders(w, allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", http.MethodPost+", "+http.MethodOptions)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}
		if !originAllowed(r, allowedOrigin) {
			http.Error(w, "Origin not allowed.", http.StatusForbidden)
			return
		}

		body := http.MaxBytesReader(w, r.Body, maxCallbackBodyBytes)
		defer body.Close()

		var payload callbackPayload
		if err := json.NewDecoder(body).Decode(&payload); err != nil {
			http.Error(w, "Invalid callback payload.", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(payload.Token) == "" {
			http.Error(w, "Callback token is required.", http.StatusBadRequest)
			return
		}

		select {
		case tokenCh <- payload.Token:
		default:
			http.Error(w, "Callback already received.", http.StatusConflict)
			return
		}

		setCorsHeaders(w, allowedOrigin)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, callbackResponseBody)
	})
}

func originAllowed(r *http.Request, allowedOrigin string) bool {
	return r.Header.Get("Origin") == allowedOrigin
}

func setCorsHeaders(w http.ResponseWriter, allowedOrigin string) {
	w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
	w.Header().Set("Vary", "Origin")
}
