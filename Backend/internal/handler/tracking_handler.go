package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"tracking-backend/internal/middleware"
	"tracking-backend/internal/models"
	"tracking-backend/internal/service"
	"tracking-backend/internal/utils"
)

type TrackingHandler struct {
	trackingSvc *service.TrackingService
}

func NewTrackingHandler(trackingSvc *service.TrackingService) *TrackingHandler {
	return &TrackingHandler{trackingSvc: trackingSvc}
}

func (h *TrackingHandler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = 20
	}

	items, total, err := h.trackingSvc.GetHistory(r.Context(), userID, page, perPage)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to fetch history")
		return
	}
	utils.SuccessWithMeta(w, http.StatusOK, "history fetched", items, map[string]interface{}{
		"page":     page,
		"per_page": perPage,
		"total":    total,
	})
}

func (h *TrackingHandler) Detail(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	date := r.URL.Query().Get("date")
	if date == "" {
		utils.Error(w, http.StatusBadRequest, "date query param required (YYYY-MM-DD)")
		return
	}
	detail, err := h.trackingSvc.GetDetail(r.Context(), userID, date)
	if err != nil {
		if err.Error() == "no data for date" {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(w, http.StatusOK, "detail fetched", detail)
}

func (h *TrackingHandler) BatchPoints(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req models.BatchPointsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	count, distance, err := h.trackingSvc.AddPoints(r.Context(), userID, req)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(w, http.StatusOK, "points saved", map[string]interface{}{
		"saved":             count,
		"distance_added_m":  distance,
	})
}

func (h *TrackingHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sess, err := h.trackingSvc.StartSession(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(w, http.StatusOK, "session started", sess)
}

func (h *TrackingHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sess, err := h.trackingSvc.EndSession(r.Context(), userID)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(w, http.StatusOK, "session ended", sess)
}
