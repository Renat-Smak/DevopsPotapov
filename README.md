# Лабораторная работа № 6. Вариант 10 «Аптека»

[![CI](https://github.com/Renat-Smak/DevopsPotapov/actions/workflows/ci.yml/badge.svg)](https://github.com/Renat-Smak/DevopsPotapov/actions/workflows/ci.yml)

Тема работы — основы микросервисной архитектуры. В модуле два отдельных приложения на Go kit. Сервис заказов повторяет базовый пример, а «Аптека» обслуживает запросы по HTTP и gRPC через один набор эндпоинтов и одно хранилище в памяти.

## Сборка и запуск

Понадобится Go 1.25 или новее. Для проверки используются `curl` (в Windows PowerShell — `curl.exe`) и `grpcurl`. Команды ниже запускаются из корня проекта.

```text
go build ./...
go test ./...
go vet ./...
```

Сначала запустите сервис заказов командой `go run ./cmd/orders`. После проверки остановите его через Ctrl+C. Затем запустите «Аптеку» командой `go run ./cmd/pharmacy`. Оба приложения по умолчанию занимают HTTP-порт 8080. «Аптека» одновременно принимает gRPC на порту 9090. Адреса можно задать флагами `-http.addr` и `-grpc.addr`.

## Проверка заказов в PowerShell

```powershell
$http = 'http://localhost:8080'
curl.exe -i "$http/orders/1"

$order = @{
    customer_id = 'cust-1'
    restaurant_id = 'rest-42'
    order_items = @(@{product_code = 'P001'; name = 'Pizza'; unit_price = 550; quantity = 1})
} | ConvertTo-Json -Depth 4 -Compress
$id = ($order | curl.exe -sS -H 'Content-Type: application/json' --data-binary '@-' "$http/orders" | ConvertFrom-Json).id
curl.exe -i "$http/orders/$id"
@{status = 'Completed'} | ConvertTo-Json -Compress | curl.exe -i -X PATCH -H 'Content-Type: application/json' --data-binary '@-' "$http/orders/$id/status"
curl.exe -i "$http/orders/$id"
```

`POST /orders` возвращает HTTP 201 и UUID. Первый `GET` созданного заказа показывает `Pending`, а `GET` после `PATCH` — `Completed`. Маршрут `/orders/1` читает учебную запись, которая создаётся при запуске.

## Проверка «Аптеки» в PowerShell

```powershell
$http = 'http://localhost:8080'
$grpc = 'localhost:9090'
$svc = 'pharmacy.v1.PharmacyService'
$medicine = @{name = 'Paracetamol'; manufacturer = 'Demo Pharma'; price = 149.5; prescription = $false; quantity = 20}

$id = ($medicine | ConvertTo-Json -Compress | curl.exe -sS -H 'Content-Type: application/json' --data-binary '@-' "$http/medicines" | ConvertFrom-Json).id
curl.exe -i "$http/medicines/$id"
@{count = 3} | ConvertTo-Json -Compress | curl.exe -i -H 'Content-Type: application/json' --data-binary '@-' "$http/medicines/$id/sell"
curl.exe -i "$http/medicines/$id"

grpcurl -plaintext $grpc list $svc
@{medicine = $medicine} | ConvertTo-Json -Depth 3 -Compress | grpcurl -plaintext -d '@' $grpc "$svc/Create"
@{id = $id} | ConvertTo-Json -Compress | grpcurl -plaintext -d '@' $grpc "$svc/GetByID"
@{id = $id; count = 4} | ConvertTo-Json -Compress | grpcurl -plaintext -d '@' $grpc "$svc/Sell"
curl.exe -i "$http/medicines/$id"
```

Количество для записи `$id` меняется с 20 до 17 после HTTP-продажи и с 17 до 13 после gRPC-продажи. В JSON-выводе `grpcurl` поле `quantity` имеет строковое представление типа Protobuf `int64`.

| Операция | HTTP | gRPC |
| --- | --- | --- |
| Создание | `POST /medicines` → 201 | `Create` |
| Получение | `GET /medicines/{id}` → 200 | `GetByID` |
| Продажа | `POST /medicines/{id}/sell` → 200 | `Sell` |
| Некорректные данные | 400 | `InvalidArgument` |
| Лекарство отсутствует | 404 | `NotFound` |
| Недостаточно товара | 409 | `FailedPrecondition` |

Операция `Sell` проверяет и изменяет остаток в одной секции под блокировкой. Записи хранятся до остановки процесса.

## Структура и генерация gRPC

- `cmd/orders`, `cmd/pharmacy` — точки входа.
- `orders`, `pharmacy` — доменные интерфейсы, реализация сервисов, репозитории, эндпоинты и адаптеры транспорта.
- `api/pharmacy/v1` — контракт Protobuf и сгенерированные файлы Go.

Сгенерированные файлы включены в проект. Для повторной генерации установите `protoc`, добавьте директорию установленных Go-плагинов в `PATH` и выполните команды.

```text
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/pharmacy/v1/pharmacy.proto
```

Документация [Go kit HTTP](https://pkg.go.dev/github.com/go-kit/kit/transport/http), [Go kit gRPC](https://pkg.go.dev/github.com/go-kit/kit/transport/grpc), [Protobuf для Go](https://protobuf.dev/reference/go/go-generated/) и [grpcurl](https://github.com/fullstorydev/grpcurl).

## Лабораторная работа № 7. CI/CD и уведомления

GitHub Actions проверяет проект на Go 1.25 и 1.26 при запросе на слияние в `main`, при обновлении `main` и при отправке тега `v*`. После успешных проверок собирается Docker-образ «Аптеки». Отчёты о покрытии доступны в артефактах запуска.

Для публикации образа задайте переменную репозитория `DOCKERHUB_IMAGE` в формате `namespace/pharmacy` и секреты `DOCKERHUB_USERNAME` и `DOCKERHUB_TOKEN`. Для уведомлений о статусе сборки при push добавьте секреты `TELEGRAM_TO` и `TELEGRAM_TOKEN`.
