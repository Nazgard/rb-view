# ============ Этап 1: Сборка приложения ============
FROM golang:1.23-alpine AS builder

# Устанавливаем необходимые инструменты
RUN apk add --no-cache git

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum (для кэширования зависимостей)
#COPY go.mod go.sum ./

# Загружаем зависимости
#RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем статический бинарник (без CGO, для максимальной совместимости с scratch)
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -o bot-rb main.go

# ============ Этап 2: Финальный минимальный образ ============
FROM scratch

# Копируем только исполняемый файл из этапа сборки
COPY --from=builder /app/bot-rb /bot-rb

# Указываем точку входа
ENTRYPOINT ["/bot-rb"]

# Порт, на котором работает приложение (опционально, для документации)
EXPOSE 8080