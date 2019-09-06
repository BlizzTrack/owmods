package routes

import (
	"errors"
	"fmt"
	"github.com/kataras/iris"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/battlenet"
	"github.com/blizztrack/owmods/database"
	"github.com/blizztrack/owmods/system"
	"gopkg.in/alecthomas/kingpin.v2"
)

type authRoutes struct{}

var (
	clientID     = kingpin.Flag("client_id", "Battle.net Client ID").Envar("CLIENT_ID").Default("").String()
	clientSecret = kingpin.Flag("client_secret", "Battle.net Client Secret").Envar("CLIENT_SECRET").Default("").String()
	hostServer   = kingpin.Flag("client_host_server", "Battle.net Client Host Server").Envar("CLIENT_HOST").Default("http://127.0.0.1:1337").String()
)

const oauth = "battlenet"

func NewAuthRoutes(party iris.Party) {
	a := authRoutes{}

	party.Get("/start", a.authStart)
	party.Get("/callback", a.authCallBack)
	party.Get("/leave", a.authLogout)

	goth.UseProviders(
		battlenet.New(*clientID, *clientSecret, fmt.Sprintf("%s/auth/callback", *hostServer)),
	)
}

func (authRoutes) authStart(ctx iris.Context) {
	if gothUser, err := completeUserAuth(ctx); err == nil {
		user := database.UpsertUser(gothUser)
		system.Session().Get(ctx).Set("User", user)
		system.Session().Get(ctx).Set("UserID", user.ID.Hex())

		ctx.Redirect("/")
	} else {
		beginAuthHandler(ctx)
	}
}

func (authRoutes) authCallBack(ctx iris.Context) {
	gothUser, err := completeUserAuth(ctx)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.Writef("%v", err)
		return
	}

	user := database.UpsertUser(gothUser)
	system.Session().Get(ctx).Set("User", user)
	system.Session().Get(ctx).Set("UserID", user.ID.Hex())

	ctx.Redirect("/")
}


func (authRoutes) authLogout(ctx iris.Context) {
	sessionsManager := system.Session()
	session := sessionsManager.Get(ctx)
	session.Destroy()

	ctx.Redirect("/")
}

func beginAuthHandler(ctx iris.Context) {
	url, err := getAuthURL(ctx)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.Writef("%v", err)
		return
	}

	ctx.Redirect(url, iris.StatusTemporaryRedirect)
}

func getAuthURL(ctx iris.Context) (string, error) {
	providerName := oauth
	provider, err := goth.GetProvider(providerName)
	if err != nil {
		return "", err
	}
	sess, err := provider.BeginAuth(setState(ctx))
	if err != nil {
		return "", err
	}

	url, err := sess.GetAuthURL()
	if err != nil {
		return "", err
	}
	session := system.Session().Get(ctx)
	session.Set(providerName, sess.Marshal())
	return url, nil
}

var setState = func(ctx iris.Context) string {
	state := ctx.URLParam("state")
	if len(state) > 0 {
		return state
	}

	return "state"
}

var getState = func(ctx iris.Context) string {
	return ctx.URLParam("state")
}

var completeUserAuth = func(ctx iris.Context) (goth.User, error) {
	providerName := oauth
	sessionsManager := system.Session()

	provider, err := goth.GetProvider(providerName)
	if err != nil {
		return goth.User{}, err
	}

	session := sessionsManager.Get(ctx)
	value := session.GetString(providerName)
	if value == "" {
		return goth.User{}, errors.New("session value for " + providerName + " not found")
	}

	sess, err := provider.UnmarshalSession(value)
	if err != nil {
		return goth.User{}, err
	}

	user, err := provider.FetchUser(sess)
	if err == nil {
		// user can be found with existing session data
		return user, err
	}

	// get new token and retry fetch
	_, err = sess.Authorize(provider, ctx.Request().URL.Query())
	if err != nil {
		return goth.User{}, err
	}

	session.Set(providerName, sess.Marshal())
	return provider.FetchUser(sess)
}
