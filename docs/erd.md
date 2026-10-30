# نمودار رابطه موجودیت‌ها (ERD) و معماری پایگاه‌داده

این سند معماری پایگاه‌داده رابطه ای (PostgreSQL 16) را برای **پلتفرم داده‌ی تیک و بازار اختیار معامله بورس تهران (TSETMC)** مستند می‌کند. این طراحی در پاسخ به نیازمندی‌های **کارت E3-1** و اصول فاز ۱ درس الگوریتم‌های معاملاتی تنظیم شده است.

---

## ۱. دیاگرام ERD (Mermaid Diagram)

```mermaid
erDiagram
    UNDERLYING_ASSETS ||--o{ OPTION_CONTRACTS : "manages"
    UNDERLYING_ASSETS ||--o{ DAILY_HISTORY : "has daily prices"
    OPTION_CONTRACTS ||--o{ DAILY_HISTORY : "has daily prices"
    UNDERLYING_ASSETS ||--o{ TICKS_RAW : "has raw ticks"
    OPTION_CONTRACTS ||--o{ TICKS_RAW : "has raw ticks"
    UNDERLYING_ASSETS ||--o{ TICKS_CLEAN : "has clean ticks"
    OPTION_CONTRACTS ||--o{ TICKS_CLEAN : "has clean ticks"
    UNDERLYING_ASSETS ||--o{ DATA_QUALITY_LOGS : "evaluated in"
    OPTION_CONTRACTS ||--o{ DATA_QUALITY_LOGS : "evaluated in"

    UNDERLYING_ASSETS {
        int id PK
        string symbol_fa UK "نماد فارسی سهم پایه (مثل اهرم، خودرو)"
        string name_fa "نام کامل شرکت یا صندوق"
        string tsetmc_id UK "کد اختصاصی ۱۲ تا ۱۸ رقمی در TSETMC"
        string isin UK "شناسه بین‌المللی نماد (ISIN)"
        string market_type "نوع بازار (Bourse / Farabourse)"
        boolean is_active "وضعیت فعال بودن نماد"
        timestamptz created_at
        timestamptz updated_at
    }

    OPTION_CONTRACTS {
        int id PK
        int underlying_asset_id FK "کلید خارجی به دارایی پایه"
        string symbol_fa UK "نماد قرارداد (مثل ضخود1204)"
        string name_fa "عنوان قرارداد اختیار"
        string contract_type "نوع قرارداد: CALL یا PUT"
        numeric strike_price "قیمت اعمال توافق‌شده (ریال)"
        int contract_size "اندازه قرارداد (معمولاً ۱۰۰۰ سهم)"
        date expiration_date "تاریخ سررسید قرارداد"
        date begin_date "تاریخ شروع بازگشایی نماد"
        string tsetmc_id UK "کد ۱۲ تا ۱۸ رقمی در TSETMC"
        string isin UK "شناسه بین‌المللی ۱۲ رقمی"
        boolean is_active "فعال یا منقضی شده"
        string status "وضعیت: ACTIVE / EXPIRED / DELISTED"
        timestamptz created_at
        timestamptz updated_at
    }

    DAILY_HISTORY {
        bigint id PK
        string instrument_type "نوع ابزار: UNDERLYING یا OPTION"
        int instrument_id "شناسه ابزار مربوطه"
        date trade_date "تاریخ روز معاملاتی"
        numeric open_price "قیمت اولین معامله"
        numeric high_price "بیشترین قیمت روز"
        numeric low_price "کمترین قیمت روز"
        numeric close_price "قیمت پایانی/آخرین"
        numeric yesterday_price "قیمت پایانی روز قبل"
        numeric settlement_price "قیمت تسویه (ویژه اختیار)"
        bigint volume "حجم کل معاملات روز"
        numeric value "ارزش کل معاملات روز (ریال)"
        int trade_count "تعداد دفعات معامله"
        int open_interest "موقعیت‌های باز (Open Interest)"
        timestamptz created_at
    }

    TICKS_RAW {
        bigint id PK
        string instrument_type "نوع ابزار: UNDERLYING یا OPTION"
        int instrument_id "شناسه ابزار"
        date trade_date "تاریخ معامله"
        time trade_time "زمان دقیق معامله (HH:MM:SS)"
        int sequence_number "شماره ردیف معامله در روز"
        numeric price "قیمت معامله (ریال)"
        bigint volume "حجم معامله (تعداد برگه/قرارداد)"
        jsonb raw_payload "متن یا آبجکت دست‌نخورده دریافتی از TSETMC"
        timestamptz created_at
    }

    TICKS_CLEAN {
        bigint id PK
        string instrument_type "نوع ابزار: UNDERLYING یا OPTION"
        int instrument_id "شناسه ابزار"
        date trade_date "تاریخ معامله"
        timestamptz trade_timestamp "زمان استاندارد شده (UTC/Tehran)"
        int sequence_number "ترتیب زمانی تیک‌ها بدون گسستگی"
        numeric price "قیمت پالایش‌شده"
        bigint volume "حجم معامله"
        boolean is_canceled "نشانگر باطل شدن معامله"
        boolean is_outlier "نشانگر جهش قیمتی نامتعارف"
        timestamptz created_at
    }

    DATA_QUALITY_LOGS {
        bigint id PK
        string instrument_type "نوع ابزار: UNDERLYING یا OPTION"
        int instrument_id "شناسه ابزار"
        date trade_date "تاریخ بررسی"
        string check_category "دسته بررسی: DUPLICATE / CHRONO / GAPS"
        string status "وضعیت: PASSED / WARNING / FAILED"
        int gap_count "تعداد شکاف‌های زمانی تشخیص‌داده‌شده"
        jsonb details "جزئیات آماری و لاگ خطاها"
        text notes "توضیحات تکمیلی مهندس داده"
        timestamptz created_at
    }
```

