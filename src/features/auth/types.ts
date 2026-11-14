export type Role = 'DISPATCHER' | 'TECHNICIAN' | 'SUBSCRIBER';
export type User = {
  id: string;
  username: string;
  name: string;
  role: Role;
  staff_id?: string;
  disabled: boolean;
};
export const roleNames: Record<Role, string> = {
  DISPATCHER: 'Диспетчер · управление',
  TECHNICIAN: 'Исполнитель',
  SUBSCRIBER: 'Абонент',
};
export const roleHome: Record<Role, string> = {
  DISPATCHER: '/app/dispatch',
  TECHNICIAN: '/app/technician',
  SUBSCRIBER: '/app/subscriber',
};
