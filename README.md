# Сервисы региона — мини-приложение MAX

Каталог официальных каналов, чатов, ботов и сайтов РСО-Алания. Открывается внутри
MAX на iOS, Android, десктопе и в веб-версии. Это мини-приложение к чат-боту региона.

- **Бэкенд:** Go, один статический бинарник, фронтенд встроен в него.
- **База:** SQLite (файл `/data/bmax.db`). В ней каталог, аватарки и статистика запусков.
- **Фронтенд:** HTML/CSS/JS без сборки, работает через [MAX Bridge](https://dev.max.ru/docs/webapps/bridge) (`max-web-app.js`).
- **Деплой:** Docker Compose. Caddy сам получает и продлевает HTTPS-сертификат Let's Encrypt.

## Структура

```
cmd/bmax/            точка входа и команды (serve, import, export, backup, healthcheck)
internal/config/     настройки из переменных окружения
internal/maxauth/    проверка подписи initData от MAX
internal/catalog/    формат каталога, валидация, публичный снимок
internal/store/      SQLite: миграции, запросы, импорт/экспорт
internal/server/     HTTP: публичное API, админ-API, раздача фронтенда
web/                 мини-приложение (index.html, css, js)
seed/                исходный каталог (catalog.json + media/), импортируется в пустую базу
deploy/Caddyfile     HTTPS reverse proxy
```

## Запуск на сервере

1. Укажите в DNS A-запись домена на IP сервера.
2. Установите Docker с плагином Compose. Откройте порты 80 и 443.
3. Настройте окружение:
   ```sh
   cp .env.example .env
   # заполните DOMAIN, ACME_EMAIL, BOT_TOKEN, при необходимости ADMIN_TOKEN
   ```
4. Запустите:
   ```sh
   docker compose up -d --build
   ```
   При первом старте база создаётся и заполняется из `seed/`.
5. В настройках бота на платформе MAX для партнёров укажите адрес мини-приложения
   `https://<DOMAIN>/`.

Обновление после изменений в репозитории: `git pull && docker compose up -d --build`.
Данные лежат в томе `app-data`, обновление их не трогает.

## Как это работает

1. MAX открывает `https://<DOMAIN>/` и передаёт в `window.WebApp.initData` строку, подписанную токеном бота.
2. Приложение запрашивает `GET /api/v1/catalog` с заголовком `X-Max-Init-Data`.
   Сервер проверяет подпись (HMAC-SHA256) и срок `auth_date`, затем отдаёт каталог из памяти (с ETag).
3. `POST /api/v1/session` записывает пользователя и число его запусков в таблицу `app_users`.
4. Ссылки на `max.ru` открываются внутри MAX через `WebApp.openMaxLink`, остальные — в браузере через `WebApp.openLink`.

Deep-link параметры (`https://max.ru/<бот>?startapp=...`):

- `category_<id>` — открыть категорию, например `category_health`;
- `org_<id>` — открыть категорию организации и подсветить её.

Плашки-ссылки (`link`) по deep-link не открываются: MAX разрешает открыть ссылку только по нажатию пользователя.

## Схема базы

| Таблица | Назначение |
|---|---|
| `categories` | Плашки: id, название, иконка, `wide`, `link` (сразу открыть организацию), подзаголовки, порядок |
| `organizations` | Каналы, чаты, боты, сайты: id, название, `url` (уникален), иконка, аватар, подзаголовок |
| `organization_categories` | Связь «организация — категория», первая категория идёт подзаголовком |
| `featured` | Верхняя лента «Официальные каналы» и её порядок |
| `media` | Картинки (аватарки), отдаются по `/media/<имя>` |
| `app_users` | Пользователи MAX, открывавшие приложение, и число запусков |

Миграции лежат в `internal/store/migrations` и применяются автоматически при старте.

## Как менять каталог

### Вариант 1: правка файла (проще всего)

```sh
docker compose exec app /app/bmax export /data/export   # выгрузить текущий каталог
docker compose cp app:/data/export ./export
# отредактируйте export/catalog.json, положите картинки в export/media/
docker compose cp ./export app:/data/export
docker compose exec app /app/bmax import /data/export   # заменить каталог целиком
docker compose restart app                              # перечитать каталог
```

Формат `catalog.json`:

```json
{
  "categories": [
    { "id": "doctor", "title": "Запись к врачу", "icon": "🩺", "wide": false,
      "link": "bot_minzdrav", "groups": [] },
    { "id": "government", "title": "Органы власти РСО-Алания", "icon": "🏛️", "wide": false,
      "link": "", "groups": ["Министерства", "Комитеты", "Службы и управления"] }
  ],
  "organizations": [
    {
      "id": "minzdrav",
      "name": "Министерство здравоохранения РСО-Алания",
      "url": "https://max.ru/minzdravrso",
      "icon": "🏥",
      "avatar": "minzdrav.jpg",
      "group": "Министерства",
      "categories": ["government"]
    }
  ],
  "featured": ["governor", "gov", "parlament", "mfc"]
}
```

Правила:

- `id` — латиница в нижнем регистре, цифры, `_` или `-`.
- `url` — только `https://`, у каждой организации свой.
- `wide`: `true` — плашка на всю ширину, `false` — половинка. Половинки встают парами слева направо.
- `link` — id организации. Плашка открывает её сразу, без списка (топливо, МФЦ, запись к врачу, Водоканал).
- `groups` — подзаголовки внутри категории по порядку. Организация выбирает подзаголовок полем `group`.
- `avatar` — имя файла из `media/` (`.jpg` или `.png`) или пустая строка.
- Порядок категорий и ленты `featured` задаётся порядком в массиве.

### Аватарки

Положите картинку в `media/` с именем, равным `id` организации, например `media/minzdrav.jpg`.
При импорте она сама подставится организации без аватарки. Картинку можно брать любого размера
(до 10 МБ): она обрезается до квадрата по центру и уменьшается до 256×256.
Если аватарки нет, показывается эмодзи из поля `icon`.
- Организации внутри категорий сортируются по алфавиту автоматически.

### Вариант 2: админ-API

Включается, если задан `ADMIN_TOKEN`. Изменения сразу видны в приложении, перезапуск не нужен.

```sh
T="Authorization: Bearer $ADMIN_TOKEN"; H=https://<DOMAIN>

curl -H "$T" $H/api/admin/stats                       # пользователи, запуски, активность
curl -H "$T" $H/api/admin/catalog > catalog.json      # выгрузить каталог
curl -H "$T" -X PUT --data-binary @catalog.json $H/api/admin/catalog   # заменить целиком

# добавить или изменить организацию
curl -H "$T" -X PUT $H/api/admin/organizations/eirc_fuel -d '{
  "name": "Топливные карты ЕИРЦ", "url": "https://eirc15.ru/", "icon": "⛽",
  "avatar": "", "categories": ["fuel"] }'
curl -H "$T" -X DELETE $H/api/admin/organizations/eirc_fuel

# категории
curl -H "$T" -X PUT $H/api/admin/categories/transport -d '{"title":"Транспорт","icon":"🚌","wide":false}'
curl -H "$T" -X PUT $H/api/admin/categories -d '["government","law","transport", "..."]'   # порядок, нужно перечислить все
curl -H "$T" -X DELETE $H/api/admin/categories/transport

# лента «Официальные каналы»
curl -H "$T" -X PUT $H/api/admin/featured -d '["governor","gov","comdigital","mchs","minzdrav","mfc"]'

# картинки (обрезаются до квадрата и уменьшаются до 256×256)
curl -H "$T" -X PUT --data-binary @logo.jpg $H/api/admin/media/minzdrav.jpg
curl -H "$T" -X DELETE $H/api/admin/media/logo.jpg
```

Ошибки возвращаются как `{"error": "..."}` с кодом 400, 401 или 404.

## Резервные копии

```sh
docker compose exec app /app/bmax backup /data/backup-$(date +%F).db
docker compose cp app:/data/backup-$(date +%F).db .
```

## Разработка

```sh
go test ./...
BOT_TOKEN=<токен> SEED_DIR=seed go run ./cmd/bmax    # http://localhost:8080
```

Вне MAX приложение покажет просьбу открыть его в мессенджере: без подписанного initData каталог не отдаётся.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `BOT_TOKEN` | — (обязательна) | Токен бота MAX, к которому привязано мини-приложение |
| `ADMIN_TOKEN` | пусто (API выключено) | Токен админ-API, не короче 32 символов |
| `ADDR` | `:8080` | Адрес HTTP-сервера |
| `DATABASE_PATH` | `data/bmax.db` (в Docker `/data/bmax.db`) | Файл SQLite |
| `SEED_DIR` | пусто (в Docker `/app/seed`) | Каталог для первичного заполнения пустой базы |
| `INIT_DATA_TTL` | `1h` | Максимальный возраст initData |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` или `error` |
| `DOMAIN`, `ACME_EMAIL` | — | Только для Caddy в `docker-compose.yml` |
