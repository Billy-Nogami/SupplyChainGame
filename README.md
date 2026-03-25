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

## Следующие шаги

- real-time обновления комнаты через `WebSocket` или `SSE`;
- перенос активных сессий и ходов в `Redis`;
- архивирование завершенных игр в `PostgreSQL`;
- экспорт истории сессии в `Excel`;
- simulation mode и Monte Carlo.
