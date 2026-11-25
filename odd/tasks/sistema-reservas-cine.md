# Sistema de reservas de cine — tareas

Objetivo: sistema web para consultar disponibilidad y crear, modificar y cancelar reservas de asientos de cine. Usa Vite + React, Go y PostgreSQL en contenedores Docker.
Rama: `feat/sistema-reservas` · Entrega: un commit convencional por tarea · Tests: `go test ./...` (backend), `vitest` (frontend).

## Specs

Requisitos citados textualmente de `Sistema_de_reservas.pdf`:

- **S1** — "Es un sistema de reservas que permite a los usuarios consultar disponibilidad, crear, modificar y cancelar reservaciones desde una interfaz web, mientras la información se almacena y gestiona de forma centralizada y segura."
- **S2** — "Aplicación frontend funcional (Vite + React)."
- **S3** — "API backend en Go con sus endpoints (crear, consultar, modificar y cancelar reservas)."
- **S4** — "Base de datos PostgreSQL con su esquema y diagrama entidad-relación."
- **S5** — "Documentación: descripción del sistema, arquitectura y manual de instalación o uso."
- **S6** — "Repositorio de código (GitHub) con README."
- **S7** — "Contenedores: un Dockerfile por servicio (frontend y backend) y un docker-compose.yml que levanta frontend, backend y PostgreSQL."
- **S8** — "PostgreSQL con un volumen para conservar los datos" y "Docker Compose los levanta juntos en una red compartida con un solo comando."

Decisiones del equipo (no vienen del PDF):

- **S9** — El dominio es un **cine**: película → función (sala + horario) → asientos. Un asiento no se puede reservar dos veces en la misma función.
- **S10** — La autenticación usa un JWT de 1 h guardado en `localStorage` y enviado como `Authorization: Bearer`, sin nginx (ver ADR-001 en `docs/arquitectura.md`).
- **S11** — La documentación está en español; el código y los identificadores, en inglés.
- **S12** *(supuesto)* — Máximo 10 asientos por reserva. Solo se puede modificar o cancelar antes de que empiece la función.

## Tasks

| ID | Specs | Tarea | Responsable sugerido | Estado |
|---|---|---|---|---|
| T0 | S6, S8 | `git init`, `.gitignore`, `.env.example`, README, CHANGELOG, docs | Nahum | ✅ hecho |
| T1 | S4, S9 | Migraciones SQL (esquema + índices + `UNIQUE(showtime_id, seat_id)`) y seed (películas, 2 salas, funciones, usuarios de prueba) | Héctor | ✅ hecho |
| T2 | S3, S7 | Backend base: config por env, `slog`, pool `pgx`, `goose` embebido, `GET /api/healthz`, `http.Server` con timeouts y apagado ordenado, Dockerfile multi-stage (distroless, non-root) | Nahum | ✅ hecho |
| T3 | S1, S10 | Auth: `register`, `login`, `me`; bcrypt; emisión y middleware JWT; CORS con allowlist; rate limit en login/registro; tests | Nahum | ✅ hecho |
| T4 | S1, S3, S9 | Catálogo: películas, funciones y disponibilidad de asientos (una sola query); tests | Héctor | ✅ hecho |
| T5 | S1, S3, S9, S12 | Reservas: crear, listar, ver, modificar y cancelar; transacciones; `409` si el asiento está ocupado; autorización por dueño; tests de service y handler | Héctor | ⬜ |
| T6 | S3 | Admin: CRUD de películas y funciones (rol `admin`) | Héctor | ⬜ |
| T7 | S2, S7 | Frontend base: Vite + React + TS, React Router, TanStack Query, `shared/api/client.ts`, layout, Dockerfile (`serve`) + `serve.json` (CSP, headers, cache) | Belén | ✅ hecho |
| T8 | S2, S10 | Front auth: login, registro, `tokenStorage`, rutas protegidas, logout, manejo de `401` | Belén | ⬜ |
| T9 | S1, S2, S9 | Front catálogo: cartelera, detalle, funciones, mapa de sala para elegir asientos | Belén | ⬜ |
| T10 | S1, S2 | Front "Mis reservas": listar, modificar asientos, cancelar | Belén | ⬜ |
| T11 | S2 | Front admin mínimo (películas y funciones) | Belén | ⬜ |
| T12 | S7, S8 | `docker-compose.yml` con healthchecks, `depends_on: service_healthy` y volumen; probar `docker compose up --build` | Nahum | ⬜ |
| T13 | S5, S6 | Actualizar docs con capturas, ejemplos reales de la API y el CHANGELOG `0.1.0` | Todos | ⬜ |
| T14 | — | *(Opcional)* GitHub Actions: `go vet`, `go test`, `npm run lint`, `npm test`, `npm run build`, `npm audit` | Nahum | ⬜ |

