# Лабораторная работа № 8. Основы Docker

Вариант 10 — cron-задача. Проект `cron/` добавляет в Alpine расписание для записи времени UTC в `/data/dates.log`.

## Проверки

В Linux или Git Bash с подключённым Docker выполните из папки лабораторной:

```bash
bash check-cron.sh
bash check-docker.sh
```

Для второго сценария также нужны `curl` и `jq`. Исходники «Аптеки» и Dockerfile из ЛР № 7 должны находиться на уровень выше или в соседней папке `lab06-07`. Сценарий запускает nginx, PostgreSQL и «Аптеку», сравнивает контексты сборки и удаляет созданные им контейнеры, тома, сеть и тестовые образы.

## Вариант 10 вручную

```bash
docker build -t pharmacy-cron:lab08 ./cron
docker volume create lab08-dates
docker run -d --name lab08-cron -v lab08-dates:/data pharmacy-cron:lab08
docker exec lab08-cron crontab -l
docker exec lab08-cron cat /data/dates.log
```

Первая запись появляется на ближайшей границе минуты. Повторите последнюю команду ещё через минуту. Для проверки сохранности удалите контейнер и запустите новый с томом `lab08-dates`.

## Публикация образа «Аптеки»

Запустите команды из корня исходного Go-проекта после входа в Docker Hub и создания репозитория `pharmacy`. Замените `NAMESPACE` именем аккаунта Docker Hub.

```bash
docker build -t pharmacy:lab08 .
docker tag pharmacy:lab08 NAMESPACE/pharmacy:lab08
docker push NAMESPACE/pharmacy:lab08
docker pull NAMESPACE/pharmacy:lab08
docker run --rm -p 127.0.0.1:8082:8080 NAMESPACE/pharmacy:lab08
```

После запуска проверьте HTTP через `curl -i http://localhost:8082/medicines/unknown`. Для отсутствующей записи ожидается HTTP 404.
