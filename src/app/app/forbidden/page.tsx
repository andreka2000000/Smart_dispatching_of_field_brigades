import Link from 'next/link';
import { requireUser } from '@/features/auth/server';
import { roleHome } from '@/features/auth/types';
export default async function Page() {
  const user = await requireUser();
  return (
    <div className="page-content">
      <section className="live-panel state-panel">
        <span className="eyebrow">ПЕРСОНАЛЬНОЕ ПРОСТРАНСТВО</span>
        <h1>Этот раздел доступен другой роли</h1>
        <p>Ваши инструменты и данные находятся в вашем рабочем пространстве.</p>
        <Link className="primary-button" href={roleHome[user.role]}>
          Перейти к своей работе
        </Link>
      </section>
    </div>
  );
}
