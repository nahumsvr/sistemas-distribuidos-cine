# Manual de instalación

## Requisitos

| Herramienta | Versión | Para qué |
|---|---|---|
| Docker Desktop / Docker Engine | 24+ con Compose v2 | Ejecutar todo el sistema |
| Git | cualquiera | Clonar el repositorio |
| Go | 1.23+ | *(Opcional)* Desarrollar el backend fuera de Docker |
| Node.js | 22 LTS | *(Opcional)* Desarrollar el frontend fuera de Docker |

## 1. Clonar y configurar

```bash
git clone <url-del-repositorio> proyecto-cine
cd proyecto-cine
cp .env.example .env
```

Edita `.env` y cambia, como mínimo:

- `POSTGRES_PASSWORD`, y la misma contraseña dentro de `DATABASE_URL`.
- `JWT_SECRET`: genera uno con `openssl rand -base64 48`.

### Variables de entorno

| Variable | Servicio | Descripción |
|---|---|---|
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | db | Credenciales y nombre de la base |
| `DATABASE_URL` | backend | Cadena de conexión (host `db` dentro de Compose) |
| `JWT_SECRET` | backend | Secreto para firmar tokens (≥ 32 bytes) |
| `JWT_TTL` | backend | Duración del token (por defecto `1h`) |
| `FRONTEND_ORIGIN` | backend | Origen permitido por CORS |
| `API_PORT` | backend | Puerto de la API (por defecto `8080`) |
| `VITE_API_URL` | frontend (build) | URL de la API que usa el navegador |

> `VITE_API_URL` se incrusta al **compilar** el frontend. Si la cambias, vuelve a construir con `--build` y actualiza `connect-src` en `frontend/serve.json`.

## 2. Levantar el sistema

```bash
docker compose up --build
```

Espera a que los tres servicios estén `healthy` (`docker compose ps`). Después abre:

- Aplicación: <http://localhost:3000>
- API: <http://localhost:8080/api/healthz>

Las migraciones y el seed se aplican solos al arrancar el backend.

## 3. Comandos útiles

| Comando | Efecto |
|---|---|
| `docker compose up -d --build` | Levantar en segundo plano |
| `docker compose logs -f backend` | Ver los logs de la API |
| `docker compose ps` | Ver el estado y los healthchecks |
| `docker compose down` | Detener (los datos se conservan) |
| `docker compose down -v` | Detener y **borrar** la base de datos |
| `docker compose exec db psql -U cine cine` | Abrir una consola SQL |

## 4. Desarrollo local (sin reconstruir imágenes)

```bash
# Solo la base de datos en Docker. Para exponer el 5432 en desarrollo, crea un
# docker-compose.override.yml (no versionado) con: services.db.ports: ["5432:5432"]
docker compose up -d db

# Backend (en otra terminal). Cambia el host de DATABASE_URL a localhost.
cd backend && go run ./cmd/api

# Frontend (en otra terminal)
cd frontend && npm install && npm run dev   # http://localhost:5173
```

Para desarrollar el frontend con Vite, define `FRONTEND_ORIGIN=http://localhost:5173` en el backend para que CORS lo permita.

## 5. Solución de problemas

| Síntoma | Causa probable | Solución |
|---|---|---|
| `port is already allocated` | El 3000 o el 8080 está ocupado | Libera el puerto o cambia el mapeo en `docker-compose.yml` |
| El backend se reinicia en bucle | `DATABASE_URL` o `JWT_SECRET` inválidos | Revisa `.env` y `docker compose logs backend` |
| Error de CORS en el navegador | `FRONTEND_ORIGIN` no coincide con la URL del front | Ajusta la variable (incluye esquema y puerto) |
| La app no carga datos y la consola muestra un error de CSP | `connect-src` no incluye la URL de la API | Actualiza `frontend/serve.json` y reconstruye |
| Cambié el esquema y no se aplica | El volumen conserva la base vieja | Crea una migración nueva, o `docker compose down -v` en desarrollo |
