package system

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

type sessionHandler struct {
	session *sessions.Sessions
}

var lSession *sessionHandler

func newSessionHandler() *sessionHandler {
	sess := sessions.New(sessions.Config{
		Cookie:       "owmods.session",
		Expires:      0, // defaults to 0: unlimited life. Another good value is: 45 * time.Minute,
		AllowReclaim: true,
	})
	sess.UseDatabase(Redis().Client())

	return &sessionHandler{session: sess}
}

func Session() *sessionHandler {
	if lSession == nil {
		lSession = newSessionHandler()
	}

	return lSession
}

func (s *sessionHandler) Handler() *sessions.Sessions {
	return s.session
}

func (s *sessionHandler) Get(ctx iris.Context) *sessions.Session{
	return s.session.Start(ctx)
}