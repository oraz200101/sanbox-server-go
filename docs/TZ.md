# ТЗ: sandbox-server на Go

Дата: 2026-09-16. Источник: Java-проект `~/IdeaProjects/arman` (greetgo `sandbox-server`, Spring Boot 2.7, Java 11, ~10k строк main + ~9.5k строк тестов).
Цель: переписать тот же функционал на Go (1.26) как учебный проект. Внешний контракт (HTTP API, топики Kafka, схема Postgres, индексы Elastic, формат CIA-файлов) сохраняем 1:1, внутреннюю архитектуру делаем идиоматичной для Go.

---

## 1. Что это за система

Сервис управления клиентами (customer) с тремя хранилищами и событийной синхронизацией между ними:

```
HTTP (create/update/delete)
        │
        ▼
    MongoDB  ──(publish)──►  Kafka ──┬──► consumer → Elasticsearch  (поиск/таблица/пагинация)
  (источник                          └──► consumer → PostgreSQL     (отчёты PDF/XLSX, справочник charm)
   истины)
```

Плюс отдельная batch-утилита «миграция CIA»: забирает по SSH архивы `*.tar.bz2` с XML, льёт во временные таблицы Postgres, валидирует, ищет дубли, апсертит в основные таблицы, пишет файл ошибок и выгружает результат обратно по SFTP.

Плюс «учебные тикеты»: hot-config через Zookeeper, планировщик, email, внешний сервис (ИВС), логирование.

---

## 2. Скоуп

### 2.1 Обязательно (ядро)

| # | Модуль | Java-аналог |
|---|---|---|
| A | HTTP API: Customer CRUD, Charm, Filter, Report, TestModelA CRUD/Table | `controller/*` |
| B | Mongo-хранилище customer + testModelA | `mongo/*`, `impl/CustomerCrudRegisterImpl`, `impl/TestARegisterImpl` |
| C | Kafka: продюсер + 3 консьюмера | `prod/kafka`, `kafka/consumer/*` |
| D | Elasticsearch: индексы, CRUD документов, фильтр/сортировка/count | `elastic/*`, `impl/CustomerElasticRegisterImpl`, `impl/TestAElasticRegisterImpl` |
| E | PostgreSQL: схема + CRUD customer/address/phone/charm | `postgres/*`, `db/changelog/*` |
| F | Отчёты PDF и XLSX по таблице customer | `report/*` |
| G | Миграция CIA (SSH → XML → tmp-таблицы → валидация → upload) | `migration/*` |
| H | Генератор тестовых CIA-файлов | `migration/GenerateInputFiles` |
| I | Ожидание готовности зависимостей при старте, автосоздание индексов | `spring_config/connection/*`, `elastic/ElasticCreator` |
| J | Конфигурация с горячей перезагрузкой (ElasticConfig, TestConfig, BatchConfig, SshConfig, PostgreSQLConfig) | `prod/zookeeper/*`, `config/*` |
| K | Планировщик с задачей раз в минуту, расписание из конфига | `scheduler/TestScheduler`, `spring_config/scheduler/*` |

### 2.2 Опционально (учебные тикеты, делать после ядра)

- Email-отправка (SMTP) с контроллером `POST /email/send`.
- Внешний сервис «эко-проверка компаний» (in-services) с fake- и real-реализациями.
- Пример hot-config из файла.
- Пример логирования с уровнями и трассировкой.

### 2.3 Не переносим

- Spring-специфика (`BeanConfigAll`, `@Lazy`, `SpringBootServletInitializer`).
- Liquibase как инструмент: заменяем на Go-миграции, SQL-файлы переносим как есть.
- Zookeeper как хранилище конфигов: заменяем файлами с hot-reload (см. §7). Zookeeper остаётся в docker-compose только потому, что он нужен Kafka.
- Mongo-express, Kafdrop, Kibana, ZooNavigator: оставляем в docker-compose как есть, кода не требуют.

---

## 3. Доменная модель

### 3.1 Customer (Mongo, коллекция `Customer`, БД `sandbox`)

