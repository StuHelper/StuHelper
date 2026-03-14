-- StuHelper 种子数据（开发环境）
-- 使用方法: psql -U stuhelper -d stuhelper -f seed.sql
-- 包含: 院系、教师、课程、用户、测评、回复、投票
-- 注意: 需要先执行 init.sql 创建表结构

BEGIN;

-- ============================================
-- 1. 院系数据
-- ============================================
INSERT INTO departments (id, name, short_name, category, sort_order) VALUES
    (1, '计算机学院', '计算机', '工科', 1),
    (2, '数学科学学院', '数学', '理科', 2),
    (3, '外国语学院', '外语', '文科', 3),
    (4, '经济管理学院', '经管', '文科', 4),
    (5, '电子信息工程学院', '电信', '工科', 5),
    (6, '马克思主义学院', '马院', '思政', 6),
    (7, '体育部', '体育', '体育', 7),
    (8, '新媒体与艺术设计学院', '艺术', '文科', 8)
ON CONFLICT (id) DO NOTHING;

SELECT setval('departments_id_seq', 8, true);

-- ============================================
-- 2. 教师数据
-- ============================================
INSERT INTO teachers (id, name, department_id) VALUES
    (1,  '张明远', 1),
    (2,  '李思涵', 1),
    (3,  '王建国', 1),
    (4,  '陈晓峰', 2),
    (5,  '刘雅琴', 2),
    (6,  '赵文博', 3),
    (7,  '孙丽华', 3),
    (8,  '周志强', 4),
    (9,  '吴海燕', 4),
    (10, '郑大伟', 5),
    (11, '黄晓明', 5),
    (12, '林小红', 6),
    (13, '杨健', 7),
    (14, '何美玲', 8),
    (15, '马天宇', 1)
ON CONFLICT (id) DO NOTHING;

SELECT setval('teachers_id_seq', 15, true);

-- ============================================
-- 3. 课程数据
-- ============================================
INSERT INTO courses (id, name, code, department_id, credits, category) VALUES
    -- 通识课
    (1,  '大学计算机基础', 'CS1001', 1, 2.0, '通识'),
    (2,  '信息技术导论', 'CS1002', 1, 2.0, '通识'),
    -- 计算机专业课
    (3,  '数据结构与算法', 'CS2001', 1, 4.0, ''),
    (4,  'Java程序设计', 'CS2002', 1, 3.5, ''),
    (5,  '操作系统原理', 'CS3001', 1, 4.0, ''),
    (6,  '计算机网络', 'CS3002', 1, 3.5, ''),
    (7,  '数据库系统概论', 'CS2003', 1, 3.5, ''),
    -- 数学课
    (8,  '高等数学A', 'MA1001', 2, 5.0, ''),
    (9,  '线性代数', 'MA1002', 2, 3.0, ''),
    (10, '概率论与数理统计', 'MA2001', 2, 3.5, ''),
    -- 英语课
    (11, '大学英语(一)', 'EN1001', 3, 3.0, '英语'),
    (12, '大学英语(二)', 'EN1002', 3, 3.0, '英语'),
    (13, '英语口语实训', 'EN2001', 3, 2.0, '英语'),
    -- 经管课
    (14, '微观经济学', 'EC2001', 4, 3.0, ''),
    (15, '管理学原理', 'EC2002', 4, 3.0, ''),
    -- 电信课
    (16, '电路分析基础', 'EE2001', 5, 3.5, ''),
    (17, '数字电子技术', 'EE2002', 5, 3.5, ''),
    -- 思政课
    (18, '毛泽东思想和中国特色社会主义理论体系概论', 'MX1001', 6, 3.0, '思政'),
    (19, '思想道德与法治', 'MX1002', 6, 2.5, '思政'),
    -- 体育课
    (20, '大学体育-篮球', 'PE1001', 7, 1.0, '体育'),
    (21, '大学体育-羽毛球', 'PE1002', 7, 1.0, '体育'),
    -- 艺术课
    (22, '设计色彩', 'AR2001', 8, 2.0, ''),
    (23, 'UI/UX设计基础', 'AR2002', 8, 2.5, '')
ON CONFLICT (id) DO NOTHING;

SELECT setval('courses_id_seq', 23, true);

