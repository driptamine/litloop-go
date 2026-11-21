# LitLoop — Go backend

Самостоятельный перенос Django-бэкенда на Go. Проект расположен в `~/Desktop/litloop-go`, отдельно от `litloop_backend_v2`. `y_tinkering/` исключён.

Go 1.26+, Chi, GORM, PostgreSQL; SQLite для разработки. JWT, WebSocket, Redis Pub/Sub, S3/R2/GCS, SQL-очередь фоновых задач, FFmpeg. Python и Django для запуска не требуются.

## Запуск локально

```sh
cd ~/Desktop/litloop-go
cp .env.example .env
```

Укажите постоянный `JWT_SECRET` в `.env` (сгенерировать: `openssl rand -hex 32`). Переменные окружения имеют приоритет над `.env`. При пустом `DATABASE_URL` используется SQLite; для него нужен C-компилятор из Xcode Command Line Tools или GCC.

```sh
go run ./cmd/migrate
go run ./cmd/litloop-go-server
```

API: `http://localhost:8000`. Проверки: `/healthz`, `/readyz`. Список маршрутов в OpenAPI: `/openapi.json`.

Для фонового запуска API и worker: `./scripts/start-local.sh`; остановка: `./scripts/stop-local.sh`. Логи находятся в `var/log/`, PID-файлы — в `var/run/`, загруженные файлы — в `var/media/`. Скрипт запуска добавляет `bin/` в `PATH` для локальных инструментов обработки медиа.

Для PostgreSQL задайте `DATABASE_URL=postgres://user:password@localhost:5432/litloop?sslmode=disable`. Отдельный worker запускается командой `go run ./cmd/worker`. Для обработки видео установите FFmpeg. Сервер и worker должны использовать одну БД, одинаковые `JWT_SECRET`, `PUBLIC_URL` и общий `MEDIA_DIR`.

Для выгрузки новых просмотров, лайков и снятий лайка постов, альбомов и фильмов создайте Astra DB Serverless с keyspace `engagement` и задайте серверу и worker настройки `ASTRA_DB_ID` и `ASTRA_DB_APPLICATION_TOKEN`. Другое имя keyspace можно задать через `ASTRA_DB_KEYSPACE`. Токену нужны разрешения подключения к базе, создания таблицы и записи данных. Worker создаёт CQL-таблицу `events_by_resource_day` при первой выгрузке. Счётчики и состояние лайков хранятся в PostgreSQL.

```sh
docker compose up --build
```

Compose поднимает PostgreSQL, Redis, миграции, HTTP-сервер и worker. Учётные данные PostgreSQL в compose предназначены для локальной разработки. Внешние облачные сервисы настраиваются через `.env`.

## Структура

Пакеты предметных областей находятся в корне:

```text
users/          posts/          comments/       communities/
chats/          notifications/ todos/          notes/
artists/        albums/         tracks/         playlists/
photos/         videos/         movies/         memes/
links/          jobs/           queries/        websites/
images/         suggestions/    broadcasts/     magazines/
auth_cookie/    views/
```

В папках приложений находятся обработчики, сериализаторы, сервисы и `urls.go`. Например: `users/oauth.go`, `posts/uploads.go`, `chats/websocket.go`, `chats/calls.go`, `notes/blocks.go`. GORM-модели и связи расположены в `<приложение>/model/models.go`; `models.go` экспортирует типы пакета. Такое разделение исключает циклические импорты между моделями и обработчиками.

`app/` связывает сервисы приложений и регистрирует их маршруты. В `internal/httpapi` остаются общая HTTP-обвязка, контекст запроса, обработка ошибок, интерфейсы операций и транспорт WebSocket/Redis. `internal/models` содержит реестр миграций; `internal/database`, `internal/config`, `internal/storage`, `internal/tree`, `internal/worker` — инфраструктуру. Доменные обработчики в `internal/` не хранятся. `cmd/litloop-go-server`, `cmd/worker`, `cmd/migrate`, `cmd/admin`, `cmd/routes` — точки входа. Имя процесса API в Activity Monitor — `litloop-go-server`.

