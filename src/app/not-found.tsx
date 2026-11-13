import Link from 'next/link';
export default function NotFound() {
  return (
    <div className="page-content">
      <h1>Страница не найдена</h1>
      <Link href="/">Вернуться в диспетчерскую</Link>
    </div>
  );
}
