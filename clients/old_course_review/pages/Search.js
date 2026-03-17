import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Container,
  TextField,
  Button,
  Typography,
  Grid,
  Card,
  CardContent,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  CircularProgress,
  Alert,
  InputAdornment,
  IconButton,
  Snackbar,
  Chip,
  Divider,
  Tooltip,
  Link
} from '@mui/material';
import { useNavigate } from 'react-router-dom';
import SearchIcon from '@mui/icons-material/Search';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import SentimentVeryDissatisfiedIcon from '@mui/icons-material/SentimentVeryDissatisfied';
import SentimentDissatisfiedIcon from '@mui/icons-material/SentimentDissatisfied';
import SentimentNeutralIcon from '@mui/icons-material/SentimentNeutral';
import SentimentSatisfiedIcon from '@mui/icons-material/SentimentSatisfied';
import SentimentVerySatisfiedIcon from '@mui/icons-material/SentimentVerySatisfied';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import WarningIcon from '@mui/icons-material/Warning';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import InfoIcon from '@mui/icons-material/Info';
import courseService from '../services/courseService';
import searchService from '../services/searchService';
import reviewService from '../services/reviewService';
import { voteReview } from '../services/reviewService';
import cacheService from '../services/cacheService';
import { getBrowserFingerprint } from '../utils/fingerprintUtils';
import { Link as RouterLink } from 'react-router-dom';
import { setDocumentTitle } from '../utils/titleUtils';

