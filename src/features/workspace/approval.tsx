'use client';
import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { request } from '@/shared/api/client';
import { useWorkspace } from './context';
import { Modal } from '@/components/ui/modal';
import { fromLocal } from './model';
export function ApprovalDialog({
  id,
  service,
  address,
  preferredDate,
  onClose,
}: {
  id: string;
  service: string;
  address: string;
  preferredDate: string;
  onClose: () => void;
}) {
  const w = useWorkspace();
  const d = w.data;
  const client = useQueryClient();
  const [place, setPlace] = useState('');
  const [work, setWork] = useState('');
  const save = useMutation({
    mutationFn: (input: object) =>
      request('', `/ui/v1/requests/${encodeURIComponent(id)}/approve`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: ['service-requests'] }),
        w.refresh(),
      ]);
      onClose();
    },
  });
  if (!d?.plan)
    return (
      <Modal title="Одобрение обращения" onClose={onClose}>
        <p>Выберите загруженную смену и сначала постройте её план в диспетчерской.</p>
      </Modal>
    );
  const choices = d.reference.access_nodes.flatMap((n) =>
    n.buildings.map((b) => ({ key: `${n.access_node_id}/${b.building_id}`, node: n, building: b })),
  );
  const selected = choices.find((x) => x.key === place);
  const works = d.reference.work_types.filter(
    (t) =>
      t.profiles.includes(service === 'REPAIR' ? 'LTP3' : 'INSTALL') &&
      (!selected ||
        ((t.code !== 'INST_PON_MKD' ||
          (selected.node.technology === 'PON' && selected.building.building_type === 'MKD')) &&
          (t.code !== 'INST_FTTB_MKD' ||
            (selected.node.technology === 'FTTB' && selected.building.building_type === 'MKD')) &&
          (!['INST_PRIVATE_VLS', 'SURVEY_PRIVATE'].includes(t.code) ||
            selected.building.building_type === 'PRIVATE'))),
  );
  return (
    <Modal title="Одобрить и распределить" onClose={onClose}>
      <form
        className="form-stack live-form"
        onSubmit={(e) => {
          e.preventDefault();
          if (!selected) return;
          const f = new FormData(e.currentTarget);
          save.mutate({
            date: d.day.date,
            building_id: selected.building.building_id,
            access_node_id: selected.node.access_node_id,
            work_type_code: work,
            slot: {
              start: fromLocal(String(f.get('start'))),
              end: fromLocal(String(f.get('end'))),
            },
            required_equipment: {
              ONT: Number(f.get('ONT')),
              ROUTER: Number(f.get('ROUTER')),
              STB: Number(f.get('STB')),
            },
            decision_note: String(f.get('note')),
          });
        }}
      >
        <p>
          Адрес из обращения: <strong>{address}</strong>
        </p>
        <p className="muted">
          Рабочая смена: {d.day.date}. Желаемая дата: {preferredDate || 'не указана'}. Согласуйте
          выбранное окно с абонентом.
        </p>
        <label>
          Адрес и технология из справочника
          <select
            required
            value={place}
            onChange={(e) => {
              setPlace(e.target.value);
              setWork('');
            }}
          >
            <option value="">Сопоставьте адрес</option>
            {choices.map((c) => (
              <option key={c.key} value={c.key}>
                {c.building.address} · {c.node.technology}
              </option>
            ))}
          </select>
        </label>
        <label>
          Норматив работ
          <select required value={work} onChange={(e) => setWork(e.target.value)}>
            <option value="">Выберите норматив</option>
            {works.map((t) => (
              <option key={t.code} value={t.code}>
                {t.name} · {t.duration_min} мин
              </option>
            ))}
          </select>
        </label>
        <div className="form-columns">
          <label>
            Начало окна
            <input
              name="start"
              type="datetime-local"
              required
              defaultValue={`${d.day.date}T14:00`}
            />
          </label>
          <label>
            Конец окна
            <input name="end" type="datetime-local" required defaultValue={`${d.day.date}T16:00`} />
          </label>
        </div>
        <div className="form-columns">
          {['ONT', 'ROUTER', 'STB'].map((k) => (
            <label key={k}>
              {k}
              <input name={k} type="number" min="0" max="100" defaultValue={0} required />
            </label>
          ))}
        </div>
        <label>
          Комментарий абоненту
          <textarea
            name="note"
            maxLength={2000}
            rows={2}
            placeholder="Согласованное окно и детали услуги"
          />
        </label>
        <label className="checkbox-label">
          <input type="checkbox" required />
          Окно визита согласовано с абонентом
        </label>
        {save.error && (
          <p className="form-error" role="alert">
            {save.error.message}
          </p>
        )}
        <button className="primary-button" disabled={save.isPending}>
          {save.isPending ? 'Распределяем…' : 'Одобрить и распределить'}
        </button>
        <p className="muted">
          Если свободного исполнителя нет, наряд сохранится без назначения. Повторное одобрение не
          создаёт дубликат.
        </p>
      </form>
    </Modal>
  );
}
