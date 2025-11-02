package http

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/internal/clients"
	"ride-sharing/services/api-gateway/internal/dto"
	"ride-sharing/shared/contracts"
)

type TripHandler struct {
	tripClient clients.TripServiceClient
}

func NewTripHandler(tripClient clients.TripServiceClient) *TripHandler {
	return &TripHandler{
		tripClient: tripClient,
	}
}

// Http handler to preview a trip
func (h *TripHandler) HandleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody dto.PreviewTripRequest

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	if reqBody.UserID == "" {
		http.Error(w, "UserID is required", http.StatusBadRequest)
		return
	}

	tripPreview, err := h.tripClient.PreviewTrip(r.Context(), reqBody.ToProto())
	if err != nil {
		log.Printf("failed to preview trip: %v", err)
		http.Error(w, "failed to preview trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: tripPreview}
	WriteJSON(w, http.StatusCreated, response)
}

// Http handler to create a trip
func (h *TripHandler) HandleCreateTrip(w http.ResponseWriter, r *http.Request) {
	var reqBody dto.StartTripRequest

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Failed to parse JSON data", http.StatusBadRequest)
		return
	}

	trip, err := h.tripClient.CreateTrip(r.Context(), reqBody.ToProto())
	if err != nil {
		log.Printf("Failed to create trip: %v", err)
		http.Error(w, "Failed to create trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: trip}
	WriteJSON(w, http.StatusCreated, response)
}
