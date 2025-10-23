# Публичное API (без префикса /panel)

Этот документ описывает дополнительное публичное API, доступное по пути `{basePath}api/*`.

## Авторизация

- Заголовок: `X-API-Key: <token>` или `Authorization: Bearer <token>`
- Публичное API активно, если в окружении задана переменная `XUI_API_KEY`.
- Старые маршруты `/panel/api/*` ключ не затрагивает — их логика не менялась.
- `GET {basePath}api/ping` доступен без ключа.

## Базовый URL

- Базовый путь: `{basePath}api/` (например, `/api/`, если `basePath=/`).

## Эндпоинты

1) Проверка доступности
- `GET {basePath}api/ping` → 200 OK, текст `ok` (без авторизации)

2) Inbounds (проксирование в существующие хендлеры)
- `GET {basePath}api/list` — список инбаундов пользователя
- `GET {basePath}api/get/:id` — детали инбаунда
- `POST {basePath}api/addClient` — добавить клиента(ов)
  - Тело (JSON): `{ "id": <inboundId>, "settings": "{\"clients\":[ ... ]}" }`
- `POST {basePath}api/:id/delClient/:clientId` — удалить клиента
- `POST {basePath}api/updateClient/:clientId` — обновить одного клиента
  - Тело (JSON): `{ "id": <inboundId>, "settings": "{\"clients\":[ oneClientJson ]}" }`

3) Статистика (агрегации)
- `GET {basePath}api/stats/users` → `{ "count": <int> }`
  - Подсчёт как в веб‑панели (вкладка инбаундов):
    - VMESS/VLESS/TROJAN/SHADOWSOCKS — количество записей в `settings.clients`
    - WIREGUARD — количество записей в `settings.peers`
- `GET {basePath}api/stats/online` → `{ "count": <int> }`
  - Используются текущие «онлайн» e‑mail из работающего процесса;
  - Пересечение с включёнными клиентами включённых инбаундов для VMESS/VLESS/TROJAN/SHADOWSOCKS;
  - Для WireGuard сопоставления с e‑mail нет, поэтому сейчас в этот счётчик не попадает.

## Формат ответа

Большинство эндпоинтов проксируют существующие и возвращают:
```
{ "success": boolean, "msg": string, "obj": any }
```

## Примеры

Замените плейсхолдеры: `HOST`, `PORT`, `API_KEY`, `ID`, `CLIENT_ID`.

Список инбаундов
```
curl -s -H "X-API-Key: $API_KEY" \
  http://$HOST:$PORT/api/list
```

Инбаунд по id
```
curl -s -H "Authorization: Bearer $API_KEY" \
  http://$HOST:$PORT/api/get/$ID
```

Добавление клиента (VMESS/VLESS — обязательны: id, email)
```
DATA='{
  "id":'$ID',
  "settings":"{\"clients\":[{
    \"id\":\"<uuid-v4>\",              \"security\":\"auto\",      \"password\":\"\",
    \"flow\":\"\",                      \"email\":\"user1\",         \"limitIp\":0,
    \"totalGB\":0,                         \"expiryTime\":0,               \"enable\":true,
    \"tgId\":0,                            \"subId\":\"<subIdA>\",       \"comment\":\"\",
    \"reset\":0
  },{
    \"id\":\"<uuid-v4>\",              \"security\":\"auto\",      \"password\":\"\",
    \"flow\":\"\",                      \"email\":\"user2\",         \"limitIp\":2,
    \"totalGB\":1073741824,                \"expiryTime\":1735689600000,    \"enable\":true,
    \"tgId\":123456789,                    \"subId\":\"<subIdB>\",       \"comment\":\"VIP\",
    \"reset\":30
  }]}"
}'
curl -s -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" \
  -d "$DATA" http://$HOST:$PORT/api/addClient
```

Добавление клиента (TROJAN — обязателен password; email желателен для статистики)
```
DATA='{
  "id":'$ID',
  "settings":"{\"clients\":[{
    \"password\":\"<strong-pass>\",     \"email\":\"user-trojan\",   \"enable\":true,
    \"limitIp\":0,                         \"totalGB\":0,                   \"expiryTime\":0,
    \"tgId\":0,                            \"subId\":\"<subTrojan>\",     \"comment\":\"\",
    \"reset\":0
  }]}"
}'
curl -s -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" \
  -d "$DATA" http://$HOST:$PORT/api/addClient
```

Добавление клиента (SHADOWSOCKS — обязателен email; метод шифрования берётся из настроек инбаунда)
```
DATA='{
  "id":'$ID',
  "settings":"{\"clients\":[{
    \"email\":\"user-ss\",              \"enable\":true,                \"limitIp\":0,
    \"totalGB\":524288000,                 \"expiryTime\":0,               \"tgId\":0,
    \"subId\":\"<subSS>\",               \"comment\":\"Mobile\",        \"reset\":0
  }]}"
}'
curl -s -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" \
  -d "$DATA" http://$HOST:$PORT/api/addClient
```

Удаление клиента
```
# clientId зависит от протокола:
# - trojan: clientId = password
# - shadowsocks: clientId = email
# - vmess/vless (и др.): clientId = id (UUID)
curl -s -H "X-API-Key: $API_KEY" \
  -X POST http://$HOST:$PORT/api/$ID/delClient/$CLIENT_ID
```

Обновление клиента (в `settings` — ровно один клиент) со всеми параметрами
```
UPD='{
  "id":'$ID',
  "settings":"{\"clients\":[{
    \"id\":\"$CLIENT_ID\",              \"security\":\"auto\",      \"password\":\"\",
    \"flow\":\"\",                      \"email\":\"user-upd\",      \"limitIp\":3,
    \"totalGB\":2147483648,                \"expiryTime\":1735689600000,    \"enable\":false,
    \"tgId\":0,                            \"subId\":\"<newSub>\",        \"comment\":\"paused\",
    \"reset\":0
  }]}"
}'
curl -s -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" \
  -d "$UPD" http://$HOST:$PORT/api/updateClient/$CLIENT_ID
```

Статистика
```
curl -s -H "X-API-Key: $API_KEY" http://$HOST:$PORT/api/stats/users
curl -s -H "X-API-Key: $API_KEY" http://$HOST:$PORT/api/stats/online
```

WireGuard (подсчёт пользователей)
- В `stats/users` учитываются peer'ы из `settings.peers`.
- Управление peer'ами через эти эндпоинты не поддержано; используйте обновление всего инбаунда (`/panel/api/inbounds/update/:id`) с изменением `settings.peers`.

## Примечания

- Чтобы отключить авторизацию публичного API, не задавайте `XUI_API_KEY`.
- Учитывается `basePath` из настроек (в примерах показан случай с `basePath=/`).