-- ============================================
-- 4. 用户数据
-- ============================================
INSERT INTO users (id, external_id, username, email) VALUES
    (1, 'casdoor_user_001', '匿名用户A', 'user_a@example.com'),
    (2, 'casdoor_user_002', '匿名用户B', 'user_b@example.com'),
    (3, 'casdoor_user_003', '匿名用户C', 'user_c@example.com'),
    (4, 'casdoor_user_004', '匿名用户D', 'user_d@example.com'),
    (5, 'casdoor_user_005', '匿名用户E', 'user_e@example.com')
ON CONFLICT (id) DO NOTHING;

SELECT setval('users_id_seq', 5, true);

-- ============================================
-- 5. 测评数据
-- ============================================
-- user_hash 使用固定的测试哈希值
-- ratings JSONB 键对应 rating_dimensions 中的 key: difficulty, workload, usefulness, teaching, grading

-- 数据结构与算法 (course_id=3, 张明远)
INSERT INTO reviews (id, course_id, teacher_id, term_id, user_hash, title, content, grade, ratings, avg_rating, like_count, reply_count, status, created_at) VALUES
(
    '019462a0-0001-7000-8000-000000000001', 3, 1, '2025-2',
    'hash_user_a_00000000000000000000000000000001',
    '张老师讲得很清楚',
    '数据结构这门课张老师讲得非常好，PPT做得很用心，每个算法都会用动画演示。期末考试难度适中，平时作业量不大但需要认真完成。推荐选张老师的课。',
    'A', '{"difficulty": 4, "workload": 3, "usefulness": 5, "teaching": 5, "grading": 4}',
    4.20, 5, 2, 'published', NOW() - INTERVAL '30 days'
),
(
    '019462a0-0002-7000-8000-000000000002', 3, 1, '2025-2',
    'hash_user_b_00000000000000000000000000000002',
    '有一定难度但收获很大',
    '课程内容比较硬核，树和图那部分需要花时间理解。张老师会在课后答疑，态度很好。给分还行，认真学的话拿A-没问题。',
    'A-', '{"difficulty": 4, "workload": 4, "usefulness": 5, "teaching": 4, "grading": 4}',
    4.20, 3, 1, 'published', NOW() - INTERVAL '25 days'
),

-- Java程序设计 (course_id=4, 李思涵)
(
    '019462a0-0003-7000-8000-000000000003', 4, 2, '2025-2',
    'hash_user_c_00000000000000000000000000000003',
    '适合零基础入门',
    '李老师的Java课非常适合编程入门，从变量讲起，循序渐进。实验课有助教辅导，不用担心跟不上。期末项目是做一个小系统，工作量不算大。',
    'A', '{"difficulty": 2, "workload": 3, "usefulness": 4, "teaching": 5, "grading": 5}',
    3.80, 8, 1, 'published', NOW() - INTERVAL '20 days'
),

-- 高等数学A (course_id=8, 陈晓峰)
(
    '019462a0-0004-7000-8000-000000000004', 8, 4, '2025-2',
    'hash_user_a_00000000000000000000000000000001',
    '高数噩梦',
    '陈老师讲课速度很快，板书也比较潦草。内容本身就难，老师讲得又快，课后需要花大量时间自学。给分倒是还可以，会捞一下。',
    'B+', '{"difficulty": 5, "workload": 5, "usefulness": 4, "teaching": 3, "grading": 4}',
    4.20, 12, 2, 'published', NOW() - INTERVAL '45 days'
),
(
    '019462a0-0005-7000-8000-000000000005', 8, 4, '2025-2',
    'hash_user_d_00000000000000000000000000000004',
    '没那么夸张',
    '我觉得陈老师讲得还行，可能需要一定的数学基础。建议提前预习，课后多刷题。期末考试有原题，好好复习问题不大。',
    'A-', '{"difficulty": 4, "workload": 4, "usefulness": 5, "teaching": 4, "grading": 4}',
    4.20, 2, 0, 'published', NOW() - INTERVAL '40 days'
),

-- 大学英语(一) (course_id=11, 赵文博)
(
    '019462a0-0006-7000-8000-000000000006', 11, 6, '2025-2',
    'hash_user_b_00000000000000000000000000000002',
    '轻松愉快的英语课',
    '赵老师上课很有趣，经常放英文电影片段讨论。平时分给得很高，期末考试也不难。想轻松拿高分的同学推荐选。',
    'A', '{"difficulty": 2, "workload": 2, "usefulness": 3, "teaching": 4, "grading": 5}',
    3.20, 6, 0, 'published', NOW() - INTERVAL '35 days'
),

