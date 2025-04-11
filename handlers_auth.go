package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/paokimsiwoong/chirpy/internal/auth"
	"github.com/paokimsiwoong/chirpy/internal/database"
)

// /api/login path POST handler : 로그인 요청 처리
func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	// request body의 json 데이터를 담을 구조체
	reqBody := uReqBody{}

	// @@@ 모든 타입은 empty interface를 구현하므로 임의 타입을 담을 container로 사용가능

	// request body decoding
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding resquest body json", fmt.Errorf("error decoding resquest body json: %w", err))
		// code는 500
		return
	}

	user, err := cfg.ptrDB.GetUserByEmail(r.Context(), reqBody.Email)
	// http.Request의 Context() method는 req의 context.Context를 반환
	// ==> 만약 접속이 끊기거나 타임아웃이 되면 그 정보가 context로 전달되서 db 쿼리를 알아서 중단시켜준다
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error incorrect email or password", fmt.Errorf("error incorrect email or password: %w", err)) // 4번째 인자 error는 response로 가지 않으므로 이메일 오류인지 비밀번호 오류인지 요청자가 알 수 없다.
		// code는 401
		return
	}

	if err := auth.CheckPasswordHash(user.HashedPassword, reqBody.Password); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error incorrect email or password", fmt.Errorf("error incorrect email or password: %w", err)) // 4번째 인자 error는 response로 가지 않으므로 이메일 오류인지 비밀번호 오류인지 요청자가 알 수 없다.
		// code는 401
		return
	}
	// err == nil 이면 비밀번호 일치

	// token 수명 설정값
	// var expiresIn time.Duration
	// if reqBody.ExpiresInSeconds > 3600 || reqBody.ExpiresInSeconds == 0 {
	// 	expiresIn = time.Hour
	// } else {
	// 	// expiresIn = time.Duration(reqBody.ExpiresInSeconds)
	// 	// @@@ 해답 참조 후 수정
	// 	expiresIn = time.Duration(reqBody.ExpiresInSeconds) * time.Second
	// }
	// @@@ 1시간으로 고정

	// JWT 생성
	tokenString, err := auth.MakeJWT(user.ID, cfg.tokenSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating token", fmt.Errorf("error creating token: %w", err))
		// code는 500
		return
	}

	// refresh token (32 byte hex-encoded string) 생성
	refreshTokenString, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating refresh token string", fmt.Errorf("error creating refresh token string: %w", err))
		// code는 500
		return
	}

	// db에 생성한 refresh token 입력
	_, err = cfg.ptrDB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:  refreshTokenString,
		UserID: user.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating refresh token in DB", fmt.Errorf("error creating refresh token in DB: %w", err))
		// code는 500
		return
	}

	// json에 저장할 데이터들 구조체에 저장
	resBody := uResBodySuccess{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        tokenString,
		RefreshToken: refreshTokenString,
		// @@@ hashed password는 절대 response로 반환하면 안된다 => 보안문제
	}

	respondWithJSON(w, http.StatusOK, resBody)
	// code는 200
}

// /api/refresh path POST handler : valid한 refresh token이 있으면 새로운 1시간짜리 jwt 생성
func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {

	type rResBodySuccess struct {
		Token string `json:"token"`
	}

	// refresh token이 Authorization header에 저장되어 있는지 확인
	refreshTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error parsing header", fmt.Errorf("error parsing header: %w", err))
		// code 401
		return
	}

	// db에서 refresh token 찾기
	refreshToken, err := cfg.ptrDB.GetRefreshToken(r.Context(), refreshTokenString)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error refresh token invalid or expired", fmt.Errorf("error refresh token invalid or expired: %w", err))
		// code는 401
		return
	}

	// expire, revoke 여부 확인
	if time.Now().After(refreshToken.ExpiresAt) || refreshToken.RevokedAt.Valid { // sql.NullTime.Valid 가 true => revoked_at이 NULL이 아님 ==> revoke
		respondWithError(w, http.StatusUnauthorized, "Error refresh token invalid or expired", errors.New("error refresh token invalid or expired"))
		// code는 401
		return
	}
	// @@@ 해답은 sql 쿼리 안에서 expired, revoked 여부 걸러서 반환하므로 이 조건문 블록 필요없음

	// 새로 발급할 1시간짜리 JWT 생성
	tokenString, err := auth.MakeJWT(refreshToken.UserID, cfg.tokenSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating token", fmt.Errorf("error creating token: %w", err))
		// code는 500
		return
	}

	// json에 저장할 데이터들 구조체에 저장
	resBody := rResBodySuccess{
		Token: tokenString,
	}

	respondWithJSON(w, http.StatusOK, resBody)
	// code는 200
}

// /api/revoke path POST handler : refresh token revoke
func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	// refresh token이 Authorization header에 저장되어 있는지 확인
	refreshTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error parsing header", fmt.Errorf("error parsing header: %w", err))
		// code 401
		return
	}

	// db의 해당 refresh token의 revoked_at, updated_at 필드 업데이트
	if err := cfg.ptrDB.RevokeRefreshToken(r.Context(), refreshTokenString); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error invalid refresh token", fmt.Errorf("error invalid refresh token: %w", err))
		// code는 401
		return
	}

	w.WriteHeader(http.StatusNoContent)
	// code 204
}
