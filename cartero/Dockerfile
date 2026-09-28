# =============================================================================
# El cartero de Esfinge. Contexto de construcción: esta misma carpeta.
# =============================================================================
FROM node:22-bookworm-slim AS base
ENV PNPM_HOME=/pnpm
ENV PATH=/pnpm:$PATH
RUN corepack enable
WORKDIR /app

# ---- construcción -----------------------------------------------------------
FROM base AS build
COPY package.json pnpm-lock.yaml tsconfig.json tsconfig.construir.json ./
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile
COPY src ./src
RUN pnpm run construir
RUN pnpm install --prod --frozen-lockfile

# ---- imagen final -----------------------------------------------------------
FROM base AS cartero
ENV NODE_ENV=production
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/package.json ./package.json
COPY --from=build /app/dist ./dist
# Cargar el módulo de verdad: un `test -f` no detecta un enlace roto.
RUN node --input-type=module -e "await import('nodemailer'); console.log('la imagen carga sus módulos')"
USER node
EXPOSE 8791
CMD ["node", "dist/cartero.js"]