Подробная раскладка по файлам: [docs/STRUCTURE.md](docs/STRUCTURE.md).

## Основные возможности

- Регистрация, вход по email/username, access/refresh JWT, Google/VK OAuth, сброс пароля по SMTP, роли. Проверка существующих Django `pbkdf2_sha256` и bcrypt-паролей.
- Посты с медиа, лента, лайки, просмотры, показы, древовидные комментарии.
- Сообщества, членство, роли, заявки и модерация постов, изображения.
- Личные и групповые чаты, избранное, вложения, голосовые сообщения, WebSocket, уведомления, typing и WebRTC-сигналы. `REDIS_URL` включает обмен событиями между экземплярами API.
- Задачи, списки и сортировка; страницы заметок, вложенные страницы, блоки, таблицы, теги.
- Музыкальный каталог, фотоальбомы, видео, история воспроизведения, фильмы, IMDb CSV-импорт watchlist и рейтингов, TMDB, мемы.
- Ссылки, теги, CSV-импорт; вакансии, компании, резюме, отклики; поисковые адаптеры Google/Brave/Bing.
- Multipart-загрузка в S3, R2, GCS HMAC; resumable GCS через service account; локальная загрузка. Медиа получают подписанные URL.

## Примеры API

```sh
curl -X POST http://localhost:8000/users/signup/ \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","username":"alice","password":"change-me-123"}'
```

Ответ содержит `user`, `access`, `refresh` и совместимый алиас `token`. Защищённые запросы используют `Authorization: Bearer <access>`. Конечный `/` поддерживается.

```sh
curl -X POST http://localhost:8000/posts/create/ \
  -H 'Authorization: Bearer <access>' \
  -H 'Content-Type: application/json' \
  -d '{"title":"Hello","description":"Первый пост","photo_ids":[]}'
```

WebSocket: `/ws/chat/<id>/?token=<access>` и `/ws/notifications/?token=<access>`. Сообщение в чат: `{"type":"chat_message","text":"Привет","attachments":[]}`. Уведомления также поддерживают первый кадр `{"type":"auth","token":"..."}`.

Локальная загрузка: `POST /videos/create_presigned_url/` с `filename`, `content_type`; затем `PUT` файла на возвращённый `upload_url` с Bearer-токеном; затем `POST /videos/complete_upload/` с `upload_id`, `key`, `title`. Для S3/R2/GCS HMAC между initiate и complete получите URL частей через `get_presigned_url`, загрузите части и передайте массив `parts` с `PartNumber`, `ETag`. `upload_id` — непрозрачный идентификатор сессии сервера.

`POST /videos/<id>/encode/` ставит HLS-конвертацию и создание thumbnail в очередь. Готовый `hls_url` появляется в деталях видео. Worker переписывает локальный HLS-плейлист в подписанные URL сегментов при выдаче через API.

Транскриптор на `TRANSCRIPTION_URL` принимает POST с `voice_id`, `url`, `callback_url` и Bearer `TRANSCRIPTION_SECRET`. Он возвращает JSON `{"text":"...","language":"ru"}` или HTTP 202 и позже вызывает callback с `voice_id`, `status`, `transcription`, `language` и тем же секретом.

Создание администратора:

```sh
go run ./cmd/admin --email admin@example.com --username admin
```

Пароль запрашивается в терминале; для автоматизации используйте `ADMIN_PASSWORD`. `/admin/` предоставляет JSON API моделей и требует superuser. Таблицы пользователей и внутренние служебные записи изменяются через специальные API/CLI.

## Проверка и совместимость

```sh
go test ./...
go vet ./...
go build ./...
```

Тесты проверяют основные HTTP-сценарии, права доступа, WebSocket, CSV, пароли Django, сброс пароля, адаптеры внешних API и наличие всех 237 исходных объявлений URL. Наличие маршрута не означает побайтовую совместимость ответов. Подробности и ограничения: [docs/MIGRATION.md](docs/MIGRATION.md). Исходные файлы Django не изменялись.
