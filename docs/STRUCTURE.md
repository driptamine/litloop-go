# Структура Go-проекта

Каждое приложение располагается отдельным пакетом в корне. Здесь находятся его HTTP-обработчики, бизнес-логика, сериализация и маршруты.

```text
users/
  auth.go              JWT и аутентификация
  handlers.go          регистрация, вход, профиль, роли
  oauth.go             Google/VK
  password_reset.go    сброс пароля
  media.go             медиа пользователя
  security.go          совместимость паролей Django
  urls.go              маршруты пользователей
  module.go            зависимости обработчика
  models.go            экспорт моделей
  model/models.go      GORM-модель пользователя
posts/
  handlers.go          создание, изменение, удаление, лента, лайки
  service.go           связывание медиа с постом
  uploads.go           multipart-вложения
  serializers.go
  urls.go
  module.go
  models.go
  model/models.go
chats/
  handlers.go          личные/групповые чаты и отправка сообщений
  service.go           сохранение сообщений и проверка участников
  serializers.go
  websocket.go
  calls.go
  ai.go
  transcription.go
  urls.go
  module.go
  models.go
  model/models.go
notes/
  handlers.go
  service.go
  blocks.go
  tags.go
  serializers.go
  urls.go
  module.go
  models.go
  model/models.go
app/
  app.go               сборка зависимостей приложений
  urls.go              health и OpenAPI
  *_test.go            интеграционные сценарии
internal/
  httpapi/             общая HTTP-обвязка, контекст, ошибки, транспорт событий
  database/            подключение и миграции
  models/              реестр моделей и служебные записи
  config/              конфигурация
  storage/             S3/R2/GCS и подпись URL
  worker/              очередь задач и выполнение
  tree/                поддержка MPTT-индексов
  types/               общий SQL/JSON-тип
  testutil/            изоляция тестовых баз
```

Аналогично организованы `communities`, `comments`, `todos`, `movies`, `tracks`, `photos`, `playlists`, `jobs`, `links`, `websites`, `queries`, `notifications` и остальные приложения. Набор файлов соответствует реализованным функциям: пустые файлы для симметрии не создаются. Например, у `users` сериализация профиля выполняется обработчиком, отдельного `serializers.go` в этом пакете нет.

Модели выделены в подпа́кеты `model`, чтобы SQL-связи между приложениями не создавали циклические зависимости обработчиков. `models.go` каждого приложения предоставляет алиасы этих же типов. В `internal/models/registry.go` собран общий реестр для миграций. Там нет второй реализации доменных моделей.

`internal/httpapi/operations.go` описывает функции, которыми обмениваются приложения. Их реализации находятся в соответствующих папках приложений и связываются в `app.New`. В `internal/httpapi` нет обработчиков постов, пользователей, чатов, заметок и других предметных областей.