```
id          ObjectId (Mongo _id)
name        string   (обязательно)
surname     string   (обязательно)
patronymic  string
charm       Charm{id int64, name string, description string}   (обязательно, все 3 поля)
gender      enum MALE | FEMALE   (обязательно)
birthDate   date, JSON-формат "yyyy-MM-dd"   (обязательно)
phones      []Phone{type enum HOME|WORK|MOBILE, number string}
addresses   []Address{addressType enum FACT|REG, street, house, apartment string}
```

Поведение геттеров в Java: null-поля отдаются как пустая строка / пустой список / `MALE` / «сейчас». В Go: хранить как есть, нормализовать при маппинге в ответ (пустые строки и пустые слайсы вместо `null`).

### 3.2 Правила валидации Customer (create и update)

- `name`, `surname` непустые (не только пробелы).
- `charm` не nil, `charm.id` не nil, `charm.name` и `charm.description` непустые.
- `gender`, `birthDate` не nil.
- `phones` непустой список; у каждого телефона есть `type`, `number` непустой и состоит только из цифр (регэксп `\D` не должен матчиться); минимум один телефон типа `MOBILE`.
- `addresses` непустой список; у каждого адреса `house`, `street`, `apartment` непустые и `addressType` задан; **ровно один** адрес типа `REG`.
- Для update дополнительно: `id` непустой и парсится как ObjectId.

Ошибка валидации → HTTP 400 с текстом сообщения (Java кидал `IllegalArgumentException`). Не найдено → 404 (`NoElementWasFoundException`).

### 3.3 TestModelA (Mongo, коллекция `TestModelA`)

```
id        ObjectId
strField  string   (обязательно)
boolField bool     (обязательно)
intField  int      (обязательно, 0 ≤ x ≤ 1_000_000)
```

### 3.4 Charm (Postgres, справочник)

`id bigserial PK, name text UNIQUE, description text`. Сидится 10 записями на русском (см. `001-initial-schema.sql` в исходнике — перенести дословно).

### 3.5 Схема PostgreSQL (перенести дословно из `001-create-tables.sql`)

```sql
charm            (id bigserial PK, name text UNIQUE, description text)
customer         (id text PK, name, surname, patronymic, gender text, birth_date date, charm_id text, cia_id text)
customer_address (customer_id text, type text, street, house, apartment text, PK(customer_id, type))
customer_phone   (number text, customer_id text, type text, PK(number, customer_id))
```

Известная странность оригинала: `customer.charm_id` — `text`, хотя `charm.id` — `bigserial`. Сохраняем ради совместимости, в отчёте так и отдаётся строкой.

### 3.6 Индексы Elasticsearch

`customer`:
```json
{"mappings":{"properties":{
  "id":{"type":"keyword"},"name":{"type":"keyword"},"surname":{"type":"keyword"},
  "patronymic":{"type":"keyword"},"charm":{"type":"keyword"},"gender":{"type":"keyword"},
  "birthDate":{"type":"date","format":"yyyy-MM-dd"}}}}
```
`model_a`:
```json
{"mappings":{"properties":{"id":{"type":"keyword"},"strField":{"type":"text"}}}}
```
Документ `customer` в ES содержит `charm` как **строку** (имя), не объект. `_id` документа = id из Mongo.

---

## 4. HTTP API (контракт сохраняем 1:1, порт 8080, CORS `*`)

Все тела — JSON. Даты — `"yyyy-MM-dd"`.

### 4.1 Customer CRUD — `/customer/crud`

| Метод и путь | Тело запроса | Ответ |
|---|---|---|
| `POST /read` | `{"id": string}` | `CustomerReadRes` (все поля §3.1, id строкой) |
| `POST /create` | `CustomerReadRes` без id | `string` — новый id (JSON-строка) |
| `POST /create-fake-data` | `{"size": int}` | пусто |
| `POST /update` | `CustomerReadRes` с id | пусто |
| `POST /delete` | `{"ids": [string]}` | пусто |

Семантика:
- `create`: валидация → сгенерировать ObjectId → insertOne в Mongo → publish в `CUSTOMER` и в `CUSTOMER_POSTGRES` с `changeVariant=CREATE`. Возвращает id.
- `create-fake-data`: загрузить все charm из Postgres → сгенерировать `size` случайных клиентов (случайные строки, случайный charm из списка, случайный пол, дата, 1+ телефон в т.ч. MOBILE, адреса FACT+REG) → insertMany → по каждому publish в оба топика CREATE.
- `update`: валидация → updateOne по `_id` (set всех 8 полей) → перечитать документ → publish UPDATE в оба топика. Если документ не найден после update — молча ничего не публиковать (лог error).
- `delete`: пустой список → no-op. Иначе: найти документы по `_id in ids` → deleteMany → по каждому найденному publish DELETE в оба топика.

