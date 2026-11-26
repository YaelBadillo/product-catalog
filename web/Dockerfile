# syntax=docker/dockerfile:1

FROM node:24-alpine AS build
WORKDIR /app
# pnpm version comes from "packageManager" in package.json.
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
# Vite inlines VITE_* variables at build time.
ARG VITE_API_URL
RUN test -n "$VITE_API_URL" || (echo "VITE_API_URL build arg is required" >&2 && exit 1)
ENV VITE_API_URL=${VITE_API_URL}
RUN pnpm build

FROM nginxinc/nginx-unprivileged:1.30-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
