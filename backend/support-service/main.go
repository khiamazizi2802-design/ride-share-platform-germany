package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// ---------------------------------------------------------------------------
// GDPR-Hinweis: Alle personenbezogenen Daten (Kundendaten, Agentendaten,
// Ticketinhalte) werden gemäß DSGVO (EU 2016/679) verarbeitet.
// Die Speicherung erfolgt ausschließlich auf EU-Servern.
// Daten werden nach Ablauf der gesetzlichen Aufbewahrungsfristen gelöscht.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// MODELS
// ---------------------------------------------------------------------------

// Ticket repräsentiert ein Support-Ticket.
// Enthält personenbezogene Daten gemäß DSGVO Art. 4.
type Ticket struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`       // open, in_progress, waiting, resolved, closed
	Priority    string     `json:"priority"`     // low, medium, high, urgent
	Category    string     `json:"category"`     // billing, technical, ride, account, other
	CustomerID  int64      `json:"customer_id"`
	AgentID     *int64     `json:"agent_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// TicketComment repräsentiert einen Kommentar zu einem Ticket.
// Interne Kommentare sind nur für Agenten sichtbar (DSGVO-konform).
type TicketComment struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticket_id"`
	AuthorID   int64     `json:"author_id"`
	AuthorType string    `json:"author_type"` // customer, agent
	Content    string    `json:"content"`
	IsInternal bool      `json:"is_internal"` // Interne Notizen nicht an Kunden weitergeben
	CreatedAt  time.Time `json:"created_at"`
}

// KnowledgeBaseArticle repräsentiert einen Wissensdatenbank-Artikel.
type KnowledgeBaseArticle struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	Language    string    `json:"language"` // de, en
	IsPublished bool      `json:"is_published"`
	ViewCount   int64     `json:"view_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SupportAgent repräsentiert einen Support-Mitarbeiter.
// Enthält personenbezogene Daten gemäß DSGVO Art. 4.
type SupportAgent struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"` // Verarbeitung nach DSGVO Art. 6 Abs. 1 lit. b
	Role           string    `json:"role"`  // agent, senior_agent, team_lead
	Department     string    `json:"department"`
	IsActive       bool      `json:"is_active"`
	MaxTickets     int       `json:"max_tickets"`
	CurrentTickets int       `json:"current_tickets"`
	CreatedAt      time.Time `json:"created_at"`
}

// ---------------------------------------------------------------------------
// REQUEST / RESPONSE TYPES
// ---------------------------------------------------------------------------

type CreateTicketRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	Category    string     `json:"category"`
	CustomerID  int64      `json:"customer_id"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type UpdateTicketRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Priority    *string    `json:"priority,omitempty"`
	Category    *string    `json:"category,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type AddCommentRequest struct {
	Content    string `json:"content"`
	IsInternal bool   `json:"is_internal"`
}

type AssignTicketRequest struct {
	AgentID int64 `json:"agent_id"`
}

type CreateArticleRequest struct {
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Language    string   `json:"language"`
	IsPublished bool     `json:"is_published"`
}

type UpdateArticleRequest struct {
	Title       *string  `json:"title,omitempty"`
	Content     *string  `json:"content,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Language    *string  `json:"language,omitempty"`
	IsPublished *bool    `json:"is_published,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// ---------------------------------------------------------------------------
// JWT CLAIMS
// ---------------------------------------------------------------------------

type Claims struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// ---------------------------------------------------------------------------
// RATE LIMITER
// ---------------------------------------------------------------------------

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientInfo
	limit    int
	window   time.Duration
	cleanup  time.Duration
}

