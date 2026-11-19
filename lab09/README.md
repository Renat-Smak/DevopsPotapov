# Лабораторная работа № 9. Работа с образами Docker

Вариант 10 — сервис «Счётчик посещений» на Go. `GET /` увеличивает счётчик, `GET /count` читает его, `GET /health` проверяет работоспособность. Счётчик живёт в памяти одного процесса.

## Проверка

Из папки `lab09` выполните на Linux с запущенным Docker, Python 3 и curl:

```bash
python3 -u verify.py
```

Сценарий проверяет три варианта образа, сравнивает контексты сборки, проставляет теги, измеряет запуск и HTTP-ответ, проверяет состояния `healthy` и `unhealthy`. Дополнительно отправляет два тега во временный локальный registry, удаляет тестовые теги и проверяет скачанный образ. Результаты выводятся в терминал. Созданные сценарием ресурсы удаляются при выходе.

## Ручной запуск

```bash
docker build -f Dockerfile.naive -t visit-counter:naive .
docker build --target plain -t visit-counter:plain .
docker build --target release -t visit-counter:1.0.0 .
docker build --target unhealthy -t visit-counter:unhealthy .
docker run --rm -p 127.0.0.1:8080:8080 visit-counter:1.0.0
```

Во втором терминале:

```bash
curl -fsS http://localhost:8080/count
curl -fsS http://localhost:8080/
curl -fsS http://localhost:8080/count
curl -i http://localhost:8080/health
```

Стадия `unhealthy` включает учебный режим, в котором `/health` отвечает HTTP 500. Обычные образы продолжают отвечать HTTP 200.

## Docker Hub

После настройки доступа выполните команды из текущего каталога. Замените `NAMESPACE` на имя аккаунта Docker Hub.

```bash
docker login
docker tag visit-counter:1.0.0 NAMESPACE/visit-counter:1.0.0
docker tag visit-counter:1.0.0 NAMESPACE/visit-counter:latest
docker push NAMESPACE/visit-counter:1.0.0
docker push NAMESPACE/visit-counter:latest
docker image rm NAMESPACE/visit-counter:1.0.0 NAMESPACE/visit-counter:latest
docker pull NAMESPACE/visit-counter:1.0.0
docker run --rm -p 127.0.0.1:8080:8080 NAMESPACE/visit-counter:1.0.0
```
