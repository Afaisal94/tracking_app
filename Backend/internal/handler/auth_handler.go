package handler

import (
	"encoding/json"
	"net/http"

	"tracking-backend/internal/middleware"
	"tracking-backend/internal/models"
	"tracking-backend/internal/service"
	"tracking-backend/internal/utils"
)

type AuthHandler struct {
	authService *service.AuthService
	trackingSvc *service.TrackingService
	redisSvc    *service.RedisService
}

func NewAuthHandler(authService *service.AuthService, trackingSvc *service.TrackingService, redisSvc *service.RedisService) *AuthHandler {
	return &AuthHandler{authService: authService, trackingSvc: trackingSvc, redisSvc: redisSvc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.authService.Register(r.Context(), req)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(w, http.StatusCreated, "registration successful", map[string]interface{}{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"phone": user.Phone,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	// Auto-start tracking session on login
	if resp.User != nil {
		_, _ = h.trackingSvc.StartSession(r.Context(), resp.User.ID)
	}
	utils.Success(w, http.StatusOK, "login successful", resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	utils.Success(w, http.StatusOK, "token refreshed", resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r)

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	_ = h.authService.Logout(r.Context(), body.RefreshToken, userID)

	// End active tracking session
	if userID.String() != "00000000-0000-0000-0000-000000000000" {
		_, _ = h.trackingSvc.EndSession(r.Context(), userID)
		// Clear Redis session state
		if h.redisSvc != nil {
			_ = h.redisSvc.DeleteActiveSession(r.Context(), userID)
			_ = h.redisSvc.DeleteWSConn(r.Context(), userID)
			// Publish force disconnect for WS coordination
			_ = h.redisSvc.PublishForceDisconnect(r.Context(), userID)
		}
	}

	utils.Success(w, http.StatusOK, "logout successful", nil)
}
