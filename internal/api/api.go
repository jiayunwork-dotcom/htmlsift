package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"htmlsift/internal/htmllink"
	"htmlsift/internal/htmlparse"
	"htmlsift/internal/htmlsanitize"
)

type Server struct {
	mux    *http.ServeMux
	policy htmlsanitize.Policy
	addr   string
	webDir string
}

type Config struct {
	Addr   string
	WebDir string
}

func DefaultConfig() Config {
	return Config{Addr: ":8080", WebDir: "web"}
}

func New(cfg Config) *Server {
	s := &Server{
		mux:    http.NewServeMux(),
		policy: htmlsanitize.DefaultPolicy(),
		addr:   cfg.Addr,
		webDir: cfg.WebDir,
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) Addr() string { return s.addr }

func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/sanitize", s.handleSanitize)
	s.mux.HandleFunc("/api/parse", s.handleParse)
	s.mux.HandleFunc("/api/links", s.handleLinks)
	s.mux.HandleFunc("/api/text", s.handleText)
	s.mux.HandleFunc("/api/health", s.handleHealth)
	if s.webDir != "" {
		fs := http.FileServer(http.Dir(s.webDir))
		s.mux.Handle("/", fs)
	}
}

type SanitizeRequest struct {
	HTML     string `json:"html"`
	Fragment bool   `json:"fragment"`
}

type SanitizeResponse struct {
	Output          string `json:"output"`
	RemovedElements int    `json:"removed_elements"`
	RemovedAttrs    int    `json:"removed_attrs"`
	RemovedURLs     int    `json:"removed_urls"`
}

type ParseRequest struct {
	HTML string `json:"html"`
}

type ParseResponse struct {
	Elements  int            `json:"elements"`
	TextNodes int            `json:"text_nodes"`
	Comments  int            `json:"comments"`
	MaxDepth  int            `json:"max_depth"`
	Links     int            `json:"links"`
	Images    int            `json:"images"`
	TextBytes int            `json:"text_bytes"`
	Tags      map[string]int `json:"tags"`
}

type LinksRequest struct {
	HTML    string `json:"html"`
	BaseURL string `json:"base_url"`
}

type LinkItem struct {
	Tag      string `json:"tag"`
	Href     string `json:"href"`
	Text     string `json:"text"`
	Resolved string `json:"resolved,omitempty"`
	Class    string `json:"class"`
}

type LinksResponse struct {
	Links []LinkItem `json:"links"`
	Count int        `json:"count"`
}

type TextRequest struct {
	HTML string `json:"html"`
}

type TextResponse struct {
	Text string `json:"text"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleSanitize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req SanitizeRequest
	if err := readJSON(r, &req); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.HTML) == "" {
		httpError(w, http.StatusBadRequest, "html field is required")
		return
	}
	var out string
	var rep htmlsanitize.Report
	var err error
	if req.Fragment {
		out, err = s.policy.SanitizeFragment(req.HTML)
	} else {
		out, rep, err = s.policy.SanitizeReport(req.HTML)
	}
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if out != "" {
		again, err2 := s.policy.Sanitize(out)
		if err2 == nil {
			out = again
		}
	}
	writeJSON(w, SanitizeResponse{
		Output:          out,
		RemovedElements: rep.RemovedElements,
		RemovedAttrs:    rep.RemovedAttrs,
		RemovedURLs:     rep.RemovedURLs,
	})
}

func (s *Server) handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req ParseRequest
	if err := readJSON(r, &req); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.HTML) == "" {
		httpError(w, http.StatusBadRequest, "html field is required")
		return
	}
	d, err := htmlparse.Parse(req.HTML)
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	st := d.Stats()
	writeJSON(w, ParseResponse{
		Elements:  st.Elements,
		TextNodes: st.TextNodes,
		Comments:  st.Comments,
		MaxDepth:  st.MaxDepth,
		Links:     st.Links,
		Images:    st.Images,
		TextBytes: st.TotalBytes,
		Tags:      st.Tags,
	})
}

func (s *Server) handleLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req LinksRequest
	if err := readJSON(r, &req); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.HTML) == "" {
		httpError(w, http.StatusBadRequest, "html field is required")
		return
	}
	d, err := htmlparse.Parse(req.HTML)
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	links, err := htmllink.Extract(d, req.BaseURL)
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	items := make([]LinkItem, len(links))
	for i, l := range links {
		items[i] = LinkItem{
			Tag:      l.Tag,
			Href:     l.Href,
			Text:     l.Text,
			Resolved: l.Resolved,
			Class:    htmllink.Classify(l.Href).String(),
		}
	}
	writeJSON(w, LinksResponse{Links: items, Count: len(items)})
}

func (s *Server) handleText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req TextRequest
	if err := readJSON(r, &req); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.HTML) == "" {
		httpError(w, http.StatusBadRequest, "html field is required")
		return
	}
	d, err := htmlparse.Parse(req.HTML)
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	text := htmlparse.VisibleText(d)
	writeJSON(w, TextResponse{Text: text})
}

func readJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(body) == 0 {
		return fmt.Errorf("empty request body")
	}
	return json.Unmarshal(body, v)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func httpError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
