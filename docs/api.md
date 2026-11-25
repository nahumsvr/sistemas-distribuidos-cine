# API

URL base: `http://localhost:8080/api` · Formato: JSON · Fechas: ISO 8601 en UTC.

## Autenticación

Las rutas protegidas exigen el header:

```
Authorization: Bearer <token>
```

El token se obtiene con `/auth/login` o `/auth/register` y vence en 1 hora. Ver ADR-001 en [arquitectura.md](arquitectura.md).

## Formato de error

Todos los errores tienen la misma forma:

```json
{ "error": { "code": "SEAT_TAKEN", "message": "One or more seats are already reserved" } }
```

| HTTP | `code` | Cuándo |
|---|---|---|
| 400 | `VALIDATION_ERROR` | JSON inválido, campos faltantes o fuera de rango |
| 401 | `UNAUTHORIZED` | Falta el token, o es inválido o expiró; credenciales incorrectas |
| 403 | `FORBIDDEN` | El usuario no tiene el rol necesario (admin) |
| 404 | `NOT_FOUND` | El recurso no existe o pertenece a otro usuario |
| 409 | `SEAT_TAKEN` | Algún asiento ya está reservado para esa función |
| 409 | `EMAIL_TAKEN` | El email ya está registrado |
| 413 | `PAYLOAD_TOO_LARGE` | El cuerpo supera 1 MB |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | Un `POST`/`PUT` con cuerpo no trae `Content-Type: application/json` |
| 422 | `SHOWTIME_STARTED` | Se intenta reservar, modificar o cancelar una función que ya empezó |
| 429 | `RATE_LIMITED` | Demasiados intentos de login o registro |
| 500 | `INTERNAL` | Error inesperado (el detalle solo se registra en el log) |
| 503 | `UNAVAILABLE` | La base de datos no responde (solo `/healthz`) |

## Endpoints

### Salud

| Método | Ruta | Auth | Respuesta |
|---|---|---|---|
| GET | `/healthz` | — | `200 {"status":"ok"}` si la DB responde; `503 UNAVAILABLE` si no |

Las rutas inexistentes responden `404 NOT_FOUND` con el mismo formato JSON de error. Todas las respuestas incluyen los headers `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` y `Content-Security-Policy`.

### Auth

| Método | Ruta | Auth | Body | Respuesta |
|---|---|---|---|---|
| POST | `/auth/register` | — | `{ "name", "email", "password" }` | `201 { "token", "user" }` |
| POST | `/auth/login` | — | `{ "email", "password" }` | `200 { "token", "user" }` |
| GET | `/auth/me` | user | — | `200 { "user" }` |

Validaciones:
- `email` válido; se guarda en minúsculas y sin espacios.
- `password` de 8 a 72 **bytes** (límite de bcrypt; un emoji ocupa 4 bytes).
- `name` de 1 a 100 caracteres.
- Se rechazan los campos desconocidos: por ejemplo, enviar `"role"` responde `400`. El rol siempre es `user`.

Seguridad:
- **Login:** un email inexistente y una contraseña incorrecta dan la misma respuesta (`401`) y tardan lo mismo, así que no se puede averiguar qué emails están registrados.
- **Rate limit por IP:** login y registro comparten un límite de unos 5 intentos por minuto (burst de 5, recarga de 1 cada 12 s). Al superarlo se responde `429` con el header `Retry-After` en segundos.
- **Header de autenticación:** el esquema `Bearer` no distingue mayúsculas. Sin header, con otro esquema o con un token inválido o expirado se responde `401` con `WWW-Authenticate: Bearer`.

```json
// 200 /auth/login
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": { "id": 2, "name": "Ana", "email": "ana@mail.com", "role": "user", "createdAt": "2026-11-24T17:23:06Z" }
}
```

### Catálogo (público)

| Método | Ruta | Respuesta |
|---|---|---|
| GET | `/movies` | `200 [movie]` |
| GET | `/movies/{id}` | `200 movie` |
| GET | `/movies/{id}/showtimes` | `200 [showtime]` (solo funciones futuras) |
| GET | `/showtimes/{id}/seats` | `200 { showtime, seats: [seat] }` |