type clientInfo struct {
	count    int
	resetAt  time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientInfo),
		limit:   limit,
		window:  window,
		cleanup: 5 * time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, info := range rl.clients {
			if now.After(info.resetAt) {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	info, exists := rl.clients[ip]
	if !exists || now.After(info.resetAt) {
		rl.clients[ip] = &clientInfo{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if info.count >= rl.limit {
		return false
	}
	info.count++
	return true
}

// ---------------------------------------------------------------------------
// SERVER
// ---------------------------------------------------------------------------

type Server struct {
	db          *sql.DB
	router      *mux.Router
	jwtSecret   []byte
	rateLimiter *RateLimiter
	logger      *log.Logger
}

func NewServer(db *sql.DB, jwtSecret string) *Server {
	s := &Server{
		db:          db,
		router:      mux.NewRouter(),
		jwtSecret:   []byte(jwtSecret),
		rateLimiter: NewRateLimiter(100, time.Minute),
		logger:      log.New(os.Stdout, "[SUPPORT-SERVICE] ", log.LstdFlags|log.Lshortfile),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/v1").Subrouter()
	api.Use(s.loggingMiddleware)
	api.Use(s.rateLimitMiddleware)

	// Ticket-Routen
	api.HandleFunc("/tickets", s.authMiddleware(s.createTicket)).Methods(http.MethodPost)
	api.HandleFunc("/tickets", s.authMiddleware(s.listTickets)).Methods(http.MethodGet)
	api.HandleFunc("/tickets/{id}", s.authMiddleware(s.getTicket)).Methods(http.MethodGet)
	api.HandleFunc("/tickets/{id}", s.authMiddleware(s.updateTicket)).Methods(http.MethodPut)
	api.HandleFunc("/tickets/{id}/comments", s.authMiddleware(s.addComment)).Methods(http.MethodPost)
	api.HandleFunc("/tickets/{id}/comments", s.authMiddleware(s.getComments)).Methods(http.MethodGet)
	api.HandleFunc("/tickets/{id}/assign", s.authMiddleware(s.assignTicket)).Methods(http.MethodPost)
	api.HandleFunc("/tickets/{id}/resolve", s.authMiddleware(s.resolveTicket)).Methods(http.MethodPost)

	// Wissensdatenbank-Routen
	api.HandleFunc("/kb/articles", s.listArticles).Methods(http.MethodGet)
	api.HandleFunc("/kb/articles/{id}", s.getArticle).Methods(http.MethodGet)
	api.HandleFunc("/kb/articles", s.authMiddleware(s.createArticle)).Methods(http.MethodPost)
	api.HandleFunc("/kb/articles/{id}", s.authMiddleware(s.updateArticle)).Methods(http.MethodPut)

	// Agenten-Routen
	api.HandleFunc("/agents", s.authMiddleware(s.listAgents)).Methods(http.MethodGet)
	api.HandleFunc("/agents/{id}/tickets", s.authMiddleware(s.getAgentTickets)).Methods(http.MethodGet)

	// Gesundheitscheck
	s.router.HandleFunc("/health", s.healthCheck).Methods(http.MethodGet)
}

// ---------------------------------------------------------------------------
// MIDDLEWARE
// ---------------------------------------------------------------------------

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// DSGVO: IP-Adressen werden nur für Sicherheitszwecke protokolliert (Art. 6 Abs. 1 lit. f)
		// und nach 7 Tagen gelöscht.
		s.logger.Printf("method=%s path=%s remote_addr=%s",
			r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		s.logger.Printf("method=%s path=%s duration=%s",
			r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = strings.Split(fwd, ",")[0]
		}
		if !s.rateLimiter.Allow(strings.TrimSpace(ip)) {
			s.writeError(w, http.StatusTooManyRequests, "Zu viele Anfragen. Bitte warten Sie einen Moment.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.writeError(w, http.StatusUnauthorized, "Authentifizierung erforderlich")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			s.writeError(w, http.StatusUnauthorized, "Ungültiges Authentifizierungsformat")
			return
		}
		tokenStr := parts[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unerwartete Signiermethode: %v", token.Header["alg"])
			}
			return s.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			s.writeError(w, http.StatusUnauthorized, "Ungültiges oder abgelaufenes Token")
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, contextKeyRole, claims.Role)
		next(w, r.WithContext(ctx))
	}
}

type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyRole   contextKey = "role"
)

// ---------------------------------------------------------------------------
// HELPER FUNCTIONS
// ---------------------------------------------------------------------------

func (s *Server) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Printf("Fehler beim Schreiben der JSON-Antwort: %v", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, APIResponse{Success: false, Error: message})
}

func (s *Server) writeSuccess(w http.ResponseWriter, status int, data interface{}) {
	s.writeJSON(w, status, APIResponse{Success: true, Data: data})
}

func (s *Server) parseID(r *http.Request, key string) (int64, error) {
	vars := mux.Vars(r)
	val, ok := vars[key]
	if !ok {
		return 0, fmt.Errorf("fehlender Parameter: %s", key)
	}
	return strconv.ParseInt(val, 10, 64)
}

func getQueryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	if i, err := strconv.Atoi(val); err == nil && i > 0 {
		return i
	}
	return defaultVal
}

func getUserID(r *http.Request) int64 {
	if v, ok := r.Context().Value(contextKeyUserID).(int64); ok {
		return v
	}
	return 0
}

func getRole(r *http.Request) string {
	if v, ok := r.Context().Value(contextKeyRole).(string); ok {
		return v
	}
	return ""
}

// ---------------------------------------------------------------------------
// TICKET HANDLERS
// ---------------------------------------------------------------------------

// createTicket erstellt ein neues Support-Ticket.
// DSGVO: Ticketinhalte können personenbezogene Daten enthalten (Art. 4 Nr. 1).
func (s *Server) createTicket(w http.ResponseWriter, r *http.Request) {
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Anfragedaten")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		s.writeError(w, http.StatusBadRequest, "Titel ist erforderlich")
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		s.writeError(w, http.StatusBadRequest, "Beschreibung ist erforderlich")
		return
	}
	if req.CustomerID == 0 {
		s.writeError(w, http.StatusBadRequest, "Kunden-ID ist erforderlich")
		return
	}
	validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
	if !validPriorities[req.Priority] {
		req.Priority = "medium"
	}
	validCategories := map[string]bool{"billing": true, "technical": true, "ride": true, "account": true, "other": true}
	if !validCategories[req.Category] {
		req.Category = "other"
	}
	now := time.Now().UTC()
	ticket := &Ticket{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      "open",
		Priority:    req.Priority,
		Category:    req.Category,
		CustomerID:  req.CustomerID,
		DueDate:     req.DueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	query := `INSERT INTO tickets
		(title, description, status, priority, category, customer_id, due_date, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`
	err := s.db.QueryRowContext(r.Context(), query,
		ticket.Title, ticket.Description, ticket.Status, ticket.Priority,
		ticket.Category, ticket.CustomerID, ticket.DueDate,
		ticket.CreatedAt, ticket.UpdatedAt,
	).Scan(&ticket.ID)
	if err != nil {
		s.logger.Printf("Fehler beim Erstellen des Tickets: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Ticket konnte nicht erstellt werden")
		return
	}
	s.logger.Printf("Ticket erstellt: id=%d customer_id=%d", ticket.ID, ticket.CustomerID)
	s.writeSuccess(w, http.StatusCreated, ticket)
}

