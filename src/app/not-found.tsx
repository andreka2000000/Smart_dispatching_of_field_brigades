import Link from 'next/link';
export default function NotFound() {
  return (
    <main style={{ padding: '80px 24px', textAlign: 'center' }}>
      <h1>Страница не найдена</h1>
      <p style={{ marginTop: 20 }}>
        <Link href="/">Вернуться на главную</Link>
      </p>
    </main>
  );
}
