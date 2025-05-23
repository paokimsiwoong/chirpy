package main

import (
	"fmt"
	"net/http"
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
