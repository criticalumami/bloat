package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"mastodon"
	"web/model"
	"web/renderer"
	"web/util"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidClient   = errors.New("invalid client")
)

type Service interface {
	ServeHomePage(ctx context.Context, client io.Writer) (err error)
	GetAuthUrl(ctx context.Context, instance string) (url string, sessionID string, err error)
	GetUserToken(ctx context.Context, sessionID string, c *mastodon.Client, token string) (accessToken string, err error)
	ServeErrorPage(ctx context.Context, client io.Writer, err error)
	ServeSigninPage(ctx context.Context, client io.Writer) (err error)
	ServeTimelinePage(ctx context.Context, client io.Writer, c *mastodon.Client, maxID string, sinceID string, minID string) (err error)
	ServeThreadPage(ctx context.Context, client io.Writer, c *mastodon.Client, id string, reply bool) (err error)
	Like(ctx context.Context, client io.Writer, c *mastodon.Client, id string) (err error)
	UnLike(ctx context.Context, client io.Writer, c *mastodon.Client, id string) (err error)
	Retweet(ctx context.Context, client io.Writer, c *mastodon.Client, id string) (err error)
	UnRetweet(ctx context.Context, client io.Writer, c *mastodon.Client, id string) (err error)
	PostTweet(ctx context.Context, client io.Writer, c *mastodon.Client, content string, replyToID string) (err error)
}

type service struct {
	clientName    string
	clientScope   string
	clientWebsite string
	renderer      renderer.Renderer
	sessionRepo   model.SessionRepository
	appRepo       model.AppRepository
}

func NewService(clientName string, clientScope string, clientWebsite string,
	renderer renderer.Renderer, sessionRepo model.SessionRepository,
	appRepo model.AppRepository) Service {
	return &service{
		clientName:    clientName,
		clientScope:   clientScope,
		clientWebsite: clientWebsite,
		renderer:      renderer,
		sessionRepo:   sessionRepo,
		appRepo:       appRepo,
	}
}

func (svc *service) GetAuthUrl(ctx context.Context, instance string) (
	redirectUrl string, sessionID string, err error) {
	if !strings.HasPrefix(instance, "https://") {
		instance = "https://" + instance
	}

	sessionID = util.NewSessionId()
	err = svc.sessionRepo.Add(model.Session{
		ID:          sessionID,
		InstanceURL: instance,
	})
	if err != nil {
		return
	}

	app, err := svc.appRepo.Get(instance)
	if err != nil {
		if err != model.ErrAppNotFound {
			return
		}

		var mastoApp *mastodon.Application
		mastoApp, err = mastodon.RegisterApp(ctx, &mastodon.AppConfig{
			Server:       instance,
			ClientName:   svc.clientName,
			Scopes:       svc.clientScope,
			Website:      svc.clientWebsite,
			RedirectURIs: svc.clientWebsite + "/oauth_callback",
		})
		if err != nil {
			return
		}

		app = model.App{
			InstanceURL:  instance,
			ClientID:     mastoApp.ClientID,
			ClientSecret: mastoApp.ClientSecret,
		}

		err = svc.appRepo.Add(app)
		if err != nil {
			return
		}
	}

	u, err := url.Parse(path.Join(instance, "/oauth/authorize"))
	if err != nil {
		return
	}

	q := make(url.Values)
	q.Set("scope", "read write follow")
	q.Set("client_id", app.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", svc.clientWebsite+"/oauth_callback")
	u.RawQuery = q.Encode()

	redirectUrl = u.String()

	return
}

func (svc *service) GetUserToken(ctx context.Context, sessionID string, c *mastodon.Client,
	code string) (token string, err error) {
	if len(code) < 1 {
		err = ErrInvalidArgument
		return
	}

	session, err := svc.sessionRepo.Get(sessionID)
	if err != nil {
		return
	}

	app, err := svc.appRepo.Get(session.InstanceURL)
	if err != nil {
		return
	}

	data := &bytes.Buffer{}
	err = json.NewEncoder(data).Encode(map[string]string{
		"client_id":     app.ClientID,
		"client_secret": app.ClientSecret,
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  svc.clientWebsite + "/oauth_callback",
	})
	if err != nil {
		return
	}

	resp, err := http.Post(app.InstanceURL+"/oauth/token", "application/json", data)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var res struct {
		AccessToken string `json:"access_token"`
	}

	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return
	}
	/*
		err = c.AuthenticateToken(ctx, code, svc.clientWebsite+"/oauth_callback")
		if err != nil {
			return
		}
		err = svc.sessionRepo.Update(sessionID, c.GetAccessToken(ctx))
	*/

	return res.AccessToken, nil
}

