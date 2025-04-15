# Этап сборки приложения
FROM node:16 AS builder

WORKDIR /app

# Копируем package.json и package-lock.json (если есть)
COPY package*.json ./
RUN npm install

# Копируем исходный код приложения
COPY . .

# Собираем приложение для production
RUN npm run build

# Финальный образ на базе Nginx
FROM nginx:stable-alpine
# Копируем собранное приложение в папку Nginx для отдачи контента
COPY --from=builder /app/build /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
