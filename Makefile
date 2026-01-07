# Makefile для сборки и деплоя Docker-образа в Yandex Cloud Container Registry

# Название образа и тег (можно переопределить при вызове: make deploy TAG=latest)
REGISTRY    := cr.yandex/crpbjrij9hsbb1q5fj6s
IMAGE_NAME  := bot-rb
TAG         ?= latest  # по умолчанию latest, можно указать другой

FULL_IMAGE  := $(REGISTRY)/$(IMAGE_NAME):$(TAG)

# Путь к Dockerfile (если он в корне проекта — оставляем как есть)
DOCKERFILE  := Dockerfile

# Цели
.PHONY: build push deploy login clean

# Сборка образа локально
build:
	@echo "Сборка Docker-образа $(FULL_IMAGE)..."
	docker build -t $(FULL_IMAGE) -f $(DOCKERFILE) .

# Авторизация в Yandex Container Registry
login:
	@echo "Авторизация в Yandex Container Registry..."
	@yc container registry configure-docker
	# Или альтернативно, если используете docker login с токеном:
	# docker login --username iam --password $(yc iam create-token) cr.yandex

# Пуш образа в реестр
push: login
	@echo "Пуш образа $(FULL_IMAGE) в реестр..."
	docker push $(FULL_IMAGE)

# Полный деплой: сборка + пуш
deploy: build push
	@echo "Деплой завершен: $(FULL_IMAGE)"

# Очистка локальных образов (опционально)
clean:
	@echo "Удаление локального образа $(FULL_IMAGE)..."
	docker rmi $(FULL_IMAGE) || true

# Показать текущие переменные (для отладки)
info:
	@echo "Registry: $(REGISTRY)"
	@echo "Image:    $(IMAGE_NAME)"
	@echo "Tag:      $(TAG)"
	@echo "Full:     $(FULL_IMAGE)"