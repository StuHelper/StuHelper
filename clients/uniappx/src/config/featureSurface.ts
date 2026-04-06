import { translate } from '../i18n'

export interface UniappxFeatureLink {
  title: string
  desc?: string
  icon?: string
  path: string
}

export function getUniappxAppNotice(): string {
  return translate('feature.notice')
}

export function getHomeFeatures(): UniappxFeatureLink[] {
  return [
    {
      title: translate('feature.home.course.title'),
      desc: translate('feature.home.course.desc'),
      icon: '📚',
      path: '/pages/course/index',
    },
    {
      title: translate('feature.home.review.title'),
      desc: translate('feature.home.review.desc'),
      icon: '⭐',
      path: '/pages/review/index',
    },
    {
      title: translate('feature.home.user.title'),
      desc: translate('feature.home.user.desc'),
      icon: '👤',
      path: '/pages/user/index',
    },
  ]
}

export function getUserMenuItems(): UniappxFeatureLink[] {
  return [
    { title: translate('feature.user.reviews'), icon: '📝', path: '/pages/user/reviews' },
    { title: translate('feature.user.votes'), icon: '👍', path: '/pages/user/votes' },
    { title: translate('feature.user.favorites'), icon: '⭐', path: '/pages/user/favorites' },
    {
      title: translate('feature.user.notifications'),
      icon: '🔔',
      path: '/pages/user/notifications',
    },
  ]
}
