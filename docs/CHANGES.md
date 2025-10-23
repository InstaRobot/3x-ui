# Изменения в форке

Этот документ описывает все изменения, внесённые в данный форк относительно оригинального репозитория.

## Оглавление

- [Публичное API](#публичное-api)
- [Архитектура изменений](#архитектура-изменений)
- [Технические детали](#технические-детали)

---

## Публичное API

### Описание

Добавлено новое публичное API для программного доступа к панели без использования префикса `/panel`. API предназначено для интеграции с внешними системами мониторинга, автоматизации и биллинга.

### Основные возможности

- **Управление клиентами**: добавление, удаление, обновление
- **Просмотр инбаундов**: список и детальная информация
- **Статистика**: подсчёт пользователей и онлайн-подключений
- **Авторизация**: через API ключ в заголовках (без session cookies)

### Маршруты

Базовый путь: `{basePath}api/*`

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/ping` | Проверка доступности (без авторизации) |
| `GET` | `/api/list` | Список всех инбаундов |
| `GET` | `/api/get/:id` | Получить инбаунд по ID |
| `POST` | `/api/addClient` | Добавить клиента(ов) |
| `POST` | `/api/:id/delClient/:clientId` | Удалить клиента |
| `POST` | `/api/updateClient/:clientId` | Обновить клиента |
| `GET` | `/api/stats/users` | Общее количество пользователей |
| `GET` | `/api/stats/online` | Количество онлайн пользователей |

### Авторизация

API защищено ключом, который настраивается через переменную окружения:

```bash
export XUI_API_KEY="your-secret-key-here"
```

Ключ передаётся в одном из заголовков:
- `X-API-Key: <token>`
- `Authorization: Bearer <token>`

Если `XUI_API_KEY` не задан, авторизация через API ключ отключена.

### Примеры использования

Подробные примеры с curl находятся в [public-api.md](./public-api.md).

Быстрый пример:
```bash
# Получить список инбаундов
curl -H "X-API-Key: $XUI_API_KEY" http://localhost:2053/api/list

# Получить статистику пользователей
curl -H "X-API-Key: $XUI_API_KEY" http://localhost:2053/api/stats/users

# Проверить доступность (без ключа)
curl http://localhost:2053/api/ping
```

---

## Архитектура изменений

### Новые файлы

#### 1. `web/controller/public_api.go`

Новый контроллер, реализующий публичное API:

```go
type PublicAPIController struct {
    BaseController
    inboundController *InboundController
    inboundService    service.InboundService
}
```

**Особенности:**
- Переиспользует существующие хендлеры через проксирование
- Добавляет агрегирующие эндпоинты (`stats/users`, `stats/online`)
- Не дублирует бизнес-логику

**Логика подсчёта:**
- `stats/users`: подсчёт клиентов в `settings.clients` (VMESS/VLESS/Trojan/SS) и `settings.peers` (WireGuard)
- `stats/online`: пересечение онлайн e-mail из процесса xray с включёнными клиентами

#### 2. `docs/public-api.md`

Полная документация API на русском языке с примерами для всех протоколов.

### Изменённые файлы

#### 1. `web/middleware/api_key_auth.go`

**Что изменилось:**
- Middleware теперь работает **только** для нового API (`{basePath}api/*`)
- Старые маршруты `/panel/api/*` не затрагиваются
- Добавлено исключение для `/api/ping` (доступен без авторизации)

**Было:**
```go
if !strings.HasPrefix(path, basePath+"panel/api/") {
    c.Next()
    return
}
```

**Стало:**
```go
allowed := strings.HasPrefix(path, basePath+"api/")
if !allowed {
    c.Next()
    return
}

// Исключение для ping
if path == pingPath {
    c.Next()
    return
}
```

#### 2. `web/web.go`

**Что изменилось:**
- Удалён глобальный middleware `engine.Use(middleware.ApiKeyAuthMiddleware())`
- Добавлена регистрация нового контроллера после старого API

**Код:**
```go
// Регистрация публичного API без префикса '/panel'
trimmedBase := strings.TrimRight(basePath, "/")
controller.NewPublicAPIController(engine.Group(trimmedBase))
```

---

## Технические детали

### Безопасность

1. **Изоляция scope**: middleware применяется только к новым эндпоинтам
2. **Обратная совместимость**: старые маршруты работают как раньше
3. **Опциональность**: функция отключается отсутствием `XUI_API_KEY`
4. **Session reuse**: при валидном ключе создаётся временная сессия

### Переиспользование кода

Новый контроллер не дублирует логику:
- `getInbounds`, `getInbound` → прямое проксирование
- `addClient`, `delClient`, `updateClient` → проксирование с той же сигнатурой
- `InboundService.GetAllInbounds()` → для агрегации статистики

### Совместимость с basePath

Все маршруты учитывают `basePath` из настроек панели:
- Если `basePath = "/"` → `/api/list`
- Если `basePath = "/x-ui/"` → `/x-ui/api/list`

### Docker

Для использования в Docker добавьте переменную окружения:

```yaml
# docker-compose.yml
services:
  3x-ui:
    environment:
      - XUI_API_KEY=your-secret-key
```

или

```bash
docker run -e XUI_API_KEY=your-secret-key ...
```

---

## Отличия от оригинала

### Что НЕ изменилось

- ✅ Старые маршруты `/panel/api/*` работают без изменений
- ✅ Веб-интерфейс не затронут
- ✅ База данных и модели данных не изменены
- ✅ Логика работы с xray не изменена
- ✅ Существующая авторизация через сессии работает

### Что добавлено

- ✨ Новый набор эндпоинтов `{basePath}api/*`
- ✨ Авторизация через API ключ (опционально)
- ✨ Агрегирующие эндпоинты статистики
- ✨ Документация на русском

### Преимущества подхода

1. **Безопасность**: независимая авторизация для API
2. **Гибкость**: можно использовать оба API одновременно
3. **Простота**: не требует изменений в существующем коде
4. **Расширяемость**: легко добавить новые эндпоинты в `PublicAPIController`

---

## Использование

### Настройка

1. Задайте API ключ:
```bash
export XUI_API_KEY="my-secret-key-123"
```

2. Перезапустите панель:
```bash
systemctl restart x-ui
# или
docker-compose restart
```

3. Проверьте работу:
```bash
curl http://your-server:2053/api/ping
# Должно вернуть: ok
```

### Интеграция с мониторингом

Пример скрипта для Prometheus/Grafana:

```bash
#!/bin/bash
API_KEY="your-key"
HOST="localhost:2053"

# Получить метрики
users=$(curl -s -H "X-API-Key: $API_KEY" http://$HOST/api/stats/users | jq '.obj.count')
online=$(curl -s -H "X-API-Key: $API_KEY" http://$HOST/api/stats/online | jq '.obj.count')

echo "xui_total_users $users"
echo "xui_online_users $online"
```

### Биллинг системы

```python
import requests

class XUIClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.headers = {"X-API-Key": api_key}
    
    def add_user(self, inbound_id, email, total_gb=0, expiry_time=0):
        data = {
            "id": inbound_id,
            "settings": json.dumps({
                "clients": [{
                    "id": str(uuid.uuid4()),
                    "email": email,
                    "totalGB": total_gb,
                    "expiryTime": expiry_time,
                    "enable": True
                }]
            })
        }
        r = requests.post(
            f"{self.base_url}/api/addClient",
            json=data,
            headers=self.headers
        )
        return r.json()
    
    def get_stats(self):
        users = requests.get(
            f"{self.base_url}/api/stats/users",
            headers=self.headers
        ).json()
        online = requests.get(
            f"{self.base_url}/api/stats/online",
            headers=self.headers
        ).json()
        return {
            "total": users["obj"]["count"],
            "online": online["obj"]["count"]
        }
```

---

## Дальнейшее развитие

### Возможные улучшения

- [ ] Поддержка управления WireGuard peers через API
- [ ] Эндпоинт для массового обновления клиентов
- [ ] Детальная статистика трафика по клиентам
- [ ] Webhooks для событий (новый клиент, превышение лимита и т.д.)
- [ ] Rate limiting для API запросов
- [ ] Логирование API запросов

### Обратная связь

Если у вас есть предложения или вы нашли проблемы, создайте issue в репозитории.

---

## Changelog

### v1.0 (текущая версия)

- ✨ Добавлено публичное API `{basePath}api/*`
- ✨ Авторизация через `XUI_API_KEY`
- ✨ Эндпоинты управления клиентами
- ✨ Эндпоинты статистики (`stats/users`, `stats/online`)
- 📝 Документация на русском языке
- 🔧 Изменён scope middleware API key auth
- 🔧 Регистрация нового контроллера в роутере

---

## Лицензия

Все изменения распространяются под той же лицензией, что и оригинальный проект.

