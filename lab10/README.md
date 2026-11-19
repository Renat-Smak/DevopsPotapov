# Лабораторная работа № 10. Docker Compose

Вариант 10 — миграции и начальное заполнение. Стек использует «Аптеку» из ЛР № 6. HTTP и gRPC обращаются к одному репозиторию PostgreSQL.

## Запуск

Скопируйте `.env.example` в `.env` и задайте пароль PostgreSQL. Если локальный `.env` уже есть, используйте его.

```bash
docker compose up -d --build --wait
docker compose ps -a
curl -i http://localhost:8080/health
curl -fsS http://localhost:8080/medicines/00000000-0000-4000-8000-000000000010
```

Compose ждёт готовности PostgreSQL. Затем одноразовый сервис `migrate` применяет миграции и завершается с кодом 0. После этого стартует приложение. Миграция 1 создаёт таблицу `medicines`, миграция 2 добавляет карточки Paracetamol и Ibuprofen. gRPC доступен на `localhost:9090`.

Изменение `APP_PORT` и команда `docker compose up -d --wait` перепубликуют HTTP-порт. Изменение пароля в `.env` применяется к новому тому PostgreSQL. Для существующего тома пароль нужно менять в самой базе.

```bash
docker compose logs -f app
docker compose exec app sh
docker compose stop
docker compose start
docker compose down
```

`stop` оставляет контейнеры. `down` удаляет контейнеры и сеть, сохраняя данные в томе. `down -v` удаляет и том. Для просмотра итоговой конфигурации используйте `docker compose config`. Полный вывод может содержать пароль, поэтому в отчёте показаны только выбранные поля.

## Автоматическая проверка

```bash
bash verify.sh
```

Нужны Docker Compose 2.24.4 или новее, Bash, curl, jq и OpenSSL. Проверка создаёт отдельные Compose-проекты с временными настройками, меняет HTTP-порт с 18080 на 18081 и удаляет только свои ресурсы. Файл `compose.failure.yaml` используется для эксперимента с запуском приложения до готовности PostgreSQL.
