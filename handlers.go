package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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

	user, err := cfg.ptrDB.CreateUser(r.Context(), reqBody.Email)
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
	}

	// HTTP 201 Created는 http.StatusCreated
	code = http.StatusCreated

	respondWithJSON(w, code, resBody)
}

// /api/chirps path POST handler : 새로운 chirp post 생성
// apiConfig의 ptrDB에 접근해야 하므로 apiConfig의 method으로 정의
func (cfg *apiConfig) handlerChirpsPOST(w http.ResponseWriter, r *http.Request) {
	// request body의 json 데이터를 담을 구조체
	reqBody := cReqBody{}
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
		UserID: reqBody.UserID,
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
