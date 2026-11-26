# Catálogo de productos

[English version](README.en.md)

Gestión de un catálogo de productos: API REST en Go y aplicación web en React, en un solo repositorio.

## Requisitos

- Docker con Compose v2
- make

Para desarrollar o correr los tests también necesitas Go 1.27.4+, Node.js 22.22+ y pnpm (`corepack enable`).

## Levantar el proyecto

```sh
make up     # Postgres, API y web; espera a que estén listos
make seed   # opcional: 20 productos de ejemplo
```

- Web: <http://localhost:3000>
- API: <http://localhost:8080> (documentación en <http://localhost:8080/docs>)

`make down` lo detiene. Funciona sin configurar nada; para cambiar valores, copia `.env.example` a `.env`.

## Modo desarrollo

Con recarga en caliente (necesita Go y Node):

```sh
cp .env.example .env
make db        # solo Postgres
make dev-api   # API en :8080
make dev-web   # web en http://localhost:5173
```

## Tests y calidad

```sh
make test   # API y web (con Docker encendido: los tests de integración crean su propio Postgres)
make lint   # lint, formato y typecheck
```

`make help` lista todos los comandos.

## Estructura

| Ruta | Contenido |
| --- | --- |
| `api/` | API en Go + Gin ([api/CLAUDE.md](api/CLAUDE.md)) |
| `web/` | Web en React + Vite + TypeScript ([web/CLAUDE.md](web/CLAUDE.md)) |
| `openapi.yaml` | Contrato de la API, fuente única de verdad |
| `specs/` | Especificaciones de cada feature (Spec Kit) |

## Variables de entorno

`.env.example` las documenta todas. En producción viven en Railway y Vercel.

| Variable | Usada por | Descripción |
| --- | --- | --- |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | `db` (compose) | Credenciales locales (`catalog` por defecto). |
| `DATABASE_URL` | API | Conexión a Postgres. |
| `PORT` | API | Puerto HTTP (`8080`). |
| `CORS_ALLOWED_ORIGINS` | API | Orígenes permitidos separados por comas, `http(s)://host[:puerto]`. No acepta `*`. |
| `GIN_MODE` | API | `release` en producción. |
| `CLIENT_IP_HEADER` | API | Cabecera con la IP real del cliente para el rate limit. Vacía en local; `X-Real-IP` en Railway. |
| `VITE_API_URL` | web (al compilar) | URL raíz de la API, p. ej. `http://localhost:8080`. |

## Despliegue

Cada merge a `main` despliega la API en Railway y la web en Vercel.

**API y Postgres en Railway**

1. Crea un proyecto con un servicio **PostgreSQL** 18.
2. Agrega un servicio desde este repositorio: Root Directory `/api`, Config file path `/api/railway.toml`, Watch Paths `/api/**` y **Wait for CI** activado.
3. Variables: `DATABASE_URL=${{Postgres.DATABASE_URL}}`, `CORS_ALLOWED_ORIGINS=<URL de la web>`, `PORT=8080`, `GIN_MODE=release`, `CLIENT_IP_HEADER=X-Real-IP`.
4. Genera un dominio público en **Networking**: es el `VITE_API_URL` de la web.

**Web en Vercel**

1. Importa el repositorio con Root Directory `web` y el preset **Vite**.
2. Define `VITE_API_URL` con el dominio de la API. Si el build no encuentra pnpm 12, agrega `ENABLE_EXPERIMENTAL_COREPACK=1`.

**GitHub**: protege `main` exigiendo PR, los checks `OpenAPI`, `API (Go)` y `Web (React)`, y ramas actualizadas antes del merge. Vercel no espera a CI, así que esta protección es la que garantiza que solo llegue código probado.
