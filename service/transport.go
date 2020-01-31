package service

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"time"

	"bloat/model"

	"github.com/gorilla/mux"
)

func newClient(w io.Writer) *model.Client {
	return &model.Client{
		Writer: w,
	}
}

func newCtxWithSesion(req *http.Request) context.Context {
	ctx := context.Background()
	sessionID, err := req.Cookie("session_id")
	if err != nil {
		return ctx
	}
	return context.WithValue(ctx, "session_id", sessionID.Value)
}

func newCtxWithSesionCSRF(req *http.Request, csrfToken string) context.Context {
	ctx := newCtxWithSesion(req)
	return context.WithValue(ctx, "csrf_token", csrfToken)
}

func getMultipartFormValue(mf *multipart.Form, key string) (val string) {
	vals, ok := mf.Value[key]
	if !ok {
		return ""
	}
	if len(vals) < 1 {
		return ""
	}
	return vals[0]
}

func serveJson(w io.Writer, data interface{}) (err error) {
	var d = make(map[string]interface{})
	d["data"] = data
	return json.NewEncoder(w).Encode(d)
}

func serveJsonError(w http.ResponseWriter, err error) {
	var d = make(map[string]interface{})
	d["error"] = err.Error()
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(d)
	return
}

func NewHandler(s Service, staticDir string) http.Handler {
	r := mux.NewRouter()

	rootPage := func(w http.ResponseWriter, req *http.Request) {
		sessionID, _ := req.Cookie("session_id")

		location := "/signin"
		if sessionID != nil && len(sessionID.Value) > 0 {
			location = "/timeline/home"
		}

		w.Header().Add("Location", location)
		w.WriteHeader(http.StatusFound)
	}

	signinPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := context.Background()
		err := s.ServeSigninPage(ctx, c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	timelinePage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		tType, _ := mux.Vars(req)["type"]
		maxID := req.URL.Query().Get("max_id")
		minID := req.URL.Query().Get("min_id")

		err := s.ServeTimelinePage(ctx, c, tType, maxID, minID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	timelineOldPage := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Location", "/timeline/home")
		w.WriteHeader(http.StatusFound)
	}

	threadPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		id, _ := mux.Vars(req)["id"]
		reply := req.URL.Query().Get("reply")

		err := s.ServeThreadPage(ctx, c, id, len(reply) > 1)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	likedByPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		id, _ := mux.Vars(req)["id"]

		err := s.ServeLikedByPage(ctx, c, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	retweetedByPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		id, _ := mux.Vars(req)["id"]

		err := s.ServeRetweetedByPage(ctx, c, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	followingPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		id, _ := mux.Vars(req)["id"]
		maxID := req.URL.Query().Get("max_id")
		minID := req.URL.Query().Get("min_id")

		err := s.ServeFollowingPage(ctx, c, id, maxID, minID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	followersPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		id, _ := mux.Vars(req)["id"]
		maxID := req.URL.Query().Get("max_id")
		minID := req.URL.Query().Get("min_id")

		err := s.ServeFollowersPage(ctx, c, id, maxID, minID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	notificationsPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		maxID := req.URL.Query().Get("max_id")
		minID := req.URL.Query().Get("min_id")

		err := s.ServeNotificationPage(ctx, c, maxID, minID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	userPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		id, _ := mux.Vars(req)["id"]
		maxID := req.URL.Query().Get("max_id")
		minID := req.URL.Query().Get("min_id")

		err := s.ServeUserPage(ctx, c, id, maxID, minID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	userSearchPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		id, _ := mux.Vars(req)["id"]
		q := req.URL.Query().Get("q")
		offsetStr := req.URL.Query().Get("offset")

		var offset int
		var err error
		if len(offsetStr) > 1 {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				s.ServeErrorPage(ctx, c, err)
				return
			}
		}

		err = s.ServeUserSearchPage(ctx, c, id, q, offset)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	aboutPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)

		err := s.ServeAboutPage(ctx, c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	emojisPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)

		err := s.ServeEmojiPage(ctx, c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	searchPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		q := req.URL.Query().Get("q")
		qType := req.URL.Query().Get("type")
		offsetStr := req.URL.Query().Get("offset")

		var offset int
		var err error
		if len(offsetStr) > 1 {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				s.ServeErrorPage(ctx, c, err)
				return
			}
		}

		err = s.ServeSearchPage(ctx, c, q, qType, offset)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	settingsPage := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)

		err := s.ServeSettingsPage(ctx, c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}
	}

	signin := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := context.Background()
		instance := req.FormValue("instance")

		url, sessionID, err := s.NewSession(ctx, instance)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "session_id",
			Value:   sessionID,
			Expires: time.Now().Add(365 * 24 * time.Hour),
		})

		w.Header().Add("Location", url)
		w.WriteHeader(http.StatusFound)
	}

	oauthCallback := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesion(req)
		token := req.URL.Query().Get("code")

		_, err := s.Signin(ctx, c, "", token)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		w.Header().Add("Location", "/timeline/home")
		w.WriteHeader(http.StatusFound)
	}

	post := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		err := req.ParseMultipartForm(4 << 20)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(context.Background(), c, err)
			return
		}

		ctx := newCtxWithSesionCSRF(req,
			getMultipartFormValue(req.MultipartForm, "csrf_token"))
		content := getMultipartFormValue(req.MultipartForm, "content")
		replyToID := getMultipartFormValue(req.MultipartForm, "reply_to_id")
		format := getMultipartFormValue(req.MultipartForm, "format")
		visibility := getMultipartFormValue(req.MultipartForm, "visibility")
		isNSFW := "on" == getMultipartFormValue(req.MultipartForm, "is_nsfw")
		files := req.MultipartForm.File["attachments"]

		id, err := s.Post(ctx, c, content, replyToID, format, visibility, isNSFW, files)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		location := "/timeline/home" + "#status-" + id
		if len(replyToID) > 0 {
			location = "/thread/" + replyToID + "#status-" + id
		}
		w.Header().Add("Location", location)
		w.WriteHeader(http.StatusFound)
	}

	like := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]
		retweetedByID := req.FormValue("retweeted_by_id")

		_, err := s.Like(ctx, c, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		rID := id
		if len(retweetedByID) > 0 {
			rID = retweetedByID
		}
		w.Header().Add("Location", req.Header.Get("Referer")+"#status-"+rID)
		w.WriteHeader(http.StatusFound)
	}

	unlike := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]
		retweetedByID := req.FormValue("retweeted_by_id")

		_, err := s.UnLike(ctx, c, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		rID := id
		if len(retweetedByID) > 0 {
			rID = retweetedByID
		}
		w.Header().Add("Location", req.Header.Get("Referer")+"#status-"+rID)
		w.WriteHeader(http.StatusFound)
	}

	retweet := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]
		retweetedByID := req.FormValue("retweeted_by_id")

		_, err := s.Retweet(ctx, c, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		rID := id
		if len(retweetedByID) > 0 {
			rID = retweetedByID
		}
		w.Header().Add("Location", req.Header.Get("Referer")+"#status-"+rID)
		w.WriteHeader(http.StatusFound)
	}

	unretweet := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]
		retweetedByID := req.FormValue("retweeted_by_id")

		_, err := s.UnRetweet(ctx, c, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		rID := id
		if len(retweetedByID) > 0 {
			rID = retweetedByID
		}

		w.Header().Add("Location", req.Header.Get("Referer")+"#status-"+rID)
		w.WriteHeader(http.StatusFound)
	}

	follow := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]

		err := s.Follow(ctx, c, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		w.Header().Add("Location", req.Header.Get("Referer"))
		w.WriteHeader(http.StatusFound)
	}

	unfollow := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]

		err := s.UnFollow(ctx, c, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		w.Header().Add("Location", req.Header.Get("Referer"))
		w.WriteHeader(http.StatusFound)
	}

	settings := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		visibility := req.FormValue("visibility")
		copyScope := req.FormValue("copy_scope") == "true"
		threadInNewTab := req.FormValue("thread_in_new_tab") == "true"
		maskNSFW := req.FormValue("mask_nsfw") == "true"
		fluorideMode := req.FormValue("fluoride_mode") == "true"
		darkMode := req.FormValue("dark_mode") == "true"

		settings := &model.Settings{
			DefaultVisibility: visibility,
			CopyScope:         copyScope,
			ThreadInNewTab:    threadInNewTab,
			MaskNSFW:          maskNSFW,
			FluorideMode:      fluorideMode,
			DarkMode:          darkMode,
		}

		err := s.SaveSettings(ctx, c, settings)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s.ServeErrorPage(ctx, c, err)
			return
		}

		w.Header().Add("Location", req.Header.Get("Referer"))
		w.WriteHeader(http.StatusFound)
	}

	signout := func(w http.ResponseWriter, req *http.Request) {
		// TODO remove session from database
		http.SetCookie(w, &http.Cookie{
			Name:    "session_id",
			Value:   "",
			Expires: time.Now(),
		})
		w.Header().Add("Location", "/")
		w.WriteHeader(http.StatusFound)
	}

	fLike := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]

		count, err := s.Like(ctx, c, id)
		if err != nil {
			serveJsonError(w, err)
			return
		}

		err = serveJson(w, count)
		if err != nil {
			serveJsonError(w, err)
			return
		}
	}

	fUnlike := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]
		count, err := s.UnLike(ctx, c, id)
		if err != nil {
			serveJsonError(w, err)
			return
		}

		err = serveJson(w, count)
		if err != nil {
			serveJsonError(w, err)
			return
		}
	}

	fRetweet := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]

		count, err := s.Retweet(ctx, c, id)
		if err != nil {
			serveJsonError(w, err)
			return
		}

		err = serveJson(w, count)
		if err != nil {
			serveJsonError(w, err)
			return
		}
	}

	fUnretweet := func(w http.ResponseWriter, req *http.Request) {
		c := newClient(w)
		ctx := newCtxWithSesionCSRF(req, req.FormValue("csrf_token"))
		id, _ := mux.Vars(req)["id"]

		count, err := s.UnRetweet(ctx, c, id)
		if err != nil {
			serveJsonError(w, err)
			return
		}

		err = serveJson(w, count)
		if err != nil {
			serveJsonError(w, err)
			return
		}
	}

	r.HandleFunc("/", rootPage).Methods(http.MethodGet)
	r.HandleFunc("/signin", signinPage).Methods(http.MethodGet)
	r.HandleFunc("/timeline/{type}", timelinePage).Methods(http.MethodGet)
	r.HandleFunc("/timeline", timelineOldPage).Methods(http.MethodGet)
	r.HandleFunc("/thread/{id}", threadPage).Methods(http.MethodGet)
	r.HandleFunc("/likedby/{id}", likedByPage).Methods(http.MethodGet)
	r.HandleFunc("/retweetedby/{id}", retweetedByPage).Methods(http.MethodGet)
	r.HandleFunc("/following/{id}", followingPage).Methods(http.MethodGet)
	r.HandleFunc("/followers/{id}", followersPage).Methods(http.MethodGet)
	r.HandleFunc("/notifications", notificationsPage).Methods(http.MethodGet)
	r.HandleFunc("/user/{id}", userPage).Methods(http.MethodGet)
	r.HandleFunc("/usersearch/{id}", userSearchPage).Methods(http.MethodGet)
	r.HandleFunc("/about", aboutPage).Methods(http.MethodGet)
	r.HandleFunc("/emojis", emojisPage).Methods(http.MethodGet)
	r.HandleFunc("/search", searchPage).Methods(http.MethodGet)
	r.HandleFunc("/settings", settingsPage).Methods(http.MethodGet)
	r.HandleFunc("/signin", signin).Methods(http.MethodPost)
	r.HandleFunc("/oauth_callback", oauthCallback).Methods(http.MethodGet)
	r.HandleFunc("/post", post).Methods(http.MethodPost)
	r.HandleFunc("/like/{id}", like).Methods(http.MethodPost)
	r.HandleFunc("/unlike/{id}", unlike).Methods(http.MethodPost)
	r.HandleFunc("/retweet/{id}", retweet).Methods(http.MethodPost)
	r.HandleFunc("/unretweet/{id}", unretweet).Methods(http.MethodPost)
	r.HandleFunc("/follow/{id}", follow).Methods(http.MethodPost)
	r.HandleFunc("/unfollow/{id}", unfollow).Methods(http.MethodPost)
	r.HandleFunc("/settings", settings).Methods(http.MethodPost)
	r.HandleFunc("/signout", signout).Methods(http.MethodGet)
	r.HandleFunc("/fluoride/like/{id}", fLike).Methods(http.MethodPost)
	r.HandleFunc("/fluoride/unlike/{id}", fUnlike).Methods(http.MethodPost)
	r.HandleFunc("/fluoride/retweet/{id}", fRetweet).Methods(http.MethodPost)
	r.HandleFunc("/fluoride/unretweet/{id}", fUnretweet).Methods(http.MethodPost)
	r.PathPrefix("/static").Handler(http.StripPrefix("/static",
		http.FileServer(http.Dir(path.Join(".", staticDir)))))

	return r
}
