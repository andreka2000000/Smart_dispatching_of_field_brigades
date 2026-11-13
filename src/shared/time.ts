export function formatModelTime(asOf: string): string {
  // Presentation zone of the case, independent of the device clock and timezone.
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Asia/Yekaterinburg',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).format(new Date(asOf));
}
