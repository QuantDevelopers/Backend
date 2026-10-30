# QuantBackend — پلتفرم داده‌ی تیک بازار اختیار معامله (TSETMC)

پلتفرم مقیاس‌پذیر، ساختاریافته و سریع جهت جمع‌آوری، پاک‌سازی، ذخیره‌سازی و سرویس‌دهی داده‌های تیک (ریز معاملات درون‌روزی) و تاریخچه روزانه بازار اختیار معامله بورس تهران.

---

## 🏗 معماری پروژه (Architecture)

این سرویس بر اساس اصول **Clean Architecture / DDD** (الهام گرفته از الگوی مدرن سرویس‌ها در `fastapi-ms`) پیاده‌سازی شده است:

```
QuantBackend/
├── src/
│   ├── config.py                 # تنظیمات محیطی با Pydantic Settings
│   ├── container.py              # مخزن تزریق وابستگی (Dependency Injection Container)
│   ├── main.py                   # کارخانه FastAPI، هندلر Lifespan و مدیریت CORS
│   ├── domain/                   # موجودیت‌های مستقل دامنه (Entities & Exceptions)
│   ├── application/              # سرویس‌ها و موارد کاربرد (Application Services)
│   ├── infrastructure/           # پایگاه داده PostgreSQL (SQLAlchemy 2.0 Async + Repositories)
│   └── interfaces/http/          # اندپوینت‌های REST API v1 و اسکیمای Pydantic
├── tests/                        # تست‌های خودکار با Pytest و AsyncClient
├── docs/                         # مستندات و کوک‌بوک اندپوینت‌های بورس
├── docker-compose.yml            # اجرای کامل دیتابیس PostgreSQL 16 و سرویس بک‌اند
├── Dockerfile                    # تصویر بهینه‌شده پایتون
└── .github/                      # قالب PR و پایپ‌لاین CI
```

---

## 🚀 راه‌اندازی سریع با Docker Compose (پیشنهادی)

برای بالا آوردن کل پلتفرم (پایگاه‌داده PostgreSQL 16 به همراه وب‌سرویس FastAPI) تنها یک دستور کافی است:

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

### ۱. نصب وابستگی‌ها
پیش‌نیاز: پایتون 3.11 یا بالاتر
```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

### ۲. تنظیم متغیرهای محیطی
```bash
cp .env.example .env
```

### ۳. بالا آوردن پایگاه‌داده محلی با داکر
```bash
docker compose up -d postgres
```

### ۴. اجرای وب‌سرور
```bash
uvicorn src.main:app --host 0.0.0.0 --port 8000 --reload
```

---

## 📡 اندپوینت‌های در دسترس (API Endpoints)

پس از اجرای سرویس، مستندات تعاملی Swagger و Redoc در دسترس هستند:
* **مستندات تعاملی Swagger:** [http://localhost:8000/docs](http://localhost:8000/docs)
* **مستندات Redoc:** [http://localhost:8000/redoc](http://localhost:8000/redoc)

| متد | مسیر (Route) | توضیحات |
| :---: | :--- | :--- |
| `GET` | `/health` | وضعیت سلامت سرور و بررسی اتصال زنده به دیتابیس PostgreSQL |
| `GET` | `/api/v1/underlyings` | دریافت فهرست کامل ۲۵ دارایی پایه بازار اختیار معامله |
| `GET` | `/api/v1/underlyings/{symbol_fa}` | دریافت اطلاعات یک دارایی پایه بر اساس نماد فارسی (مثل `اهرم` یا `خودرو`) |

---

## 🧪 اجرای تست‌ها و اعتبارسنجی کد

### اجرای تست‌های واحد با Pytest:
```bash
pytest -v
```

### بررسی کیفیت و استایل کد با Ruff:
```bash
ruff check src tests
```

---

## 🤖 شفافیت و اعلام استفاده از هوش مصنوعی (AI Usage Statement)
مطابق بند ۱۲ راهنمای درس، از ابزار هوش مصنوعی (Antigravity AI Agent) به عنوان دستیار توسعه جهت پیاده‌سازی ساختار اولیه (Scaffolding)، طراحی لایه‌بندی Clean Architecture بر اساس الگوی میکروسرویس مرجع، تنظیم تست‌های غیرهمگام (Async Unit Tests) و کانفیگ Dockerfile/Docker-compose استفاده شده است. کلیه خطوط کد، منطق دامنه‌ای و تست‌ها توسط اعضای تیم بازبینی و تایید شده است.
