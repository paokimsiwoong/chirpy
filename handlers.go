package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	// fileserverHits 값을 0으로 초기화
	cfg.fileserverHits.Store(0)

	// header 설정
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	// .Set은 기존값이 있으면 그걸 대체하지만 .Add는 기존값에 추가로 append
	// @@@ 해답은 header 맵 설정 없음

	// status code 는 200 ok 지정
	w.WriteHeader(http.StatusOK)

	// response body 설정
	w.Write([]byte("Hits reset"))
}

// /api/validate_chirp path handler : POST request를 받아 json body를 decoding하고 적절한 처리 결과를 json에 담아 response로 전송
func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {

	// request body의 json 데이터를 담을 구조체
	reqBody := vcReqBody{}
	// status code 담을 int
	var code int
	// resBodyFail, resBodySuccess 둘다 vcbody interface를 구현
	// var resBody vcBody
	var resBody interface{}
	// @@@ 모든 타입은 empty interface를 구현하므로 임의 타입을 담을 container로 사용가능

	// request body decoding
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		code = 500
		errMessage := fmt.Sprintf("Error decoding resquest body json: %v", err)
		resBody = vcResBodyFail{Error: errMessage}
		// @@@ 해답처럼 log 사용하기
		log.Println(errMessage)
	} else {
		// @@@ 길이 확인 부분을 else 부분에 넣지 않고 바깥에 두면 decoding err가 nil이 아닐 때도,
		// @@@ reqBody.Body는 zero value로 len 사용가능하기때문에 길이 확인 부분이 실행된다
		// @@@ ==> len(reqBody.Body)는 0 이기 때문에 code가 500에서 200으로 바뀌어 버린다

		// 문자열 길이 확인 후 140 초과면 에러, 아니면 성공
		if len(reqBody.Body) > 140 {
			code = 400
			resBody = vcResBodyFail{Error: "Chirp is too long"}
		} else {
			code = 200

			resBody = vcResBodySuccess{CleanedBody: censor(reqBody.Body)} // utils.go의 censor 함수로 비속어 처리
		}
	}

	// data, err := json.Marshal(resBody)
	// if err != nil {
	// 	code = 500
	// 	errMessage := fmt.Sprintf("Error marshalling response body json: %v", err)
	// 	// resBody = vcResBodyFail{Error: errMessage} marshal 에러 후 block이므로 새로이 marshalling이 필요한 struct 필요 없음
	// 	// @@@ 해답처럼 log 사용하기
	// 	log.Println(errMessage)

	// 	w.WriteHeader(code)
	// 	return
	// 	// w.Write에 넣을 data가 없으므로 여기서 바로 return
	// }

	// // header 설정
	// w.Header().Add("Content-Type", "application/json")

	// // status code 설정
	// w.WriteHeader(code)
	// // @@@ int 숫자대신 해답처럼 http 패키지의 const
	// // http.StatusOK(200), http.StatusBadRequest(400), http.StatusInternalServerError(500)
	// // @@@ 사용해도 된다

	// // response body 쓰기
	// w.Write(data)

	// @@@ 해답의 DRY
	respondWithJSON(w, code, resBody)
}
