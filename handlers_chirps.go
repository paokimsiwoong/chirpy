package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/paokimsiwoong/chirpy/internal/auth"
	"github.com/paokimsiwoong/chirpy/internal/database"
)

// /api/chirps path POST handler : 새로운 chirp post 생성
// apiConfig의 ptrDB에 접근해야 하므로 apiConfig의 method으로 정의
func (cfg *apiConfig) handlerChirpsPOST(w http.ResponseWriter, r *http.Request) {
	// tokenString이 Authorization header에 저장되어 있는지 확인
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error parsing header", fmt.Errorf("error parsing header: %w", err))
		// code 401
		return
	}

	// JWT 검증
	userID, err := auth.ValidateJWT(tokenString, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error invalid token", fmt.Errorf("error invalid token: %w", err))
		// code 401
		return
	}

	// request body의 json 데이터를 담을 구조체
	reqBody := cReqBody{}
	// status code 담을 int
	var code int

	// request body decoding
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		code = 500
		respondWithError(w, code, "Error decoding resquest body json", fmt.Errorf("error decoding resquest body json: %w", err))
		// respondWithError는 입력된 error를 log.Println으로 출력하고 입력된 msg를 json에 담아 response하는 함수
		return
	}

	// 문자열 길이 확인 후 140 초과면 에러, 아니면 성공
	if len(reqBody.Body) > 140 {
		code = 400
		respondWithError(w, code, "Error posting chirp : Chirp is too long", errors.New("chirp is too long"))
		return
	}

	cleaned := censor(reqBody.Body)
	// 특정 단어들 검열

	chirp, err := cfg.ptrDB.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleaned,
		UserID: userID,
	})
	// http.Request의 Context() method는 req의 context.Context를 반환
	// ==> 만약 접속이 끊기거나 타임아웃이 되면 그 정보가 context로 전달되서 db 쿼리를 알아서 중단시켜준다
	if err != nil {
		code = 500
		respondWithError(w, code, "Error creating chirp in DB", fmt.Errorf("error creating chirp in DB: %w", err))
		return
	}

	// json에 저장할 데이터들 구조체에 저장
	resBody := cResBodySuccess{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	// HTTP 201 Created는 http.StatusCreated
	code = http.StatusCreated

	respondWithJSON(w, code, resBody)
}

