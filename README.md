# QuantBackend — پلتفرم داده‌ی تیک بازار اختیار معامله (TSETMC)

پلتفرم مقیاس‌پذیر، ساختاریافته و سریع به زبان **Go** و فریم‌ورک **Echo** جهت جمع‌آوری، پاک‌سازی، ذخیره‌سازی و سرویس‌دهی داده‌های تیک (ریز معاملات درون‌روزی) و تاریخچه روزانه بازار اختیار معامله بورس تهران.

---

## 🏗 معماری پروژه (Clean Architecture + Wire + Echo)

این سرویس با الگوبرداری از معماری ماژولار **Clean Architecture / DDD** پیاده‌سازی شده است:

```
QuantBackend/
├── bootstrap/                     # مدیریت متغیرهای محیطی (Env) و ثابت‌های سیستم (Constants)
├── cmd/
│   ├── app/main.go                # نقطه ورود سرور Echo (و اجرای --seed-only برای مهاجرت و بذر اولیه)
│   └── benchmark/main.go          # اسکریپت بنچ‌مارک تاخیر و دریافت نمونه داده‌های خام TSETMC
├── wire/
│   ├── wire.go                    # تعریف ProviderSetها و تزریق وابستگی با Google Wire
│   └── wire_gen.go                # کد تولیدشده خودکار توسط Wire
├── internal/
│   ├── domain/                    # لایه دامنه خالص (entity, enum, exception, logger, localization, metrics, options, repository)
│   ├── application/               # لایه موارد کاربرد (dto, usecase, service + testify/suite)
│   ├── infrastructure/            # لایه زیرساخت (database/GORM, zap logger, localization, prometheus, repository, seed)
│   └── presentation/              # لایه ارائه HTTP با Echo (controller, middleware, routes)
├── mocks/                         # ماک‌های Testify برای تست‌های واحد ایزوله
├── monitoring/prometheus/         # تنظیمات اسکریپ متریک‌ها با Prometheus
├── tests/                         # تست‌های موجودیت‌ها، کوک‌بوک و یکپارچگی PostgreSQL
├── docs/                          # مستندات، ERD و کوک‌بوک اندپوینت‌های بورس
├── Makefile                       # دستورات کمکی توسعه، تست، سید و داکر
├── docker-compose.yml             # هماهنگ‌سازی سرویس‌های postgres، seed، backend و پروفایل monitoring
├── Dockerfile                     # بیلد چندمرحله‌ای Go + تصویر Alpine با پشتیبانی از مخزن آروان‌کلاد
├── .golangci.yml                  # پیکربندی golangci-lint v2
└── .github/workflows/ci.yml       # پایپ‌لاین CI روی هر PR (lint, test + postgres, docker build)
```

---

## 🚀 راه‌اندازی کامل با Docker Compose (E7-3 · همه سرویس‌ها + Seed)

سرویس‌های تعریف‌شده در [docker-compose.yml](docker-compose.yml):
1. **`postgres` (`postgres:16-alpine`):** پایگاه‌داده با حجم پایدار `quant_postgres_data` و بررسی سلامت (`pg_isready`).
2. **`seed`:** پس از سلامت `postgres` اجرا می‌شود، با فلگ `--seed-only` جدول‌ها را `AutoMigrate` کرده و **۲۵ دارایی پایه پیش‌فرض** را در دیتابیس ثبت می‌کند و سپس با موفقیت خارج می‌شود.
3. **`backend`:** پس از اتمام موفق `seed` بالا می‌آید و وب‌سرویس Echo را روی پورت `8000` در دسترس قرار می‌دهد.
4. **`prometheus` (اختیاری با پروفایل `monitoring`):** متریک‌های مسیر `/metrics` سرویس بک‌اند را روی پورت `9090` جمع‌آوری می‌کند.

### تنظیم محیط و بالا آوردن کل پلتفرم:
```bash
cp .env.example .env
docker compose up --build -d
# یا با Makefile:
make up
```

### راه‌اندازی همراه با مانیتورینگ Prometheus:
```bash
docker compose --profile monitoring up --build -d
```

### اجرای دستی سرویس Seed:
```bash
docker compose run --rm seed
# یا:
make seed
```

### بررسی سلامت، لاگ‌ها و توقف سرویس‌ها:
```bash
docker compose ps
curl http://localhost:8000/health
docker compose logs -f backend
docker compose down
# برای پاک کردن کامل دیتابیس (Volume):
docker compose down -v
```