func (svc *service) ServeHomePage(ctx context.Context, client io.Writer) (err error) {
	err = svc.renderer.RenderHomePage(ctx, client)
	if err != nil {
		return
	}

	return
}

func (svc *service) ServeErrorPage(ctx context.Context, client io.Writer, err error) {
	svc.renderer.RenderErrorPage(ctx, client, err)
}

func (svc *service) ServeSigninPage(ctx context.Context, client io.Writer) (err error) {
	err = svc.renderer.RenderSigninPage(ctx, client)
	if err != nil {
		return
	}

	return
}

func (svc *service) ServeTimelinePage(ctx context.Context, client io.Writer,
	c *mastodon.Client, maxID string, sinceID string, minID string) (err error) {

	var hasNext, hasPrev bool
	var nextLink, prevLink string

	var pg = mastodon.Pagination{
		MaxID:   maxID,
		SinceID: sinceID,
		MinID:   minID,
		Limit:   20,
	}

	statuses, err := c.GetTimelineHome(ctx, &pg)
	if err != nil {
		return err
	}

	if len(pg.MaxID) > 0 {
		hasNext = true
		nextLink = fmt.Sprintf("/timeline?max_id=%s", pg.MaxID)
	}
	if len(pg.SinceID) > 0 {
		hasPrev = true
		prevLink = fmt.Sprintf("/timeline?since_id=%s", pg.SinceID)
	}

	data := renderer.NewTimelinePageTemplateData(statuses, hasNext, nextLink, hasPrev, prevLink)
	err = svc.renderer.RenderTimelinePage(ctx, client, data)
	if err != nil {
		return
	}

	return
}

func (svc *service) ServeThreadPage(ctx context.Context, client io.Writer, c *mastodon.Client, id string, reply bool) (err error) {
	status, err := c.GetStatus(ctx, id)
	if err != nil {
		return
	}

	context, err := c.GetStatusContext(ctx, id)
	if err != nil {
		return
	}

	var content string
	if reply {
		content += status.Account.Acct + " "
		for _, m := range status.Mentions {
			content += m.Acct + " "
		}
	}

	fmt.Println("content", content)

	data := renderer.NewThreadPageTemplateData(status, context, reply, id, content)
	err = svc.renderer.RenderThreadPage(ctx, client, data)
	if err != nil {
		return
	}

	return
}

func (svc *service) Like(ctx context.Context, client io.Writer, c *mastodon.Client, id string) (err error) {
	_, err = c.Favourite(ctx, id)
	return
}

func (svc *service) UnLike(ctx context.Context, client io.Writer, c *mastodon.Client, id string) (err error) {
	_, err = c.Unfavourite(ctx, id)
	return
}

func (svc *service) Retweet(ctx context.Context, client io.Writer, c *mastodon.Client, id string) (err error) {
	_, err = c.Reblog(ctx, id)
	return
}

func (svc *service) UnRetweet(ctx context.Context, client io.Writer, c *mastodon.Client, id string) (err error) {
	_, err = c.Unreblog(ctx, id)
	return
}

func (svc *service) PostTweet(ctx context.Context, client io.Writer, c *mastodon.Client, content string, replyToID string) (err error) {
	tweet := &mastodon.Toot{
		Status:      content,
		InReplyToID: replyToID,
	}
	_, err = c.PostStatus(ctx, tweet)
	return
}
