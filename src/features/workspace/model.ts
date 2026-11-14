import type { Snapshot, WorkOrder, Staff, Stop } from './types';
import { formatModelTime } from '@/shared/time';
export const eventNames: Record<string, string> = {
  NEW_ORDER: 'Новая заявка',
  ORDER_CANCELLED: 'Отмена заявки',
  VISIT_DELAYED: 'Задержка визита',
  ORDER_TRANSFERRED: 'Передача в подразделение',
  STAFF_UNAVAILABLE: 'Недоступность мастера',
  TIME_TICK: 'Обновление времени',
};
export const stopNames: Record<string, string> = {
  DEPOT_START: 'Выезд из СЦ',
  VISIT: 'Визит',
  BREAK: 'Перерыв',
  RESTOCK: 'Пополнение',
  DEPOT_END: 'Возвращение в СЦ',
};
export const profileNames: Record<string, string> = {
  INSTALL: 'Подключение',
  LTP3: 'Восстановление связи',
};
export const time = formatModelTime;
export function countText(count: number, one: string, few: string, many: string) {
  const mod100 = count % 100;
  const mod10 = count % 10;
  const word =
    mod100 >= 11 && mod100 <= 14 ? many : mod10 === 1 ? one : mod10 >= 2 && mod10 <= 4 ? few : many;
  return `${count} ${word}`;
}
export function localInput(value: string) {
  return new Intl.DateTimeFormat('sv-SE', {
    timeZone: 'Asia/Yekaterinburg',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hourCycle: 'h23',
  })
    .format(new Date(value))
    .replace(' ', 'T');
}
export function fromLocal(value: string) {
  return `${value.length === 16 ? value + ':00' : value}+05:00`;
}
export function staffName(s: Staff) {
  return s.display_name || s.staff_id;
}
export function initials(name: string) {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map((n) => n[0])
    .join('')
    .toUpperCase();
}
export type ViewOrder = {
  source: WorkOrder;
  stop?: Stop;
  staffId?: string;
  status: string;
  tone: string;
  title: string;
  groupId?: string;
  reason?: string;
};
export function ordersView(data: Snapshot): ViewOrder[] {
  const visits = new Map<string, { stop: Stop; staffId: string }>();
  for (const r of data.plan?.routes || [])
    for (const stop of r.stops)
      if (stop.type === 'VISIT' && stop.order_id)
        visits.set(stop.order_id, { stop, staffId: r.staff_id });
  const groups = new Map<string, string>();
  for (const g of data.plan?.pgp_groups || [])
    for (const id of g.order_ids) groups.set(id, g.group_id);
  const cancelled = new Set(data.plan?.cancelled.map((o) => o.order_id));
  const unassigned = new Map(data.plan?.unassigned.map((o) => [o.order_id, o.reason_text]));
  const now = Date.parse(data.day.now);
  return data.day.orders.map((source) => {
    const visit = visits.get(source.order_id);
    const groupId = groups.get(source.order_id);
    let status = 'Без назначения',
      tone = 'amber';
    if (cancelled.has(source.order_id)) {
      status = 'Отменена';
      tone = 'muted';
    } else if (groupId) {
      status = 'ПГП';
      tone = 'violet';
    } else if (visit) {
      if (Date.parse(visit.stop.end) <= now) {
        status = 'Ожидает подтверждения';
        tone = 'amber';
      } else if (Date.parse(visit.stop.start) <= now) {
        status = 'Окно визита идёт';
        tone = 'violet';
      } else {
        status = 'Назначена';
        tone = 'blue';
      }
      const fact = data.facts?.find((f) => f.order_id === source.order_id);
      if (fact?.status === 'IN_PROGRESS') {
        status = 'На объекте';
        tone = 'violet';
      }
      if (fact?.status === 'COMPLETED') {
        status = 'Выполнена';
        tone = 'green';
      }
    }
    return {
      source,
      ...visit,
      status,
      tone,
      groupId,
      title:
        data.reference.work_types.find((w) => w.code === source.work_type_code)?.name ||
        source.work_type_code,
      reason: unassigned.get(source.order_id),
    };
  });
}
export function locationPoints(data: Snapshot): Record<string, [number, number]> {
  const out: Record<string, [number, number]> = {};
  for (const c of data.reference.service_centers) out[c.service_center_id] = [c.lon, c.lat];
  for (const n of data.reference.access_nodes)
    for (const b of n.buildings) out[b.building_id] = [b.lon, b.lat];
  return out;
}
export function errorText(error: unknown) {
  const e = error as { code?: string; message?: string };
  const errors: Record<string, string> = {
    NOT_FOUND: 'Данные недоступны. Обновите страницу или обратитесь к администратору.',
    BACKEND_UNAVAILABLE: 'Нет связи с сервисом. Проверьте, что Docker запущен.',
    EVENTS_ALREADY_ACCEPTED:
      'День уже принимает события. План обновляется через изменения, а не первоначальный расчёт.',
    EVENT_OUT_OF_ORDER: 'Время события меньше текущего времени смены. Обновите данные.',
    PLANNING_INPUT_CHANGED: 'Данные сменились во время расчёта. Обновите экран и повторите.',
    EVENT_DEADLINE:
      'Операция не подтвердилась вовремя. Повторите: сохранённый идентификатор защитит от дублирования.',
    EVENT_INFEASIBLE: 'Изменение противоречит уже начатым работам или ограничениям смены.',
    PLAN_NOT_BUILT: 'Сначала постройте первоначальный план.',
    ROUTING_UNAVAILABLE:
      'Локальная маршрутизация недоступна. Запустите OSRM по инструкции настройки дорожного графа.',
    ROAD_ROUTE_NOT_FOUND:
      'Не найден автомобильный путь. Проверьте координаты остановок и границы загруженного графа.',
    ROUTE_REVISION_CHANGED: 'План смены обновился. Обновите данные для расчёта дороги.',
    ROUTE_LOCATION_UNKNOWN: 'В справочнике отсутствуют координаты остановки.',
  };
  return errors[e?.code || ''] || e?.message || 'Не удалось выполнить запрос.';
}
