# Stage 1: Build admin SPA
FROM node:22-alpine AS admin-build
WORKDIR /app
COPY src/admin/package*.json ./
RUN npm ci
COPY src/admin/ ./
RUN npm run build

# Stage 2: Build server
FROM node:22-alpine AS server-build
WORKDIR /app
COPY package*.json tsconfig*.json eslint.config.js vitest.config.ts ./
RUN npm ci
COPY src/server/ ./src/server/
COPY src/types/ ./src/types/
RUN npm run build

# Stage 3: Production
FROM node:22-alpine
RUN apk add --no-cache dumb-init
WORKDIR /app
COPY --from=admin-build /server/admin/public/ ./admin/public/
COPY --from=server-build /app/dist/ ./dist/
COPY --from=server-build /app/node_modules/ ./node_modules/
COPY package*.json ./
EXPOSE 3000
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD node -e "fetch('http://localhost:3000/health').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"
CMD ["dumb-init", "node", "dist/server/index.js"]
