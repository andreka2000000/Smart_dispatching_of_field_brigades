# Frontend Магистраль

Репозиторий: https://github.com/andreka2000000/Smart_dispatching_of_field_brigades

Стек: Next.js App Router, React, TypeScript, TanStack Query, MapLibre и Lucide.
Локальная рабочая ветка — `frontend/setup`, отдельная ветка на GitHub — `frontend`.
Слияние в `main` выполняется отдельно после проверки frontend.
Backend на Go в `backend/` не изменяется.

## Дизайн

Визуальный источник: https://dispatch-two-bay.vercel.app/.
Сохранены тёмная тема, фиолетовый знак, боковая навигация, карточки сводки, три колонки диспетчерской, карточки заявок и мастеров.
Маршруты: `/`, `/technician`, `/admin`.
Исходников Vercel-сайта в GitHub нет, поэтому это локальная реализация по публичному интерфейсу, не экспорт его исходного кода.
Защищённый preview https://dispatch-9evx945cv-ngame062-1133.vercel.app/ требует входа в Vercel.

## Запуск в WebStorm

Требуются Node.js 24 LTS и npm. Откройте корень проекта.

```sh
npm ci
npm run dev
```

Адрес: http://127.0.0.1:3000. В WebStorm используйте npm run configuration для `dev`.

- `npm run check`: TypeScript, ESLint, тесты, production build.
- `npm run build` и `npm start`: production-режим Next.js.
- `npm run generate:api`: типы из `contracts/api_spec.yaml`.
- `npm run format` и `npm run format:check`: Prettier.

Vite и React Router удалены из приложения. Их прежний scaffold сохранён в игнорируемом `work/vite-starter-backup`.

## Структура

```text
src/app/                 layout, providers и страницы Next.js
src/components/layout/   общая навигация и шапка
src/components/ui/       общие элементы интерфейса
src/features/dispatch/   диспетчерская и диалог назначения
src/features/technician/ приложение мастера
src/features/admin/      пользователи, SLA и интеграции
src/features/map/        клиентская карта с lazy loading
src/lib/api/index.ts     публичная граница интеграции с Go
src/lib/demo-context.tsx состояние демо между переходами страниц
src/lib/demo-data.ts     демонстрационные данные публичного сайта
src/entities/plan/       типы плана из OpenAPI
src/shared/              конфигурация, транспорт API, время
contracts/               неизменённый контракт организаторов
```

App Router отвечает за страницы; интерактивные части объявлены Client Components.
MapLibre загружается через `next/dynamic` с `ssr: false` внутри клиентского компонента.
Next.js не содержит алгоритмов планирования, реализации Go API или ключей ИИ.

## Демо

По умолчанию включён `NEXT_PUBLIC_DATA_MODE=mock`.
Работают поиск, фильтры статуса и региона, фильтр по мастеру, открытие карточки из списка и карты, создание тестовой заявки, тестовое назначение, переходы статусов мастера и вкладки админ-панели.
Изменения живут в памяти frontend до перезагрузки. Это не сохранение на backend и не расчёт маршрутов.
Заблокированы назначения с отсутствующими навыками или сотруднику не в сети. Проверка полной допустимости H1–H10 остаётся задачей Go.
Московские данные сохранены для визуального соответствия исходному сайту; при интеграции кейса они заменяются данными Екатеринбурга.
Позиции мастеров и новых тестовых заявок условные, GPS и геокодирование не имитируются.

## Подключение Go

Скопируйте `.env.example` в `.env.local` и задайте `NEXT_PUBLIC_DATA_MODE=api`.
Dev rewrite направляет `/api/*` на Go :8080; другой адрес задаётся через `API_PROXY_TARGET`.
В production используйте прямой HTTPS Go endpoint через `NEXT_PUBLIC_API_BASE_URL`; сервер должен разрешать CORS для домена frontend.
Не направляйте синхронный расчёт до 300 секунд через ограниченный по времени serverless proxy.
`NEXT_PUBLIC_*` доступны браузеру; секреты Nemotron/OpenRouter в них хранить нельзя.

Режим API сейчас выводит реальный GET плана, версию, модельное время и нераспределённые наряды.
Полный UI-адаптер справочников, дня и плана ещё нужен. Мастер и админ-панель не подставляют demo records в режиме API.
Ошибка Go отображается явно. Мутации автоматически не повторяются; запросы принимают AbortSignal.
После POST события нужно обновлять query плана; версию и модельное время задаёт Go.
Бизнес-логика не должна использовать системные часы клиента.

## Карта

MapLibre использует raster-подложку OpenStreetMap с видимой атрибуцией.
Для публичной нагрузки следует выбрать провайдера с подходящими лимитами и соблюдать https://operations.osmfoundation.org/policies/tiles/.
Карта отображает точки; дорожные линии и расчёт маршрутов ещё не подключены.
Время расписания должно поступать из матрицы организаторов, независимо от геометрии дорог.

## Следующие frontend задачи

1. Адаптер справочников, сотрудников, нарядов и плана для существующих UI-компонентов.
2. Загрузка дня и обработка событий через согласованный Go API.
3. Дорожная геометрия и синхронизация выделения маршрута.
4. Гант по Stop: DEPOT_START, VISIT, BREAK, RESTOCK, DEPOT_END.
5. Отдельные списки Unassigned, LATE_RISK, ПГП и сравнение версий.
6. Контекстный помощник после основного сценария.

## Зависимости

При проверке runtime-зависимости не имеют известных audit findings. В dev-цепочке ESLint/typescript-eslint остаются moderate advisories minimatch.
Обычный audit fix не устраняет их; принудительный downgrade не применён.
