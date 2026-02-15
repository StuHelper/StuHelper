/**
 * 评课模块文本 - 中文
 */
export default {
  title: '课程测评',
  courseList: '课程列表',
  courseDetail: '课程详情',
  latestReviews: '最新测评',
  postReview: '发布测评',
  searchPlaceholder: '搜索课程名称或教师...',
  filters: {
    department: '院系',
    all: '全部',
    latest: '最新',
    hottest: '最热',
    featured: '精选',
    following: '关注的课程'
  },
  course: {
    teacher: '授课教师',
    reviewCount: '{count} 条测评',
    noReviews: '暂无测评',
    credits: '{n} 学分',
    overview: '概览',
    reviews: '测评',
    stats: '统计',
    teachers: '教师',
    basedOnReviews: '基于 {count} 条测评',
    reviewUnit: '条测评',
    shareExperience: '分享你对这门课的体验...',
    noTeacherData: '暂无教师数据',
    creditsBadge: '{n}学分',
    reviewCountBadge: '{count}评'
  },
  review: {
    anonymous: '匿名用户',
    like: '有用',
    dislike: '没用',
    report: '举报',
    noReplies: '暂无回复',
    voteFailed: '操作失败，请重试',
    replySuccess: '回复成功',
    replyFailed: '回复失败',
    replyLoadFailed: '加载回复失败',
    deleteFailed: '删除失败',
    commentBtn: '查看回复',
    expandContent: '展开全文'
  },
  post: {
    selectCourse: '选择课程',
    selectTeacher: '选择教师',
    title: '标题（可选）',
    titleLabel: '测评标题',
    titleRequired: '标题',
    titlePlaceholder: '一句话总结您的观点，例如：绝世好课',
    titleMissing: '请填写标题',
    content: '测评内容',
    contentPlaceholder: '分享你的课程体验...',
    contentLabel: '测评内容',
    detailedReview: '详细评价',
    contentMinError: '请至少填写 {min} 个字的评价内容',
    contentMinErrorNoTemplate: '请至少填写 {min} 个字的评价内容（不含模板文字）',
    ratingMissing: '请为每个维度打分',
    grade: '成绩',
    gradeLabel: '成绩',
    gradeOptional: '选填',
    gradePlaceholder: '例如：90、A+、优秀',
    templateListening: '课程听感:',
    templateWorkload: '作业/任务量:',
    templateExam: '关于考试:',
    rating: '评分',
    submit: '发布',
    success: '测评发布成功',
    failed: '发布失败，请重试',
    searchCourse: '搜索课程...',
    searchCourseLabel: '搜索课程'
  },
  rating: {
    overall: '综合评分',
    content: '课程内容',
    teaching: '教学质量',
    workload: '课业负担',
    grading: '给分情况',
    veryBad: '很差',
    bad: '较差',
    average: '一般',
    good: '较好',
    excellent: '很好',
    countSuffix: '{count}人评价',
    level5: '强烈推荐',
    level4: '推荐',
    level3: '一般',
    level2: '不推荐',
    level1: '强烈不推荐',
    unknown: '未知',
    face5: '超赞',
    face4: '不错',
    face3: '一般',
    face2: '较差',
    face1: '很差'
  },
  chart: {
    title: '评分统计',
    selectTerm: '选择学期',
    overall: '总体',
    emptyTitle: '暂无评分数据',
    emptyHint: '成为第一个评价者吧',
    radarAria: '课程评分雷达图'
  },
  reply: {
    count: '{count} 条回复',
    placeholder: '写下你的回复...',
    emptyTitle: '暂无回复',
    emptyDesc: '成为第一个回复的人',
    sendFailed: '发送失败，请稍后重试',
    deleteConfirm: '确定要删除这条回复吗？',
    deleteFailed: '删除失败，请稍后重试',
    loadFailed: '加载回复失败',
    inputLabel: '回复内容',
    toggleExpand: '展开回复',
    toggleCollapse: '收起回复'
  },
  vote: {
    like: '点赞',
    dislike: '踩'
  },
  favorite: {
    favorited: '已收藏',
    add: '收藏',
    failed: '操作失败'
  },
  quality: {
    warningTitle: '内容质量提示',
    errorTitle: '内容不符合要求',
    dismiss: '忽略并继续'
  },
  draft: {
    saving: '保存中...',
    saved: '草稿已保存',
    restore: '恢复草稿',
    savedAndLogin: '当前未登录，内容已暂存，即将跳转登录',
    confirmSave: '是否暂存当前内容为草稿？',
    restored: '已恢复草稿',
    saveFailed: '草稿保存失败',
    redirectCountdown: '{n}秒后跳转登录...',
    discard: '不保存',
    saveDraft: '保存草稿'
  },
  validation: {
    contentTooShort: '内容至少需要 {min} 个字符',
    contentTooLong: '内容不能超过 {max} 个字符',
    titleTooLong: '标题不能超过 {max} 个字符',
    replyTooShort: '回复至少需要 {min} 个字符',
    charCount: '{current}/{max}'
  },
  topBar: {
    title: '课程测评',
    searchPlaceholder: '搜索课程...',
    writeReview: '写测评',
    recentSearches: '最近搜索'
  },
  hub: {
    postReview: '发布测评',
    stats: '{courses} 门课程, {reviews} 条测评',
    hotRankings: '热门排行',
    communityStats: '社区统计',
    totalCourses: '课程总数',
    totalReviews: '测评总数',
    todayReviews: '今日新增',
    feedEmpty: '暂无测评，来发布第一条吧',
    loadFailed: '加载测评失败，请稍后重试',
    retry: '重试',
    loading: '正在加载测评'
  },
  stats: {
    insufficient: '数据较少',
    reviewCount: '{count} 条测评',
    insufficientHint: '评价人数不足，暂不展示详细数据',
    noData: '暂无统计数据'
  },
  ratingEmoji: {
    recommendation: '推荐度',
    content_quality: '内容',
    workload: '工作量',
    grading: '给分',
    fallback: '评分'
  },
  filter: {
    teacher: '教师筛选',
    all: '全部'
  }
}
