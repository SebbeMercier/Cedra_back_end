package handlers

import (
	"os"

	"cedra_back_end/internal/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var Providers = map[string]*auth.OAuthProvider{}

func InitProviders() {
	Providers["google"] = &auth.OAuthProvider{
		Name: "google",
		Config: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
			Scopes:       []string{"email"},
			Endpoint:     google.Endpoint,
		},
	}

	Providers["facebook"] = &auth.OAuthProvider{
		Name: "facebook",
		Config: &oauth2.Config{
			ClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
			ClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("FACEBOOK_REDIRECT_URL"),
			Scopes:       []string{"email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.facebook.com/v15.0/dialog/oauth",
				TokenURL: "https://graph.facebook.com/v15.0/oauth/access_token",
			},
		},
	}

	Providers["tiktok"] = &auth.OAuthProvider{
		Name: "tiktok",
		Config: &oauth2.Config{
			ClientID:     os.Getenv("TIKTOK_CLIENT_ID"),
			ClientSecret: os.Getenv("TIKTOK_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("TIKTOK_REDIRECT_URL"),
			Scopes:       []string{"user.info.basic"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.tiktok.com/auth/authorize/",
				TokenURL: "https://open-api.tiktokglobalshop.com/oauth/access_token",
			},
		},
	}

	// Apple sera différent (OpenID Connect), on peut l'ajouter après
}