---

## ۲. مشخصات و قواعد یکپارچگی (Integrity Constraints)

### ۱. قیود بررسی دامنه (Check Constraints)
* `option_contracts.contract_type`: الزام به یکی از دو مقدار `'CALL'` یا `'PUT'`.
* `daily_history.instrument_type`: الزام به یکی از دو مقدار `'UNDERLYING'` یا `'OPTION'`.
* `ticks_raw.instrument_type` و `ticks_clean.instrument_type`: الزام به `'UNDERLYING'` یا `'OPTION'`.
* `data_quality_logs.status`: الزام به یکی از مقادیر `'PASSED'`, `'WARNING'`, یا `'FAILED'`.

### ۲. قیود یکتایی (Unique Constraints)
* `underlying_assets.symbol_fa` و `option_contracts.symbol_fa`: جلوگیری از تکرار نماد.
* `daily_history`: ترکیب یکتای `(instrument_type, instrument_id, trade_date)` مانع ثبت رکورد تکراری برای یک تاریخ در یک نماد می‌شود.
* `ticks_clean`: ترکیب یکتای `(instrument_type, instrument_id, trade_date, sequence_number)` سلامت ردیف معاملات درون‌روزی را تضمین می‌کند.

---

## ۳. استراتژی ایندکس‌گذاری (Indexing Strategy)

به دلیل حجم چند صد هزار الی چند میلیون تیک معاملاتی:
1. **شاخص‌های مرکب زمانی (Composite Time Indexes):**
   * روی `ticks_raw`: ایندکس `ix_raw_ticks_lookup` روی `(instrument_type, instrument_id, trade_date, trade_time)`.
   * روی `ticks_clean`: ایندکس `ix_clean_ticks_ts` روی `(instrument_type, instrument_id, trade_timestamp)`.
   * این شاخص‌ها کوئری‌های بازه‌ای درون‌روزی (Intraday window queries) و عملیات **As-of Join** بین سهم پایه و قراردادهای اختیار را در کسری از میلی‌ثانیه اجرا می‌کنند.
2. **شاخص‌های زنجیره اختیار (Option Chain Indexes):**
   * روی `option_contracts`: ایندکس مرکب `ix_option_ua_type_strike` روی `(underlying_asset_id, contract_type, strike_price)` برای ساخت آنی صفحه زنجیره اختیار (Option Chain Matrix).
   * ایندکس مرکب `ix_option_ua_expiry` روی `(underlying_asset_id, expiration_date)` برای فیلتر سررسیدهای نزدیک و دور.

---

## ۴. استراتژی پارتیشن‌بندی در مقیاس بالا (Partitioning Strategy)

در اسپرینت ۲ و با لود کامل تاریخچه ۲ ساله:
* جداول `ticks_raw` و `ticks_clean` می‌توانند بر اساس محدوده ستون `trade_date` به روش **Declarative Range Partitioning** سالانه یا ماهانه افراز (Partition) شوند (مثلاً `ticks_clean_1402`, `ticks_clean_1403`, `ticks_clean_1404`).
* این کار باعث افزایش چشمگیر سرعت Scan و نگهداری آسان‌تر دیسک (Vacuuming & Dropping Old Shards) می‌شود.

---

## ۵. چرخه انتقال داده (Lifecycle)

$$\text{TSETMC API} \xrightarrow{\text{Raw Dump}} \text{ticks\_raw} \xrightarrow[\text{Validation}]{\text{Deduplicate \& Order}} \text{ticks\_clean} \xrightarrow{\text{As-of Join \& Candles}} \text{REST API / Client}$$

* **لایه خام (Raw Layer):** پاسخ دست‌نخورده بورس ذخیره می‌شود تا در صورت کشف هرگونه باگ در پارسر، بدون خزش مجدد قابل بازتولید باشد.
* **لایه تمیز (Clean Layer):** داده‌های زمانی با منطقه زمانی یکپارچه شده، معاملات باطل‌شده فیلتر و ردیف‌ها کاملاً مرتب زمانی هستند.
* **لایه کیفیت (Quality Layer):** نتایج لاگ‌ها، گپ‌های زمانی بین تیک‌ها و مغایرت‌ها در `data_quality_logs` ماندگار می‌شوند.
