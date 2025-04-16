package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // _ "github.com/lib/pq" 는 postgres driver를 사용한다고 알리는 것. main.go 내부에서 직접 코드 작성할 때 쓰이지는 않음
	"github.com/paokimsiwoong/chirpy/internal/database"
)

func main() {
	// @@@ 해답처럼 한 곳에서 const들 관리할 수 있도록 변경
	const rootPath = "."
	const port = "8080"

	// .env 파일 load해서 리눅스 환경변수에 추가
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	} // godotenv.Load(filenames ...string) 함수에 불러들일 파일들의 path들을 입력해도 된다. (입력하지 않으면 기본값 .env 파일 로드)

	// Getenv 함수로 환경변수를 불러올 수 있음
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	tokenSecret := os.Getenv("TOKEN_SECRET")
	polkaKey := os.Getenv("POLKA_KEY")

	// @@@ 해답처럼 dbURL empty string 예외처리
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	// 불러온 dbURL로 데이터베이스 연결
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to db : %v", err)
	}
	// sqlc가 생성한 database 패키지 New함수 사용
	dbQueries := database.New(db)
	// sql.DB 구조체는 database.DBTX 인터페이스를 구현하므로 New 함수에 입력 가능
	// dbQueries *database.Queries 는 db 필드에 DBTX를 저장하는 단순한 구조체

	cfg := apiConfig{
		fileserverHits: atomic.Int32{}, // @@@ 해답처럼 값 초기화 명시하기
		ptrDB:          dbQueries,
		platform:       platform,
		tokenSecret:    tokenSecret,
		polkaKey:       polkaKey,
	}

	// http.NewServeMux() 함수는 메모리에 새로 http.ServeMux를 할당하고 그 포인터를 반환
	serveMux := http.NewServeMux()
	// http.Handler 인터페이스는 ServeHTTP(ResponseWriter, *Request) method을 가진다
	// ===> http.ServeMux (type ServeMux struct) 는 ServeHTTP 메소드를 가지고 있으므로 http.Handler 인터페이스를 구현한다

	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	serveMux.HandleFunc("POST /admin/reset", cfg.handlerReset)
	// serveMux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	// POST /api/chirps에 흡수
	serveMux.HandleFunc("POST /api/users", cfg.handlerUsersPOST)
	serveMux.HandleFunc("PUT /api/users", cfg.handlerUsersPUT)

	serveMux.HandleFunc("POST /api/login", cfg.handlerLogin)
	serveMux.HandleFunc("POST /api/refresh", cfg.handlerRefresh)
	serveMux.HandleFunc("POST /api/revoke", cfg.handlerRevoke)

	serveMux.HandleFunc("POST /api/chirps", cfg.handlerChirpsPOST)
	serveMux.HandleFunc("GET /api/chirps", cfg.handlerChirpsGET)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", cfg.handlerChirpsGETOne) // {path_parameter_name}으로 path parameter 설정가능 ==> http.Request.PathValue(path_parameter_name)으로 접근
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.handlerChirpsDELETEOne)

	serveMux.HandleFunc("POST /api/polka/webhooks", cfg.handlerPolkaWebhooks)
	// handler 함수들 등록
	// pattern string의 앞부분에 HTTP method 이름을 명시해서 해당 path에 사용가능한 method을 제한할 수 있다

	// @@@ 해답처럼 server 정의 전에 Handle 메소드 실행하기
	serveMux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(rootPath)))))
	// Handle 메소드의 첫번째 인자 pattern은 URL http://localhost:8080 뒤에 따라 붙는 부분 /path?query#fragment (protocoll://username:password@domain:port/path?query#fragment)

	// http.FileServer 함수의 인자 root(http.FileSystem 타입)는 req url가 들어오면 root 인자로 지정한 경로 + url 경로 위치의 파일을 serve
	// type Dir string는 http.FileStstem 인터페이스를 구현하는 타입 ==> 단순 string을 http.Dir 타입으로 형변환

	// http.StripPrefix 함수는 request url의 특정 부분을 제거한 다음에 FileServer가 볼 수 있도록 하는 함수
	// 함수를 사용하지 않으면 /app/이 rootPath . 에 연결되지 않고 ./app에 연결되어 버린다
	// 함수 사용 후에는 /app/assets/logo.png ==> ./assets/logo.png로 연결

	// cfg.middlewareMetricsInc는 입력된 http.Handler의 ServeHTTP메소드를 그대로 호출하면서
	// 추가로 fileserverHits 가 1씩 증가시키는 ServeHTTP메소드를 가진 http.Handler를 반환

	server := http.Server{
		Addr:    ":" + port, // 지정하지 않으면 기본값 ":http" (port 80)
		Handler: serveMux,
	}

	// @@@ 해답처럼 서버가 하는 일 log
	log.Printf("Serving files from %s on port: %s\n", rootPath, port)

	if err := server.ListenAndServe(); err != nil {
		// @@@ when ListenAndServe() is called, the main function blocks until the server is shut down

		// if err != nil {
		// 	// fmt.Printf("error: %v", err)
		// }
		// @@@ ListenAndServe 의 err는 항상 non nil
		// @@@ (ListenAndServe always returns a non-nil error. After [Server.Shutdown] or [Server.Close], the returned error is [ErrServerClosed].)
		// @@@ 해답처럼 main에선 log.Fatal 쓰기
		log.Fatal(err)
	}
}
