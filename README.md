# МаксТелеком — frontend

Next.js frontend проекта «МаксТелеком — порядок там, где раньше был хаос».

## Запуск

Требуется Node.js 24. Установите зависимости командой `npm ci`, скопируйте `.env.example` в `.env.local` и запустите `npm run dev -- --port 3001`.

Сайт: http://127.0.0.1:3001. Для авторизации и рабочих экранов нужен backend из отдельной ветки `backend`, доступный по `API_PROXY_TARGET`. Учётные записи создаются средствами backend; пароли в репозитории не хранятся.

## Проверки и сборка

`npm run check` — типы, ESLint, тесты и production-сборка.

`npm run build` и `npm run start -- --port 3001` — запуск production-версии.

`docker build -f Dockerfile.frontend -t maxtelecom-frontend .` — Docker-сборка. При запуске контейнера задайте `API_PROXY_TARGET` адресом доступного backend и опубликуйте порт 3000.

Описание frontend: [docs/FRONTEND.md](docs/FRONTEND.md).