// getTicket gibt ein einzelnes Ticket zurück.
func (s *Server) getTicket(w http.ResponseWriter, r *http.Request) {
	id, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Ticket-ID")
		return
	}
	ticket, err := s.fetchTicketByID(r.Context(), id)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "Ticket nicht gefunden")
		return
	}
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen des Tickets %d: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "Ticket konnte nicht abgerufen werden")
		return
	}
	s.writeSuccess(w, http.StatusOK, ticket)
}

func (s *Server) fetchTicketByID(ctx context.Context, id int64) (*Ticket, error) {
	query := `SELECT id, title, description, status, priority, category,
		customer_id, agent_id, created_at, updated_at, resolved_at, due_date
		FROM tickets WHERE id = $1`
	ticket := &Ticket{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&ticket.ID, &ticket.Title, &ticket.Description, &ticket.Status,
		&ticket.Priority, &ticket.Category, &ticket.CustomerID, &ticket.AgentID,
		&ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ResolvedAt, &ticket.DueDate,
	)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

// updateTicket aktualisiert ein bestehendes Ticket.
func (s *Server) updateTicket(w http.ResponseWriter, r *http.Request) {
	id, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Ticket-ID")
		return
	}
	var req UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Anfragedaten")
		return
	}
	existing, err := s.fetchTicketByID(r.Context(), id)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "Ticket nicht gefunden")
		return
	}
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen des Tickets %d: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "Ticket konnte nicht aktualisiert werden")
		return
	}
	if req.Title != nil {
		existing.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		existing.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		validStatuses := map[string]bool{"open": true, "in_progress": true, "waiting": true, "resolved": true, "closed": true}
		if !validStatuses[*req.Status] {
			s.writeError(w, http.StatusBadRequest, "Ungültiger Status")
			return
		}
		existing.Status = *req.Status
	}
	if req.Priority != nil {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if !validPriorities[*req.Priority] {
			s.writeError(w, http.StatusBadRequest, "Ungültige Priorität")
			return
		}
		existing.Priority = *req.Priority
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.DueDate != nil {
		existing.DueDate = req.DueDate
	}
	existing.UpdatedAt = time.Now().UTC()
	query := `UPDATE tickets SET title=$1, description=$2, status=$3, priority=$4,
		category=$5, due_date=$6, updated_at=$7 WHERE id=$8`
	_, err = s.db.ExecContext(r.Context(), query,
		existing.Title, existing.Description, existing.Status, existing.Priority,
		existing.Category, existing.DueDate, existing.UpdatedAt, id,
	)
	if err != nil {
		s.logger.Printf("Fehler beim Aktualisieren des Tickets %d: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "Ticket konnte nicht aktualisiert werden")
		return
	}
	s.writeSuccess(w, http.StatusOK, existing)
}

// listTickets gibt eine paginierte Liste von Tickets zurück.
// Filter: status, priority, category, customer_id, agent_id
func (s *Server) listTickets(w http.ResponseWriter, r *http.Request) {
	page := getQueryInt(r, "page", 1)
	pageSize := getQueryInt(r, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var conditions []string
	var args []interface{}
	argIdx := 1

	if status := r.URL.Query().Get("status"); status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if priority := r.URL.Query().Get("priority"); priority != "" {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, priority)
		argIdx++
	}
	if category := r.URL.Query().Get("category"); category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}
	if customerID := r.URL.Query().Get("customer_id"); customerID != "" {
		if cid, err := strconv.ParseInt(customerID, 10, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("customer_id = $%d", argIdx))
			args = append(args, cid)
			argIdx++
		}
	}
	if agentID := r.URL.Query().Get("agent_id"); agentID != "" {
		if aid, err := strconv.ParseInt(agentID, 10, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("agent_id = $%d", argIdx))
			args = append(args, aid)
			argIdx++
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tickets %s", where)
	var total int64
	if err := s.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total); err != nil {
		s.logger.Printf("Fehler beim Zählen der Tickets: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Tickets konnten nicht aufgelistet werden")
		return
	}

	listArgs := append(args, pageSize, offset)
	listQuery := fmt.Sprintf(`SELECT id, title, description, status, priority, category,
		customer_id, agent_id, created_at, updated_at, resolved_at, due_date
		FROM tickets %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(r.Context(), listQuery, listArgs...)
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen der Tickets: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Tickets konnten nicht abgerufen werden")
		return
	}
	defer rows.Close()

	tickets := make([]*Ticket, 0)
	for rows.Next() {
		t := &Ticket{}
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Category,
			&t.CustomerID, &t.AgentID, &t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt, &t.DueDate,
		); err != nil {
			s.logger.Printf("Fehler beim Scannen des Tickets: %v", err)
			continue
		}
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		s.logger.Printf("Fehler beim Iterieren über Tickets: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Fehler beim Lesen der Tickets")
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	s.writeJSON(w, http.StatusOK, PaginatedResponse{
		Success:    true,
		Data:       tickets,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// ---------------------------------------------------------------------------
// COMMENT HANDLERS
// ---------------------------------------------------------------------------

// addComment fügt einen Kommentar zu einem Ticket hinzu.
// DSGVO: Interne Kommentare (IsInternal=true) werden nicht an Kunden weitergegeben.
func (s *Server) addComment(w http.ResponseWriter, r *http.Request) {
	ticketID, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Ticket-ID")
		return
	}
	var req AddCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Anfragedaten")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		s.writeError(w, http.StatusBadRequest, "Kommentarinhalt ist erforderlich")
		return
	}
	// Prüfe ob Ticket existiert
	if _, err := s.fetchTicketByID(r.Context(), ticketID); err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "Ticket nicht gefunden")
		return
	} else if err != nil {
		s.logger.Printf("Fehler beim Prüfen des Tickets %d: %v", ticketID, err)
		s.writeError(w, http.StatusInternalServerError, "Interner Fehler")
		return
	}
	userID := getUserID(r)
	role := getRole(r)
	authorType := "customer"
	if role == "agent" || role == "admin" || role == "senior_agent" || role == "team_lead" {
		authorType = "agent"
	}
	// Kunden dürfen keine internen Kommentare erstellen
	if authorType == "customer" && req.IsInternal {
		req.IsInternal = false
	}
	now := time.Now().UTC()
	comment := &TicketComment{
		TicketID:   ticketID,
		AuthorID:   userID,
		AuthorType: authorType,
		Content:    strings.TrimSpace(req.Content),
		IsInternal: req.IsInternal,
		CreatedAt:  now,
	}
	query := `INSERT INTO ticket_comments
		(ticket_id, author_id, author_type, content, is_internal, created_at)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`
	err = s.db.QueryRowContext(r.Context(), query,
		comment.TicketID, comment.AuthorID, comment.AuthorType,
		comment.Content, comment.IsInternal, comment.CreatedAt,
	).Scan(&comment.ID)
	if err != nil {
		s.logger.Printf("Fehler beim Erstellen des Kommentars für Ticket %d: %v", ticketID, err)
		s.writeError(w, http.StatusInternalServerError, "Kommentar konnte nicht erstellt werden")
		return
	}
	// Ticket updated_at aktualisieren
	_, _ = s.db.ExecContext(r.Context(),
		"UPDATE tickets SET updated_at=$1 WHERE id=$2", now, ticketID)
	s.writeSuccess(w, http.StatusCreated, comment)
}

// getComments gibt alle Kommentare eines Tickets zurück.
// DSGVO: Interne Kommentare werden nur an Agenten und Admins zurückgegeben.
func (s *Server) getComments(w http.ResponseWriter, r *http.Request) {
	ticketID, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Ticket-ID")
		return
	}
	role := getRole(r)
	var query string
	var args []interface{}
	if role == "agent" || role == "admin" || role == "senior_agent" || role == "team_lead" {
		query = `SELECT id, ticket_id, author_id, author_type, content, is_internal, created_at
			FROM ticket_comments WHERE ticket_id = $1 ORDER BY created_at ASC`
		args = []interface{}{ticketID}
	} else {
		// Kunden sehen keine internen Kommentare (DSGVO-konform)
		query = `SELECT id, ticket_id, author_id, author_type, content, is_internal, created_at
			FROM ticket_comments WHERE ticket_id = $1 AND is_internal = FALSE ORDER BY created_at ASC`
		args = []interface{}{ticketID}
	}
	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen der Kommentare für Ticket %d: %v", ticketID, err)
		s.writeError(w, http.StatusInternalServerError, "Kommentare konnten nicht abgerufen werden")
		return
	}
	defer rows.Close()
	comments := make([]*TicketComment, 0)
	for rows.Next() {
		c := &TicketComment{}
		if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.AuthorType,
			&c.Content, &c.IsInternal, &c.CreatedAt); err != nil {
			s.logger.Printf("Fehler beim Scannen des Kommentars: %v", err)
			continue
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		s.logger.Printf("Fehler beim Iterieren über Kommentare: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Fehler beim Lesen der Kommentare")
		return
	}
	s.writeSuccess(w, http.StatusOK, comments)
}

// ---------------------------------------------------------------------------
// TICKET ASSIGNMENT & RESOLUTION
// ---------------------------------------------------------------------------

// assignTicket weist ein Ticket einem Agenten zu.
func (s *Server) assignTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Ticket-ID")
		return
	}
	role := getRole(r)
	if role != "agent" && role != "admin" && role != "senior_agent" && role != "team_lead" {
		s.writeError(w, http.StatusForbidden, "Keine Berechtigung zur Zuweisung von Tickets")
		return
	}
	var req AssignTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Anfragedaten")
		return
	}
	if req.AgentID == 0 {
		s.writeError(w, http.StatusBadRequest, "Agenten-ID ist erforderlich")
		return
	}
	// Prüfe ob Agent existiert und aktiv ist
	var isActive bool
	var currentTickets, maxTickets int
	err = s.db.QueryRowContext(r.Context(),
		"SELECT is_active, current_tickets, max_tickets FROM support_agents WHERE id=$1",
		req.AgentID,
	).Scan(&isActive, &currentTickets, &maxTickets)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "Agent nicht gefunden")
		return
	}
	if err != nil {
		s.logger.Printf("Fehler beim Prüfen des Agenten %d: %v", req.AgentID, err)
		s.writeError(w, http.StatusInternalServerError, "Interner Fehler")
		return
	}
	if !isActive {
		s.writeError(w, http.StatusBadRequest, "Agent ist nicht aktiv")
		return
	}
	if currentTickets >= maxTickets {
		s.writeError(w, http.StatusBadRequest, "Agent hat die maximale Ticketanzahl erreicht")
		return
	}
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		s.logger.Printf("Fehler beim Starten der Transaktion: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Interner Fehler")
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	_, err = tx.ExecContext(r.Context(),
		"UPDATE tickets SET agent_id=$1, status='in_progress', updated_at=$2 WHERE id=$3",
		req.AgentID, now, ticketID)
	if err != nil {
		s.logger.Printf("Fehler beim Zuweisen des Tickets %d: %v", ticketID, err)
		s.writeError(w, http.StatusInternalServerError, "Ticket konnte nicht zugewiesen werden")
		return
	}
	_, err = tx.ExecContext(r.Context(),
		"UPDATE support_agents SET current_tickets=current_tickets+1 WHERE id=$1",
		req.AgentID)
	if err != nil {
		s.logger.Printf("Fehler beim Aktualisieren der Agentenzählung: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Agentenzählung konnte nicht aktualisiert werden")
		return
	}
	if err = tx.Commit(); err != nil {
		s.logger.Printf("Fehler beim Committen der Transaktion: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Interner Fehler")
		return
	}
	s.logger.Printf("Ticket %d wurde Agent %d zugewiesen", ticketID, req.AgentID)
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Ticket erfolgreich zugewiesen"})
}

// resolveTicket markiert ein Ticket als gelöst.
func (s *Server) resolveTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Ticket-ID")
		return
	}
	existing, err := s.fetchTicketByID(r.Context(), ticketID)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "Ticket nicht gefunden")
		return
	}
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen des Tickets %d: %v", ticketID, err)
		s.writeError(w, http.StatusInternalServerError, "Interner Fehler")
		return
	}
	if existing.Status == "resolved" || existing.Status == "closed" {
		s.writeError(w, http.StatusBadRequest, "Ticket ist bereits gelöst oder geschlossen")
		return
	}
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		s.logger.Printf("Fehler beim Starten der Transaktion: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Interner Fehler")
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	_, err = tx.ExecContext(r.Context(),
		"UPDATE tickets SET status='resolved', resolved_at=$1, updated_at=$1 WHERE id=$2",
		now, ticketID)
	if err != nil {
		s.logger.Printf("Fehler beim Lösen des Tickets %d: %v", ticketID, err)
		s.writeError(w, http.StatusInternalServerError, "Ticket konnte nicht gelöst werden")
		return
	}
	if existing.AgentID != nil {
		_, err = tx.ExecContext(r.Context(),
			"UPDATE support_agents SET current_tickets=GREATEST(current_tickets-1,0) WHERE id=$1",
			*existing.AgentID)
		if err != nil {
			s.logger.Printf("Fehler beim Aktualisieren der Agentenzählung: %v", err)
			s.writeError(w, http.StatusInternalServerError, "Agentenzählung konnte nicht aktualisiert werden")
			return
		}
	}
	if err = tx.Commit(); err != nil {
		s.logger.Printf("Fehler beim Committen der Transaktion: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Interner Fehler")
		return
	}
	s.logger.Printf("Ticket %d wurde gelöst", ticketID)
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Ticket erfolgreich gelöst"})
}

// ---------------------------------------------------------------------------
// KNOWLEDGE BASE HANDLERS
// ---------------------------------------------------------------------------

// listArticles gibt eine Liste von Wissensdatenbank-Artikeln zurück.
func (s *Server) listArticles(w http.ResponseWriter, r *http.Request) {
	page := getQueryInt(r, "page", 1)
	pageSize := getQueryInt(r, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var conditions []string
	var args []interface{}
	argIdx := 1

	// Standardmäßig nur veröffentlichte Artikel zeigen
	conditions = append(conditions, fmt.Sprintf("is_published = $%d", argIdx))
	args = append(args, true)
	argIdx++

	if category := r.URL.Query().Get("category"); category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}
	if lang := r.URL.Query().Get("language"); lang != "" {
		conditions = append(conditions, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, lang)
		argIdx++
	}
	if search := r.URL.Query().Get("search"); search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR content ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM kb_articles %s", where)
	var total int64
	if err := s.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total); err != nil {
		s.logger.Printf("Fehler beim Zählen der Artikel: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Artikel konnten nicht aufgelistet werden")
		return
	}

	listArgs := append(args, pageSize, offset)
	listQuery := fmt.Sprintf(`SELECT id, title, content, category, tags, language,
		is_published, view_count, created_at, updated_at
		FROM kb_articles %s ORDER BY view_count DESC, created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(r.Context(), listQuery, listArgs...)
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen der Artikel: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Artikel konnten nicht abgerufen werden")
		return
	}
	defer rows.Close()
	articles := make([]*KnowledgeBaseArticle, 0)
	for rows.Next() {
		a := &KnowledgeBaseArticle{}
		var tagsStr string
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &tagsStr,
			&a.Language, &a.IsPublished, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt); err != nil {
			s.logger.Printf("Fehler beim Scannen des Artikels: %v", err)
			continue
		}
		if tagsStr != "" {
			a.Tags = strings.Split(tagsStr, ",")
		} else {
			a.Tags = []string{}
		}
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		s.logger.Printf("Fehler beim Iterieren über Artikel: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Fehler beim Lesen der Artikel")
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	s.writeJSON(w, http.StatusOK, PaginatedResponse{
		Success:    true,
		Data:       articles,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// getArticle gibt einen einzelnen Wissensdatenbank-Artikel zurück.
func (s *Server) getArticle(w http.ResponseWriter, r *http.Request) {
	id, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Artikel-ID")
		return
	}
	article := &KnowledgeBaseArticle{}
	var tagsStr string
	query := `SELECT id, title, content, category, tags, language, is_published, view_count, created_at, updated_at
		FROM kb_articles WHERE id=$1 AND is_published=true`
	err = s.db.QueryRowContext(r.Context(), query, id).Scan(
		&article.ID, &article.Title, &article.Content, &article.Category, &tagsStr,
		&article.Language, &article.IsPublished, &article.ViewCount, &article.CreatedAt, &article.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "Artikel nicht gefunden")
		return
	}
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen des Artikels %d: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "Artikel konnte nicht abgerufen werden")
		return
	}
	if tagsStr != "" {
		article.Tags = strings.Split(tagsStr, ",")
	} else {
		article.Tags = []string{}
	}
	// Aufrufzähler asynchron erhöhen
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = s.db.ExecContext(ctx,
			"UPDATE kb_articles SET view_count=view_count+1 WHERE id=$1", id)
	}()
	s.writeSuccess(w, http.StatusOK, article)
}

