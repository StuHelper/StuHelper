const ID_HEAD_LENGTH = 8;
const ID_TAIL_LENGTH = 6;

const dateTimeFormatter = new Intl.DateTimeFormat('zh-CN', {
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  month: '2-digit',
  year: 'numeric',
});

export function compactID(value?: null | string) {
  if (!value) return '—';
  if (value.length <= ID_HEAD_LENGTH + ID_TAIL_LENGTH + 1) return value;
  return `${value.slice(0, ID_HEAD_LENGTH)}...${value.slice(-ID_TAIL_LENGTH)}`;
}

export function formatAdminDateTime(value?: null | string) {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return dateTimeFormatter.format(date);
}

export function formatNullableText(value?: null | string) {
  const normalized = value?.trim();
  return normalized || '—';
}
