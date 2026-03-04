@echo off
REM Script to install dependencies for xzdp-go project
REM Author: xzdp-go dev
REM Date: 2026-03-03

echo === Starting to install dependencies for xzdp-go ===

REM 1. Initialize go module (if not exists)
if not exist go.mod (
    echo Initializing go module: xzdp-go
    go mod init xzdp-go
)

REM 2. Install core web framework (gin)
echo Installing gin web framework...
go get github.com/gin-gonic/gin@v1.10.0

REM 3. Install ORM framework (gorm) and mysql driver
echo Installing gorm and mysql driver...
go get gorm.io/gorm@v1.25.0
go get gorm.io/driver/mysql@v1.5.2

REM 4. Install JWT for authentication
echo Installing JWT library...
go get github.com/golang-jwt/jwt/v5@v5.2.0

REM 5. Install bcrypt for password encryption
echo Installing bcrypt for password encryption...
go get golang.org/x/crypto/bcrypt@v0.21.0

REM 6. Install Redis
echo Installing dotenv for environment config...
go get github.com/go-redis/redis/v8

REM 7. Install dotenv for config management
echo Installing dotenv for environment config...
go get github.com/joho/godotenv@v1.5.1

REM 8. Tidy dependencies (clean unused)
echo Tidying dependencies...
go mod tidy

echo === All dependencies installed successfully! ===
pause