'use client';
import { createContext, useContext, useState } from 'react';
import {
  orders as initialOrders,
  staff,
  initialActivity,
  type DemoOrder,
  type OrderStatus,
} from './demo-data';
import { config } from '@/shared/config';

function useDemoState() {
  const [orders, setOrders] = useState(initialOrders);
  const [activity, setActivity] = useState(initialActivity);
  function log(text: string) {
    setActivity((a) => [{ text, time: '11:20' }, ...a]);
  }
  function assign(orderId: string, staffId: string) {
    if (config.dataMode !== 'mock') return;
    const order = orders.find((o) => o.id === orderId),
      person = staff.find((s) => s.id === staffId);
    if (
      !order ||
      !person ||
      person.status === 'Не в сети' ||
      !order.skills.every((skill) => person.skills.includes(skill))
    )
      return;
    setOrders((previous) =>
      previous.map((o) => (o.id === orderId ? { ...o, staffId, status: 'Назначена' } : o)),
    );
    log(`${orderId} назначена: ${person.name}`);
  }
  function updateStatus(id: string, status: OrderStatus) {
    if (config.dataMode !== 'mock') return;
    setOrders((previous) => previous.map((o) => (o.id === id ? { ...o, status } : o)));
    log(`${id}: ${status}`);
  }
  function addOrder(order: DemoOrder) {
    if (config.dataMode === 'mock') {
      setOrders((previous) => [order, ...previous]);
      log(`Создана заявка ${order.id}`);
    }
  }
  return { orders, activity, assign, updateStatus, addOrder };
}
type DemoState = ReturnType<typeof useDemoState>;
const DemoContext = createContext<DemoState | null>(null);
export function DemoProvider({ children }: { children: React.ReactNode }) {
  const state = useDemoState();
  return <DemoContext.Provider value={state}>{children}</DemoContext.Provider>;
}
export function useDemo() {
  const state = useContext(DemoContext);
  if (!state) throw new Error('DemoProvider is missing');
  return state;
}