const Search = () => {
  const navigate = useNavigate();

  // 自定义评分图标
  const customIcons = {
    1: <SentimentVeryDissatisfiedIcon color="error" />,
    2: <SentimentDissatisfiedIcon color="warning" />,
    3: <SentimentNeutralIcon color="action" />,
    4: <SentimentSatisfiedIcon color="success" />,
    5: <SentimentVerySatisfiedIcon color="success" />
  };

  // 状态管理
  const [searchParams, setSearchParams] = useState({
    courseName: '',
    courseCode: '',
    teacherName: '',
    department: '',
    semester: '',
  });
  const [courses, setCourses] = useState([]);
  const [reviews, setReviews] = useState([]);
  const [loading, setLoading] = useState(false);
  const [reviewsLoading, setReviewsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [departments, setDepartments] = useState([]);
  const [semesters, setSemesters] = useState([]);
  const [showWarning, setShowWarning] = useState(false);
  const [showResults, setShowResults] = useState(false);

  // 设置页面标题
  useEffect(() => {
    setDocumentTitle('搜索');
  }, []);

  // 初始化加载部门和学期数据
  useEffect(() => {
    const fetchInitialData = async () => {
      try {
        // 检查是否是页面刷新或首次加载
        const isRefresh = cacheService.isPageRefresh();
        
        // 获取院系数据（异步）
        const deptData = await courseService.getAllDepartments({ forceRefresh: isRefresh });
        // 获取学期数据（同步）
        const semesterData = courseService.getSemesters();
        
        setDepartments(deptData);
        setSemesters(semesterData);
      } catch (err) {
        setError('加载初始数据失败');
        console.error('Error fetching initial data:', err);
      }
    };
    
    fetchInitialData();
  }, []);

  // 处理输入变化
  const handleInputChange = (event) => {
    const { name, value } = event.target;
    
    // 如果修改课程名称，清空课程代码
    if (name === 'courseName' && value) {
      setSearchParams(prev => ({
        ...prev,
        [name]: value,
        courseCode: ''
      }));
      return;
    }
    
    // 如果修改课程代码，清空课程名称
    if (name === 'courseCode' && value) {
      setSearchParams(prev => ({
        ...prev,
        [name]: value,
        courseName: ''
      }));
      return;
    }
    
    setSearchParams(prev => ({
      ...prev,
      [name]: value
    }));
  };

  // 验证搜索条件
  const validateSearchParams = () => {
    // 检查是否至少输入了一个主要搜索条件（课程名称、课程代码或教师姓名）
    const hasCourseCondition = !!searchParams.courseName || !!searchParams.courseCode;
    const hasTeacherCondition = !!searchParams.teacherName;
    
    if (!hasCourseCondition && !hasTeacherCondition) {
      setShowWarning(true);
      return false;
    }
    
    setShowWarning(false);
    return true;
  };

  // 搜索课程
  const handleSearch = async () => {
    // 验证搜索条件
    if (!validateSearchParams()) {
      return;
    }
    
    setLoading(true);
    setReviewsLoading(true);
    setError(null);
    setReviews([]);
    setCourses([]);
    
    try {
      // 判断是否有课程条件
      const hasCourseCondition = !!searchParams.courseName || !!searchParams.courseCode;
      const hasReviewCondition = !!searchParams.teacherName || !!searchParams.semester || !!searchParams.department;
      
      // 如果有课程条件，搜索课程
      if (hasCourseCondition) {
      const { courses: searchResults, error: searchError } = await courseService.searchCourses(searchParams);
      
      if (searchError) {
        setError(searchError);
        } else if (searchResults.length === 0) {
          setError('未找到符合条件的课程，请尝试其他关键词');
          setShowResults(true);
          setLoading(false);
          setReviewsLoading(false);
          return; // 如果没有找到课程，直接返回，不搜索测评
      } else {
        setCourses(searchResults);
          setShowResults(true);
          
          // 搜索测评
          try {
            // 获取搜索到的课程的所有测评
            const courseIds = searchResults.map(course => course.id);
            if (courseIds.length > 0) {
              // 获取这些课程的所有测评
              const reviewPromises = courseIds.map(courseId => 
                reviewService.getCourseReviews(courseId)
                  .then(result => {
                    // 为每个测评添加对应的课程信息
                    const reviews = result.reviews || [];
                    const course = searchResults.find(c => c.id === courseId);
                    return reviews.map(review => ({
                      ...review,
                      course: course
                    }));
                  })
                  .catch(err => {
                    console.error(`获取课程ID ${courseId} 的测评失败:`, err);
                    return [];
                  })
              );
              
              const allReviewsArrays = await Promise.all(reviewPromises);
              let allReviews = allReviewsArrays.flat();
              
              // 如果有测评条件，进一步筛选
              if (hasReviewCondition) {
                allReviews = allReviews.filter(review => {
                  let matchTeacher = true;
                  let matchSemester = true;
                  
                  if (searchParams.teacherName) {
                    matchTeacher = review.teacher && review.teacher.includes(searchParams.teacherName);
                  }
                  
                  if (searchParams.semester) {
                    matchSemester = review.semester === searchParams.semester;
                  }
                  
                  return matchTeacher && matchSemester;
                });
              }
              
              // 按发布时间排序（最新的在前面）
              allReviews.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
              
              // 限制结果数量为最多15条
              setReviews(allReviews.slice(0, 15));
            }
          } catch (reviewError) {
            console.error('搜索测评失败:', reviewError);
            setReviews([]);
          }
        }
      } else if (hasReviewCondition) {
        // 只有测评条件，没有课程条件
        // 直接搜索所有测评
        setShowResults(true);
        try {
          // 获取所有课程
          const allCourses = await courseService.getAllCourses();
          
          // 先按照测评条件筛选课程，减少需要获取测评的课程数量
          let filteredCourses = allCourses;
          if (searchParams.department) {
            filteredCourses = filteredCourses.filter(course => 
              course.department === searchParams.department
            );
          }
          
          // 如果筛选后的课程数量过多，只取前20个课程
          if (filteredCourses.length > 20) {
            filteredCourses = filteredCourses.slice(0, 20);
          }
          
          // 获取筛选后课程的测评
          const courseIds = filteredCourses.map(course => course.id);
          const reviewPromises = courseIds.map(courseId => 
            reviewService.getCourseReviews(courseId)
              .then(result => {
                // 为每个测评添加对应的课程信息
                const reviews = result.reviews || [];
                const course = filteredCourses.find(c => c.id === courseId);
                return reviews.map(review => ({
                  ...review,
                  course: course
                }));
              })
              .catch(err => {
                console.error(`获取课程ID ${courseId} 的测评失败:`, err);
                return [];
              })
          );
          
          // 使用Promise.all并行获取测评
          const allReviewsArrays = await Promise.all(reviewPromises);
          let allReviews = allReviewsArrays.flat();
          
          // 根据测评条件筛选
          const filteredReviews = allReviews.filter(review => {
            let matchTeacher = true;
            let matchSemester = true;
            
            if (searchParams.teacherName) {
              matchTeacher = review.teacher && review.teacher.includes(searchParams.teacherName);
            }
            
            if (searchParams.semester) {
              matchSemester = review.semester === searchParams.semester;
            }
            
            return matchTeacher && matchSemester;
          });
          
          // 按发布时间排序
          filteredReviews.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
          
          // 限制结果数量为最多15条
          setReviews(filteredReviews.slice(0, 15));
        } catch (reviewError) {
          console.error('搜索测评失败:', reviewError);
          setReviews([]);
        }
      }
    } catch (err) {
      setError('搜索时出错，请稍后重试');
      console.error('Search error:', err);
    } finally {
      setLoading(false);
      setReviewsLoading(false);
    }
  };

  // 返回搜索表单
  const handleBack = () => {
    setShowResults(false);
    setError(null);
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

  // 测评卡片组件
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
            <Box sx={{ display: 'flex', gap: 1, mb: 1, flexWrap: 'wrap', alignItems: 'center' }}>
              <Link 
                component={RouterLink} 
                to={`/courses/view/${review.courseId}`}
                sx={{ 
                  color: '#00bfff', 
                  textDecoration: 'none',
                  fontWeight: 'bold',
                  fontSize: '0.875rem',
                  display: 'inline-flex',
                  alignItems: 'center',
                  '&:hover': { textDecoration: 'underline' }
                }}
              >
                {review.course?.name || '未知课程'}
              </Link>
              <Typography variant="caption" sx={{ fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center' }}>
                <span style={{ fontWeight: 'bold' }}>{review.teacher}</span> 老师 ({review.semester})
              </Typography>
              <Typography variant="caption" sx={{ color: 'text.secondary', fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center' }}>
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

  // 渲染搜索表单
  const renderSearchForm = () => (
    <Box component="form" noValidate sx={{ mt: 3 }}>
      <Typography variant="h6" gutterBottom>
        按课程条件搜索
      </Typography>
      <Grid container spacing={2}>
        <Grid item xs={12} sm={6}>
          <TextField
            name="courseName"
            label="课程名称"
            fullWidth
            value={searchParams.courseName}
            onChange={handleInputChange}
            disabled={!!searchParams.courseCode}
            helperText="支持中英文及拼音搜索"
            InputProps={{
              style: { opacity: searchParams.courseCode ? 0.5 : 1 }
            }}
          />
        </Grid>
        <Grid item xs={12} sm={6}>
          <TextField
            name="courseCode"
            label="课程代码"
            fullWidth
            value={searchParams.courseCode}
            onChange={handleInputChange}
            disabled={!!searchParams.courseName}
            helperText="学校内部课程编号，由于同门课程可能存在多个代码，系统内只随机保留了一个，不推荐使用"
            InputProps={{
              style: { opacity: searchParams.courseName ? 0.5 : 1 }
            }}
          />
        </Grid>
        <Grid item xs={12} sm={6}>
          <FormControl fullWidth>
            <InputLabel>开课院系</InputLabel>
            <Select
              name="department"
              value={searchParams.department}
              onChange={handleInputChange}
              label="开课院系"
            >
              <MenuItem value="">全部</MenuItem>
              {departments.map((dept) => (
                <MenuItem key={dept} value={dept}>{dept}</MenuItem>
              ))}
            </Select>
          </FormControl>
        </Grid>
      </Grid>
      
      <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>
        按测评条件搜索
      </Typography>
      <Grid container spacing={2}>
        <Grid item xs={12} sm={6}>
          <TextField
            name="teacherName"
            label="教师姓名"
            fullWidth
            value={searchParams.teacherName}
            onChange={handleInputChange}
            helperText="支持模糊搜索"
          />
        </Grid>
        <Grid item xs={12} sm={6}>
          <FormControl fullWidth>
            <InputLabel>学期</InputLabel>
            <Select
              name="semester"
              value={searchParams.semester}
              onChange={handleInputChange}
              label="学期"
            >
              <MenuItem value="">任意</MenuItem>
              {semesters.map((sem) => (
                <MenuItem key={sem} value={sem}>{sem}</MenuItem>
              ))}
            </Select>
          </FormControl>
        </Grid>
        <Grid item xs={12}>
          <Button
            fullWidth
            variant="contained"
            onClick={handleSearch}
            startIcon={<SearchIcon />}
            disabled={loading}
          >
            {loading ? <CircularProgress size={24} /> : '搜索'}
          </Button>
        </Grid>
        {showWarning && (
          <Grid item xs={12}>
            <Alert 
              severity="warning" 
              sx={{ 
                mt: 1,
                backgroundColor: 'rgba(255, 244, 229, 1)',
                '& .MuiAlert-icon': {
                  color: '#ff9800'
                },
                '& .MuiAlert-message': {
                  color: '#ff0000',
                  fontWeight: 'bold'
                }
              }}
            >
              请至少输入一个搜索条件！
            </Alert>
          </Grid>
        )}
      </Grid>
    </Box>
  );

  // 渲染搜索结果
  const renderSearchResults = () => (
    <Box sx={{ mt: 0 }}>
      {courses.length > 0 && (
        <>
          {/* 有测评的课程 */}
          {courses.filter(course => course.reviewCount > 0).length > 0 && (
            <>
              <Typography variant="h6" gutterBottom sx={{ color: 'text.primary', mb: 2 }}>
                有测评的课程
              </Typography>
        <Grid container spacing={2}>
                {courses
                  .filter(course => course.reviewCount > 0)
                  .map((course) => (
                    <Grid item xs={12} sm={6} key={course.id}>
                      <Card 
                        sx={{ 
                          bgcolor: 'rgba(52, 73, 94, 0.7)', 
                          color: 'white',
                          cursor: 'pointer',
                          transition: 'all 0.2s ease-in-out',
                          '&:hover': {
                            transform: 'translateY(-2px)',
                            boxShadow: '0 4px 8px rgba(0,0,0,0.2)'
                          }
                        }}
                        onClick={() => navigate(`/courses/view/${course.id}`)}
                      >
                        <CardContent sx={{ py: 1, px: 2, '&:last-child': { pb: 1 } }}>
                          <Box sx={{ 
                            display: 'flex', 
                            alignItems: 'center', 
                            justifyContent: 'space-between',
                            gap: 1
                          }}>
                            <Box sx={{ 
                              display: 'flex', 
                              alignItems: 'center', 
                              gap: 1,
                              flex: 1,
                              minWidth: 0 // 确保文本可以正确截断
                            }}>
                              <Typography 
                                variant="subtitle1" 
                                sx={{ 
                                  fontWeight: 'bold',
                                  whiteSpace: 'nowrap',
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis'
                                }}
                              >
                                {course.name}
                              </Typography>
                              <Chip 
                                label={course.department}
                                size="small"
                                sx={{
                                  backgroundColor: '#3498db',
                                  color: 'white',
                                  fontWeight: 'bold',
                                  height: '20px'
                                }}
                              />
                              <Typography 
                                variant="caption" 
                                sx={{ 
                                  color: 'text.secondary',
                                  whiteSpace: 'nowrap'
                                }}
                              >
                                {course.semester}
                  </Typography>
                            </Box>
                            <Chip 
                              label={`${course.reviewCount} 条测评`} 
                              size="small" 
                              sx={{ 
                                color: '#3498db',
                                fontWeight: 'bold',
                                height: '20px',
                                flexShrink: 0,
                                backgroundColor: 'transparent'
                              }} 
                            />
                          </Box>
                        </CardContent>
                      </Card>
                    </Grid>
                  ))}
              </Grid>
            </>
          )}

          {/* 无测评的课程 */}
          {courses.filter(course => !course.reviewCount).length > 0 && (
            <>
              <Typography variant="h6" gutterBottom sx={{ color: 'text.primary', mt: 4, mb: 2 }}>
                无测评的课程
                  </Typography>
              <Grid container spacing={2}>
                {courses
                  .filter(course => !course.reviewCount)
                  .map((course) => (
                    <Grid item xs={12} sm={6} key={course.id}>
                      <Card 
                        sx={{ 
                          bgcolor: 'rgba(52, 73, 94, 0.7)', 
                          color: 'white',
                          cursor: 'pointer',
                          transition: 'all 0.2s ease-in-out',
                          '&:hover': {
                            transform: 'translateY(-2px)',
                            boxShadow: '0 4px 8px rgba(0,0,0,0.2)'
                          }
                        }}
                        onClick={() => navigate(`/courses/view/${course.id}`)}
                      >
                        <CardContent sx={{ py: 1, px: 2, '&:last-child': { pb: 1 } }}>
                          <Box sx={{ 
                            display: 'flex', 
                            alignItems: 'center',
                            gap: 1
                          }}>
                            <Typography 
                              variant="subtitle1" 
                              sx={{ 
                                fontWeight: 'bold',
                                whiteSpace: 'nowrap',
                                overflow: 'hidden',
                                textOverflow: 'ellipsis'
                              }}
                            >
                              {course.name}
                  </Typography>
                            <Chip 
                              label={course.department}
                              size="small"
                              sx={{
                                backgroundColor: '#3498db',
                                color: 'white',
                                fontWeight: 'bold',
                                height: '20px',
                                flexShrink: 0
                              }}
                            />
                            <Box sx={{ flex: 1 }} />
                            <Typography 
                              variant="caption" 
                              sx={{ 
                                color: 'text.secondary',
                                whiteSpace: 'nowrap',
                                flexShrink: 0
                              }}
                            >
                              {course.semester}
                  </Typography>
                          </Box>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
            </>
          )}
        </>
      )}
    </Box>
  );

  // 渲染搜索表单页面
  const renderSearchPage = () => (
    <>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 4 }}>
        <IconButton 
          onClick={() => navigate('/')}
          sx={{ 
            mr: 2,
            color: '#3498db',
            '&:hover': {
              backgroundColor: 'rgba(52, 152, 219, 0.1)'
            }
          }}
        >
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h5" component="h1">
          高级搜索
        </Typography>
      </Box>
      <Typography variant="subtitle1" gutterBottom color="text.secondary" sx={{ mb: 3 }}>
        自由组合以下搜索条件以搜索测评，或者试试首页中较为简单的按课程搜索
      </Typography>
      {renderSearchForm()}
    </>
  );

  // 渲染搜索结果页面
  const renderResultsPage = () => {
    // 判断是否有课程条件
    const hasCourseCondition = !!searchParams.courseName || !!searchParams.courseCode;
    const hasReviewCondition = !!searchParams.teacherName || !!searchParams.semester || !!searchParams.department;
    
    return (
      <>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 4 }}>
          <IconButton 
            onClick={handleBack}
            sx={{ 
              mr: 2,
              color: '#3498db',
              '&:hover': {
                backgroundColor: 'rgba(52, 152, 219, 0.1)'
              }
            }}
          >
            <ArrowBackIcon />
          </IconButton>
          <Typography variant="h5" component="h1">
            搜索结果
          </Typography>
        </Box>
        
        {/* 错误提示 */}
        {error && (
          <Alert severity="warning" sx={{ mb: 4, '& .MuiAlert-icon': { color: '#ff9800' }, '& .MuiAlert-message': { color: '#ff9800', fontWeight: 'medium' } }}>
            {error}
          </Alert>
        )}
        
        {/* 课程结果 - 只在有课程条件且找到课程时显示 */}
        {hasCourseCondition && courses.length > 0 && renderSearchResults()}
        
        {/* 测评结果部分 */}
        {!error && (reviews.length > 0 || reviewsLoading) && (
          <Box sx={{ mt: hasCourseCondition && courses.length > 0 ? 4 : 0 }}>
            <Typography variant="h6" gutterBottom sx={{ color: 'text.primary', mb: 2 }}>
              测评结果
            </Typography>
            
            {reviewsLoading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', width: '100%', py: 4 }}>
                <CircularProgress />
              </Box>
            ) : reviews.length > 0 ? (
              reviews.map((review) => (
                <ReviewCard key={review.id} review={review} />
              ))
            ) : (
              <Alert severity="info" sx={{ mb: 2 }}>
                未找到符合条件的测评
              </Alert>
            )}
          </Box>
        )}
        
        {/* 当没有错误、不在加载中，且没有任何结果时显示 */}
        {!error && !loading && !reviewsLoading && courses.length === 0 && reviews.length === 0 && (
          <Alert severity="warning" sx={{ mb: 2, '& .MuiAlert-icon': { color: '#ff9800' }, '& .MuiAlert-message': { color: '#ff9800', fontWeight: 'medium' } }}>
            未找到任何符合条件的结果，请尝试其他搜索条件
          </Alert>
        )}
      </>
    );
  };

  return (
    <Container maxWidth="lg">
      <Box sx={{ my: 4 }}>
        {showResults ? renderResultsPage() : renderSearchPage()}
      </Box>
    </Container>
  );
};

export default Search; 