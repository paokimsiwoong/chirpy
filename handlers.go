package main

import (
	"fmt"
	"net/http"
)

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