// createArticle erstellt einen neuen Wissensdatenbank-Artikel.
// Nur für Admins und Team-Leads.
func (s *Server) createArticle(w http.ResponseWriter, r *http.Request) {
	role := getRole(r)
	if role != "admin" && role != "team_lead" {
		s.writeError(w, http.StatusForbidden, "Keine Berechtigung zum Erstellen von Artikeln")
		return
	}
	var req CreateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Anfragedaten")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		s.writeError(w, http.StatusBadRequest, "Titel ist erforderlich")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		s.writeError(w, http.StatusBadRequest, "Inhalt ist erforderlich")
		return
	}
	if req.Language == "" {
		req.Language = "de"
	}
	now := time.Now().UTC()
	tagsStr := strings.Join(req.Tags, ",")
	article := &KnowledgeBaseArticle{
		Title:       strings.TrimSpace(req.Title),
		Content:     strings.TrimSpace(req.Content),
		Category:    req.Category,
		Tags:        req.Tags,
		Language:    req.Language,
		IsPublished: req.IsPublished,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	query := `INSERT INTO kb_articles (title, content, category, tags, language, is_published, view_count, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,0,$7,$8) RETURNING id`
	err := s.db.QueryRowContext(r.Context(), query,
		article.Title, article.Content, article.Category, tagsStr,
		article.Language, article.IsPublished, article.CreatedAt, article.UpdatedAt,
	).Scan(&article.ID)
	if err != nil {
		s.logger.Printf("Fehler beim Erstellen des Artikels: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Artikel konnte nicht erstellt werden")
		return
	}
	s.writeSuccess(w, http.StatusCreated, article)
}

