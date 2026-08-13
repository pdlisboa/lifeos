# build do monorepo pnpm (apps/web + packages/*) e serve o estático via
# nginx. Contexto = frontend/ — packages/api-client/src/schema.ts já vem
# gerado e commitado, então o build não depende do backend/api/openapi.yaml
# (06-frontend.md §3, §10.1: SPA servida como estático, sem SSR).

FROM node:22-alpine AS build
RUN corepack enable && corepack prepare pnpm@11.20.0 --activate
WORKDIR /src

# copia só os package.json antes do resto pra cachear o pnpm install
# enquanto o código muda sem mudar dependência.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json
COPY packages/core/package.json packages/core/package.json
COPY packages/api-client/package.json packages/api-client/package.json
COPY packages/ui-tokens/package.json packages/ui-tokens/package.json
RUN pnpm install --frozen-lockfile

COPY . .

# VITE_API_BASE_URL é embutido no bundle em build time (Vite não lê env em
# runtime). Default aponta pro host onde o compose expõe a api (API_PORT).
ARG VITE_API_BASE_URL=http://localhost:8080/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN pnpm --filter @lifeos/web build

FROM nginx:1.27-alpine
COPY --from=build /src/apps/web/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
