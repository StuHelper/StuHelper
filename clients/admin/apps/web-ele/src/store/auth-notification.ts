import { ElNotification } from 'element-plus';

type AdminAuthNotificationOptions = Parameters<typeof ElNotification>[0];

export function showAuthNotification(options: AdminAuthNotificationOptions) {
  ElNotification(options);
}