// updateArticle aktualisiert einen bestehenden Wissensdatenbank-Artikel.
// Nur für Admins und Team-Leads.
func (s *Server) updateArticle(w http.ResponseWriter, r *http.Request) {
	role := getRole(r)
	if role != "admin" && role != "team_lead" {
		s.writeError(w, http.StatusForbidden, "Keine Berechtigung zum Aktualisieren von Artikeln")
		return
	}
	id, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Artikel-ID")
		return
	}
	var req UpdateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Anfragedaten")
		return
	}
	// Bestehenden Artikel abrufen
	article := &KnowledgeBaseArticle{}
	var tagsStr string
	existingQuery := `SELECT id, title, content, category, tags, language, is_published, view_count, created_at, updated_at
		FROM kb_articles WHERE id=$1`
	err = s.db.QueryRowContext(r.Context(), existingQuery, id).Scan(
		&article.ID, &article.Title, &article.Content, &article.Category, &tagsStr,
		&article.Language, &article.IsPublished, &article.ViewCount, &article.CreatedAt, &article.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "Artikel nicht gefunden")
		return
	}
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen des Artikels %d: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "Interner Fehler")
		return
	}
	if req.Title != nil {
		article.Title = strings.TrimSpace(*req.Title)
	}
	if req.Content != nil {
		article.Content = strings.TrimSpace(*req.Content)
	}
	if req.Category != nil {
		article.Category = *req.Category
	}
	if req.Tags != nil {
		article.Tags = req.Tags
		tagsStr = strings.Join(req.Tags, ",")
	}
	if req.Language != nil {
		article.Language = *req.Language
	}
	if req.IsPublished != nil {
		article.IsPublished = *req.IsPublished
	}
	article.UpdatedAt = time.Now().UTC()
	updateQuery := `UPDATE kb_articles SET title=$1, content=$2, category=$3, tags=$4,
		language=$5, is_published=$6, updated_at=$7 WHERE id=$8`
	_, err = s.db.ExecContext(r.Context(), updateQuery,
		article.Title, article.Content, article.Category, tagsStr,
		article.Language, article.IsPublished, article.UpdatedAt, id,
	)
	if err != nil {
		s.logger.Printf("Fehler beim Aktualisieren des Artikels %d: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "Artikel konnte nicht aktualisiert werden")
		return
	}
	s.writeSuccess(w, http.StatusOK, article)
}

