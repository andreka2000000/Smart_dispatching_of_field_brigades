'use client';
import { useState } from 'react';
import { PageHeader } from '@/components/layout/page-header';
import { Modal } from '@/components/ui/modal';
import { staff } from '@/lib/demo-data';
import { config } from '@/shared/config';
export function Admin() {
  const [tab, setTab] = useState('Пользователи');
  const [invite, setInvite] = useState(false);
  return (
    <>
      <PageHeader title="Админ-панель" subtitle="Пользователи, правила SLA и интеграции" />
      <div className="page-content">
        <div className="tabs admin-tabs" role="tablist">
          {['Пользователи', 'Правила SLA', 'Интеграции'].map((t) => (
            <button role="tab" aria-selected={tab === t} key={t} onClick={() => setTab(t)}>
              {t}
            </button>
          ))}
        </div>
        <section className="panel">
          {tab === 'Пользователи' ? (
            <>
              <div className="section-heading">
                <h2>Пользователи</h2>
                <button className="primary-button" onClick={() => setInvite(true)}>
                  Пригласить
                </button>
              </div>
              {config.dataMode === 'mock' ? (
                <div className="table-scroll">
                  <table>
                    <thead>
                      <tr>
                        <th>Сотрудник</th>
                        <th>Специализация</th>
                        <th>Статус</th>
                      </tr>
                    </thead>
                    <tbody>
                      {staff.map((s) => (
                        <tr key={s.id}>
                          <td>{s.name}</td>
                          <td>{s.skills.join(' · ')}</td>
                          <td>{s.status}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <p>Справочник пользователей ещё не подключён.</p>
              )}
            </>
          ) : tab === 'Правила SLA' ? (
            <>
              <h2>Правила SLA</h2>
              <p className="muted">
                Параметры поступают из справочника типов работ. Контрольные сроки и допустимость
                назначений проверяет Go API.
              </p>
              <div className="rule-row">
                <strong>Инсталляция</strong>
                <span>Визит в согласованный слот</span>
              </div>
              <div className="rule-row">
                <strong>Ремонт</strong>
                <span>Контрольный срок по типу работ</span>
              </div>
              <p className="muted">Редактирование правил будет доступно после согласования API.</p>
            </>
          ) : (
            <>
              <h2>Интеграции</h2>
              <div className="rule-row">
                <strong>Go REST API</strong>
                <span>{config.dataMode === 'mock' ? 'Демо-режим' : 'Включён'}</span>
              </div>
              <div className="rule-row">
                <strong>Карта</strong>
                <span>MapLibre · OpenStreetMap</span>
              </div>
              <div className="rule-row">
                <strong>ИИ-помощник</strong>
                <span>Пока не подключён</span>
              </div>
            </>
          )}
        </section>
      </div>
      {invite && (
        <Modal title="Пригласить пользователя" onClose={() => setInvite(false)}>
          <p>Приглашения станут доступны после подключения сервиса пользователей.</p>
          <p className="muted">Письмо не отправлено.</p>
          <button className="primary-button" onClick={() => setInvite(false)}>
            Понятно
          </button>
        </Modal>
      )}
    </>
  );
}
