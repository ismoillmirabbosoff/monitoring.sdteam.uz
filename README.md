# monitoring-api

OnlinePBX asosidagi call-center monitoring dashboardi uchun **Go backend**.
`monitoring.sddev.uz` (Vue 3 SPA) frontendiga xizmat qiladi va eski
`phone.sdteam.uz/api` ni almashtiradi.

## Nima qiladi

1. **OnlinePBX auth** — `auth.json` orqali sessiya tokeni (`key_and_id`) va
   websocket kalitini (`auth_key`) generatsiya qiladi.
2. **Sync worker** — har `SYNC_INTERVAL`da OnlinePBX call-history'sini
   (`mongo_history/search.json`) tortib, Postgres'ga `uuid` bo'yicha upsert qiladi.
3. **API** — frontend o'qishlari bazadan (tez), jonli operatorlar OnlinePBX'dan.

## API (frontend kontrakti)

| Endpoint | Tavsif |
|---|---|
| `GET /api/monitoring/keys` | `{key_and_id, auth_key}` — REST tokeni + websocket kaliti |
| `GET /api/monitoring/data?gateway=&from=&to=` | qo'ng'iroqlar (unix soniya oraliq) |
| `GET /api/monitoring/bigData?date=YYYY-MM-DD` | bir kunlik barcha qo'ng'iroqlar |
| `GET /api/monitoring/operatorTime?date=YYYY-MM-DD` | operator (gateway) bo'yicha jamlanma |
| `GET /api/monitoring/fifo` | jonli navbatlar/operatorlar (OnlinePBX'dan) |
| `GET /health` | tiriklik tekshiruvi |

## Ishga tushirish

```bash
cp .env.example .env
# .env ichida ONPBX_DOMAIN, ONPBX_API_KEY, ONPBX_API_ID ni to'ldiring

# To'liq stack (Postgres + API):
docker compose up --build

# Yoki lokal: avval Postgres ko'taring, keyin
go run ./cmd/server
```

## Ikki ishlash rejimi

**1) Upstream-keys (hozir, tez start):** xom `api_key`/`api_id` bo'lmasa,
backend tokenni mavjud `ONPBX_KEYS_URL` (`phone.sdteam.uz/api/monitoring/keys`)
dan oladi. Token 30 daqiqa kesh qilinadi — upstream'ni zo'riqtirmaydi.
> Eslatma: bu endpoint rate-limitli; uni tez-tez chaqirmang (kesh shuni hal qiladi).

**2) Mustaqil auth (tavsiya etiladi):** `ONPBX_API_KEY` + `ONPBX_API_ID` berilsa,
backend o'zi `auth.json` qiladi va `phone.sdteam.uz` ga umuman bog'liq bo'lmaydi.
Bunda `ONPBX_KEYS_URL` ni o'chirib qo'ying.

## OnlinePBX kalitlarini olish

OnlinePBX kabineti → **Sozlamalar → API** bo'limidan `api_key` va `api_id` ni
oling. `ONPBX_DOMAIN` — sizning domeningiz (masalan `pbx12127.onpbx.ru`).

> **Eslatma:** websocket kaliti (`auth_key`) OnlinePBX versiyasiga qarab
> `auth.json` javobida bo'lmasligi mumkin. Bunday holda `ONPBX_WS_KEY` ni
> qo'lda kiriting (frontend WebSocket `wss://<domain>:3342/?key=<auth_key>` ga ulanadi).

## Frontend ulanishi

Vue ilovasidagi hardcode qilingan manzillarni almashtiring:
- `https://phone.sdteam.uz/api` → shu backend manzili
- `pbx12127.onpbx.ru` → sizning domeningiz (fifo URL va websocket)

Eng yaxshisi — bularni frontendda `.env` (Vite `VITE_API_BASE`, `VITE_ONPBX_DOMAIN`)
orqali konfiguratsiyalanadigan qilish.