-- 毛概 (course_id=18, 林小红)
(
    '019462a0-0007-7000-8000-000000000007', 18, 12, '2025-2',
    'hash_user_c_00000000000000000000000000000003',
    '比想象中有意思',
    '林老师讲课不照本宣科，会结合时事案例分析，课堂讨论也很活跃。期末写论文，给分宽松。思政课能上成这样已经很不错了。',
    'A', '{"difficulty": 2, "workload": 2, "usefulness": 3, "teaching": 4, "grading": 5}',
    3.20, 4, 1, 'published', NOW() - INTERVAL '15 days'
),

-- 大学体育-篮球 (course_id=20, 杨健)
(
    '019462a0-0008-7000-8000-000000000008', 20, 13, '2025-2',
    'hash_user_e_00000000000000000000000000000005',
    '杨老师太严格了',
    '体测标准卡得很死，迟到一次扣5分平时分。不过篮球技术确实能学到东西，期末考运球上篮和投篮。建议有一定基础的同学选。',
    'B', '{"difficulty": 4, "workload": 3, "usefulness": 4, "teaching": 3, "grading": 2}',
    3.20, 1, 0, 'published', NOW() - INTERVAL '10 days'
),

-- 操作系统原理 (course_id=5, 王建国)
(
    '019462a0-0009-7000-8000-000000000009', 5, 3, '2025-1',
    'hash_user_d_00000000000000000000000000000004',
    '硬核但值得',
    '王老师是学院的老教授，讲课风格偏传统但内容扎实。进程调度、内存管理这些概念讲得很透彻。实验用C写一个简单的文件系统，有挑战性但很有成就感。',
    'A-', '{"difficulty": 5, "workload": 4, "usefulness": 5, "teaching": 4, "grading": 4}',
    4.40, 7, 1, 'published', NOW() - INTERVAL '50 days'
),

-- 计算机网络 (course_id=6, 马天宇)
(
    '019462a0-0010-7000-8000-000000000010', 6, 15, '2025-2',
    'hash_user_a_00000000000000000000000000000001',
    '年轻老师，讲得不错',
    '马老师是新来的老师，备课很认真，Wireshark抓包实验很有意思。就是有时候会拖堂。期末开卷考，难度不大。',
    'A', '{"difficulty": 3, "workload": 3, "usefulness": 4, "teaching": 4, "grading": 5}',
    3.80, 3, 0, 'published', NOW() - INTERVAL '8 days'
),

-- UI/UX设计基础 (course_id=23, 何美玲)
(
    '019462a0-0011-7000-8000-000000000011', 23, 14, '2025-2',
    'hash_user_e_00000000000000000000000000000005',
    '设计入门好课',
    '何老师本身是设计师出身，作品很厉害。课程从Figma基础操作教起，到最后能独立完成一个App的UI设计。作业量偏大但能学到真东西。',
    'A-', '{"difficulty": 3, "workload": 4, "usefulness": 5, "teaching": 5, "grading": 4}',
    4.20, 5, 0, 'published', NOW() - INTERVAL '5 days'
),

-- 数据库系统概论 (course_id=7, 李思涵)
(
    '019462a0-0012-7000-8000-000000000012', 7, 2, '2025-1',
    'hash_user_d_00000000000000000000000000000004',
    '实用性很强',
    '李老师的数据库课理论和实践结合得很好，SQL写得多了自然就熟练了。期末项目是设计一个完整的数据库系统，可以写进简历。给分中规中矩。',
    'B+', '{"difficulty": 3, "workload": 4, "usefulness": 5, "teaching": 4, "grading": 3}',
    3.80, 4, 0, 'published', NOW() - INTERVAL '3 days'
)
ON CONFLICT (id) DO NOTHING;

-- 更新课程的 review_count
UPDATE courses SET review_count = sub.cnt
FROM (SELECT course_id, COUNT(*) AS cnt FROM reviews WHERE status = 'published' GROUP BY course_id) sub
WHERE courses.id = sub.course_id;

