package main

import (
	"fmt"
	"net/http"
)

func main() {
	// http.NewServeMux() 함수는 메모리에 새로 http.ServeMux를 할당하고 그 포인터를 반환
	ServeMux := http.NewServeMux()
	// http.Handler 인터페이스는 ServeHTTP(ResponseWriter, *Request) method을 가진다
	// ===> http.ServeMux (type ServeMux struct) 는 ServeHTTP 메소드를 가지고 있으므로 http.Handler 인터페이스를 구현한다
	Server := http.Server{
		Addr:    ":8080", // 지정하지 않으면 기본값 ":http" (port 80)
		Handler: ServeMux,
	}

	err := Server.ListenAndServe()

	if err != nil {
		fmt.Printf("error: %v", err)
	}
}