### 4.2 Charm — `/charm/crud`

| `POST /read` | без тела | `[]Charm` из Postgres |

### 4.3 Фильтр по Elastic — `/customer/filter`

Запрос `CustomerFilterReq`:
```
search       string|null
limit        int   (0 → 10)
offset       int
orderByField enum name|surname|patronymic|charm|age|gender|birthDate | null
orderBy      enum ASC|DESC
```

| `POST /table` | → `[]CustomerFilterTableRes` = `{id,name,surname,patronymic,charm,age,gender,birthDate}` |
| `POST /pagination` | → `{"total": int}` |

Логика ES-запроса (`table`):
- `search == null` → `match_all`; иначе `bool.should` из трёх `wildcard` `*search*` по `name`, `surname`, `patronymic`.
- Сортировка: если `orderByField` задан — `sort` по нему. Для `age` и `birthDate` сортируем по `birthDate`, **инвертируя** направление (ASC по возрасту = DESC по дате рождения).
- `size=limit`, `from=offset`.
- `age` вычисляется на стороне сервера как полные годы между `birthDate` и сегодня.
- `pagination`: тот же query в `_count`, без сортировки и пагинации.

Требование к Go-версии: собирать ES-запрос через структуры + `encoding/json`, а не конкатенацией строк (в оригинале `search` вставляется в JSON без экранирования).

### 4.4 Отчёты — `/customer/report`

| `POST /pdf` | без тела | `application/pdf`, байты |
| `POST /xlsx` | без тела | `application/vnd.ms-excel`, байты |

Содержимое: `SELECT id, name, surname, patronymic, gender, birth_date, charm_id FROM customer` — таблица с заголовком из 7 имён полей (`id, name, surname, patronymic, gender, birthDate, charmId`), даты в `yyyy-MM-dd`. XLSX: один лист `customer`, жирный заголовок, автоширина колонок. PDF: таблица с рамками, заголовок жирный 10pt.

### 4.5 TestModelA — `/a/crud` и `/a/table`

| `GET /a/crud/load?id=` | → `{id,strField,boolField,intField}` |
| `POST /a/crud/create` | `{strField,boolField,intField}` → `string` id |
| `POST /a/crud/update` | `{id,strField,boolField,intField}` → пусто |
| `POST /a/crud/delete?id=` | → пусто |
| `GET /a/table/all?offset=0&limit=10` | → `[]{id,strField}` из ES (`match_all`) |
| `POST /a/table/filtered?offset&limit` | тело `{"strField": string}` → `[]{id,strField}`; ES `bool.must` + `match_phrase_prefix` по каждому непустому полю; пустое тело → `match_all` |

`create/update/delete` публикуют в топик `MODEL_A`.

---

## 5. Kafka

Bootstrap: `localhost:12211`. Группа консьюмеров: `my-group`. Сериализация: JSON-строка в value, ключ пустой. Auto-commit.

| Топик | Продюсер | Консьюмер | Payload |
|---|---|---|---|
| `MODEL_A` | TestModelA CRUD | → ES `model_a` | `{id, changeVariant, strField, boolField, intField}` |
| `CUSTOMER` | Customer CRUD | → ES `customer` | `{id, changeVariant, name, surname, patronymic, charm(string=name), gender, birthDate}` |
| `CUSTOMER_POSTGRES` | Customer CRUD | → Postgres | `{id, changeVariant, name, surname, patronymic, gender(string), birthDate, charm{id,name,description}, addresses[{type,street,house,apartment}], phones[{number,type}]}` |

`changeVariant ∈ CREATE | UPDATE | DELETE`. Продюсер отвергает сообщение с пустым `changeVariant` (ошибка, не отправка).

Поведение Postgres-консьюмера:
- CREATE: insert customer, затем insert каждого адреса и телефона.
- UPDATE: update customer; **update** каждого адреса по `(customer_id, type)`; телефоны — delete all by customer_id, затем insert заново.
- DELETE: delete адресов, телефонов, затем customer.
- Неизвестный `changeVariant` → ошибка (в Java — exception, сообщение теряется; в Go — залогировать и **закоммитить оффсет**, чтобы не крутить вечную петлю).