// ---------------------------------------------------------------------------
// AGENT HANDLERS
// ---------------------------------------------------------------------------

// listAgents gibt eine Liste aller Support-Agenten zurück.
// DSGVO: Nur für authentifizierte Nutzer mit entsprechenden Rollen zugänglich.
func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	role := getRole(r)
	if role != "admin" && role != "team_lead" && role != "agent" && role != "senior_agent" {
		s.writeError(w, http.StatusForbidden, "Keine Berechtigung")
		return
	}
	page := getQueryInt(r, "page", 1)
	pageSize := getQueryInt(r, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var conditions []string
	var args []interface{}
	argIdx := 1

	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
			args = append(args, isActive)
			argIdx++
		}
	}
	if dept := r.URL.Query().Get("department"); dept != "" {
		conditions = append(conditions, fmt.Sprintf("department = $%d", argIdx))
		args = append(args, dept)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM support_agents %s", where)
	var total int64
	if err := s.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total); err != nil {
		s.logger.Printf("Fehler beim Zählen der Agenten: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Agenten konnten nicht aufgelistet werden")
		return
	}

	listArgs := append(args, pageSize, offset)
	listQuery := fmt.Sprintf(`SELECT id, user_id, name, email, role, department,
		is_active, max_tickets, current_tickets, created_at
		FROM support_agents %s ORDER BY name ASC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(r.Context(), listQuery, listArgs...)
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen der Agenten: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Agenten konnten nicht abgerufen werden")
		return
	}
	defer rows.Close()
	agents := make([]*SupportAgent, 0)
	for rows.Next() {
		a := &SupportAgent{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Email, &a.Role, &a.Department,
			&a.IsActive, &a.MaxTickets, &a.CurrentTickets, &a.CreatedAt); err != nil {
			s.logger.Printf("Fehler beim Scannen des Agenten: %v", err)
			continue
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		s.logger.Printf("Fehler beim Iterieren über Agenten: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Fehler beim Lesen der Agenten")
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	s.writeJSON(w, http.StatusOK, PaginatedResponse{
		Success:    true,
		Data:       agents,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// getAgentTickets gibt alle Tickets eines Agenten zurück.
func (s *Server) getAgentTickets(w http.ResponseWriter, r *http.Request) {
	agentID, err := s.parseID(r, "id")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Ungültige Agenten-ID")
		return
	}
	role := getRole(r)
	if role != "admin" && role != "team_lead" && role != "agent" && role != "senior_agent" {
		s.writeError(w, http.StatusForbidden, "Keine Berechtigung")
		return
	}
	page := getQueryInt(r, "page", 1)
	pageSize := getQueryInt(r, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var conditions []string
	args := []interface{}{agentID}
	argIdx := 2
	conditions = append(conditions, "agent_id = $1")

	if status := r.URL.Query().Get("status"); status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tickets %s", where)
	var total int64
	if err := s.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total); err != nil {
		s.logger.Printf("Fehler beim Zählen der Tickets für Agent %d: %v", agentID, err)
		s.writeError(w, http.StatusInternalServerError, "Tickets konnten nicht aufgelistet werden")
		return
	}

	listArgs := append(args, pageSize, offset)
	listQuery := fmt.Sprintf(`SELECT id, title, description, status, priority, category,
		customer_id, agent_id, created_at, updated_at, resolved_at, due_date
		FROM tickets %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(r.Context(), listQuery, listArgs...)
	if err != nil {
		s.logger.Printf("Fehler beim Abrufen der Tickets für Agent %d: %v", agentID, err)
		s.writeError(w, http.StatusInternalServerError, "Tickets konnten nicht abgerufen werden")
		return
	}
	defer rows.Close()
	tickets := make([]*Ticket, 0)
	for rows.Next() {
		t := &Ticket{}
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Category,
			&t.CustomerID, &t.AgentID, &t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt, &t.DueDate); err != nil {
			s.logger.Printf("Fehler beim Scannen des Tickets: %v", err)
			continue
		}
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		s.logger.Printf("Fehler beim Iterieren über Tickets: %v", err)
		s.writeError(w, http.StatusInternalServerError, "Fehler beim Lesen der Tickets")
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	s.writeJSON(w, http.StatusOK, PaginatedResponse{
		Success:    true,
		Data:       tickets,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// ---------------------------------------------------------------------------
// HEALTH CHECK
// ---------------------------------------------------------------------------

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	if err := s.db.PingContext(r.Context()); err != nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":   "unhealthy",
			"database": "unreachable",
			"error":    err.Error(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "support-service",
		"database":  "connected",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ---------------------------------------------------------------------------
// DATABASE SETUP
// ---------------------------------------------------------------------------

func setupDatabase(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("Fehler beim Öffnen der Datenbankverbindung: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("Datenbankverbindung fehlgeschlagen: %w", err)
	}
	if err := migrateDatabase(ctx, db); err != nil {
		return nil, fmt.Errorf("Datenbankmigration fehlgeschlagen: %w", err)
	}
	return db, nil
}

// migrateDatabase erstellt die erforderlichen Tabellen, falls sie nicht existieren.
// DSGVO: Alle Tabellen unterstützen das Recht auf Löschung (Art. 17 DSGVO).
func migrateDatabase(ctx context.Context, db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS tickets (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			description TEXT NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'open',
			priority VARCHAR(50) NOT NULL DEFAULT 'medium',
			category VARCHAR(100) NOT NULL DEFAULT 'other',
			customer_id BIGINT NOT NULL,
			agent_id BIGINT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			resolved_at TIMESTAMPTZ,
			due_date TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_customer_id ON tickets(customer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_agent_id ON tickets(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS ticket_comments (
			id SERIAL PRIMARY KEY,
			ticket_id BIGINT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
			author_id BIGINT NOT NULL,
			author_type VARCHAR(20) NOT NULL DEFAULT 'customer',
			content TEXT NOT NULL,
			is_internal BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_ticket_id ON ticket_comments(ticket_id)`,
		`CREATE TABLE IF NOT EXISTS kb_articles (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			category VARCHAR(100) NOT NULL DEFAULT 'general',
			tags TEXT NOT NULL DEFAULT '',
			language VARCHAR(10) NOT NULL DEFAULT 'de',
			is_published BOOLEAN NOT NULL DEFAULT FALSE,
			view_count BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_kb_articles_category ON kb_articles(category)`,
		`CREATE INDEX IF NOT EXISTS idx_kb_articles_language ON kb_articles(language)`,
		`CREATE INDEX IF NOT EXISTS idx_kb_articles_published ON kb_articles(is_published)`,
		`CREATE TABLE IF NOT EXISTS support_agents (
			id SERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL UNIQUE,
			role VARCHAR(50) NOT NULL DEFAULT 'agent',
			department VARCHAR(100) NOT NULL DEFAULT 'general',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			max_tickets INT NOT NULL DEFAULT 20,
			current_tickets INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agents_is_active ON support_agents(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_agents_department ON support_agents(department)`,
	}
	for _, migration := range migrations {
		if _, err := db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("Migration fehlgeschlagen: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// MAIN
// ---------------------------------------------------------------------------

func main() {
	logger := log.New(os.Stdout, "[SUPPORT-SERVICE] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Support-Service wird gestartet...")
	logger.Println("DSGVO-Hinweis: Alle personenbezogenen Daten werden gemäß EU-Datenschutzgrundverordnung verarbeitet.")

	// Konfiguration aus Umgebungsvariablen
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "support_db")
	dbSSLMode := getEnv("DB_SSLMODE", "require") // Verschlüsselte Verbindung (DSGVO)
	serverPort := getEnv("SERVER_PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "")

	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET Umgebungsvariable ist nicht gesetzt")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	logger.Printf("Verbinde mit Datenbank: host=%s port=%s dbname=%s", dbHost, dbPort, dbName)
	db, err := setupDatabase(dsn)
	if err != nil {
		logger.Fatalf("Datenbankverbindung fehlgeschlagen: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Printf("Fehler beim Schließen der Datenbankverbindung: %v", err)
		}
	}()
	logger.Println("Datenbankverbindung hergestellt und Migrationen abgeschlossen")

	server := NewServer(db, jwtSecret)

	httpServer := &http.Server{
		Addr:         ":" + serverPort,
		Handler:      server.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful Shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Printf("Support-Service läuft auf Port %s", serverPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP-Server Fehler: %v", err)
		}
	}()

	<-shutdownCh
	logger.Println("Shutdown-Signal empfangen. Graceful Shutdown wird eingeleitet...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Printf("Fehler beim Graceful Shutdown: %v", err)
	} else {
		logger.Println("Server erfolgreich heruntergefahren")
	}
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
