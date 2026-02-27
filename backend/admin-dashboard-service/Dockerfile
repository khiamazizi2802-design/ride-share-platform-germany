FROM node:20.11.1-alpine3.19 AS builder

WORKDIR /app
COPY package*.json .
RUN npm ci

COPY . .
RUN npm run build

ROM node:20.11.1-alpine3.19

WORKDIR /app
ENV NODE_ENV=production

COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules

EXPOSE 3001
CMD ["node", "dist/index.js"]