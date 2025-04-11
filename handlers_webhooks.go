package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// /api/polka/webhooks path POST handler : polka webhooks 처리
func (cfg *apiConfig) handlerPolkaWebhooks(w http.ResponseWriter, r *http.Request) {
	type pReqBody struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	reqBody := pReqBody{}
	// request body decoding
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding resquest body json", fmt.Errorf("error decoding resquest body json: %w", err))
		// code 500
		return
	}

	// "user.upgraded" 이벤트가 아닐 경우 204 처리
	if reqBody.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		// code 204
		return
	}

	// uuid 담은 string uuid.UUID로 변환
	userID, err := uuid.Parse(reqBody.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing string to uuid", fmt.Errorf("error parsing string to uuid: %w", err))
		// code 400
		return
	}

	_, err = cfg.ptrDB.UpdateUserMembership(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // @@@ 해답의 errors.Is 활용해보기
			respondWithError(w, http.StatusNotFound, "Couldn't find user", err)
			// code 404
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error updating user in DB", fmt.Errorf("error updating user in DB: %w", err))
		// code 500
		return
	}
	// @@@ 해답 에러처리
	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		respondWithError(w, http.StatusNotFound, "Couldn't find user", err)
	// 		return
	// 	}
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't update user", err)
	// 	return
	// }

	w.WriteHeader(http.StatusNoContent)
	// code 204
}
