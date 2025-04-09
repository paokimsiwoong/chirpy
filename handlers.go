package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/paokimsiwoong/chirpy/internal/auth"
	"github.com/paokimsiwoong/chirpy/internal/database"
)

// /api/healthz path handler
func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	// Header() returns the header map that will be sent by [ResponseWriter.WriteHeader]. The [Header] map also is the mechanism with which [Handler] implementations can set HTTP trailers.
	// w.Header()는 http.Header 타입 반환
	headerMap := w.Header()
	// http.Header (type Header map[string][]string ) : A Header represents the key-value pairs in an HTTP header.
	headerMap.Set("Content-Type", "text/plain; charset=utf-8")
	// Content-Type: text/plain; charset=utf-8
	// @@@ 해답은 w.Header().Add("Content-Type", "text/plain; charset=utf-8")

	// header 변경은 반드시 w.WriteHeader나 w.Write 함수 호출 이전에 해야한다. 일반적인 경우에서는 호출 이후에 header를 변경해도 그 변경이 적용되지 않는다.
	// Changing the header map after a call to [ResponseWriter.WriteHeader] (or [ResponseWriter.Write]) has no effect unless the HTTP status code was of the 1xx class or the modified headers are trailers.

	// status code 는 200 ok 지정
	w.WriteHeader(http.StatusOK)

	content := "OK"
	code, err := w.Write([]byte(content))
	// @@@ 해답: w.Write([]byte(http.StatusText(http.StatusOK)))
	// @@@ Write 함수 반환값 무시
	// @@@ func http.StatusText(code int) string : StatusText returns a text for the HTTP status code. It returns the empty string if the code is unknown
	fmt.Printf("status code: %d\nerror message: %v\n", code, err)
}

// /admin/metrics path handler : cfg에 저장된 fileserverHits 값을 표시
// apiConfig의 fileserverHits에 접근해야 하므로 apiConfig의 method으로 정의
func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	// header 설정
	w.Header().Add("Content-Type", "text/html")
	// html로 설정해야 html이 담긴 response body를 정상적으로 표시

	// status code 는 200 ok 지정
	w.WriteHeader(http.StatusOK)

	// response body 설정
	formattedHtml := fmt.Sprintf(
		`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`,
		cfg.fileserverHits.Load(),
	)

	w.Write([]byte(formattedHtml))
}

// /admin/reset path handler : cfg에 저장된 fileserverHits 값을 초기화
// apiConfig의 fileserverHits에 접근해야 하므로 apiConfig의 method으로 정의
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	// 개발자가 아니면 reset 사용 금지
	if cfg.platform != "dev" {
		// respondWithError(w, http.StatusForbidden, "Only admin can POST /admin/reset", errors.New("403 Forbidden"))
		// @@@ 해답처럼 json이 아니라 단순 텍스트 표시로 변경
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Only admin can POST /admin/reset"))
		return
	}

	// fileserverHits 값을 0으로 초기화
	cfg.fileserverHits.Store(0)

	cfg.ptrDB.ResetUsers(r.Context())
	// db의 user 테이블 reset

	// header 설정
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	// .Set은 기존값이 있으면 그걸 대체하지만 .Add는 기존값에 추가로 append
	// @@@ 해답은 header 맵 설정 없음

	// status code 는 200 ok 지정
	w.WriteHeader(http.StatusOK)

	// response body 설정
	w.Write([]byte("Hits and db reset"))
}

// /api/chirps path POST handler : 새로운 chirp post 생성
// apiConfig의 ptrDB에 접근해야 하므로 apiConfig의 method으로 정의
func (cfg *apiConfig) handlerChirpsPOST(w http.ResponseWriter, r *http.Request) {
	// tokenString이 Authorization header에 저장되어 있는지 확인
	tokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error parsing header", fmt.Errorf("error parsing header: %w", err))
		// code 401
	}

	// JWT 검증
	userID, err := auth.ValidateJWT(tokenString, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error invalid token", fmt.Errorf("error invalid token: %w", err))
		// code 401
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
// apiConfig의 ptrDB에 접근해야 하므로 apiConfig의 method으로 정의
func (cfg *apiConfig) handlerChirpsGET(w http.ResponseWriter, r *http.Request) {

	chirps, err := cfg.ptrDB.GetChirps(r.Context())
	// http.Request의 Context() method는 req의 context.Context를 반환
	// ==> 만약 접속이 끊기거나 타임아웃이 되면 그 정보가 context로 전달되서 db 쿼리를 알아서 중단시켜준다
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting chirp list in DB", fmt.Errorf("error getting chirp list in DB: %w", err))
		return
	}

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

// /api/login path PSO handler : 로그인 요청 처리
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
	var expiresIn time.Duration
	if reqBody.ExpiresInSeconds > 3600 || reqBody.ExpiresInSeconds == 0 {
		expiresIn = time.Hour
	} else {
		// expiresIn = time.Duration(reqBody.ExpiresInSeconds)
		// @@@ 해답 참조 후 수정
		expiresIn = time.Duration(reqBody.ExpiresInSeconds) * time.Second
	}

	tokenString, err := auth.MakeJWT(user.ID, cfg.tokenSecret, expiresIn)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating token", fmt.Errorf("error creating token: %w", err))
		// code는 500
		return
	}

	// json에 저장할 데이터들 구조체에 저장
	resBody := uResBodySuccess{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     tokenString,
		// @@@ hashed password는 절대 response로 반환하면 안된다 => 보안문제
	}

	respondWithJSON(w, http.StatusOK, resBody)
	// code는 200
}