-- ============================================
-- 6. 回复数据
-- ============================================
INSERT INTO review_replies (id, review_id, parent_id, user_hash, content, like_count, created_at) VALUES
-- 数据结构第一条测评的回复
(
    '019462b0-0001-7000-8000-000000000001',
    '019462a0-0001-7000-8000-000000000001', NULL,
    'hash_user_c_00000000000000000000000000000003',
    '同意，张老师的动画演示确实很直观，链表那节课印象深刻。',
    2, NOW() - INTERVAL '28 days'
),
(
    '019462b0-0002-7000-8000-000000000002',
    '019462a0-0001-7000-8000-000000000001',
    '019462b0-0001-7000-8000-000000000001',
    'hash_user_a_00000000000000000000000000000001',
    '对，红黑树那个动画也做得很好，一下就理解了旋转操作。',
    1, NOW() - INTERVAL '27 days'
),
-- 数据结构第二条测评的回复
(
    '019462b0-0003-7000-8000-000000000003',
    '019462a0-0002-7000-8000-000000000002', NULL,
    'hash_user_e_00000000000000000000000000000005',
    '图论部分确实难，建议配合B站视频一起看。',
    3, NOW() - INTERVAL '23 days'
),
-- 高数测评的回复
(
    '019462b0-0004-7000-8000-000000000004',
    '019462a0-0004-7000-8000-000000000004', NULL,
    'hash_user_b_00000000000000000000000000000002',
    '哈哈我也觉得板书看不清，建议坐前三排。',
    5, NOW() - INTERVAL '43 days'
),
(
    '019462b0-0005-7000-8000-000000000005',
    '019462a0-0004-7000-8000-000000000004',
    '019462b0-0004-7000-8000-000000000004',
    'hash_user_d_00000000000000000000000000000004',
    '前三排也看不清（笑哭）',
    8, NOW() - INTERVAL '42 days'
),
-- 毛概测评的回复
(
    '019462b0-0006-7000-8000-000000000006',
    '019462a0-0007-7000-8000-000000000007', NULL,
    'hash_user_a_00000000000000000000000000000001',
    '林老师确实不错，上学期选的她的课，论文给了90+。',
    2, NOW() - INTERVAL '14 days'
),
-- Java测评的回复
(
    '019462b0-0007-7000-8000-000000000007',
    '019462a0-0003-7000-8000-000000000003', NULL,
    'hash_user_d_00000000000000000000000000000004',
    '期末项目可以组队吗？',
    0, NOW() - INTERVAL '18 days'
),
-- 操作系统测评的回复
(
    '019462b0-0008-7000-8000-000000000008',
    '019462a0-0009-7000-8000-000000000009', NULL,
    'hash_user_a_00000000000000000000000000000001',
    '文件系统实验确实有挑战，建议提前两周开始写。',
    4, NOW() - INTERVAL '48 days'
)
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 7. 投票数据
-- ============================================
INSERT INTO review_votes (id, review_id, user_hash, vote_type) VALUES
    ('019462c0-0001-7000-8000-000000000001', '019462a0-0001-7000-8000-000000000001', 'hash_user_b_00000000000000000000000000000002', 'like'),
    ('019462c0-0002-7000-8000-000000000002', '019462a0-0001-7000-8000-000000000001', 'hash_user_c_00000000000000000000000000000003', 'like'),
    ('019462c0-0003-7000-8000-000000000003', '019462a0-0001-7000-8000-000000000001', 'hash_user_d_00000000000000000000000000000004', 'like'),
    ('019462c0-0004-7000-8000-000000000004', '019462a0-0003-7000-8000-000000000003', 'hash_user_a_00000000000000000000000000000001', 'like'),
    ('019462c0-0005-7000-8000-000000000005', '019462a0-0003-7000-8000-000000000003', 'hash_user_b_00000000000000000000000000000002', 'like'),
    ('019462c0-0006-7000-8000-000000000006', '019462a0-0003-7000-8000-000000000003', 'hash_user_d_00000000000000000000000000000004', 'like'),
    ('019462c0-0007-7000-8000-000000000007', '019462a0-0004-7000-8000-000000000004', 'hash_user_b_00000000000000000000000000000002', 'like'),
    ('019462c0-0008-7000-8000-000000000008', '019462a0-0004-7000-8000-000000000004', 'hash_user_c_00000000000000000000000000000003', 'like'),
    ('019462c0-0009-7000-8000-000000000009', '019462a0-0004-7000-8000-000000000004', 'hash_user_e_00000000000000000000000000000005', 'like'),
    ('019462c0-0010-7000-8000-000000000010', '019462a0-0009-7000-8000-000000000009', 'hash_user_a_00000000000000000000000000000001', 'like'),
    ('019462c0-0011-7000-8000-000000000011', '019462a0-0009-7000-8000-000000000009', 'hash_user_b_00000000000000000000000000000002', 'like'),
    ('019462c0-0012-7000-8000-000000000012', '019462a0-0009-7000-8000-000000000009', 'hash_user_c_00000000000000000000000000000003', 'like'),
    ('019462c0-0013-7000-8000-000000000013', '019462a0-0006-7000-8000-000000000006', 'hash_user_a_00000000000000000000000000000001', 'like'),
    ('019462c0-0014-7000-8000-000000000014', '019462a0-0006-7000-8000-000000000006', 'hash_user_d_00000000000000000000000000000004', 'like'),
    ('019462c0-0015-7000-8000-000000000015', '019462a0-0008-7000-8000-000000000008', 'hash_user_a_00000000000000000000000000000001', 'dislike'),
    ('019462c0-0016-7000-8000-000000000016', '019462a0-0011-7000-8000-000000000011', 'hash_user_c_00000000000000000000000000000003', 'like'),
    ('019462c0-0017-7000-8000-000000000017', '019462a0-0011-7000-8000-000000000011', 'hash_user_a_00000000000000000000000000000001', 'like')