// /api/chirps path GET handler : 모든 chirps 반환
// ? 쿼리에 "author_id" 또는 "sort" key가 있을 때와 없을 때 행동이 다름
func (cfg *apiConfig) handlerChirpsGET(w http.ResponseWriter, r *http.Request) {
	var chirps []database.Chirp
	var err error

	// query 확인하기
	// @@@ query는 ?first=name&second=age와 같이 &로 여러개의 key, value pair가 포함될 수 있다
	// @@@ ===> r.URL.Query().Has(key)와 r.URL.Query().Get(key) 활용해 key 값별로 존재 여부 확인 및 불러오기 가능
	if r.URL.Query().Has("author_id") {
		// .Get은 key값이 설정되지 않은 경우 "" 반환되지만 조건문에서 .Has로 key 설정 여부 확인 가능
		userID, err := uuid.Parse(r.URL.Query().Get("author_id"))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Error parsing string to uuid", fmt.Errorf("error parsing string to uuid: %w", err))
			// code 400
			return
		}
		chirps, err = cfg.ptrDB.GetChirpsByAuthorID(r.Context(), userID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error getting chirp list by author id in DB", fmt.Errorf("error getting chirp list by author id in DB: %w", err))
			return
		}
	} else {
		chirps, err = cfg.ptrDB.GetChirps(r.Context())
		// http.Request의 Context() method는 req의 context.Context를 반환
		// ==> 만약 접속이 끊기거나 타임아웃이 되면 그 정보가 context로 전달되서 db 쿼리를 알아서 중단시켜준다
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error getting chirp list in DB", fmt.Errorf("error getting chirp list in DB: %w", err))
			return
		}
	}
	// @@@ 해답은 user_id로 거르는 별도의 SQL 쿼리를 쓰지 않고 밑의 for문에서 userid 일치하는 chirp만 append하는 방식 사용

	// json에 저장할 데이터들 구조체에 저장
	var resBody []cResBodySuccess
	// resBody := make([]cResBodySuccess, len(chirps)) 사용하면 쓰레기 json이 두개 앞에 추가됨??

	for _, chirp := range chirps {
		resBody = append(resBody, cResBodySuccess{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}

	// query 확인하기 2
	// sort 키가 있으면 value "asc", "desc"에 따라 정렬하기(created_at 기준)
	if r.URL.Query().Get("sort") == "desc" {
		sort.Slice(resBody, func(i, j int) bool { return resBody[i].CreatedAt.After(resBody[j].CreatedAt) })
		// 이미 resBody가 asc로 정렬되어 있기때문에 slices.Reverse(resBody)도 가능
	}
	// @@@ 해답 예시
	// sortDirection := "asc"
	// sortDirectionParam := r.URL.Query().Get("sort")
	// if sortDirectionParam == "desc" {
	// 	sortDirection = "desc"
	// }
	// sort.Slice(chirps, func(i, j int) bool {
	// 	if sortDirection == "desc" {
	// 		return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
	// 	}
	// 	return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
	// })

	respondWithJSON(w, http.StatusOK, resBody)
}

// /api/chirps/{chirpID} path GET handler : 특정 id chirp 반환
func (cfg *apiConfig) handlerChirpsGETOne(w http.ResponseWriter, r *http.Request) {
	// r.PathValue(path parameter 이름)로 chirpID 가져오고
	// string 형태인 uuid를 uuid.Parse함수로 uuid.UUID 타입으로 변환
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing string to uuid", fmt.Errorf("error parsing string to uuid: %w", err))
		return
	}

	// db에서 해당 id로 chirp 가져오기
	chirp, err := cfg.ptrDB.GetChirpByID(r.Context(), chirpID)
	// http.Request의 Context() method는 req의 context.Context를 반환
	// ==> 만약 접속이 끊기거나 타임아웃이 되면 그 정보가 context로 전달되서 db 쿼리를 알아서 중단시켜준다
	if err != nil {
		// respondWithError(w, http.StatusInternalServerError, "Error getting a chirp in DB", fmt.Errorf("error getting a chirp in DB: %w", err))
		respondWithError(w, http.StatusNotFound, "Error finding a chirp in DB", fmt.Errorf("error finding a chirp in DB: %w", err))
		// code 404
		return
	}

	// json에 저장할 데이터들 구조체에 저장
	resBody := cResBodySuccess{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	respondWithJSON(w, http.StatusOK, resBody)
}

// /api/chirps/{chirpID} path DELETE handler : 특정 id chirp 삭제
func (cfg *apiConfig) handlerChirpsDELETEOne(w http.ResponseWriter, r *http.Request) {
	// jWT sting이 Authorization header에 저장되어 있는지 확인
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error parsing header", fmt.Errorf("error parsing header: %w", err))
		// code 401
		return
	}

	// JWT 검증
	userID, err := auth.ValidateJWT(tokenString, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error invalid token", fmt.Errorf("error invalid token: %w", err))
		// code 401
		return
	}

	// r.PathValue(path parameter 이름)로 chirpID 가져오고
	// string 형태인 uuid를 uuid.Parse함수로 uuid.UUID 타입으로 변환
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error parsing string to uuid", fmt.Errorf("error parsing string to uuid: %w", err))
		// code 400
		return
	}

	// chirpID에 해당하는 chirp가 있는지 확인하고 가져오기
	chirp, err := cfg.ptrDB.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error finding a chirp in DB", fmt.Errorf("error finding a chirp in DB: %w", err))
		// code 404
		return
	}

	// chirp의 작성자와 지금 지우려는 유저가 동일 유저인지 확인
	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Error can't delete other user's chirp", errors.New("error can't delete other user's chirp"))
		// code 403
		return
	}

	// chirp db에서 삭제
	if err := cfg.ptrDB.DeleteChirpByID(r.Context(), database.DeleteChirpByIDParams{
		ID:     chirpID,
		UserID: userID,
	}); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting chirp in DB", fmt.Errorf("error deleting chirp in DB: %w", err))
		return
	}

	// 정상적으로 삭제가 완료되면 status code 설정 후 함수 종료
	w.WriteHeader(http.StatusNoContent)
	// code 204
}
