# Changelog

Todos los cambios relevantes del proyecto se documentan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el proyecto usa [Versionado Semántico](https://semver.org/lang/es/).

## [Unreleased]

### Added

- Plan de arquitectura: frontend (Vite + React), API (Go) y PostgreSQL en contenedores Docker.
- Documentación inicial: `README.md`, `docs/arquitectura.md` (con ADRs), `docs/api.md`, `docs/base-de-datos.md`, `docs/manual-instalacion.md` y `docs/manual-uso.md`.
- Lista de tareas en `odd/tasks/sistema-reservas-cine.md`.
- `.gitignore`.
- Migraciones de base de datos (esquema, restricción anti doble reserva y seed de desarrollo) con script de verificación SQL.
- Backend base en Go: configuración validada por entorno, pool `pgx`, migraciones automáticas con goose, `GET /api/healthz`, logs JSON, headers de seguridad, recuperación de panics y apagado ordenado.
- Autenticación: `POST /api/auth/register`, `POST /api/auth/login` y `GET /api/auth/me`, con JWT HS256 de 1 hora en el header `Authorization: Bearer`.
- Catálogo público: `GET /api/movies`, `/api/movies/{id}`, `/api/movies/{id}/showtimes` (solo funciones futuras) y `/api/showtimes/{id}/seats` (mapa de asientos con disponibilidad calculada en una sola consulta).
- Reservas autenticadas: `GET/POST /api/reservations` y `GET/PUT/DELETE /api/reservations/{id}` con transacciones atómicas, protección anti doble reserva (`409 SEAT_TAKEN`), ventana previa al inicio (`422 SHOWTIME_STARTED`), límite de 1 a 10 asientos y aislamiento por dueño (`404`).
- Dockerfile del backend: multi-stage, imagen distroless de ~22 MB, usuario sin privilegios y healthcheck integrado.
- Frontend base en Vite + React + TypeScript: React Router con *code splitting* (`React.lazy`), TanStack Query, cliente HTTP en `shared/api/client.ts` (Bearer JWT, `ApiError`, manejo de `401` y `Retry-After`), `shared/auth/tokenStorage.ts` y layout accesible con navegación semántica.
- Dockerfile del frontend (multi-stage con `node:22-alpine` + `serve`, usuario `node` sin privilegios y healthcheck) y `frontend/serve.json` (caché inmutable en `/assets/**` y `no-cache` en HTML).

### Security

- Contraseñas con bcrypt (cost 12) y login sin enumeración de usuarios (misma respuesta y mismo tiempo).
- Validación estricta del JWT: solo HS256, issuer y `exp` obligatorios, rol válido.
- CORS con un solo origen permitido, sin credentials.
- Rate limit por IP en login y registro (`429` + `Retry-After`).
- Límite de 1 MB por cuerpo (`413`), `Content-Type: application/json` obligatorio (`415`) y rechazo de campos desconocidos.
- Headers de seguridad en el frontend (`frontend/serve.json`): CSP estricta (`script-src 'self'`, `style-src 'self'`, `connect-src 'self' http://localhost:8080`, `frame-ancestors 'none'`), `X-Content-Type-Options: nosniff`, `Referrer-Policy` y `X-Frame-Options: DENY`.

<!--
Plantilla para la primera versión entregable (mover aquí lo de [Unreleased] al publicar):

## [0.1.0] - AAAA-MM-DD

### Added
- Registro e inicio de sesión con JWT.
- Cartelera, funciones y disponibilidad de asientos.
- Crear, consultar, modificar y cancelar reservas.
- Panel de administración de películas y funciones.
- Dockerfiles por servicio y docker-compose.yml.

### Security
- Contraseñas con bcrypt, rate limit en autenticación, CORS con allowlist, CSP en el frontend.
-->
