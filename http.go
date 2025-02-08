package lardoon

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alioygur/gores"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type HTTPServer struct {
}

func getRequestReplay(w http.ResponseWriter, r *http.Request) *ReplayWithObjects {
	replayId := chi.URLParam(r, "id")
	row, err := db.Query(`SELECT * FROM replays WHERE id=?`, replayId)
	if err != nil {
		gores.Error(w, 500, fmt.Sprintf("error: %v", err))
		return nil
	}
	defer row.Close()

	if !row.Next() {
		gores.Error(w, 404, "replay not found")
		return nil
	}

	var replay ReplayWithObjects
	err = row.Scan(
		&replay.Id,
		&replay.Path,
		&replay.ReferenceTime,
		&replay.RecordingTime,
		&replay.Title,
		&replay.DataSource,
		&replay.DataRecorder,
		&replay.Duration,
		&replay.Size,
	)
	if err != nil {
		gores.Error(w, 500, fmt.Sprintf("error: %v", err))
		return nil
	}

	replay.Objects = make([]*ReplayObject, 0)
	rows, err := db.Query(`SELECT * FROM replay_objects WHERE replay_id = ?`, replay.Id)
	if err != nil {
		gores.Error(w, 500, fmt.Sprintf("error: %v", err))
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var object ReplayObject
		err := rows.Scan(
			&object.ReplayId,
			&object.Id,
			&object.Types,
			&object.Name,
			&object.Pilot,
			&object.CreatedOffset,
			&object.DeletedOffset,
		)
		if err != nil {
			gores.Error(w, 500, fmt.Sprintf("error: %v", err))
			return nil
		}
		replay.Objects = append(replay.Objects, &object)
	}

	return &replay
}

func (h *HTTPServer) serveEmbeddedFile(path string, w http.ResponseWriter, r *http.Request) {
	f, err := static.ReadFile("dist/" + path)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	fileName := filepath.Base(path)
	http.ServeContent(w, r, fileName, time.Now(), bytes.NewReader(f))
}

func (h *HTTPServer) serveEmbeddedStaticAssets(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "*")
	ext := filepath.Ext(param)
	if param == "" || (ext != ".js" && ext != ".css") {
		param = "index.html"
	}

	h.serveEmbeddedFile(param, w, r)
}

func (h *HTTPServer) downloadReplay(w http.ResponseWriter, r *http.Request) {
	replay := getRequestReplay(w, r)
	if replay == nil {
		return
	}

	start := 0
	end := -1

	startKeys, ok := r.URL.Query()["start"]
	if ok && len(startKeys) == 1 {
		startInt, err := strconv.ParseInt(startKeys[0], 10, 64)
		if err != nil {
			gores.Error(w, 400, "invalid start position")
			return
		}
		if startInt > 0 {
			start = int(startInt)
		}
	}

	endKeys, ok := r.URL.Query()["end"]
	if start != -1 && ok && len(startKeys) == 1 {
		endInt, err := strconv.ParseInt(endKeys[0], 10, 64)
		if err != nil || endInt < 0 || int(endInt) < start {
			gores.Error(w, 400, "invalid end position")
			return
		}
		end = int(endInt)
	}

	w.Header().Set("Content-Type", "text/plain")

	name := filepath.Base(replay.Path)
	if start == 0 && end == -1 {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v", name))
		http.ServeFile(w, r, replay.Path)
	} else {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v-%v-%v.acmi", replay.Title, start, end))
		err := trimTacView(replay.Path, w, start, end)
		if err != nil {
			log.Printf("failed to trim tacview: %v", err)
			gores.Error(w, 400, "failed to trim tacview")
			return
		}
	}
}

func (h *HTTPServer) getReplay(w http.ResponseWriter, r *http.Request) {
	replay := getRequestReplay(w, r)
	if replay == nil {
		return
	}

	gores.JSON(w, 200, replay)
}

func (h *HTTPServer) listReplays(w http.ResponseWriter, r *http.Request) {
	var rows *sql.Rows
	var err error

	filterKeys, ok := r.URL.Query()["filter"]
	if ok && len(filterKeys) == 1 {

		rows, err = db.Query(`
		SELECT * FROM replays r WHERE r.id IN (
			SELECT ro.replay_id FROM replay_objects ro
			WHERE LOWER(ro.pilot) LIKE ?
		) ORDER BY recording_time DESC`, "%"+strings.ToLower(filterKeys[0])+"%")
	} else {
		rows, err = db.Query(`SELECT * FROM replays ORDER BY recording_time DESC`)
	}

	if err != nil {
		gores.Error(w, 500, fmt.Sprintf("error: %v", err))
		return
	}
	defer rows.Close()

	replays := make([]*Replay, 0)
	for rows.Next() {
		var replay Replay
		err := rows.Scan(
			&replay.Id,
			&replay.Path,
			&replay.ReferenceTime,
			&replay.RecordingTime,
			&replay.Title,
			&replay.DataSource,
			&replay.DataRecorder,
			&replay.Duration,
			&replay.Size,
		)
		if err != nil {
			gores.Error(w, 500, fmt.Sprintf("error: %v", err))
			return
		}
		replays = append(replays, &replay)
	}

	gores.JSON(w, 200, replays)
}

func serveIndex(server *HTTPServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		server.serveEmbeddedFile("index.html", w, r)
	}
}

func (h *HTTPServer) Run(bind string) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.GetHead)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/", serveIndex(h))
	r.Get("/replay/*", serveIndex(h))
	r.Get("/static/*", h.serveEmbeddedStaticAssets)
	r.Get("/api/replay", h.listReplays)
	r.Get("/api/replay/{id}", h.getReplay)
	r.Get("/api/replay/{id}/download", h.downloadReplay)

	log.Printf("Serving Lardoon on http://%v", bind)
	return http.ListenAndServe(bind, r)
}
