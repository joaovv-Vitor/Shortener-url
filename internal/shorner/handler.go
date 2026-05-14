package shorner

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler holds HTTP handlers and their dependencies.
type Handler struct {
	service *Service
	baseURL string
	logger  *slog.Logger
}

// NewHandler creates a new Handler with the given service and base URL.
func NewHandler(service *Service, baseURL string, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		baseURL: baseURL,
		logger:  logger,
	}
}

// RegisterRoutes registers all HTTP routes on the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.handleHealth)
	r.Post("/api/shorten", h.handleShorten)
	r.Get("/api/urls/{code}", h.handleGetURL)
	r.Get("/{code}", h.handleRedirect)
}

// handleHealth handles GET /health
// Returns a simple status check for load balancer health probes.
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Request / Response types ---

type shortenRequest struct {
	URL string `json:"url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Handlers ---

// handleShorten handles POST /api/shorten
// Creates a new short URL or returns an existing one (idempotent).
func (h *Handler) handleShorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	result, err := h.service.Shorten(r.Context(), ShortenInput{
		OriginalURL: req.URL,
	}, h.baseURL)

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.logger.Info("url shortened",
		slog.String("code", result.Code),
		slog.String("original_url", result.OriginalURL),
	)

	h.writeJSON(w, http.StatusCreated, result)
}

// handleRedirect handles GET /{code}
// Redirects to the original URL with a 301 status.
func (h *Handler) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "code is required"})
		return
	}

	url, err := h.service.Resolve(r.Context(), code)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.logger.Info("redirecting",
		slog.String("code", code),
		slog.String("target", url.OriginalURL),
	)

	http.Redirect(w, r, url.OriginalURL, http.StatusMovedPermanently)
}

// handleGetURL handles GET /api/urls/{code}
// Returns URL information as JSON.
func (h *Handler) handleGetURL(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "code is required"})
		return
	}

	url, err := h.service.Resolve(r.Context(), code)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, url)
}

// --- Helpers ---

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", slog.String("error", err.Error()))
	}
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrURLNotFound):
		h.writeJSON(w, http.StatusNotFound, errorResponse{Error: "url not found"})
	case errors.Is(err, ErrInvalidURL):
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid url: must be a valid http or https url"})
	case errors.Is(err, ErrEmptyURL):
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "url cannot be empty"})
	default:
		h.logger.Error("unexpected error", slog.String("error", err.Error()))
		h.writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