Reglas:

- Son rutas públicas (sin token) y de solo lectura.
- Un `{id}` que no sea un entero positivo responde `400 VALIDATION_ERROR`; si no existe, `404 NOT_FOUND`.
- Las listas vacías son `[]`, nunca `null`. Las películas se ordenan por título y las funciones por hora de inicio.
- `/showtimes/{id}/seats` devuelve todos los asientos de la sala en orden fila/número. `available` es `false` si hay una reserva activa de ese asiento en esa función; al cancelar, vuelve a `true`.

```json
// 200 /movies/1
{ "id": 1, "title": "Dune", "synopsis": "...", "durationMin": 155, "rating": "PG-13", "posterUrl": "" }
```

```json
// 200 /showtimes/7/seats
{
  "showtime": { "id": 7, "movieId": 1, "auditorium": "Sala 1", "startsAt": "2026-12-05T20:00:00Z", "priceCents": 8000 },
  "seats": [
    { "id": 1, "row": "A", "number": 1, "available": true },
    { "id": 2, "row": "A", "number": 2, "available": false }
  ]
}
```

### Reservas (usuario autenticado)

| Método | Ruta | Body | Respuesta |
|---|---|---|---|
| GET | `/reservations` | — | `200 [reservation]` (solo las del usuario) |
| GET | `/reservations/{id}` | — | `200 reservation` |
| POST | `/reservations` | `{ "showtimeId": 7, "seatIds": [1, 3] }` | `201 reservation` |
| PUT | `/reservations/{id}` | `{ "seatIds": [4, 5] }` | `200 reservation` |
| DELETE | `/reservations/{id}` | — | `204` (cancela; el registro queda con `status: "cancelled"`) |

Reglas de negocio:

- De 1 a 10 asientos por reserva, sin repetir, y todos de la sala de esa función.
- Solo se puede crear, modificar o cancelar antes de `startsAt` (si no, `422 SHOWTIME_STARTED`).
- Al modificar, los asientos se reemplazan en una sola transacción: o se aplica todo o nada. Una reserva cancelada no se puede modificar (`400 VALIDATION_ERROR`).
- Al cancelar, se liberan los asientos (`seats: []`, `totalCents: 0`); repetir `DELETE` antes de `startsAt` es idempotente (`204`).
- Una reserva de otro usuario responde `404`, para no revelar que existe.
- `GET /reservations` devuelve las reservas del usuario ordenadas de la más reciente a la más antigua (lista vacía `[]`, nunca `null`).

```json
// 201 POST /reservations
{
  "id": 15,
  "status": "active",
  "showtime": { "id": 7, "movieTitle": "Dune", "startsAt": "2026-12-05T20:00:00Z" },
  "seats": [ { "id": 1, "row": "A", "number": 1 }, { "id": 3, "row": "A", "number": 3 } ],
  "totalCents": 16000,
  "createdAt": "2026-11-24T18:00:00Z"
}
```

### Administración (rol `admin`)

| Método | Ruta | Body |
|---|---|---|
| POST | `/admin/movies` | `{ "title", "synopsis", "durationMin", "rating", "posterUrl" }` |
| PUT | `/admin/movies/{id}` | igual que POST |
| DELETE | `/admin/movies/{id}` | — (`409` si tiene funciones con reservas activas) |
| POST | `/admin/showtimes` | `{ "movieId", "auditoriumId", "startsAt", "priceCents" }` |
| PUT | `/admin/showtimes/{id}` | igual que POST |
| DELETE | `/admin/showtimes/{id}` | — (`409` si tiene reservas activas) |

## Ejemplo con curl

```bash
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@cine.local","password":"User123!"}' | jq -r .token)

curl -s localhost:8080/api/showtimes/1/seats | jq '.seats[:3]'

curl -s -X POST localhost:8080/api/reservations \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"showtimeId":1,"seatIds":[1,2]}'
```
