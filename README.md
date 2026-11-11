# QuantBackend — پلتفرم داده‌ی تیک بازار اختیار معامله (TSETMC)

پلتفرم مقیاس‌پذیر، ساختاریافته و سریع به زبان **Go** و فریم‌ورک **Echo** جهت جمع‌آوری، پاک‌سازی، ذخیره‌سازی و سرویس‌دهی داده‌های تیک (ریز معاملات درون‌روزی) و تاریخچه روزانه بازار اختیار معامله بورس تهران.

---

## 🏗 معماری پروژه (Architecture)

این سرویس بر اساس اصول **Clean Architecture / DDD** در زبان Go پیاده‌سازی شده است:

```
QuantBackend/
├── cmd/
│   ├── server/main.go             # نقطه ورود وب‌سرور Echo، اتصال دیتابیس، اجرای مایگریشن و Seed
│   └── benchmark/main.go          # اسکریپت بنچ‌مارک تاخیر و دریافت نمونه داده‌های خام TSETMC
├── internal/
│   ├── config/                    # خوانش تنظیمات محیطی (.env و متغیرهای سیستم)
│   ├── container/                 # مخزن تزریق وابستگی (Dependency Injection Container)
│   ├── domain/
│   │   ├── entity/                # موجودیت‌های مستقل دامنه (Entities)
│   │   ├── repository/            # اینترفیس‌های مخازن داده (Repository Interfaces)
│   │   └── options/               # ماشین‌حساب تحلیلی اختیار معامله (E9-1 Options Primer)
│   ├── application/service/       # سرویس‌ها و موارد کاربرد (Application Services)
│   ├── infrastructure/
│   │   ├── database/              # استخر اتصال PostgreSQL (pgx/v5) و اجرای مایگریشن‌های SQL
│   │   └── persistence/postgres/  # پیاده‌سازی ریپازیتوری‌ها و بذر ۲۵ دارایی پایه
│   └── interfaces/http/           # روتر Echo، میدلورهای CORS/Logger، DTOها و اندپوینت‌های REST v1
├── migrations/                    # فایل‌های SQL مایگریشن (Embed شده در باینری نهایی)
├── tests/                         # تست‌های یکپارچگی ساختار اسکیما و نمونه داده‌های کوک‌بوک
├── docs/                          # مستندات و کوک‌بوک اندپوینت‌های بورس
├── docker-compose.yml             # اجرای کامل دیتابیس PostgreSQL 16 و سرویس بک‌اند Go
├── Dockerfile                     # بیلد چندمرحله‌ای (Multi-stage) و تصویر سبک Alpine
└── .github/workflows/ci.yml       # پایپ‌لاین CI (gofmt, go vet, go test -race, go build)
```

---

## 🚀 راه‌اندازی سریع با Docker Compose (پیشنهادی)

برای بالا آوردن کل پلتفرم (پایگاه‌داده PostgreSQL 16 به همراه وب‌سرویس Go / Echo) تنها یک دستور کافی است:

```bash
docker compose up --build -d
```

### بررسی وضعیت سلامت سرویس‌ها:
```bash
docker compose ps
curl http://localhost:8000/health
```

### مشاهده لاگ‌ها:
```bash
docker compose logs -f backend
```

### متوقف کردن سرویس‌ها:
```bash
docker compose down
# برای پاک کردن کامل حجم دیتابیس:
docker compose down -v
```

---

## 💻 راه‌اندازی در محیط محلی (Local Development)

### ۱. پیش‌نیازها و دریافت ماژول‌ها
پیش‌نیاز: Go 1.25 یا بالاتر (نسخه ماژول پروژه در `go.mod` تعریف شده است)
```bash
go mod download
```

### ۲. تنظیم متغیرهای محیطی
```bash
cp .env.example .env
```

### ۳. بالا آوردن پایگاه‌داده محلی با داکر
```bash
docker compose up -d postgres
```

### ۴. اجرای وب‌سرور Echo
```bash
go run ./cmd/server
```

### ۵. اجرای اسکریپت بنچ‌مارک TSETMC (اختیاری)
```bash
go run ./cmd/benchmark
```

---

## 📡 اندپوینت‌های در دسترس (API Endpoints)

پس از اجرای سرویس، مستندات تعاملی Swagger، ReDoc و مشخصات OpenAPI در دسترس هستند:
* **مستندات تعاملی Swagger:** [http://localhost:8000/docs](http://localhost:8000/docs)
* **مستندات ReDoc:** [http://localhost:8000/redoc](http://localhost:8000/redoc)
* **مشخصات OpenAPI JSON:** [http://localhost:8000/openapi.json](http://localhost:8000/openapi.json)

| متد | مسیر (Route) | توضیحات |
| :---: | :--- | :--- |
| `GET` | `/health` | وضعیت سلامت سرور و بررسی اتصال زنده به دیتابیس PostgreSQL |
| `GET` | `/api/v1/underlyings` | دریافت فهرست کامل ۲۵ دارایی پایه بازار اختیار معامله |
| `GET` | `/api/v1/underlyings/{symbol_fa}` | دریافت اطلاعات یک دارایی پایه بر اساس نماد فارسی (مثل `اهرم` یا `خودرو`) |

---

## 🧪 اجرای تست‌ها و اعتبارسنجی کد

### اجرای تست‌های واحد با Race Detector:
```bash
go test -v -race ./...
```

### بررسی کیفیت و استایل کد (Go Vet و Format):
```bash
gofmt -l .
go vet ./...
```

### ساخت باینری اجرایی:
```bash
go build -o bin/quant-server ./cmd/server
```

---

## 🤖 شفافیت و اعلام استفاده از هوش مصنوعی (AI Usage Statement)
مطابق بند ۱۲ راهنمای درس، از ابزار هوش مصنوعی (Antigravity AI Agent) به عنوان دستیار توسعه جهت پیاده‌سازی و مهاجرت معماری Clean Architecture به زبان Go و فریم‌ورک Echo، تنظیم تست‌های خودکار و پیکربندی Dockerfile/Docker-compose استفاده شده است. کلیه خطوط کد، منطق دامنه‌ای و تست‌ها توسط اعضای تیم بازبینی و تایید شده است.
