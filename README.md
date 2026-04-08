# АттестатПро — Система управления оценками

Полноценный fullstack проект: Go-бэкенд + SQLite + HTML-фронтенд.

---

## Структура проекта

```
gradebook/
├── cmd/server/
│   ├── main.go              # Точка входа, HTTP-сервер, роутинг
│   └── frontend/
│       ├── login.html       # Страница входа
│       ├── register.html    # Страница регистрации
│       └── app.html         # Главное приложение (учитель / ученик / родитель)
├── internal/
│   ├── auth/jwt.go          # JWT авторизация, middleware
│   ├── db/db.go             # SQLite, все запросы к БД
│   ├── handlers/handlers.go # HTTP-обработчики (API Gateway)
│   └── models/models.go     # Структуры данных
├── go.mod / go.sum
├── Makefile
└── README.md
```

---

## Быстрый старт

### 1. Установите зависимости (нужен Go 1.22+ и gcc для SQLite)

```bash
# Ubuntu/Debian
sudo apt install golang gcc

# macOS
brew install go
```

### 2. Скачайте Go-модули

```bash
go mod download
```

### 3. Запустите сервер

```bash
# Вариант 1: через Make
make dev

# Вариант 2: напрямую
cd cmd/server && go run .

# Вариант 3: сборка + запуск
make build && ./bin/gradebook
```

### 4. Откройте браузер

```
http://localhost:8080
```

---

## Аккаунты

| Роль    | Логин | Пароль    | Права |
|---------|-------|-----------|-------|
| Учитель | Лада  | БагиЛада  | Полный доступ, выставление оценок |
| Ученик  | —     | (регистрация) | Просмотр своих оценок |
| Родитель| —     | (регистрация) | Просмотр оценок ребёнка |

> Учитель создаётся автоматически при первом запуске.  
> Ученики и родители регистрируются сами через `/register.html`.

---

## API Эндпоинты

### Публичные (без токена)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/auth/login` | Вход в систему |
| POST | `/api/auth/register` | Регистрация (ученик / родитель) |
| GET  | `/api/schedule` | Расписание занятий |

### Защищённые (Bearer JWT)

| Метод | Путь | Роль | Описание |
|-------|------|------|----------|
| GET | `/api/me` | все | Текущий пользователь |
| GET | `/api/students` | teacher | Список учеников |
| GET | `/api/grades?date=YYYY-MM-DD` | teacher | Оценки за дату |
| GET | `/api/grades/all` | teacher | Все оценки |
| POST | `/api/grades` | teacher | Выставить/обновить оценку |
| DELETE | `/api/grades/{id}` | teacher | Удалить оценку |
| GET | `/api/my-grades` | student | Мои оценки |
| GET | `/api/children` | parent | Список детей |
| GET | `/api/child-grades/{studentId}` | parent | Оценки ребёнка |
| POST | `/api/link-child` | parent | Привязать ребёнка по логину |

---

## База данных (SQLite)

Файл: `gradebook.db` (создаётся автоматически)

### Таблицы

**users** — все пользователи  
```sql
id, login (UNIQUE), password (bcrypt), name, role, class, subject, created_at
```

**grades** — оценки  
```sql
id, student_id, teacher_id, date (YYYY-MM-DD), value (2-5), comment, subject, created_at, updated_at
UNIQUE(student_id, date, subject)  -- нельзя поставить 2 оценки за одну дату
```

**parent_children** — связь родитель ↔ ученик  
```sql
parent_id, student_id  (PRIMARY KEY составной)
```

---

## Расписание

Занятия проводятся: **Вторник / Четверг / Суббота**  
Начало: **7 апреля 2025**  
Генерируется автоматически на 20 занятий вперёд.

---

## Настройки через переменные окружения

| Переменная | По умолчанию | Описание |
|------------|-------------|----------|
| `PORT` | `8080` | Порт сервера |
| `DB_PATH` | `./gradebook.db` | Путь к БД |

Пример:
```bash
PORT=3000 DB_PATH=/data/grades.db ./bin/gradebook
```

---

## Безопасность

- Пароли хешируются через **bcrypt** (cost=10)
- JWT-токены живут **7 дней**
- Роли проверяются на каждом защищённом роуте
- CORS настроен (можно ограничить в `main.go`)
- Регистрация учителей **запрещена** через API

---

## Деплой (production)

```bash
# Сборка статического бинаря с CGO (нужен для SQLite)
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o gradebook ./cmd/server

# Запуск
PORT=80 DB_PATH=/var/data/gradebook.db ./gradebook
```

Для Docker добавьте в Dockerfile:
```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=1 go build -o gradebook ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /app/gradebook /gradebook
CMD ["/gradebook"]
```
