import React, { useState, useEffect } from 'react';
import {
  Container,
  Typography,
  Box,
  Grid,
  Chip,
  LinearProgress,
  CircularProgress,
  Alert,
  Button,
  Fade,
  Paper,
  Divider,
  Card,
  CardContent,
  IconButton,
  Tooltip,
  Snackbar,
  Slide
} from '@mui/material';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { getCourseDetails } from '../services/courseService';
import { getCourseReviews, voteReview } from '../services/reviewService';
import cacheService from '../services/cacheService';
import { setDocumentTitle } from '../utils/titleUtils';
import SentimentVeryDissatisfiedIcon from '@mui/icons-material/SentimentVeryDissatisfied';
import SentimentDissatisfiedIcon from '@mui/icons-material/SentimentDissatisfied';
import SentimentNeutralIcon from '@mui/icons-material/SentimentNeutral';
import SentimentSatisfiedIcon from '@mui/icons-material/SentimentSatisfied';
import SentimentVerySatisfiedIcon from '@mui/icons-material/SentimentVerySatisfied';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import InfoIcon from '@mui/icons-material/Info';
import WarningIcon from '@mui/icons-material/Warning';
import { getBrowserFingerprint } from '../utils/fingerprintUtils';

// 获取会话ID
const sessionId = cacheService.getSessionId();

function CourseDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const [selectedTeacher, setSelectedTeacher] = useState('所有');
  const [course, setCourse] = useState(null);
  const [reviews, setReviews] = useState([]);
  const [filteredReviews, setFilteredReviews] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [semesters, setSemesters] = useState([]);
  const [teachers, setTeachers] = useState([]);
  const [fadeIn, setFadeIn] = useState(false);
  const [contentReady, setContentReady] = useState(false);

  // 检查是否从侧边栏导航过来
  const isFromSidebar = location.state?.fromSidebar;
  const sidebarCourse = location.state?.course;

  // 使用useEffect来控制淡入效果，与数据加载分离
  useEffect(() => {
    // 立即设置淡入状态为false
    setFadeIn(false);
    setContentReady(false);
    
    // 短暂延迟后开始淡入，创建流畅的过渡效果
    const fadeTimer = setTimeout(() => {
      setFadeIn(true);
    }, 50);
    
    return () => clearTimeout(fadeTimer);
  }, [id]);

  // 当数据加载完成后，标记内容准备就绪
  useEffect(() => {
    if (!loading && course) {
      // 短暂延迟后标记内容准备就绪，确保DOM已更新
      const readyTimer = setTimeout(() => {
        setContentReady(true);
      }, 100);
      
      return () => clearTimeout(readyTimer);
    }
  }, [loading, course]);

  useEffect(() => {
    const fetchCourseData = async () => {
      try {
        let courseData;
        let reviewsData;
        
        // 检查是否是页面刷新或首次加载
        const isRefresh = cacheService.isPageRefresh();
        
        // 如果是从侧边栏导航且有课程数据，直接使用
        if (isFromSidebar && sidebarCourse) {
          courseData = sidebarCourse;
          setCourse(courseData);
          
          // 异步获取评论数据，不阻塞UI渲染
          setLoading(false);
          
          // 在后台获取评论数据
          try {
            // 如果是刷新或首次加载，强制从API获取
            reviewsData = await getCourseReviews(parseInt(id), { forceRefresh: isRefresh });
            
            // 获取浏览器指纹
            const browserFingerprint = getBrowserFingerprint();
            
            // 按发布时间排序评论
            const sortedReviews = (reviewsData.reviews || []).sort((a, b) => 
              new Date(b.createdAt) - new Date(a.createdAt)
            );
            
            // 处理评论数据，添加用户投票状态
            const processedReviews = sortedReviews.map(review => {
              // 如果后端已经返回了用户投票状态，直接使用
              if (review.userVote) {
                return review;
              }
              
              // 否则根据浏览器指纹和本地存储判断用户投票状态
              const voteKey = `review_vote_${review.id}_${browserFingerprint}`;
              const storedVote = localStorage.getItem(voteKey);
              
              return {
                ...review,
                userVote: storedVote ? JSON.parse(storedVote) : null
              };
            });
            
            setReviews(processedReviews);
            setFilteredReviews(processedReviews);
            
            // 处理数据
            processReviewData(courseData, processedReviews);
          } catch (err) {
            console.error('获取评论数据失败:', err);
            // 不显示错误，因为课程数据已经加载成功
          }
        } else {
          // 否则获取课程详情和评论
          setLoading(true);
          
          // 如果是刷新或首次加载，强制从API获取
          courseData = await getCourseDetails(parseInt(id), { forceRefresh: isRefresh });
          setCourse(courseData);
          
          // 如果是刷新或首次加载，强制从API获取
          reviewsData = await getCourseReviews(parseInt(id), { forceRefresh: isRefresh });
          
          // 获取浏览器指纹
          const browserFingerprint = getBrowserFingerprint();
          
          // 按发布时间排序评论
          const sortedReviews = (reviewsData.reviews || []).sort((a, b) => 
            new Date(b.createdAt) - new Date(a.createdAt)
          );
          
          // 处理评论数据，添加用户投票状态
          const processedReviews = sortedReviews.map(review => {
            // 如果后端已经返回了用户投票状态，直接使用
            if (review.userVote) {
              return review;
            }
            
            // 否则根据浏览器指纹和本地存储判断用户投票状态
            const voteKey = `review_vote_${review.id}_${browserFingerprint}`;
            const storedVote = localStorage.getItem(voteKey);
            
            return {
              ...review,
              userVote: storedVote ? JSON.parse(storedVote) : null
            };
          });
          
          setReviews(processedReviews);
          setFilteredReviews(processedReviews);
          
          // 处理数据
          processReviewData(courseData, processedReviews);
          setLoading(false);
        }

        // 设置页面标题为课程名称
        if (courseData && courseData.name) {
          setDocumentTitle(courseData.name);
        }
      } catch (err) {
        console.error('获取课程数据失败:', err);
        setError('获取课程数据失败，请稍后重试');
        setLoading(false);
      }
    };

    if (id) {
      fetchCourseData();
    }
  }, [id, isFromSidebar, sidebarCourse]);

  // 当选择的教师变化时，过滤评论
  useEffect(() => {
    if (selectedTeacher === '所有') {
      setFilteredReviews(reviews);
    } else {
      setFilteredReviews(reviews.filter(review => review.teacher === selectedTeacher));
    }
  }, [selectedTeacher, reviews]);

  // 处理评论数据，生成学期和教师统计
  const processReviewData = (courseData, reviewsData) => {
    if (!courseData || !reviewsData) return;

    // 提取所有学期和教师
    const semestersMap = new Map();
    const teachersSet = new Set(['所有']);
    
    // 只有在有评论数据时才添加学期
    if (reviewsData.length > 0) {
      // 添加全部学期
      semestersMap.set('all', {
        id: 'all',
        name: '全部学期',
        reviewCount: reviewsData.length,
        ratings: {
          recommend: 0,
          content: 0,
          workload: 0,
          exam: 0
        },
        reviews: []
      });

      // 处理每条评论
      reviewsData.forEach(review => {
        // 添加到全部学期
        const allSemester = semestersMap.get('all');
        allSemester.ratings.recommend += review.ratings?.overall || 0;
        allSemester.ratings.content += review.ratings?.content || 0;
        allSemester.ratings.workload += review.ratings?.workload || 0;
        allSemester.ratings.exam += review.ratings?.exam || 0;
        allSemester.reviews.push(review);
        
        // 按学期分类
        if (review.semester) {
          if (!semestersMap.has(review.semester)) {
            semestersMap.set(review.semester, {
              id: review.semester,
              name: review.semester,
              reviewCount: 0,
              ratings: {
                recommend: 0,
                content: 0,
                workload: 0,
                exam: 0
              },
              reviews: []
            });
          }
          
          const semesterData = semestersMap.get(review.semester);
          semesterData.reviewCount += 1;
          semesterData.ratings.recommend += review.ratings?.overall || 0;
          semesterData.ratings.content += review.ratings?.content || 0;
          semesterData.ratings.workload += review.ratings?.workload || 0;
          semesterData.ratings.exam += review.ratings?.exam || 0;
          semesterData.reviews.push(review);
        }
        
        // 收集教师
        if (review.teacher) {
          teachersSet.add(review.teacher);
        }
      });
      
      // 计算平均分
      semestersMap.forEach(semester => {
        if (semester.reviewCount > 0) {
          semester.ratings.recommend = +(semester.ratings.recommend / semester.reviewCount).toFixed(1);
          semester.ratings.content = +(semester.ratings.content / semester.reviewCount).toFixed(1);
          semester.ratings.workload = +(semester.ratings.workload / semester.reviewCount).toFixed(1);
          semester.ratings.exam = +(semester.ratings.exam / semester.reviewCount).toFixed(1);
          
          // 标记数据是否不足
          semester.dataInsufficient = semester.reviewCount < 3;
        }
      });
      
      // 过滤掉评价数量为0的学期
      const filteredSemesters = Array.from(semestersMap.values()).filter(
        semester => semester.reviewCount > 0
      );
      
      // 转换为数组并排序
      filteredSemesters.sort((a, b) => {
        if (a.id === 'all') return -1;
        if (b.id === 'all') return 1;
        return b.name.localeCompare(a.name); // 按学期降序排列
      });
      
      setSemesters(filteredSemesters);
    } else {
      // 如果没有评论数据，设置空数组
      setSemesters([]);
    }
    
    setTeachers(Array.from(teachersSet));
  };

  // 自定义评分图标
  const customIcons = {
    1: <SentimentVeryDissatisfiedIcon color="error" />,
    2: <SentimentDissatisfiedIcon color="warning" />,
    3: <SentimentNeutralIcon color="action" />,
    4: <SentimentSatisfiedIcon color="success" />,
    5: <SentimentVerySatisfiedIcon color="success" />
  };

  // 评分显示组件
  const RatingDisplay = ({ value }) => {
    return (
      <Box 
        sx={{ 
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          transform: 'scale(0.9)',
          color: value === 1 ? '#ef5350' : 
                 value === 2 ? '#ff9800' : 
                 value === 3 ? '#757575' : 
                 '#4caf50'
        }}
      >
        {customIcons[value]}
      </Box>
    );
  };

  const RatingBar = ({ value }) => (
    <LinearProgress
      variant="determinate"
      value={value * 20}
      sx={{
        height: 4,
        borderRadius: 2,
        backgroundColor: 'rgba(255, 255, 255, 0.2)',
        '& .MuiLinearProgress-bar': {
          backgroundColor: '#3498db',
          borderRadius: 2,
        },
      }}
    />
  );

  const SemesterRatings = ({ semester }) => (
    <Box sx={{ mb: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 0.5 }}>
        <Typography variant="subtitle2" sx={{ mr: 1, fontSize: '0.85rem' }}>
            {semester.name}
          </Typography>
          {semester.reviewCount > 0 && (
            <Chip 
              label={semester.reviewCount} 
              size="small" 
              sx={{ 
              height: 16, 
              backgroundColor: '#3498db',
                color: 'white',
              fontWeight: 'bold',
                '& .MuiChip-label': {
                px: 0.8,
                fontSize: '0.7rem',
                }
              }} 
            />
          )}
          {semester.dataInsufficient && (
          <Chip
            label="数据较少"
            size="small"
            sx={{
              ml: 0.5,
              height: 16,
              backgroundColor: '#ff9800',
              color: 'white',
              fontWeight: 'bold',
              '& .MuiChip-label': {
                px: 0.8,
                fontSize: '0.7rem',
              }
            }}
          />
          )}
        </Box>
      
      {!semester.dataInsufficient ? (
        <Grid container spacing={1} sx={{ mb: 1 }}>
      <Grid item xs={6}>
            <Typography variant="caption" sx={{ fontSize: '0.75rem' }}>推荐指数</Typography>
        <RatingBar value={semester.ratings.recommend} />
      </Grid>
      <Grid item xs={6}>
            <Typography variant="caption" sx={{ fontSize: '0.75rem' }}>内容质量</Typography>
        <RatingBar value={semester.ratings.content} />
      </Grid>
      <Grid item xs={6}>
            <Typography variant="caption" sx={{ fontSize: '0.75rem' }}>工作量</Typography>
        <RatingBar value={semester.ratings.workload} />
      </Grid>
      <Grid item xs={6}>
            <Typography variant="caption" sx={{ fontSize: '0.75rem' }}>考核</Typography>
        <RatingBar value={semester.ratings.exam} />
      </Grid>
    </Grid>
      ) : (
        <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block', mb: 1, fontSize: '0.75rem' }}>
          过少的数据没有统计代表性。
        </Typography>
      )}
    </Box>
  );

  // 评论卡片组件
  const ReviewCard = ({ review }) => {
    // 根据总体评价决定卡片边框颜色
    const overallRating = review.ratings?.overall || 0;
    const cardBorderColor = 
      overallRating === 1 ? '#ef5350' : 
      overallRating === 2 ? '#ff9800' : 
      'transparent';
    
    // 点赞/踩状态
    const [voteStatus, setVoteStatus] = useState(review.userVote || null); // null, 'up', 'down'
    const [upvotes, setUpvotes] = useState(review.likes || 0);
    const [downvotes, setDownvotes] = useState(review.dislikes || 0);
    const [isVoting, setIsVoting] = useState(false);
    const [alertMessage, setAlertMessage] = useState('');
    const [alertType, setAlertType] = useState('success'); // 'success', 'error', 'info'
    const [showAlert, setShowAlert] = useState(false);
    
    // 检查是否需要显示警告（点踩次数达到5次）
    const showWarning = downvotes >= 5;
    
    // 处理点赞/踩
    const handleVote = async (voteType, event) => {
      if (isVoting) return;
      
      setIsVoting(true);
      
      try {
        // 获取浏览器指纹
        const browserFingerprint = getBrowserFingerprint();
        
        // 确定API投票类型
        const apiVoteType = voteType === 'up' ? 'like' : 'dislike';
        
        // 调用API进行投票
        const result = await voteReview(review.id, {
          voteType: apiVoteType,
          browserFingerprint
        });
        
        if (result.success) {
          // 本地存储投票状态的键
          const voteKey = `review_vote_${review.id}_${browserFingerprint}`;
          
          // 如果点击的是当前已选择的投票类型，则取消投票
          if (voteStatus === voteType) {
            setVoteStatus(null);
            if (voteType === 'up') {
              setAlertMessage('已取消认同');
            } else {
              setAlertMessage('已取消反对');
            }
            setAlertType('info');
            
            // 从本地存储中删除投票状态
            localStorage.removeItem(voteKey);
          } else {
            setVoteStatus(voteType);
            if (voteType === 'up') {
              setAlertMessage('感谢您的认同！');
              setAlertType('success');
            } else {
              setAlertMessage('感谢您的反馈！');
              setAlertType('error');
            }
            
            // 在本地存储中保存投票状态
            localStorage.setItem(voteKey, JSON.stringify(voteType));
          }
          
          // 更新点赞/踩数量
          setUpvotes(result.data.likes);
          setDownvotes(result.data.dislikes);
          
          // 显示提示
          setShowAlert(true);
          
          // 2秒后自动隐藏提示
          setTimeout(() => {
            setShowAlert(false);
          }, 2000);
        } else {
          console.error('投票失败:', result.error);
          setAlertMessage(result.error || '投票失败，请稍后重试');
          setAlertType('error');
          setShowAlert(true);
          
          // 错误提示显示时间稍长
          setTimeout(() => {
            setShowAlert(false);
          }, 2000);
        }
      } catch (error) {
        console.error('投票失败:', error);
        setAlertMessage('投票失败，请稍后重试');
        setAlertType('error');
        setShowAlert(true);
        
        // 错误提示显示时间稍长
        setTimeout(() => {
          setShowAlert(false);
        }, 2000);
      } finally {
        setIsVoting(false);
      }
    };
    
    // 自定义提示图标
    const getAlertIcon = () => {
      switch (alertType) {
        case 'success':
          return <CheckCircleIcon sx={{ color: '#4caf50', fontSize: '1rem' }} />;
        case 'error':
          return <CancelIcon sx={{ color: '#f44336', fontSize: '1rem' }} />;
        case 'info':
        default:
          return <InfoIcon sx={{ color: '#2196f3', fontSize: '1rem' }} />;
      }
    };

  return (
      <Card 
        sx={{ 
          mb: 2, 
          bgcolor: 'rgba(52, 73, 94, 0.7)', 
          color: 'white',
          border: cardBorderColor !== 'transparent' ? `2px solid ${cardBorderColor}` : 'none',
          boxShadow: cardBorderColor !== 'transparent' ? `0 0 8px ${cardBorderColor}` : 'none',
          position: 'relative'
        }}
      >
        <CardContent>
          <Box sx={{ mb: 1 }}>
            <Typography variant="h6" sx={{ fontSize: '1rem', fontWeight: 'bold' }}>
              {review.title}
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, mb: 1, flexWrap: 'wrap' }}>
              <Typography variant="caption">
                <span style={{ fontWeight: 'bold' }}>{review.teacher}</span> 老师 ({review.semester})
              </Typography>
              <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                {new Date(review.createdAt).toLocaleDateString()}
        </Typography>
            </Box>
          </Box>
          
          <Typography variant="body2" sx={{ mb: 2, whiteSpace: 'pre-line' }}>
            {review.content}
          </Typography>
          
          {review.grade && review.grade.trim() !== '' && (
            <Typography variant="body2" sx={{ mb: showWarning ? 1 : 2 }}>
              成绩：<span style={{ fontWeight: 'bold' }}>{review.grade}</span>
            </Typography>
          )}
          
          {/* 警告提示 - 当点踩次数达到5次时显示 */}
          {showWarning && (
            <Box 
              sx={{ 
                display: 'flex', 
                alignItems: 'center', 
                mb: 2,
                p: 0.8,
                borderRadius: 1,
                bgcolor: 'rgba(255, 152, 0, 0.15)'
              }}
            >
              <WarningIcon sx={{ color: '#ff9800', fontSize: '0.9rem', mr: 0.8 }} />
              <Typography 
                variant="caption" 
                sx={{ 
                  color: '#ff9800', 
                  fontWeight: 'medium',
                  fontSize: '0.75rem'
                }}
              >
                本条测评被多个用户反对，请谨慎参考！
          </Typography>
            </Box>
          )}
          
          <Divider sx={{ bgcolor: 'rgba(255, 255, 255, 0.1)', mb: 1.5 }} />
          
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2, alignItems: 'center' }}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <Typography variant="caption" sx={{ color: 'text.secondary', mr: 0.5 }}>推荐指数</Typography>
                <RatingDisplay value={review.ratings?.overall || 0} />
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <Typography variant="caption" sx={{ color: 'text.secondary', mr: 0.5 }}>内容质量</Typography>
                <RatingDisplay value={review.ratings?.content || 0} />
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <Typography variant="caption" sx={{ color: 'text.secondary', mr: 0.5 }}>工作量</Typography>
                <RatingDisplay value={review.ratings?.workload || 0} />
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <Typography variant="caption" sx={{ color: 'text.secondary', mr: 0.5 }}>考核</Typography>
                <RatingDisplay value={review.ratings?.exam || 0} />
        </Box>
      </Box>

            <Box sx={{ display: 'flex', alignItems: 'center', ml: 2, position: 'relative' }}>
              {/* 嵌入式提示框 */}
              <Box
                sx={{
                  position: 'absolute',
                  right: '100%',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  mr: 1,
                  opacity: showAlert ? 1 : 0,
                  visibility: showAlert ? 'visible' : 'hidden',
                  transition: 'opacity 0.3s ease-in-out, visibility 0.3s ease-in-out',
                  display: 'flex',
                  alignItems: 'center',
                  whiteSpace: 'nowrap'
                }}
              >
                <Alert 
                  severity={alertType} 
                  variant="filled" 
                  icon={getAlertIcon()}
                  sx={{ 
                    py: 0.2,
                    px: 1,
                    minWidth: '120px',
                    width: 'auto',
                    borderRadius: 1,
                    boxShadow: '0 2px 8px rgba(0, 0, 0, 0.2)',
                    height: '24px',
                    '& .MuiAlert-icon': {
                      mr: 0.5,
                      p: 0,
                      display: 'flex',
                      alignItems: 'center'
                    },
                    '& .MuiAlert-message': {
                      p: 0,
                      fontSize: '0.75rem',
                      whiteSpace: 'nowrap',
                      display: 'flex',
                      alignItems: 'center',
                      color: 'white'
                    },
                    '& .MuiAlert-action': {
                      display: 'none'
                    }
                  }}
                >
                  {alertMessage}
                </Alert>
              </Box>
              
              <Tooltip title="有用">
                <IconButton 
                  size="small" 
                  onClick={(e) => handleVote('up', e)}
                  disabled={isVoting}
                  sx={{ 
                    color: voteStatus === 'up' ? '#4caf50' : 'rgba(255, 255, 255, 0.5)',
                    '&:hover': { color: '#4caf50' }
                  }}
                >
                  <ThumbUpIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Typography variant="caption" sx={{ mx: 0.5, color: 'text.secondary' }}>
                {upvotes}
              </Typography>
              
              <Tooltip title="没用">
                <IconButton 
                  size="small" 
                  onClick={(e) => handleVote('down', e)}
                  disabled={isVoting}
                  sx={{ 
                    color: voteStatus === 'down' ? '#f44336' : 'rgba(255, 255, 255, 0.5)',
                    '&:hover': { color: '#f44336' },
                    ml: 1
                  }}
                >
                  <ThumbDownIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Typography variant="caption" sx={{ mx: 0.5, color: 'text.secondary' }}>
                {downvotes}
              </Typography>
            </Box>
          </Box>
        </CardContent>
      </Card>
    );
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  if (!course) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="warning">未找到课程信息</Alert>
      </Box>
    );
  }

  // 将学期分组，每行最多3个
  const groupedSemesters = [];
  for (let i = 0; i < semesters.length; i += 3) {
    groupedSemesters.push(semesters.slice(i, i + 3));
  }

  return (
    <Fade in={fadeIn} timeout={400}>
      <Container maxWidth="lg" sx={{ py: 3, minHeight: 'calc(100vh - 64px)' }}>
        {loading && !course ? (
          <Box display="flex" justifyContent="center" alignItems="center" minHeight="300px">
            <CircularProgress />
          </Box>
        ) : error ? (
          <Alert severity="error" sx={{ mt: 2 }}>{error}</Alert>
        ) : course ? (
          <Box 
            sx={{ 
              opacity: contentReady ? 1 : 0, 
              transition: 'opacity 400ms ease-in-out, transform 400ms ease-in-out',
              transform: contentReady ? 'translateX(0)' : 'translateX(40px)',
              height: '100%',
              overflow: 'auto',
              pb: 4
            }}
          >
            {/* 课程标题和基本信息 */}
            <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap' }}>
                <Typography variant="h4" component="h1" gutterBottom sx={{ mr: 2, mb: 0 }}>
                  {course.name}
                </Typography>
                <Chip 
                  label={`${course.department || '未知学院'}`} 
                  size="medium" 
                  sx={{ 
                    bgcolor: '#3498db', 
                    color: 'white', 
                    fontWeight: 'bold', 
                    fontSize: '0.95rem',
                    height: '32px',
                    borderRadius: '16px',
                    boxShadow: '0 2px 4px rgba(52, 152, 219, 0.3)'
                  }}
                />
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <Chip 
                  icon={<InfoIcon sx={{ fontSize: '1.1rem', color: 'white !important' }} />}
                  label={`${reviews.length || 0} 条测评`} 
                  size="medium" 
                  sx={{ 
                    bgcolor: 'rgba(52, 152, 219, 0.2)', 
                    color: 'white',
                    fontWeight: 'medium',
                    height: '32px',
                    borderRadius: '16px',
                    border: '1px solid rgba(52, 152, 219, 0.3)',
                    '& .MuiChip-icon': {
                      color: 'white'
                    }
                  }}
                />
                <Button 
                  variant="contained" 
                  color="primary"
                  size="medium"
                  onClick={() => navigate(`/reviews/post?courseId=${id}`)}
                  sx={{ 
                    height: '32px',
                    borderRadius: '16px',
                    boxShadow: '0 2px 4px rgba(52, 152, 219, 0.3)',
                    fontWeight: 'bold'
                  }}
                >
                  写测评
                </Button>
              </Box>
            </Box>

            {/* 评分和教师区域 */}
            <Paper 
              elevation={0} 
              sx={{ 
                p: 2, 
                mb: 3, 
                bgcolor: 'rgba(44, 62, 80, 0.7)', 
                border: '1px solid rgba(255, 255, 255, 0.1)',
                borderRadius: 1
              }}
            >
      {/* 评分标题 */}
              <Typography variant="h6" sx={{ fontSize: '1.1rem', mb: 2, color: '#3498db', fontWeight: 'bold' }}>
        评分 {'>'}
      </Typography>

      {/* 学期评分列表 */}
              <Box sx={{ mb: 3 }}>
                {semesters.length > 0 ? (
                  <Grid container spacing={2}>
                    {groupedSemesters.map((semesterGroup, groupIndex) => (
                      <Grid item xs={12} key={groupIndex}>
                        <Grid container spacing={2}>
                          {semesterGroup.map((semester) => (
                            <Grid item xs={12} sm={4} key={semester.id}>
                              <Paper 
                                elevation={0} 
                                sx={{ 
                                  p: 1.5, 
                                  bgcolor: 'rgba(52, 73, 94, 0.7)',
                                  borderRadius: 1
                                }}
                              >
            <SemesterRatings semester={semester} />
                              </Paper>
                            </Grid>
                          ))}
                        </Grid>
                      </Grid>
                    ))}
                  </Grid>
                ) : (
                  <Box sx={{ textAlign: 'center', py: 3 }}>
                    <Typography variant="body1" sx={{ color: 'text.secondary', mb: 2 }}>
                      暂无数据，快来添加这门课的第一条测评吧！
                    </Typography>
                    <Button 
                      variant="contained" 
                      color="primary"
                      onClick={() => navigate(`/reviews/post?courseId=${id}`)}
                    >
                      写第一条测评
                    </Button>
                  </Box>
                )}
      </Box>

      {/* 授课教师 */}
              <Box>
                <Typography variant="h6" sx={{ mb: 1.5, fontSize: '1.1rem', color: '#3498db', fontWeight: 'bold' }}>
          授课教师 {'>'}
        </Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.8 }}>
                  {teachers.length > 0 ? (
                    teachers.map((teacher) => (
            <Chip
              key={teacher}
              label={teacher}
                        size="small"
                        onClick={() => setSelectedTeacher(teacher)}
              sx={{
                          bgcolor: selectedTeacher === teacher ? '#2ecc71' : '#3498db',
                color: 'white',
                          fontSize: '0.8rem',
                          height: '24px',
                          fontWeight: 'bold',
                          boxShadow: selectedTeacher === teacher ? '0 0 8px rgba(46, 204, 113, 0.7)' : 'none',
                          transform: selectedTeacher === teacher ? 'scale(1.05)' : 'scale(1)',
                          transition: 'all 0.2s ease',
                '&:hover': {
                            bgcolor: selectedTeacher === teacher ? '#27ae60' : '#2980b9',
                            transform: 'scale(1.05)',
                },
              }}
            />
                    ))
                  ) : (
                    <Typography variant="body2" sx={{ color: 'text.secondary' }}>
                      暂无教师数据
                    </Typography>
                  )}
        </Box>
              </Box>
            </Paper>

            {/* 测评内容列表 */}
            <Box>
              {filteredReviews.length > 0 ? (
                filteredReviews.map(review => (
                  <ReviewCard key={review.id} review={review} />
                ))
              ) : (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                  <Typography variant="body1" sx={{ color: 'text.secondary' }}>
                    {selectedTeacher !== '所有' 
                      ? `没有找到 ${selectedTeacher} 的测评` 
                      : '暂无测评内容'}
                  </Typography>
                </Box>
              )}
      </Box>
    </Box>
        ) : null}
      </Container>
    </Fade>
  );
}

export default CourseDetail; 