> **نکته میررور Alpine و Go Proxy در ایران:**  
> فایل [Dockerfile](Dockerfile) به‌صورت پیش‌فرض مخزن APK در Alpine را به میررور ابر آروان (`http://mirror.arvancloud.ir/alpine`) تغییر می‌دهد و برای دریافت ماژول‌های Go از `goproxy.io` استفاده می‌کند. در محیط‌های خارج از ایران (مانند CI گیت‌هاب) می‌توانید با `--build-arg ALPINE_MIRROR=""` از مخزن اصلی Alpine استفاده کنید.

---

## 💻 راه‌اندازی در محیط محلی (Local Development)

### ۱. دریافت ماژول‌ها و تنظیم `.env`
پیش‌نیاز: Go 1.25 یا بالاتر
```bash
go mod download
cp .env.example .env
```

### ۲. بالا آوردن پایگاه‌داده PostgreSQL
```bash
docker compose up -d postgres
```

### ۳. اجرای Seed یا اجرای کامل سرور Echo
```bash
# فقط اجرای AutoMigrate و ثبت ۲۵ دارایی پایه:
go run ./cmd/app --seed-only

# اجرای کامل وب‌سرور Echo (شامل AutoMigrate و Seed خودکار):
go run ./cmd/app
```

### ۴. بازتولید کد تزریق وابستگی Wire (در صورت تغییر وابستگی‌ها)
```bash
make wire
```

---

## 📡 اندپوینت‌های در دسترس (API Endpoints)

* **مستندات تعاملی Swagger:** [http://localhost:8000/docs](http://localhost:8000/docs)
* **مستندات ReDoc:** [http://localhost:8000/redoc](http://localhost:8000/redoc)
* **مشخصات OpenAPI JSON:** [http://localhost:8000/openapi.json](http://localhost:8000/openapi.json)
* **متریک‌های Prometheus:** [http://localhost:8000/metrics](http://localhost:8000/metrics)

| متد | مسیر (Route) | توضیحات |
| :---: | :--- | :--- |
| `GET` | `/health` | وضعیت سلامت سرور و بررسی اتصال زنده به دیتابیس PostgreSQL |
| `GET` | `/api/v1/underlyings` | دریافت فهرست کامل ۲۵ دارایی پایه بازار اختیار معامله |
| `GET` | `/api/v1/underlyings/{symbol_fa}` | دریافت اطلاعات یک دارایی پایه بر اساس نماد فارسی (مثل `اهرم` یا `خودرو`) |
| `GET` | `/metrics` | خروجی متریک‌های Prometheus |

پاسخ‌های API در قالب استاندارد پاکتی (`{"statusCode": ..., "message": ..., "data": ...}`) برگردانده می‌شوند و خطاها بر اساس هدر `Accept-Language` (`fa_IR` / `en_US`) بومی‌سازی می‌شوند.

---

## 🧪 تست‌ها و پایپ‌لاین CI (E7-2)

پایپ‌لاین [ci.yml](.github/workflows/ci.yml) روی هر **Pull Request** به‌صورت خودکار موارد زیر را اجرا می‌کند:
1. **`lint`:** بررسی `go mod verify`، یکدستی فرمت با `gofmt`، تحلیل `go vet` و اجرای `golangci-lint` (v2).
2. **`test`:** اجرای تمام تست‌های واحد و تست یکپارچگی PostgreSQL 16 با `-race` و تولید گزارش پوشش کد (`coverage.out`).
3. **`build`:** کامپایل باینری‌های `cmd/app` و `cmd/benchmark`، اعتبارسنجی `docker compose config` و تست بیلد `Dockerfile`.

### اجرای محلی تست‌ها و بررسی کیفیت:
```bash
make lint
make test
make cover
make build
# اجرای تست یکپارچگی به همراه کانتینر Postgres محلی:
make test-integration
```

---

## 🤖 شفافیت و اعلام استفاده از هوش مصنوعی (AI Usage Statement)
مطابق بند ۱۲ راهنمای درس، از ابزار هوش مصنوعی (Antigravity AI Agent) به عنوان دستیار توسعه جهت پیاده‌سازی و مهاجرت پروژه به زبان Go، فریم‌ورک Echo، Google Wire و GORM، تنظیم پایپ‌لاین CI و کانتینرسازی Docker استفاده شده است. کلیه خطوط کد، منطق دامنه‌ای و تست‌ها توسط اعضای تیم بازبینی و تایید شده است.
