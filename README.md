# QuantBackend — پلتفرم داده‌ی تیک بازار اختیار معامله (TSETMC)

پلتفرم مقیاس‌پذیر، ساختاریافته و سریع به زبان **Go** و فریم‌ورک **Echo** جهت جمع‌آوری، پاک‌سازی، ذخیره‌سازی و سرویس‌دهی داده‌های تیک (ریز معاملات درون‌روزی) و تاریخچه روزانه بازار اختیار معامله بورس تهران.

---

## 🏗 معماری پروژه (Clean Architecture + Wire + Echo)

این سرویس با الگوبرداری از معماری ماژولار **Clean Architecture / DDD** (مشابه ساختار `BargheNo-Backend`) پیاده‌سازی شده است:

```
QuantBackend/
├── bootstrap/                     # مدیریت متغیرهای محیطی (Env) و ثابت‌های سیستم (Constants)
├── cmd/
│   ├── app/main.go                # نقطه ورود سرور Echo، اجرای Wire DI، AutoMigrate و Seed
│   └── benchmark/main.go          # اسکریپت بنچ‌مارک تاخیر و دریافت نمونه داده‌های خام TSETMC
├── wire/
│   ├── wire.go                    # تعریف ProviderSetها و تزریق وابستگی با Google Wire
│   └── wire_gen.go                # کد تولیدشده خودکار توسط Wire
├── internal/
│   ├── domain/                    # لایه دامنه خالص
│   │   ├── entity/                # موجودیت‌های دیتابیس و دامنه (UnderlyingAsset, OptionContract, ...)
│   │   ├── enum/                  # انواع شمارشی دامنه (ContractType, Moneyness, QualityStatus, ...)
│   │   ├── exception/             # خطاهای استاندارد دامنه (NotFoundError, ValidationErrors, ...)
│   │   ├── localization/          # اینترفیس چندزبانه‌سازی خطاها (fa_IR / en_US)
│   │   ├── logger/                # اینترفیس مستقل لاگ‌گیری
│   │   ├── metrics/               # اینترفیس متریک‌های Prometheus
│   │   ├── options/               # ماشین‌حساب تحلیلی اختیار معامله (E9-1 Options Primer)
│   │   └── repository/postgres/   # اینترفیس‌های مخازن داده (Repositories)
│   ├── application/               # لایه موارد کاربرد و منطق برنامه
│   │   ├── dto/                   # مدل‌های ورودی/خروجی سرویس‌ها (Request / Response DTOs)
│   │   ├── usecase/               # اینترفیس سرویس‌های کاربردی
│   │   └── service/               # پیاده‌سازی سرویس‌ها و تست‌های واحد (testify/suite)
│   ├── infrastructure/            # لایه زیرساخت
│   │   ├── database/              # مدل پایه GORM و اتصال PostgreSQL با تراکنش
│   │   ├── localization/          # پیاده‌سازی ترجمه فارسی و انگلیسی خطاها
│   │   ├── logger/                # لاگر ساخت‌یافته با Zap
│   │   ├── metrics/               # پیاده‌سازی متریک‌ها با Prometheus
│   │   ├── repository/postgres/   # پیاده‌سازی ریپازیتوری‌ها با GORM
│   │   └── seed/                  # بذر اولیه ۲۵ دارایی پایه بازار اختیار
│   └── presentation/              # لایه ارائه HTTP با فریم‌ورک Echo
│       ├── controller/            # Base controller، Validator، Pagination، Response Envelope و کنترلرهای v1
│       ├── middleware/            # میدلورهای CORS، Logger، Localization، Recovery، RateLimit و Prometheus
│       └── routes/                # ثبت روت‌ها و گروه اندپوینت‌های HTTP v1 و تست‌های روتر
├── mocks/                         # ماک‌های Testify برای لایه‌های سرویس، ریپازیتوری و دیتابیس
├── tests/                         # تست‌های موجودیت‌ها و نمونه داده‌های کوک‌بوک
├── docs/                          # مستندات و کوک‌بوک اندپوینت‌های بورس
├── Makefile                       # دستورات کمکی توسعه، تست و داکر
├── docker-compose.yml             # اجرای کامل دیتابیس PostgreSQL 16 و سرویس بک‌اند Go
├── Dockerfile                     # بیلد چندمرحله‌ای (Multi-stage) و تصویر سبک Alpine
└── .github/workflows/ci.yml       # پایپ‌لاین CI (gofmt, go vet, go test -race, go build)
```

---

## 🚀 راه‌اندازی سریع با Docker Compose (پیشنهادی)

```bash
docker compose up --build -d
# یا با Makefile:
make up
```

### بررسی وضعیت سلامت سرویس‌ها:
```bash
docker compose ps
curl http://localhost:8000/health
```

---

## 💻 راه‌اندازی در محیط محلی (Local Development)

### ۱. پیش‌نیازها و دریافت ماژول‌ها
پیش‌نیاز: Go 1.25 یا بالاتر
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
go run ./cmd/app
```

### ۵. بازتولید فایل تزریق وابستگی Wire (در صورت تغییر وابستگی‌ها)
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

پاسخ‌های API در قالب استاندارد پاکتی (`{"statusCode": ..., "message": ..., "data": ...}`) برگردانده می‌شوند و خطاها از طریق هدر `Accept-Language` (`fa_IR` / `en_US`) بومی‌سازی می‌شوند.

---

## 🧪 اجرای تست‌ها و اعتبارسنجی کد

```bash
make test
make lint
make build
```

---

## 🤖 شفافیت و اعلام استفاده از هوش مصنوعی (AI Usage Statement)
مطابق بند ۱۲ راهنمای درس، از ابزار هوش مصنوعی (Antigravity AI Agent) به عنوان دستیار توسعه جهت بازطراحی و مهاجرت معماری Clean Architecture پروژه به زبان Go، فریم‌ورک Echo، Google Wire و GORM، تنظیم تست‌های واحد با Testify و پیکربندی Docker استفاده شده است. کلیه خطوط کد، منطق دامنه‌ای و تست‌ها توسط اعضای تیم بازبینی و تایید شده است.