ON CONFLICT (review_id, user_hash) DO NOTHING;

-- ============================================
-- 8. 课程评分统计（从 reviews 自动聚合）
-- ============================================
DELETE FROM course_rating_stats;

WITH expanded AS (
    SELECT r.course_id, r.term_id, d.key AS dimension_key, d.value::int AS rating_value
    FROM reviews r
    CROSS JOIN LATERAL jsonb_each_text(r.ratings) AS d(key, value)
    WHERE r.status = 'published'
),
dist_per_term AS (
    SELECT course_id, term_id, dimension_key,
        ROUND(AVG(rating_value), 2) AS avg_rating,
        COUNT(*) AS rating_count,
        jsonb_build_object(
            '1', COUNT(*) FILTER (WHERE rating_value = 1),
            '2', COUNT(*) FILTER (WHERE rating_value = 2),
            '3', COUNT(*) FILTER (WHERE rating_value = 3),
            '4', COUNT(*) FILTER (WHERE rating_value = 4),
            '5', COUNT(*) FILTER (WHERE rating_value = 5)
        ) AS rating_dist
    FROM expanded GROUP BY course_id, term_id, dimension_key
),
dist_overall AS (
    SELECT course_id, NULL::varchar(20) AS term_id, dimension_key,
        ROUND(AVG(rating_value), 2) AS avg_rating,
        COUNT(*) AS rating_count,
        jsonb_build_object(
            '1', COUNT(*) FILTER (WHERE rating_value = 1),
            '2', COUNT(*) FILTER (WHERE rating_value = 2),
            '3', COUNT(*) FILTER (WHERE rating_value = 3),
            '4', COUNT(*) FILTER (WHERE rating_value = 4),
            '5', COUNT(*) FILTER (WHERE rating_value = 5)
        ) AS rating_dist
    FROM expanded GROUP BY course_id, dimension_key
)
INSERT INTO course_rating_stats (id, course_id, term_id, dimension_key, avg_rating, rating_count, rating_dist)
SELECT gen_random_uuid()::varchar, course_id, term_id, dimension_key, avg_rating, rating_count, rating_dist
FROM dist_per_term
UNION ALL
SELECT gen_random_uuid()::varchar, course_id, term_id, dimension_key, avg_rating, rating_count, rating_dist
FROM dist_overall;

-- ============================================
-- 9. 教师评分统计（从 reviews 自动聚合）
-- ============================================
DELETE FROM teacher_rating_stats;

