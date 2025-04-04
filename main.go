package main

import (
	"log"
	"net/http"
)

func main() {
	// @@@ 해답처럼 한 곳에서 const들 관리할 수 있도록 변경
	const rootPath = "."
	const port = "8080"

	// http.NewServeMux() 함수는 메모리에 새로 http.ServeMux를 할당하고 그 포인터를 반환
	serveMux := http.NewServeMux()
	// http.Handler 인터페이스는 ServeHTTP(ResponseWriter, *Request) method을 가진다
	// ===> http.ServeMux (type ServeMux struct) 는 ServeHTTP 메소드를 가지고 있으므로 http.Handler 인터페이스를 구현한다

	serveMux.HandleFunc("/healthz", handlerReadiness)
	// handler 함수 등록

	// @@@ 해답처럼 server 정의 전에 Handle 메소드 실행하기
	serveMux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir(rootPath))))
	// Handle 메소드의 첫번째 인자 pattern은 URL http://localhost:8080 뒤에 따라 붙는 부분 /path?query#fragment (protocoll://username:password@domain:port/path?query#fragment)
	// http.StripPrefix 함수는 request url의 특정 부분을 제거한 다음에 FileServer가 볼 수 있도록 하는 함수
	// 함수를 사용하지 않으면 /app/이 rootPath . 에 연결되지 않고 ./app에 연결되어 버린다
	// 함수 사용 후에는 /app/assets/logo.png ==> ./assets/logo.png로 연결

	// http.FileServer 함수의 인자 root(http.FileSystem 타입)는 req url가 들어오면 root 인자로 지정한 경로 + url 경로 위치의 파일을 serve
	// type Dir string는 http.FileStstem 인터페이스를 구현하는 타입 ==> 단순 string을 http.Dir 타입으로 형변환

	server := http.Server{
		Addr:    ":" + port, // 지정하지 않으면 기본값 ":http" (port 80)
		Handler: serveMux,
	}

	// @@@ 해답처럼 서버가 하는 일 log
	log.Printf("Serving files from %s on port: %s\n", rootPath, port)

	err := server.ListenAndServe()
	// @@@ when ListenAndServe() is called, the main function blocks until the server is shut down

	// if err != nil {
	// 	// fmt.Printf("error: %v", err)
	// }
	// @@@ ListenAndServe 의 err는 항상 non nil
	// @@@ (ListenAndServe always returns a non-nil error. After [Server.Shutdown] or [Server.Close], the returned error is [ErrServerClosed].)
	// @@@ 해답처럼 main에선 log.Fatal 쓰기
	log.Fatal(err)
}
