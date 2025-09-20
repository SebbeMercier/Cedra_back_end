package auth

import (
	"context"

	"golang.org/x/oauth2"
)

type OAuthProvider struct {
	Name   string
	Config *oauth2.Config
}

func (p *OAuthProvider) GetAuthURL(state string) string {
	return p.Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *OAuthProvider) Exchange(code string) (*oauth2.Token, error) {
	return p.Config.Exchange(context.Background(), code)
}