Все три консьюмера работают внутри того же процесса, что и HTTP-сервер (как в оригинале).

---

## 6. Миграция CIA (batch-утилита, отдельный бинарник `cmd/migrate`)

### 6.1 Цикл обработки

```
loop:
  files = ssh exec `ls <remoteDir>*.bz2`
  f = первый файл с суффиксом .tar.bz2 ; нет → выход
  renamed = sftp rename f → f + "." + random   (захват файла)
  local   = sftp get renamed → <localDir>/<имя без последнего расширения>
  xml     = tar -xjf local --strip-components=2 --wildcards 'build/out_files/*.xml'
  recordCount = из имени файла по регэкспу `-\d+-(\d+)\.xml$` (может отсутствовать)
  errorFile = build/migration/errors/error_<имя>.txt
  tx-соединение к Postgres (autocommit off):
     migrate(xml, errorFile, recordCount)
  sftp put local     → <remoteDir>migrated/migrated_<имя>
  sftp put errorFile → <remoteDir>migrated/<errorFile имя>
  удалить xml, local, errorFile
```

SSH: логин по паролю, `StrictHostKeyChecking=no`. В Go: `golang.org/x/crypto/ssh` + `github.com/pkg/sftp`; распаковку делать нативно (`compress/bzip2` + `archive/tar`), а не через `exec tar`.

### 6.2 Формат входного XML

```xml
<cia>
  <client id="1-AB-CD-EF-xxxxxxxxxx"> <!-- N -->
    <surname value="..."/>  <name value="..."/>  <patronymic value="..."/>
    <birth value="yyyy-MM-dd"/>  <charm value="..."/>  <gender value="MALE|FEMALE"/>
    <address>
      <fact street="..." house="..." flat="..."/>
      <register street="..." house="..." flat="..."/>
    </address>
    <homePhone>+7-xxx-xxx-xx-xx</homePhone>
    <mobilePhone>...</mobilePhone>   <workPhone>...</workPhone>
  </client>
</cia>
```

Порядок тегов внутри `client` случайный; телефонов 2–6; любой тег может отсутствовать. Файлы до 1 млн клиентов, поэтому парсинг **потоковый** (`encoding/xml` `Decoder.Token`, без загрузки дерева). Маппинг: `fact→FACT`, `register→REG`, `flat→apartment`, `homePhone→HOME`, `mobilePhone→MOBILE`, `workPhone→WORK`.

### 6.3 Временные таблицы

Имена: `tmp_<yyyyMMdd_HHmmss_SSS>_customer_<recordCount>` (и `_address`, `_phone`). Если `recordCount` не извлёкся — случайное число 0..1_000_000.

```sql
tmp_customer (id text, customer_id text, name, surname, patronymic, gender, birth, charm text,
              status integer NULL, number_record bigint PK)
tmp_address  (street, house, apartment, type text, status integer NULL, number_record bigint)
tmp_phone    (number, type text, status integer NULL, number_record bigint)
```

`id` = cia-id из XML; `customer_id` = свежесгенерированный ObjectId (для новых) либо существующий (см. валидацию); `number_record` = порядковый номер клиента в файле, связывает три таблицы.

### 6.4 Вставка (batch)

Параметры из `BatchConfig` (значения по умолчанию): `batchMaxSize=65` (адреса/телефоны), `batchCustomerMaxSize=100000` (клиенты, при достижении — execute + commit), `commitMaxSize=50000` (общий счётчик строк, при достижении — commit). Использовать `pgx` `CopyFrom` или `Batch`; печатать `record/sec` раз в 10 секунд.

### 6.5 Поиск дублей

По `tmp_customer`: `ROW_NUMBER() OVER (PARTITION BY id ORDER BY number_record DESC)`; всё, что `> 1`, помечается `status = 13 (DUPLICATE)`. Т.е. **выигрывает последняя запись в файле**. Обновление статусов пачками по 300 id.

### 6.6 Валидация (только SQL-UPDATE по tmp-таблицам, строго в этом порядке, каждый `WHERE status IS NULL`)

