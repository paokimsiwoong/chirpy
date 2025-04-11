package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/paokimsiwoong/chirpy/internal/auth"
	"github.com/paokimsiwoong/chirpy/internal/database"
)

// /api/users path POST handler : 유저 생성 및 db 저장
// apiConfig의 ptrDB에 접근해야 하므로 apiConfig의 method으로 정의
func (cfg *apiConfig) handlerUsersPOST(w http.ResponseWriter, r *http.Request) {

	// request body의 json 데이터를 담을 구조체
	reqBody := uReqBody{}
	// status code 담을 int
	var code int

	// @@@ 모든 타입은 empty interface를 구현하므로 임의 타입을 담을 container로 사용가능

	// request body decoding
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		code = 500
		respondWithError(w, code, "Error decoding resquest body json", fmt.Errorf("error decoding resquest body json: %w", err))
		// respondWithError는 입력된 error를 log.Println으로 출력하고 입력된 msg를 json에 담아 response하는 함수
		return
	}

	hashed, err := auth.HashPassword(reqBody.Password)
	if err != nil {
		code = http.StatusInternalServerError // 500
		respondWithError(w, code, "Error hashing password", fmt.Errorf("error hashing password: %w", err))
		return
	}

	user, err := cfg.ptrDB.CreateUser(r.Context(), database.CreateUserParams{
		Email:          reqBody.Email,
		HashedPassword: hashed,
	})
	// http.Request의 Context() method는 req의 context.Context를 반환
	// ==> 만약 접속이 끊기거나 타임아웃이 되면 그 정보가 context로 전달되서 db 쿼리를 알아서 중단시켜준다
	if err != nil {
		code = 500
		respondWithError(w, code, "Error creating user in DB", fmt.Errorf("error creating user in DB: %w", err))
		// respondWithError는 입력된 error를 log.Println으로 출력하고 입력된 msg를 json에 담아 response하는 함수
		return
	}

	// json에 저장할 데이터들 구조체에 저장
	resBody := uResBodySuccess{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		// @@@ hash는 절대 response로 반환하면 안된다 => 보안문제
	}

	// HTTP 201 Created는 http.StatusCreated
	code = http.StatusCreated

	respondWithJSON(w, code, resBody)
}

// /api/users path PUT handler : 유저 정보 수정
func (cfg *apiConfig) handlerUsersPUT(w http.ResponseWriter, r *http.Request) {
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

	// request body의 json 데이터를 담을 구조체
	reqBody := uReqBody{}

	// request body decoding
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding resquest body json", fmt.Errorf("error decoding resquest body json: %w", err))
		// code 500
		return
	}

	// 암호 해쉬 생성
	hashed, err := auth.HashPassword(reqBody.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing password", fmt.Errorf("error hashing password: %w", err))
		// code 500
		return
	}

	// db 안의 user 데이터 수정
	user, err := cfg.ptrDB.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          reqBody.Email,
		HashedPassword: hashed,
		ID:             userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating user in DB", fmt.Errorf("error updating user in DB: %w", err))
		// code 500
		return
	}

	// json에 저장할 데이터들 구조체에 저장
	resBody := uResBodySuccess{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		// @@@ hash는 절대 response로 반환하면 안된다 => 보안문제
	}

	respondWithJSON(w, http.StatusOK, resBody)
	// code 200
}
