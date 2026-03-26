# Supply Chain Simulator

Веб-сервис на Go для моделирования цепи поставок в формате многопользовательской игры.

## Архитектурный подход

Проект строится по принципам Clean Architecture:

- `internal/domain` - бизнес-сущности и инварианты
- `internal/usecase` - сценарии использования и порты
- `internal/interface/http` - HTTP-транспорт и DTO
- `internal/infrastructure` - реализации хранилищ, генераторов ID, времени и экспорта
- `cmd/api` - точка входа приложения

## Ближайший MVP

Сейчас в проекте уже реализовано:

- создание комнат;
- подключение игроков;
- ручное назначение ролей;
- запуск игровой сессии;
- прием заказов по игрокам;
- переход на следующую неделю;
- хранение истории недель;
- расчет базовой аналитики по сессии.
- экспорт сессии в `Excel`;
- real-time обновления комнаты через `SSE`;
- `Redis`-адаптеры для комнат, сессий, ходов и room events.

## Текущее API

- `POST /rooms`
- `GET /rooms/{roomId}`
- `POST /rooms/{roomId}/players`
- `POST /rooms/{roomId}/roles`
- `POST /rooms/{roomId}/start`
- `POST /rooms/{roomId}/orders`
- `POST /rooms/{roomId}/next`
- `GET /rooms/{roomId}/session`
- `GET /rooms/{roomId}/weeks`
- `GET /rooms/{roomId}/analytics`
- `GET /rooms/{roomId}/decisions`
- `GET /rooms/{roomId}/export`
- `GET /rooms/{roomId}/events` (`SSE`)

## Запуск

Локально:

```bash
go run ./cmd/api
```

Через Docker:

```bash
docker compose up --build
```

Сервис слушает порт `8080` по умолчанию. Его можно переопределить через переменную окружения `PORT`.
Если задан `REDIS_ADDR`, backend использует `Redis` для активных комнат, сессий, ходов и room events.
Если `REDIS_ADDR` не задан, backend работает на in-memory хранилищах.

Для frontend-деплоя можно задать `ALLOWED_ORIGINS`, например:

```bash
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

Также в проекте есть:

- [`.env.example`](/Users/george/ Учёба/Мат Модели/StocksAndRequests/.env.example)
- [`Makefile`](/Users/george/ Учёба/Мат Модели/StocksAndRequests/Makefile)

## Следующие шаги

- frontend для комнат и дашборда;
- архивирование завершенных игр в `PostgreSQL`;
- simulation mode и Monte Carlo.
