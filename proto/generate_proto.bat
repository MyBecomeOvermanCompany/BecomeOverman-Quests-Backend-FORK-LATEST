@echo off
REM Генерация Go gRPC кода из proto файла
REM Требуется: protoc, protoc-gen-go, protoc-gen-go-grpc
REM Установка плагинов:
REM   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
REM   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

cd /d "%~dp0.."
mkdir internal\generated\recommendation 2>NUL
protoc --proto_path=proto --go_out=internal/generated/recommendation --go_opt=paths=source_relative --go-grpc_out=internal/generated/recommendation --go-grpc_opt=paths=source_relative proto/recommendation.proto
echo Done! Generated files in internal/generated/recommendation/
pause
