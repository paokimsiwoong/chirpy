# chirpy
## a toy project for practicing go HTTP server
### BOOT.DEV 백엔드 과정에서 진행한 간단한 Go, HTTP server, postgreSQL 토이 프로젝트로 가상의 메신저 앱 Chirpy를 위한 백엔드 api를 Go의 net/http 패키지를 이용해 만들었습니다.
### 사용자가 보낸 여러 http 리퀘스트들(아이디 생성, 로그인, JWT 갱신, 게시글 생성 및 삭제)들을 처리하고 사용자 정보, 게시글 정보 등을 DB에서 관리합니다. 

***
***
<details>
<summary> <h2> Prerequisites </h2> </summary>
<div markdown="1">

### 1. Install go v1.24 or later
```bash
curl -sS https://webi.sh/golang | sh
```

### 2. Install Postgres v15 or later
#### 2-1. Install
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
```
#### 2-2. Set a password for user postgres
```bash
sudo passwd postgres
# set a password for user postgres
```
#### 2-3. Start the Postgres server in the background
```bash
sudo service postgresql start
```
#### 2-4. Enter the `psql` shell
```bash
sudo -u postgres psql
# psql shell should show a new prompt : postgres=#
```
#### 2-5. Create a new database
```bash
# while in psql shell
CREATE DATABASE <db_name>;
# ex: CREATE DATABASE chirpy;
```
#### 2-6. Set the database user's password
```bash
# while in psql shell
# connect to the new database
\c <db_name>
# then psql shell should show a new prompt : <db_name>=#

# set the database user's password
ALTER USER postgres PASSWORD '<your_password>';
# this password is the one used in your connection string
``` 

</div>
</details>

***

> ## How to Install
### 1. clone the project.
```bash
git clone https://github.com/paokimsiwoong/chirpy
```
### 2. create .env file at the root of the project
#### the .env file must contain the connection string to your sql database and a secret key for JWT generation.
```
# see .env_example file
DB_URL="postgres://<username>:<password>@localhost:5432/<dbname>?sslmode=disable"
TOKEN_SECRET="create your secret key for JWT gen and store here. you could use `openssl rand -base64 64` to create"
```

### 3. install goose and run up migrations in the project's sql/schema directory 
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```
#### `cd` into the project's sql/schema directory and run
```bash
goose postgres <connection_string> up
```
> #### your connection string should look like this
> ```
> "postgres://postgres:<database user's password>@localhost:5432/<database name>"
> ```
>> *Postgres' default port is :`5432`*

### 4. build the project and run the server
```bash
go build -o <name> && ./<name>
```