WITH expanded AS (
    SELECT r.teacher_id, r.term_id, d.key AS dimension_key, d.value::int AS rating_value
    FROM reviews r
    CROSS JOIN LATERAL jsonb_each_text(r.ratings) AS d(key, value)
    WHERE r.status = 'published' AND r.teacher_id IS NOT NULL
),
dist_per_term AS (
    SELECT teacher_id, term_id, dimension_key,
        ROUND(AVG(rating_value), 2) AS avg_rating,
        COUNT(*) AS rating_count,
        jsonb_build_object(
            '1', COUNT(*) FILTER (WHERE rating_value = 1),
            '2', COUNT(*) FILTER (WHERE rating_value = 2),
            '3', COUNT(*) FILTER (WHERE rating_value = 3),
            '4', COUNT(*) FILTER (WHERE rating_value = 4),
            '5', COUNT(*) FILTER (WHERE rating_value = 5)
        ) AS rating_dist
    FROM expanded GROUP BY teacher_id, term_id, dimension_key
),
dist_overall AS (
    SELECT teacher_id, NULL::varchar(20) AS term_id, dimension_key,
        ROUND(AVG(rating_value), 2) AS avg_rating,
        COUNT(*) AS rating_count,
        jsonb_build_object(
            '1', COUNT(*) FILTER (WHERE rating_value = 1),
            '2', COUNT(*) FILTER (WHERE rating_value = 2),
            '3', COUNT(*) FILTER (WHERE rating_value = 3),
            '4', COUNT(*) FILTER (WHERE rating_value = 4),
            '5', COUNT(*) FILTER (WHERE rating_value = 5)
        ) AS rating_dist
    FROM expanded GROUP BY teacher_id, dimension_key
)
INSERT INTO teacher_rating_stats (id, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist)
SELECT gen_random_uuid()::varchar, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist
FROM dist_per_term
UNION ALL
SELECT gen_random_uuid()::varchar, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist
FROM dist_overall;

-- ============================================
-- 10. 开发环境：测试用学籍数据
-- ============================================
INSERT INTO academic.buaa_students (xh, xm, sfzjlxdm, sfzjh, yxdm, zydm, bjdm, xznj, rxnj, pyccdm, xslbdm, sjh, dzxx, xjztdm, sfzx, sfzj) VALUES
    ('20211001', '张三', '1', '110101200301010011', '001', '0812', '210101', '4', '2021', '01', '01', '13800138001', 'zhangsan@buaa.edu.cn', '01', '1', '1'),
    ('20211002', '李四', '1', '110101200301020022', '001', '0812', '210101', '4', '2021', '01', '01', '13800138002', 'lisi@buaa.edu.cn', '01', '1', '1'),
    ('20211003', '王五', '1', '110101200301030033', '003', '0502', '210301', '4', '2021', '01', '01', '13800138003', 'wangwu@buaa.edu.cn', '01', '1', '1'),
    ('20241001', '张三', '1', '110101200301010011', '001', '0812', '241001', '1', '2024', '02', '02', '13800138001', 'zhangsan@buaa.edu.cn', '01', '1', '1'),
    ('20211004', 'John Smith', 'A', 'H12345678', '001', '0812', '210102', '4', '2021', '01', '01', NULL, 'john@buaa.edu.cn', '01', '1', '1')
ON CONFLICT (xh) DO NOTHING;

-- ============================================
-- 11. 开发环境：测试用户认证数据
-- ============================================
-- 为测试用户1设置实名认证（已通过）和学生认证（已通过）
INSERT INTO user_identities (user_id, doc_type, doc_number_enc, person_uid, real_name, verified, verify_method, verified_at)
SELECT id, 'MAINLAND_ID', '\x00'::bytea, 'test_person_uid_001', '测试用户', TRUE, 'academic_db_match', NOW()
FROM users WHERE username = 'test_admin'
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO user_profiles (user_id, school_id, student_ids, active_student_id, verification_status, verification_method, phone, phone_verified, consent_given_at, verified_at)
SELECT id, '10006', '["20211001"]'::jsonb, '20211001', 'verified', 'ldap', '13800138001', TRUE, NOW(), NOW()
FROM users WHERE username = 'test_admin'
ON CONFLICT (user_id) DO NOTHING;

-- 为测试用户分配 super_admin 角色
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r
WHERE u.username = 'test_admin' AND r.name = 'super_admin'
ON CONFLICT DO NOTHING;

COMMIT;
