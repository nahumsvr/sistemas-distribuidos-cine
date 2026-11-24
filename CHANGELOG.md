# Changelog

Todos los cambios relevantes del proyecto se documentan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y el proyecto usa [Versionado Semántico](https://semver.org/lang/es/).

## [Unreleased]

### Added

- Plan de arquitectura: frontend (Vite + React), API (Go) y PostgreSQL en contenedores Docker.
- Documentación inicial: `README.md`, `docs/arquitectura.md` (con ADRs), `docs/api.md`, `docs/base-de-datos.md`, `docs/manual-instalacion.md` y `docs/manual-uso.md`.
- Lista de tareas en `odd/tasks/sistema-reservas-cine.md`.
- `.gitignore` y `.env.example` con las variables de entorno del proyecto.

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
