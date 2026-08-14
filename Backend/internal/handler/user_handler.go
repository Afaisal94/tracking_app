package handler

import (
	"encoding/json"
	"net/http"

	"tracking-backend/internal/middleware"
	"tracking-backend/internal/models"
	"tracking-backend/internal/repository"
	"tracking-backend/internal/utils"
)

type UserHandler struct {
	userRepo *repository.UserRepo
}

func NewUserHandler(userRepo *repository.UserRepo) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to fetch profile")
		return
	}
	if user == nil {
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}
	utils.Success(w, http.StatusOK, "profile fetched", map[string]interface{}{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"phone":      user.Phone,
		"avatar_url": user.AvatarURL,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil || user == nil {
		utils.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	utils.Success(w, http.StatusOK, "profile updated", map[string]interface{}{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"phone":      user.Phone,
		"avatar_url": user.AvatarURL,
	})
}