Orden recomendado: T1 → T2 → T3 → (T4, T7 en paralelo) → T5 → T8 → T9 → T10 → T6 → T11 → T12 → T13 → T14.

### Criterios de aceptación globales

1. `docker compose up --build` levanta los tres servicios en estado `healthy`.
2. Un usuario puede registrarse, reservar asientos, modificarlos y cancelarlos desde la web.
3. Dos reservas simultáneas del mismo asiento terminan en un `201` y un `409`.
4. Una petición sin token, o con un token expirado o alterado, recibe `401`.
5. Los datos persisten después de `docker compose down` y `docker compose up`.

## Log

- **L1** (2026-11-24) — Pedido original: "necesito que mires el pdf donde se especifica el proyecto, que sea simple, nada complejo, planea la arquitectura, usa las mejores prácticas actuales (separación de responsabilidades, optimización en el bacj y front, buena seguridad, cookies que protejan tokens, etc) siempre teniendo en cuenta el alcance del proyecto (proyecto escolar de fin de semestre) y una arquitectura basada en contenedores de docker, genera la lista de tareas, changelog, documentación, readme, etc"
- **L2** (2026-11-24) — Decisión: el dominio es de asientos de cine y la documentación va en español.
- **L3** (2026-11-24) — Corrección del usuario: "mejor si pon las cookies en el local storage, para evitar el uso de ngnix, pon en la documentación por que se tomó esta desición". Se le explicó que una cookie `HttpOnly` también funciona sin nginx (vía CORS); el usuario eligió "localStorage + Bearer". Queda documentado en ADR-001.
- **L4** (2026-11-24) — T0 completada: esqueleto del repo y documentación inicial (commit `e884aba`).
- **L5** (2026-11-24) — T1 completada: `backend/internal/migrations/` (esquema, seed y `verify.sql`). Se verificó en `postgres:17-alpine`: up → verify → down/up del seed → verify → down completo. Resultados: 2 usuarios, 2 salas, 128 asientos, 4 películas, 56 funciones sin solapamiento; doble reserva rechazada (`unique_violation`); asiento asociado a otra función rechazado (`foreign_key_violation`); cancelar libera el asiento; el down deja 0 tablas. `migrations.go` todavía no se compiló porque `go.mod` llega con T2. Commit `094aa01`.
- **L6** (2026-11-24) — T2 completada. Módulo `proyecto-cine/backend` con Go 1.27, `pgx/v5` y `goose/v3` (Provider API). La capa HTTP se llama `internal/httpapi` para no chocar con `net/http`. Las migraciones embebidas usan el patrón `[0-9]*_*.sql`, que deja afuera a `verify.sql`. Se hizo con TDD: RED observado (símbolos indefinidos) → GREEN. Tests: config (valores por defecto, overrides, 5 casos inválidos, el secreto no se filtra), httpapi (healthz 200/503, 404 en JSON, headers de seguridad, recover → 500 en JSON) e integración con Postgres real (migración idempotente, versión de goose 2, 56 funciones, fallo rápido). E2E con Docker: imagen de 22 MB que corre como `nonroot`, healthcheck `healthy`, `/api/healthz` 200 → 503 al desconectar la DB, SIGTERM → "shutting down" con exit 0, y la config inválida falla con un mensaje claro y sin filtrar secretos. Commit `2415d32`.
- **L7** (2026-11-24) — T3 completada. Paquetes nuevos: `domain` (User, errores), `auth` (JWT), `service` (AuthService) y `store/postgres/users.go`. En `httpapi`: handlers, `RequireAuth`, `CORS` y el rate limit `ipLimiter`. Se hizo con TDD: RED → GREEN, `go test -race` en verde e integración con Postgres en verde. E2E con curl contra el contenedor: login del seed (bcrypt de pgcrypto compatible con Go), `me`, registro, `409` por email duplicado, `401` con contraseña incorrecta, preflight CORS y `429` del rate limit. Verificador independiente (riesgo alto: auth): **PASS en S1 y S10, sin bloqueantes**. Se corrigieron sus advisories 1–5: `415` si el `Content-Type` no es JSON (cierra los formularios cross-site sin preflight), esquema `Bearer` sin distinguir mayúsculas, y docs con el `413`, la contraseña medida en bytes y `createdAt`. Pendiente menor: el método equivocado responde `404` en lugar de `405` por el fallback `/` (ya pasaba antes de T3). Commit `7bc4451`. Evaluación nativa: riesgo `high` → revisión ofrecida (lentes risk y reliability); el usuario eligió **"Omitir esta vez"** (`declined_this_candidate`). Siguiente paso: T4.
- **L8** (2026-11-24) — T4 completada. Paquetes nuevos: `domain/catalog.go`, `service/catalog.go` (`CatalogService`, reloj inyectable), `httpapi/catalog.go` y `store/postgres/catalog.go`. Rutas públicas de solo lectura: `GET /api/movies`, `/api/movies/{id}`, `/api/movies/{id}/showtimes` (solo futuras) y `/api/showtimes/{id}/seats`. La disponibilidad sale de **una sola query** (`NOT EXISTS` sobre `reservation_seats`, servida por `uq_reservation_seats_showtime_seat`); al cancelar se borran esas filas, así que el asiento vuelve a aparecer libre. Un `{id}` inválido da `400` y uno inexistente `404`; las listas vacías son `[]`. Se hizo con TDD: RED observado (símbolos indefinidos) → GREEN, `go test -race` en verde e integración con Postgres 17 real (4 películas, 80 asientos de la sala 1, reserva → ocupado → cancelar → libre, dentro de una transacción con rollback). E2E con curl contra el servidor real: 200/404/400 correctos. Riesgo: ninguno (solo se agregan rutas GET; no cambia auth, datos ni contratos existentes). Commit `5fd6d50`. Siguiente paso: T5 (reservas).
- **L9** (2026-11-25) — T7 completada. Proyecto `frontend/` con Vite 8 + React 19 + TypeScript, React Router 7 con *code splitting* (`React.lazy` por vista) y TanStack Query 5. Se implementaron `shared/api/client.ts` (`ApiError`, `Authorization: Bearer`, `credentials: 'omit'`, `Retry-After` y manejo de `401` solo cuando se envió token), `shared/auth/tokenStorage.ts`, `shared/api/queryClient.ts`, `shared/ui/Layout.tsx` (landmarks semánticos, skip link, `aria-current="page"` y actualización de `<title>`), estilos con `@layer` y tokens sin reset global, `serve.json` (CSP estricta, `X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options`, `immutable` en `/assets/**` y `no-cache` en HTML) y `Dockerfile` multi-stage (`node:22-alpine` + `serve@14.2.8`, usuario `node` sin privilegios y healthcheck con `fetch`). Verificación: `npm audit` con 0 vulnerabilidades, `npm run lint` sin errores, `vitest` (3 archivos, 13 tests en verde), `npm run build` con 8 chunks lazy; E2E en Docker (`uid=1000(node)`, healthcheck `healthy`, SPA fallback en `/reservations`, caché inmutable en `/assets/*`, headers de seguridad presentes y `serve.json` no expuesto). Commit `ced8db9`. Siguiente paso: integrar `worktree-t07-frontend` en `feat/sistema-reservas` y continuar con T5 o T8.
