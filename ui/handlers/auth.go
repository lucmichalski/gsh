package handlers

import (
	"net/http"

	oidc "github.com/coreos/go-oidc"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/gommon/random"
	"golang.org/x/oauth2"
)

// Auth is a method that provides authentication flow using OpenID Connect
func (h AppHandler) Auth(c echo.Context) error {
	sess, _ := session.Get("gsh", c)

	// generate radom state
	state := random.String(32)
	sess.Values["state"] = state

	// save session
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusFound, h.oauth2config.AuthCodeURL(state))
}

// AuthCallback is a method that provides authentication flow using OpenID Connect
func (h AppHandler) AuthCallback(c echo.Context) error {
	sess, _ := session.Get("gsh", c)

	// verify state
	stateCookie := sess.Values["state"].(string)
	stateOauth2 := c.QueryParam("state")
	if stateCookie != stateOauth2 {
		return c.String(http.StatusInternalServerError, "Invalid state")
	}

	oauth2Token, err := h.oauth2config.Exchange(c.Request().Context(), c.Request().URL.Query().Get("code"))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error converting an authorization code into a token")
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return c.String(http.StatusInternalServerError, "Missing token")
	}

	// Parse and verify ID Token payload.
	oauth2verifier := h.oauth2provider.Verifier(&oidc.Config{ClientID: h.config.GetString("AUTH_RESOURCE")})
	_, err = oauth2verifier.Verify(c.Request().Context(), rawIDToken)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Verify id_token error")
	}

	// Get info about user
	userInfo, err := h.oauth2provider.UserInfo(c.Request().Context(), oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Verify id_token error")
	}
	claims := map[string]string{}
	userInfo.Claims(&claims)

	// save session values
	sess.Values["rawIDToken"] = rawIDToken
	sess.Values["subject"] = userInfo.Subject
	sess.Values[h.config.GetString("AUTH_USERNAME_CLAIM")] = claims[h.config.GetString("AUTH_USERNAME_CLAIM")]

	// save session
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusFound, "/")
}

// AuthLogout is a method that logout user expiring gsh cookie
func (h AppHandler) AuthLogout(c echo.Context) error {
	sess, _ := session.Get("gsh", c)

	// expire cookie
	sess.Options.MaxAge = -1

	// save session
	sess.Save(c.Request(), c.Response())

	// Please note the the second parameter "logout.html" is the template name and should
	// be equal to one of the keys in the TemplateRegistry array defined in main.go
	return c.Render(http.StatusOK, "logout.html", map[string]interface{}{
		"name": "You are logged out!",
	})
}