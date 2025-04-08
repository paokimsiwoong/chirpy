package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// 검열할 단어 리스트와 텍스트를 받아서 검열하는 함수
func censor(text string) string {

	splitBody := strings.Split(text, " ")
	toJoin := make([]string, len(splitBody))
	for _, word := range splitBody {
		lowered := strings.ToLower(word)
		if strings.Contains(lowered, "sharbert") && !strings.Contains(lowered, "!") {
			toJoin = append(toJoin, strings.Replace(lowered, "sharbert", "****", 1))
			continue
		}
		if strings.Contains(lowered, "kerfuffle") {
			toJoin = append(toJoin, strings.Replace(lowered, "kerfuffle", "****", 1))
			continue
		}
		if strings.Contains(lowered, "fornax") {
			toJoin = append(toJoin, strings.Replace(lowered, "fornax", "****", 1))
			continue
		}
		// @@@ 단어 양옆에 . , " ' 등 특수 문자가 붙어 있는 경우도 대처 가능하지만
		// @@@ 검열할 단어가 늘어날 수록 map[string]struct{}에 검열할 단어를 key로 저장하는 해답 방식이 더 좋다

		toJoin = append(toJoin, word)
	}

	return strings.Trim(strings.Join(toJoin, " "), " ")
}

// @@@ 해답 예시
// func getCleanedBody(body string, badWords map[string]struct{}) string {
// @@@@ empty struct는 메모리에서 0 byte 차지 ==> go의 모든 타입 중 제일 작다
// @@@@ ====> 이 예제의 map[string]struct{} 형태로 단어 리스트를 만든 뒤 map key 확인(value, ok := map[key]) 구문 활용
// 	words := strings.Split(body, " ")
// 	for i, word := range words {
// 		loweredWord := strings.ToLower(word)
// 		if _, ok := badWords[loweredWord]; ok {
// 			words[i] = "****"
// 		}
// 	}
// 	cleaned := strings.Join(words, " ")
// 	return cleaned
// }

// @@@ 해답의 DRY 코드1 :
// respondWithError는 입력된 error를 log.Println으로 출력하고 입력된 msg를 json에 담아 response하는 함수
func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

// @@@ 해답의 DRY 코드2 : writer, code, 데이터를 받아 status code 설정하고, 데이터를 JSON으로 변환해 response body에 담아 response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}