| code | Условие | errMsg |
|---|---|---|
| 1 | `id` null/'' | ID is null or blank |
| 2 | `name` null/'' | Name is null or blank |
| 3 | `surname` null/'' | Surname is null or blank |
| 4 | `gender` null/'' | Gender is null or blank |
| 5 | `gender NOT IN ('MALE','FEMALE')` | Gender is illegal |
| 6 | `birth` null/'' | Birth is null or blank |
| 7 | `birth !~ '^\d{4}-\d{2}-\d{2}$'` | Birth is illegal value or illegal format date |
| 8 | возраст > 100 или < 18 (`AGE(NOW(), TO_DATE(birth))`) | Age is > 100 and < 18 |
| 9 | `charm` null/'' | Charm is null or blank |
| 10 | нет адреса `type='REG'` с непустыми street/house/apartment | Reg address is null |
| 12 | (tmp_phone) `number !~ '^[0-9+-]+$'` | Phone number is illegal value or illegal format phone |
| 11 | нет телефона `type='MOBILE'` с непустым number | Mobile phone is null or blank |
| 13 | дубль (см. §6.5, ставится **до** валидации) | This record is duplicate |

Последний шаг: `UPDATE tmp_customer t SET customer_id = c.id FROM customer c WHERE c.cia_id = t.id` — для уже существующих клиентов подставляем их реальный id (по нему пойдёт upsert).

Замечание: `patronymic` не валидируется, пустой/пробельный допустим.

### 6.7 Upload в основные таблицы (только `status IS NULL`)

1. `INSERT INTO charm(name) SELECT charm FROM tmp_customer ... ON CONFLICT (name) DO NOTHING`.
2. `INSERT INTO customer(...) SELECT ... LEFT JOIN charm ON tmp.charm = charm.name ... ON CONFLICT (id) DO UPDATE SET name, surname, patronymic, gender, birth_date, charm_id`.
3. `INSERT INTO customer_address ... ON CONFLICT (customer_id, type) DO UPDATE SET ...`.
4. `INSERT INTO customer_phone ... ON CONFLICT (number, customer_id) DO UPDATE SET ...`.

### 6.8 Файл ошибок

Строки вида:
```
cia_id: <id>, errMsg: <текст>, from tmp_customer
cia_id: <id>, errMsg: <текст>, from tmp_address
cia_id: <id>, errMsg: <текст>, value: <number>, from tmp_phone
```

### 6.9 Генератор тестовых файлов (`cmd/gen-cia`)

Порт `GenerateInputFiles`: пишет `build/out_files/from_cia_<ts>-<n>-<count>.xml`, файлы по 300 / 3 000 / 30 000 / 300 000 / до 1 000 000 записей, ~10 % записей с намеренными ошибками (нет/пустая фамилия, имя, charm, дата рождения отсутствует/мусор/слишком старая/слишком молодая), ~1/5 — повторный `id` уже выданного клиента (дубли и «существующие»). Упаковывает в `.tar.bz2` (в Go bzip2-**сжатия** в stdlib нет: либо `exec tar -cjf`, либо `github.com/dsnet/compress/bzip2`). FRS-часть (счета/транзакции, `.json_row.txt`) в этом проекте нигде не читается — **не переносим**.

---

## 7. Конфигурация и горячая перезагрузка

Оригинал: интерфейсы-конфиги с `@DefaultXValue`, файлы в Zookeeper `sandbox/configs/<Name>.txt`, автосоздание файла с дефолтами при первом чтении, перечитывание раз в 2 с.

Go-версия: тот же принцип, но файлы на диске `configs/<Name>.txt` в формате `key=value`:
- При старте, если файла нет — создать с дефолтами и комментариями-описаниями.
- Перечитывать по `fsnotify` (или тикер 2 с); чтение через атомарную ссылку (`atomic.Pointer[T]`), без блокировок в горячем пути.

| Конфиг | Поля и дефолты |
|---|---|
| `ElasticConfig` | `updateImmediately=false` — после каждой записи в ES делать `_refresh` (нужно в тестах) |
| `TestConfig` | `value="It is value"` — печатает планировщик |
| `BatchConfig` | см. §6.4 |
| `SshConfig` | `sshUser, sshHost=localhost, sshPassword, remoteDir, localDir` |
| `PostgreSQLConfig` | `url, username, password` (для утилиты миграции) |

