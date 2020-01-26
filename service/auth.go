package service

import (
	"context"
	"errors"
	"io"
	"mime/multipart"

	"bloat/model"
	"mastodon"
)

var (
	ErrInvalidSession   = errors.New("invalid session")
	ErrInvalidCSRFToken = errors.New("invalid csrf token")
)

type authService struct {
	sessionRepo model.SessionRepository
	appRepo     model.AppRepository
	Service
}

func NewAuthService(sessionRepo model.SessionRepository, appRepo model.AppRepository, s Service) Service {
	return &authService{sessionRepo, appRepo, s}
}

func (s *authService) getClient(ctx context.Context) (c *model.Client, err error) {
	sessionID, ok := ctx.Value("session_id").(string)
	if !ok || len(sessionID) < 1 {
		return nil, ErrInvalidSession
	}
	session, err := s.sessionRepo.Get(sessionID)
	if err != nil {
		return nil, ErrInvalidSession
	}
	client, err := s.appRepo.Get(session.InstanceDomain)
	if err != nil {
		return
	}
	mc := mastodon.NewClient(&mastodon.Config{
		Server:       client.InstanceURL,
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		AccessToken:  session.AccessToken,
	})
	c = &model.Client{Client: mc, Session: session}
	return c, nil
}

func checkCSRF(ctx context.Context, c *model.Client) (err error) {
	csrfToken, ok := ctx.Value("csrf_token").(string)
	if !ok || csrfToken != c.Session.CSRFToken {
		return ErrInvalidCSRFToken
	}
	return nil
}

func (s *authService) GetAuthUrl(ctx context.Context, instance string) (
	redirectUrl string, sessionID string, err error) {
	return s.Service.GetAuthUrl(ctx, instance)
}

func (s *authService) GetUserToken(ctx context.Context, sessionID string, c *model.Client,
	code string) (token string, err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}

	token, err = s.Service.GetUserToken(ctx, c.Session.ID, c, code)
	if err != nil {
		return
	}

	c.Session.AccessToken = token
	err = s.sessionRepo.Add(c.Session)
	if err != nil {
		return
	}

	return
}

func (s *authService) ServeErrorPage(ctx context.Context, client io.Writer, c *model.Client, err error) {
	c, _ = s.getClient(ctx)
	s.Service.ServeErrorPage(ctx, client, c, err)
}

func (s *authService) ServeSigninPage(ctx context.Context, client io.Writer) (err error) {
	return s.Service.ServeSigninPage(ctx, client)
}

func (s *authService) ServeTimelinePage(ctx context.Context, client io.Writer,
	c *model.Client, timelineType string, maxID string, sinceID string, minID string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeTimelinePage(ctx, client, c, timelineType, maxID, sinceID, minID)
}

func (s *authService) ServeThreadPage(ctx context.Context, client io.Writer, c *model.Client, id string, reply bool) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeThreadPage(ctx, client, c, id, reply)
}

func (s *authService) ServeNotificationPage(ctx context.Context, client io.Writer, c *model.Client, maxID string, minID string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeNotificationPage(ctx, client, c, maxID, minID)
}

func (s *authService) ServeUserPage(ctx context.Context, client io.Writer, c *model.Client, id string, maxID string, minID string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeUserPage(ctx, client, c, id, maxID, minID)
}

func (s *authService) ServeAboutPage(ctx context.Context, client io.Writer, c *model.Client) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeAboutPage(ctx, client, c)
}

func (s *authService) ServeEmojiPage(ctx context.Context, client io.Writer, c *model.Client) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeEmojiPage(ctx, client, c)
}

func (s *authService) ServeLikedByPage(ctx context.Context, client io.Writer, c *model.Client, id string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeLikedByPage(ctx, client, c, id)
}

func (s *authService) ServeRetweetedByPage(ctx context.Context, client io.Writer, c *model.Client, id string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeRetweetedByPage(ctx, client, c, id)
}

func (s *authService) ServeFollowingPage(ctx context.Context, client io.Writer, c *model.Client, id string, maxID string, minID string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeFollowingPage(ctx, client, c, id, maxID, minID)
}

func (s *authService) ServeFollowersPage(ctx context.Context, client io.Writer, c *model.Client, id string, maxID string, minID string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeFollowersPage(ctx, client, c, id, maxID, minID)
}

func (s *authService) ServeSearchPage(ctx context.Context, client io.Writer, c *model.Client, q string, qType string, offset int) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeSearchPage(ctx, client, c, q, qType, offset)
}

func (s *authService) ServeSettingsPage(ctx context.Context, client io.Writer, c *model.Client) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	return s.Service.ServeSettingsPage(ctx, client, c)
}

func (s *authService) SaveSettings(ctx context.Context, client io.Writer, c *model.Client, settings *model.Settings) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	err = checkCSRF(ctx, c)
	if err != nil {
		return
	}
	return s.Service.SaveSettings(ctx, client, c, settings)
}

func (s *authService) Like(ctx context.Context, client io.Writer, c *model.Client, id string) (count int64, err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	err = checkCSRF(ctx, c)
	if err != nil {
		return
	}
	return s.Service.Like(ctx, client, c, id)
}

func (s *authService) UnLike(ctx context.Context, client io.Writer, c *model.Client, id string) (count int64, err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	err = checkCSRF(ctx, c)
	if err != nil {
		return
	}
	return s.Service.UnLike(ctx, client, c, id)
}

func (s *authService) Retweet(ctx context.Context, client io.Writer, c *model.Client, id string) (count int64, err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	err = checkCSRF(ctx, c)
	if err != nil {
		return
	}
	return s.Service.Retweet(ctx, client, c, id)
}

func (s *authService) UnRetweet(ctx context.Context, client io.Writer, c *model.Client, id string) (count int64, err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	err = checkCSRF(ctx, c)
	if err != nil {
		return
	}
	return s.Service.UnRetweet(ctx, client, c, id)
}

func (s *authService) PostTweet(ctx context.Context, client io.Writer, c *model.Client, content string, replyToID string, format string, visibility string, isNSFW bool, files []*multipart.FileHeader) (id string, err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	err = checkCSRF(ctx, c)
	if err != nil {
		return
	}
	return s.Service.PostTweet(ctx, client, c, content, replyToID, format, visibility, isNSFW, files)
}

func (s *authService) Follow(ctx context.Context, client io.Writer, c *model.Client, id string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	err = checkCSRF(ctx, c)
	if err != nil {
		return
	}
	return s.Service.Follow(ctx, client, c, id)
}

func (s *authService) UnFollow(ctx context.Context, client io.Writer, c *model.Client, id string) (err error) {
	c, err = s.getClient(ctx)
	if err != nil {
		return
	}
	err = checkCSRF(ctx, c)
	if err != nil {
		return
	}
	return s.Service.UnFollow(ctx, client, c, id)
}