Статическая конфигурация процесса (порты, адреса) — через env с дефолтами:

```
HTTP_PORT=8080  MONGO_URI=mongodb://localhost:12217  KAFKA_BROKERS=localhost:12211
KAFKA_GROUP=my-group  ES_URL=http://localhost:12216
PG_DSN=postgres://sandbox:t30my2ayTWsGKC0lf7P0SfCFc421fF@localhost:12218/sandbox_db
CONFIG_DIR=./configs
```

---

## 8. Старт приложения (`cmd/server`)

Строго в этом порядке:
1. Прочитать env, поднять логгер (`log/slog`, JSON).
2. **Ждать готовности зависимостей**: цикл раз в 1 с, пока не ответит ES (`GET /`); расширить на Postgres и Mongo (в оригинале только ES).
3. Прогнать SQL-миграции (`goose` или `golang-migrate`, каталог `../main/internal/database/migrations/`).
4. Создать индексы ES, если не существуют (`HEAD /<index>` → `PUT`).
5. Запустить Kafka-консьюмеры (по горутине на топик).
6. Запустить планировщик (§9).
7. Запустить HTTP-сервер.
8. Graceful shutdown по SIGINT/SIGTERM: остановить HTTP → консьюмеры → закрыть клиентов.

---

## 9. Планировщик

Одна задача: раз в минуту печатать `FROM SCHEDULER: <TestConfig.value>`. Расписание должно читаться из файла `configs/scheduler/TestScheduler.scheduler.txt` (дефолт `repeat every 1 minute`) с горячей перезагрузкой — тот же механизм, что в §7. Поддержать хотя бы `repeat every N (second|minute|hour)` и cron-строку (`robfig/cron/v3`).

---

## 10. Нефункциональные требования и стек

| Область | Решение |
|---|---|
| Go | 1.26 (`~/sdk/go1.26.4`), модули, `go vet` + `staticcheck` чисто |
| Layout | `cmd/{server,migrate,gen-cia}`, `internal/{customer,charm,testmodela,migration,report}` (по домену), `internal/platform/{mongo,kafka,es,pg,config,sched}` (адаптеры), `../main/internal/database/migrations/`, `configs/`, `deploy/docker-compose.yml` |
| Архитектура | domain-типы и интерфейсы портов в пакете домена; адаптеры реализуют интерфейсы; хендлеры зависят только от сервиса. Никаких глобальных синглтонов кроме логгера |
| HTTP | `net/http` + `ServeMux` с паттернами методов (Go 1.22+), без фреймворка. Middleware: recover, request-id, логирование, CORS `*` |
| Mongo | `go.mongodb.org/mongo-driver/v2` |
| Kafka | `github.com/segmentio/kafka-go` (проще) или `github.com/twmb/franz-go` |
| ES | `github.com/elastic/go-elasticsearch/v8` low-level `esapi` (сохраняет дух оригинала, где всё через raw REST) |
| Postgres | `github.com/jackc/pgx/v5` + `pgxpool`; только параметризованные запросы; для tmp-таблиц имя подставляется через `pgx.Identifier{}.Sanitize()` |
| Миграции | `github.com/pressly/goose/v3`, SQL-файлы |
| XLSX | `github.com/xuri/excelize/v2` |
| PDF | `github.com/go-pdf/fpdf` |
| SSH/SFTP | `golang.org/x/crypto/ssh`, `github.com/pkg/sftp` |
| Логи | `log/slog` |
| Ошибки | sentinel-ошибки домена (`ErrNotFound`, `ErrValidation{Msg}`) → маппинг в HTTP-коды в одном месте |
| Контексты | каждый I/O принимает `context.Context`; таймауты на ES/PG/Mongo-вызовы |

Известные баги оригинала, которые в Go-версии **исправляем** (и фиксируем в README):
1. Инъекция в ES-запрос через `search` (строковая конкатенация JSON).
2. Инъекция имени таблицы в `TestPostgreRegister.createTable` — модуль не переносим вовсе.
3. UPDATE-консьюмер Postgres не удаляет адреса, которых больше нет у клиента (только апдейтит существующие типы). В Go: delete + insert, как для телефонов.
4. Исключение в консьюмере на неизвестном `changeVariant` — см. §5.
5. `refresh` вызывается на `/_refresh` для всех индексов, а не для конкретного — делать `/<index>/_refresh`.

---

## 11. Инфраструктура

`deploy/docker-compose.yml` — перенести из `arman/debug/docker-compose.yaml` без изменений портов:

| Сервис | Порт хоста |
|---|---|
| postgres 13.4 | 12218 (user `sandbox`, db `sandbox_db`, init-скрипт создаёт роль и БД) |
| mongo 4.4 | 12217; mongo-express 12213 |
| zookeeper | 12212; zoo-navigator 12210 |
| kafka 3.2 | 12211 (localhost-listener); kafdrop 12214 |
| elasticsearch 7.12 | 12216; kibana 12219 |

Makefile: `make up/down/recreate` (с очисткой `~/volumes/sandbox`), `make run`, `make migrate`, `make gen-cia`, `make test`, `make lint`.

---

## 12. Тестирование

Оригинал: TestNG, все тесты интеграционные против живого docker-compose (Kafka в тестах заменён на `KafkaProducerSimulator`, конфиги — на in-memory реализации).

Go-версия:
- **Unit** (без внешних систем): валидатор Customer/TestModelA; XML-парсер CIA (в т.ч. перемешанный порядок тегов, отсутствующие теги, пустой patronymic); построитель ES-запросов (сравнивать JSON); маппинги Kafka-payload ↔ модели; парсер расписания; извлечение `recordCount` из имени файла.
- **Интеграционные** через `testcontainers-go` (Postgres, Mongo, Elasticsearch; Kafka — `testcontainers` `kafka` модуль или `redpanda`): Mongo CRUD; Postgres CRUD всех трёх таблиц + charm; ES create/update/delete/table/pagination с сортировкой по каждому полю; полный конвейер миграции на файле из 300 записей с известным числом ошибок каждого типа; отчёты PDF/XLSX (проверять, что файл открывается и содержит N строк).
- **E2E** (по желанию): create через HTTP → сообщение в Kafka → документ появился в ES и строка в Postgres.
- Продюсер Kafka за интерфейсом → в unit-тестах сервиса подменяется на фейк, который копит сообщения.
- Цель покрытия: ≥ 70 % по `internal/`.

---

## 13. План работ (учебный порядок, каждая фаза — рабочая программа)

| Фаза | Содержимое | Что осваивается |
|---|---|---|
| 0 | Скелет: layout, `go.mod`, Makefile, docker-compose, slog, env-конфиг, graceful shutdown, `/healthz` | модули, контексты, сигналы |
| 1 | TestModelA: Mongo CRUD + HTTP, unit-тесты валидатора | `net/http`, mongo-driver, table-driven tests |
| 2 | Kafka producer + консьюмер `MODEL_A` → ES, автосоздание индексов, ожидание зависимостей, `/a/table` | горутины, каналы, kafka-go, esapi |
| 3 | Customer: Mongo CRUD, валидация, fake-data, продюсер в 2 топика | структуры, ошибки, interface-порты |
| 4 | Консьюмеры `CUSTOMER` → ES и `CUSTOMER_POSTGRES` → PG; goose-миграции; `/charm/crud`; `/customer/filter` | pgx, транзакции, testcontainers |
| 5 | Отчёты PDF/XLSX | стриминг ResultSet → writer, `io.Writer` |
| 6 | Hot-config из файлов + планировщик | fsnotify, `atomic.Pointer`, тикеры |
| 7 | Миграция CIA: XML-стрим → tmp-таблицы (batch/CopyFrom) → дубли → валидация → upload → error-файл; генератор файлов | потоковый парсинг, batch-вставка, работа с 1 млн записей, профилирование `pprof` |
| 8 | SSH/SFTP-обвязка миграции, tar.bz2 | x/crypto/ssh, archive/tar |
| 9 | Опциональные тикеты (email, in-services, примеры) | net/smtp, http-клиент с ретраями |

Критерий готовности ядра: фазы 0–8 закрыты, `make test` зелёный, все эндпоинты из §4 отвечают идентично Java-версии на одном docker-compose (можно проверить, гоняя одни и те же Postman-запросы в обе реализации).
