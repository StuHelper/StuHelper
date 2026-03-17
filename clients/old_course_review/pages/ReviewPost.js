import React, { useState, useEffect, useRef } from 'react';
import { 
  Box, 
  Typography, 
  TextField, 
  Button, 
  Container, 
  Paper, 
  IconButton,
  InputAdornment,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Grid,
  Rating,
  TextareaAutosize,
  Chip,
  Divider,
  FormHelperText,
  List,
  ListItemButton,
  ListItemText,
  Popper,
  Fade,
  CircularProgress,
  Snackbar,
  Alert,
  Link
} from '@mui/material';
import { useNavigate, useLocation } from 'react-router-dom';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import SearchIcon from '@mui/icons-material/Search';
import ClearIcon from '@mui/icons-material/Clear';
import SentimentVeryDissatisfiedIcon from '@mui/icons-material/SentimentVeryDissatisfied';
import SentimentDissatisfiedIcon from '@mui/icons-material/SentimentDissatisfied';
import SentimentNeutralIcon from '@mui/icons-material/SentimentNeutral';
import SentimentSatisfiedIcon from '@mui/icons-material/SentimentSatisfied';
import SentimentVerySatisfiedIcon from '@mui/icons-material/SentimentVerySatisfied';
import { pinyin } from 'pinyin-pro';
import courseService from '../services/courseService';
import reviewService from '../services/reviewService';
import cacheService from '../services/cacheService';
import { Link as RouterLink } from 'react-router-dom';
import { setDocumentTitle } from '../utils/titleUtils';

// 模拟学期数据
const semesters = [
  '2024-2025 第二学期',
  '2024-2025 第一学期',
  '2023-2024 第二学期',
  '2023-2024 第一学期',
  '2022-2023 第二学期',
  '2022-2023 第一学期',
  '2021-2022 第二学期'
];

function ReviewPost() {
  const navigate = useNavigate();
  const location = useLocation();
  
  // 从URL中获取courseId参数
  const searchParams = new URLSearchParams(location.search);
  const courseIdFromUrl = searchParams.get('courseId');
  
  const [courseSearch, setCourseSearch] = useState('');
  const [selectedCourse, setSelectedCourse] = useState(null);
  const [anchorEl, setAnchorEl] = useState(null);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [filteredCourses, setFilteredCourses] = useState([]);
  const [allCourses, setAllCourses] = useState([]);
  const [teacherSearch, setTeacherSearch] = useState('');
  const [semester, setSemester] = useState(semesters[0]);
  const [ratings, setRatings] = useState({
    overall: 0,
    content: 0,
    workload: 0,
    exam: 0
  });
  const [title, setTitle] = useState('');
  
  // 初始化评价内容的默认值
  const defaultReviewContent = `课程听感：
作业/任务量：
关于考试：`;

  const [review, setReview] = useState(defaultReviewContent);
  const [grade, setGrade] = useState('');
  const [errors, setErrors] = useState({});
  const [submitting, setSubmitting] = useState(false);
  const [snackbarMessage, setSnackbarMessage] = useState('');
  const [snackbarSeverity, setSnackbarSeverity] = useState('success');
  const [snackbarOpen, setSnackbarOpen] = useState(false);

  // 组件加载时获取所有课程
  useEffect(() => {
    // 设置页面标题
    setDocumentTitle('发布评价');
    
    const fetchCourses = async () => {
      try {
        // 检查是否是页面刷新或首次加载
        const isRefresh = cacheService.isPageRefresh();
        
        // 如果是刷新或首次加载，强制从API获取
        const courses = await courseService.getAllCourses({ forceRefresh: isRefresh });
        
        // 为每个课程添加拼音和首字母属性
        const coursesWithPinyin = courses.map(course => {
          const fullPinyin = pinyin(course.name, { toneType: 'none', type: 'array' }).join('').toLowerCase();
          const firstLetters = pinyin(course.name, { pattern: 'first', toneType: 'none', type: 'array' }).join('').toLowerCase();
          const pinyinArray = pinyin(course.name, { toneType: 'none', type: 'array' });
          const pinyinParts = pinyin(course.name, { toneType: 'none', type: 'array' });
          
          return {
            ...course,
            pinyin: fullPinyin,
            firstLetters,
            pinyinArray,
            pinyinParts
          };
        });
        
        setAllCourses(coursesWithPinyin);
        
        // 初始显示前10个课程
        setFilteredCourses(coursesWithPinyin.slice(0, 10));
        
        // 如果URL中有courseId参数，自动选择对应课程
        if (courseIdFromUrl) {
          const selectedCourse = coursesWithPinyin.find(c => c.id === parseInt(courseIdFromUrl));
          if (selectedCourse) {
            handleCourseSelect(selectedCourse);
          }
        }
      } catch (error) {
        console.error('获取课程列表失败:', error);
        setSnackbarMessage('获取课程列表失败，请稍后重试');
        setSnackbarSeverity('error');
        setSnackbarOpen(true);
      }
    };
    
    fetchCourses();
  }, [courseIdFromUrl]);

  // 搜索词变化时更新过滤结果
  useEffect(() => {
    const searchCourses = async () => {
      try {
        // 如果搜索词为空，显示热门课程
        if (!courseSearch.trim()) {
          setFilteredCourses(
            allCourses
              .sort((a, b) => (b.reviewCount || 0) - (a.reviewCount || 0))
              .slice(0, 10)
          );
          return;
        }
        
        // 本地搜索实现
        const searchTermLower = courseSearch.toLowerCase().trim();
        const results = allCourses.filter(course => {
          // 课程名称匹配
          if (course.name && course.name.toLowerCase().includes(searchTermLower)) {
            return true;
          }
          
          // 拼音匹配
          if (course.pinyin && course.pinyin.includes(searchTermLower)) {
            return true;
          }
          
          // 首字母匹配
          if (course.firstLetters && course.firstLetters.includes(searchTermLower)) {
            return true;
          }
          
          return false;
        });
        
        // 按评价数量排序
        const sortedResults = results.sort((a, b) => (b.reviewCount || 0) - (a.reviewCount || 0));
        setFilteredCourses(sortedResults.slice(0, 20)); // 限制结果数量
      } catch (err) {
        console.error('搜索课程失败:', err);
        setFilteredCourses([]);
      }
    };

    searchCourses();
  }, [courseSearch, allCourses]);

  const handleCourseSearchFocus = (event) => {
    setAnchorEl(event.currentTarget);
  };

  const handleCourseSearchBlur = () => {
    // 不再关闭候选框，让它保持展开状态
  };

  const handleCourseKeyDown = (event) => {
    if (!filteredCourses.length) return;

    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault();
        setSelectedIndex(prev => 
          prev < filteredCourses.length - 1 ? prev + 1 : prev
        );
        break;
      case 'ArrowUp':
        event.preventDefault();
        setSelectedIndex(prev => prev > 0 ? prev - 1 : prev);
        break;
      case 'Enter':
        event.preventDefault();
        if (filteredCourses[selectedIndex]) {
          handleCourseSelect(filteredCourses[selectedIndex]);
        }
        break;
      default:
        break;
    }
  };

  const handleCourseSelect = (course) => {
    setSelectedCourse(course);
    setCourseSearch(course.name);
    setAnchorEl(null);
  };

  const handleRatingChange = (category, value) => {
    setRatings(prev => ({
      ...prev,
      [category]: prev[category] === value ? 0 : value
    }));
    // 清除对应的错误提示
    setErrors(prev => ({
      ...prev,
      [category]: undefined
    }));
  };

  // 自定义评分图标
  const customIcons = {
    1: <SentimentVeryDissatisfiedIcon color="error" />,
    2: <SentimentDissatisfiedIcon color="warning" />,
    3: <SentimentNeutralIcon color="action" />,
    4: <SentimentSatisfiedIcon color="success" />,
    5: <SentimentVerySatisfiedIcon color="success" />
  };

  // 评分说明
  const ratingDescriptions = {
    overall: {
      1: '非常差',
      2: '较差',
      3: '一般',
      4: '良好',
      5: '优秀'
    },
    content: {
      1: '内容质量很差',
      2: '内容质量较差',
      3: '内容质量一般',
      4: '内容质量良好',
      5: '内容质量优秀'
    },
    workload: {
      1: '工作量很少',
      2: '工作量较少',
      3: '工作量适中',
      4: '工作量较多',
      5: '工作量很多'
    },
    exam: {
      1: '给分很低',
      2: '给分较低',
      3: '给分一般',
      4: '给分较高',
      5: '给分很高'
    }
  };

  const handleSubmit = async () => {
    // 重置错误状态
    const newErrors = {};

    // 验证课程选择
    if (!selectedCourse) {
      newErrors.course = '请选择课程';
    }

    // 验证教师姓名
    if (!teacherSearch.trim()) {
      newErrors.teacher = '请输入教师姓名';
    }

    // 验证学期
    if (!semester) {
      newErrors.semester = '请选择学期';
    }

    // 验证评分
    if (ratings.overall === 0) {
      newErrors.overall = '请选择总体评价';
    }
    if (ratings.content === 0) {
      newErrors.content = '请选择内容质量评分';
    }
    if (ratings.workload === 0) {
      newErrors.workload = '请选择工作量评分';
    }
    if (ratings.exam === 0) {
      newErrors.exam = '请选择考核给分评分';
    }

    // 验证标题
    if (!title.trim()) {
      newErrors.title = '请输入评价标题';
    }

    // 验证评价内容
    if (!review.trim()) {
      newErrors.review = '请输入详细评价';
    }

    // 注意：不再验证成绩字段，允许为空

    // 如果有错误，更新错误状态并返回
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    try {
      // 设置提交状态
      setSubmitting(true);
      
      // 构造提交数据
      const reviewData = {
        courseId: selectedCourse.id,
        courseName: selectedCourse.name,
        courseCode: selectedCourse.code,
        teacher: teacherSearch.trim(),
        semester,
        ratings: {
          overall: ratings.overall,
          content: ratings.content,
          workload: ratings.workload,
          exam: ratings.exam
        },
        title: title.trim(),
        content: review.trim(),
        grade: grade.trim(), // 保留grade字段，但允许为空
        createdAt: new Date().toISOString()
      };

      // 提交测评数据
      const result = await reviewService.submitReview(reviewData);
      
      if (result.success) {
        // 提交成功后显示成功消息
        alert('测评提交成功！');
        // 跳转到评论列表页
        navigate('/reviews');
      } else {
        // 处理提交错误
        throw new Error(result.error || '提交失败，请稍后重试');
      }
    } catch (error) {
      // 处理提交错误
      console.error('提交测评失败:', error);
      alert(error.message || '提交测评失败，请稍后重试');
    } finally {
      // 无论成功或失败，都重置提交状态
      setSubmitting(false);
    }
  };

  // 判断是否应该显示候选框
  const shouldShowPopper = Boolean(anchorEl) && courseSearch.trim().length > 0;

  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
      <Box sx={{ mb: 4, display: 'flex', alignItems: 'center' }}>
        <IconButton 
          onClick={() => navigate(-1)} 
          sx={{ mr: 2, color: 'primary.main' }}
          aria-label="返回"
        >
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h4" component="h1">
          发布测评
        </Typography>
      </Box>

      <Paper 
        elevation={0} 
        sx={{ 
          p: 3, 
          bgcolor: '#2c3e50', 
          color: 'white',
          borderRadius: 2
        }}
      >
        <Box sx={{ mb: 4 }}>
          <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'bold' }}>
            选择课程
          </Typography>
          
          <Box sx={{ position: 'relative' }}>
            <TextField
              fullWidth
              variant="outlined"
              placeholder="输入名称、拼音或首字母搜索，如: 高等数学、gaodengshuxue、gdsx"
              value={courseSearch}
              onChange={(e) => setCourseSearch(e.target.value)}
              onFocus={handleCourseSearchFocus}
              onBlur={handleCourseSearchBlur}
              onKeyDown={handleCourseKeyDown}
              error={!!errors.course}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon />
                  </InputAdornment>
                ),
                endAdornment: courseSearch && (
                  <InputAdornment position="end">
                    <IconButton
                      aria-label="清除搜索"
                      onClick={() => setCourseSearch('')}
                      edge="end"
                      sx={{ color: 'rgba(255, 255, 255, 0.7)' }}
                    >
                      <ClearIcon />
                    </IconButton>
                  </InputAdornment>
                ),
              }}
              sx={{
                '& .MuiOutlinedInput-root': {
                  bgcolor: 'rgba(255, 255, 255, 0.1)',
                  color: 'white',
                  '& fieldset': {
                    borderColor: errors.course ? '#ef5350' : 'transparent',
                  },
                  '&:hover fieldset': {
                    borderColor: errors.course ? '#ef5350' : 'rgba(255, 255, 255, 0.3)',
                  },
                  '&.Mui-focused fieldset': {
                    borderColor: errors.course ? '#ef5350' : 'rgba(255, 255, 255, 0.5)',
                  }
                }
              }}
            />
            
            <Popper
              open={shouldShowPopper}
              anchorEl={anchorEl}
              placement="bottom-start"
              transition
              style={{ width: anchorEl ? anchorEl.clientWidth : 'auto', maxWidth: '800px', zIndex: 1300 }}
            >
              {({ TransitionProps }) => (
                <Fade {...TransitionProps} timeout={350}>
                  <Paper 
                    elevation={3}
                    sx={{
                      mt: 1,
                      maxHeight: '400px',
                      overflow: 'auto',
                      bgcolor: '#2c3e50'
                    }}
                  >
                    {filteredCourses.length > 0 ? (
                      <List component="nav" sx={{ p: 0 }}>
                        {filteredCourses.map((course, index) => (
                          <ListItemButton
                            key={course.id}
                            selected={index === selectedIndex}
                            onClick={() => handleCourseSelect(course)}
                            sx={{
                              py: 1,
                              px: 2,
                              borderBottom: '1px solid rgba(255, 255, 255, 0.1)',
                              '&:hover': {
                                bgcolor: 'rgba(255, 255, 255, 0.1)',
                              },
                              '&.Mui-selected': {
                                bgcolor: 'rgba(46, 204, 113, 0.2)',
                                '&:hover': {
                                  bgcolor: 'rgba(46, 204, 113, 0.3)',
                                },
                              },
                            }}
                          >
                            <ListItemText
                              primary={
                                <Typography
                                  variant="body1"
                                  sx={{
                                    color: 'white',
                                    fontWeight: index === selectedIndex ? 'bold' : 'normal',
                                  }}
                                >
                                  {course.name}
                                </Typography>
                              }
                              secondary={
                                <Typography
                                  variant="caption"
                                  sx={{ color: 'rgba(255, 255, 255, 0.7)' }}
                                >
                                  {`${course.department} · ${course.reviewCount || 0} 条评价`}
                                </Typography>
                              }
                            />
                          </ListItemButton>
                        ))}
                      </List>
                    ) : (
                      <Box sx={{ p: 2, textAlign: 'center' }}>
                        <Typography sx={{ color: 'white' }}>
                          搜索的课程暂未被收录，
                          <Link
                            component={RouterLink}
                            to="/about"
                            sx={{ color: '#2ecc71', textDecoration: 'none' }}
                          >
                            联系我们反馈
                          </Link>
                          。或者试试
                          <Link
                            component={RouterLink}
                            to="/search"
                            sx={{ color: '#2ecc71', textDecoration: 'none', ml: 1 }}
                          >
                            高级搜索
                          </Link>
                        </Typography>
                      </Box>
                    )}
                  </Paper>
                </Fade>
              )}
            </Popper>
          </Box>
          {errors.course && (
            <FormHelperText error sx={{ mt: 1, color: '#ef5350' }}>
              {errors.course}
            </FormHelperText>
          )}
        </Box>

        <Box sx={{ mb: 4 }}>
          <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'bold' }}>
            授课老师
          </Typography>
          <TextField
            fullWidth
            variant="outlined"
            placeholder="薛峰"
            value={teacherSearch}
            onChange={(e) => {
              setTeacherSearch(e.target.value);
              // 清除教师姓名的错误提示
              setErrors(prev => ({
                ...prev,
                teacher: undefined
              }));
            }}
            error={!!errors.teacher}
            InputProps={{
              endAdornment: (
                <InputAdornment position="end">
                  <Chip label="老师" size="small" sx={{ bgcolor: 'primary.main', color: 'white' }} />
                </InputAdornment>
              ),
            }}
            sx={{
              '& .MuiOutlinedInput-root': {
                bgcolor: 'rgba(255, 255, 255, 0.1)',
                color: 'white',
                '& fieldset': {
                  borderColor: errors.teacher ? '#ef5350' : 'transparent',
                },
                '&:hover fieldset': {
                  borderColor: errors.teacher ? '#ef5350' : 'rgba(255, 255, 255, 0.3)',
                },
                '&.Mui-focused fieldset': {
                  borderColor: errors.teacher ? '#ef5350' : 'rgba(255, 255, 255, 0.5)',
                }
              }
            }}
          />
          {errors.teacher && (
            <FormHelperText error sx={{ mt: 1, color: '#ef5350' }}>
              {errors.teacher}
            </FormHelperText>
          )}
        </Box>

        <Box sx={{ mb: 4 }}>
          <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'bold' }}>
            学期
          </Typography>
          <FormControl fullWidth error={!!errors.semester}>
            <Select
              value={semester}
              onChange={(e) => setSemester(e.target.value)}
              displayEmpty
              IconComponent={KeyboardArrowDownIcon}
              sx={{
                bgcolor: 'rgba(255, 255, 255, 0.1)',
                color: 'white',
                '& .MuiOutlinedInput-notchedOutline': {
                  borderColor: errors.semester ? '#ef5350' : 'transparent',
                },
                '&:hover .MuiOutlinedInput-notchedOutline': {
                  borderColor: errors.semester ? '#ef5350' : 'rgba(255, 255, 255, 0.3)',
                },
                '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
                  borderColor: errors.semester ? '#ef5350' : 'rgba(255, 255, 255, 0.5)',
                },
                '& .MuiSvgIcon-root': {
                  color: 'white',
                }
              }}
            >
              {semesters.map((sem) => (
                <MenuItem key={sem} value={sem}>{sem}</MenuItem>
              ))}
            </Select>
          </FormControl>
          {errors.semester && (
            <FormHelperText error sx={{ mt: 1, color: '#ef5350' }}>
              {errors.semester}
            </FormHelperText>
          )}
        </Box>

        <Box sx={{ mb: 4 }}>
          <Typography variant="subtitle1" sx={{ mb: 2, fontWeight: 'bold' }}>
            评分
          </Typography>
          
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={3}>
              <Typography variant="body2" sx={{ textAlign: 'right', pt: 1 }}>
                总体评价
              </Typography>
            </Grid>
            <Grid item xs={9}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  {[1, 2, 3, 4, 5].map((value) => (
                    <IconButton 
                      key={value}
                      onClick={() => handleRatingChange('overall', value)}
                      sx={{ 
                        color: ratings.overall === value 
                          ? (value === 1 ? '#ef5350' : 
                             value === 2 ? '#ff9800' : 
                             value === 3 ? '#757575' : 
                             '#4caf50')
                          : 'rgba(255, 255, 255, 0.3)',
                        p: 1,
                        m: 0.5,
                        transition: 'all 0.2s',
                        transform: ratings.overall === value ? 'scale(1.2)' : 'scale(1)',
                        boxShadow: ratings.overall === value ? '0 0 10px rgba(255, 255, 255, 0.7)' : 'none',
                        backgroundColor: ratings.overall === value ? 'rgba(255, 255, 255, 0.15)' : 'transparent',
                        border: ratings.overall === value ? '2px solid rgba(255, 255, 255, 0.7)' : '2px solid transparent',
                        borderRadius: '50%',
                        '&:hover': {
                          color: value === 1 ? '#ef5350' : 
                                 value === 2 ? '#ff9800' : 
                                 value === 3 ? '#757575' : 
                                 '#4caf50',
                          transform: 'scale(1.2)',
                          backgroundColor: 'rgba(255, 255, 255, 0.1)'
                        }
                      }}
                    >
                      {customIcons[value]}
                    </IconButton>
                  ))}
                </Box>
                {errors.overall && (
                  <Typography 
                    variant="caption" 
                    sx={{ 
                      color: '#ef5350',
                      ml: 2,
                      animation: 'fadeIn 0.3s ease-in-out',
                      '@keyframes fadeIn': {
                        '0%': { opacity: 0, transform: 'translateX(-10px)' },
                        '100%': { opacity: 1, transform: 'translateX(0)' }
                      }
                    }}
                  >
                    {errors.overall}
                  </Typography>
                )}
              </Box>
            </Grid>
          </Grid>

          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={3}>
              <Typography variant="body2" sx={{ textAlign: 'right', pt: 1 }}>
                内容质量
              </Typography>
            </Grid>
            <Grid item xs={9}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  {[1, 2, 3, 4, 5].map((value) => (
                    <IconButton 
                      key={value}
                      onClick={() => handleRatingChange('content', value)}
                      sx={{ 
                        color: ratings.content === value 
                          ? (value === 1 ? '#ef5350' : 
                             value === 2 ? '#ff9800' : 
                             value === 3 ? '#757575' : 
                             '#4caf50')
                          : 'rgba(255, 255, 255, 0.3)',
                        p: 1,
                        m: 0.5,
                        transition: 'all 0.2s',
                        transform: ratings.content === value ? 'scale(1.2)' : 'scale(1)',
                        boxShadow: ratings.content === value ? '0 0 10px rgba(255, 255, 255, 0.7)' : 'none',
                        backgroundColor: ratings.content === value ? 'rgba(255, 255, 255, 0.15)' : 'transparent',
                        border: ratings.content === value ? '2px solid rgba(255, 255, 255, 0.7)' : '2px solid transparent',
                        borderRadius: '50%',
                        '&:hover': {
                          color: value === 1 ? '#ef5350' : 
                                 value === 2 ? '#ff9800' : 
                                 value === 3 ? '#757575' : 
                                 '#4caf50',
                          transform: 'scale(1.2)',
                          backgroundColor: 'rgba(255, 255, 255, 0.1)'
                        }
                      }}
                    >
                      {customIcons[value]}
                    </IconButton>
                  ))}
                </Box>
                {errors.content && (
                  <Typography 
                    variant="caption" 
                    sx={{ 
                      color: '#ef5350',
                      ml: 2,
                      animation: 'fadeIn 0.3s ease-in-out',
                      '@keyframes fadeIn': {
                        '0%': { opacity: 0, transform: 'translateX(-10px)' },
                        '100%': { opacity: 1, transform: 'translateX(0)' }
                      }
                    }}
                  >
                    {errors.content}
                  </Typography>
                )}
              </Box>
            </Grid>
          </Grid>

          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={3}>
              <Typography variant="body2" sx={{ textAlign: 'right', pt: 1 }}>
                工作量
              </Typography>
            </Grid>
            <Grid item xs={9}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  {[1, 2, 3, 4, 5].map((value) => (
                    <IconButton 
                      key={value}
                      onClick={() => handleRatingChange('workload', value)}
                      sx={{ 
                        color: ratings.workload === value 
                          ? (value === 1 ? '#ef5350' : 
                             value === 2 ? '#ff9800' : 
                             value === 3 ? '#757575' : 
                             '#4caf50')
                          : 'rgba(255, 255, 255, 0.3)',
                        p: 1,
                        m: 0.5,
                        transition: 'all 0.2s',
                        transform: ratings.workload === value ? 'scale(1.2)' : 'scale(1)',
                        boxShadow: ratings.workload === value ? '0 0 10px rgba(255, 255, 255, 0.7)' : 'none',
                        backgroundColor: ratings.workload === value ? 'rgba(255, 255, 255, 0.15)' : 'transparent',
                        border: ratings.workload === value ? '2px solid rgba(255, 255, 255, 0.7)' : '2px solid transparent',
                        borderRadius: '50%',
                        '&:hover': {
                          color: value === 1 ? '#ef5350' : 
                                 value === 2 ? '#ff9800' : 
                                 value === 3 ? '#757575' : 
                                 '#4caf50',
                          transform: 'scale(1.2)',
                          backgroundColor: 'rgba(255, 255, 255, 0.1)'
                        }
                      }}
                    >
                      {customIcons[value]}
                    </IconButton>
                  ))}
                </Box>
                {errors.workload && (
                  <Typography 
                    variant="caption" 
                    sx={{ 
                      color: '#ef5350',
                      ml: 2,
                      animation: 'fadeIn 0.3s ease-in-out',
                      '@keyframes fadeIn': {
                        '0%': { opacity: 0, transform: 'translateX(-10px)' },
                        '100%': { opacity: 1, transform: 'translateX(0)' }
                      }
                    }}
                  >
                    {errors.workload}
                  </Typography>
                )}
              </Box>
            </Grid>
          </Grid>

          <Grid container spacing={2}>
            <Grid item xs={3}>
              <Typography variant="body2" sx={{ textAlign: 'right', pt: 1 }}>
                考核给分
              </Typography>
            </Grid>
            <Grid item xs={9}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  {[1, 2, 3, 4, 5].map((value) => (
                    <IconButton 
                      key={value}
                      onClick={() => handleRatingChange('exam', value)}
                      sx={{ 
                        color: ratings.exam === value 
                          ? (value === 1 ? '#ef5350' : 
                             value === 2 ? '#ff9800' : 
                             value === 3 ? '#757575' : 
                             '#4caf50')
                          : 'rgba(255, 255, 255, 0.3)',
                        p: 1,
                        m: 0.5,
                        transition: 'all 0.2s',
                        transform: ratings.exam === value ? 'scale(1.2)' : 'scale(1)',
                        boxShadow: ratings.exam === value ? '0 0 10px rgba(255, 255, 255, 0.7)' : 'none',
                        backgroundColor: ratings.exam === value ? 'rgba(255, 255, 255, 0.15)' : 'transparent',
                        border: ratings.exam === value ? '2px solid rgba(255, 255, 255, 0.7)' : '2px solid transparent',
                        borderRadius: '50%',
                        '&:hover': {
                          color: value === 1 ? '#ef5350' : 
                                 value === 2 ? '#ff9800' : 
                                 value === 3 ? '#757575' : 
                                 '#4caf50',
                          transform: 'scale(1.2)',
                          backgroundColor: 'rgba(255, 255, 255, 0.1)'
                        }
                      }}
                    >
                      {customIcons[value]}
                    </IconButton>
                  ))}
                </Box>
                {errors.exam && (
                  <Typography 
                    variant="caption" 
                    sx={{ 
                      color: '#ef5350',
                      ml: 2,
                      animation: 'fadeIn 0.3s ease-in-out',
                      '@keyframes fadeIn': {
                        '0%': { opacity: 0, transform: 'translateX(-10px)' },
                        '100%': { opacity: 1, transform: 'translateX(0)' }
                      }
                    }}
                  >
                    {errors.exam}
                  </Typography>
                )}
              </Box>
            </Grid>
          </Grid>
          
          <Box sx={{ mt: 2, pl: 3, pr: 3 }}>
            <Typography variant="caption" sx={{ color: 'rgba(255, 255, 255, 0.6)' }}>
              提示：点击表情图标进行评分，点击已选中的图标可取消选择
            </Typography>
          </Box>
        </Box>

        <Box sx={{ mb: 4 }}>
          <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'bold' }}>
            标题
          </Typography>
          <TextField
            fullWidth
            variant="outlined"
            placeholder="一句话总结你的观点，例如: 力推这门课程"
            value={title}
            onChange={(e) => {
              setTitle(e.target.value);
              // 清除标题的错误提示
              setErrors(prev => ({
                ...prev,
                title: undefined
              }));
            }}
            error={!!errors.title}
            sx={{
              '& .MuiOutlinedInput-root': {
                bgcolor: 'rgba(255, 255, 255, 0.1)',
                color: 'white',
                '& fieldset': {
                  borderColor: errors.title ? '#ef5350' : 'transparent',
                },
                '&:hover fieldset': {
                  borderColor: errors.title ? '#ef5350' : 'rgba(255, 255, 255, 0.3)',
                },
                '&.Mui-focused fieldset': {
                  borderColor: errors.title ? '#ef5350' : 'rgba(255, 255, 255, 0.5)',
                }
              }
            }}
          />
          {errors.title && (
            <FormHelperText error sx={{ mt: 1, color: '#ef5350' }}>
              {errors.title}
            </FormHelperText>
          )}
        </Box>

        <Box sx={{ mb: 4 }}>
          <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 'bold' }}>
            详细评价
          </Typography>
          <TextField
            fullWidth
            multiline
            minRows={8}
            placeholder="请详细描述你对这门课程的评价"
            value={review}
            onChange={(e) => {
              setReview(e.target.value);
              // 清除详细评价的错误提示
              setErrors(prev => ({
                ...prev,
                review: undefined
              }));
            }}
            error={!!errors.review}
            sx={{
              '& .MuiOutlinedInput-root': {
                bgcolor: 'rgba(255, 255, 255, 0.1)',
                color: 'white',
                '& fieldset': {
                  borderColor: errors.review ? '#ef5350' : 'transparent',
                },
                '&:hover fieldset': {
                  borderColor: errors.review ? '#ef5350' : 'rgba(255, 255, 255, 0.3)',
                },
                '&.Mui-focused fieldset': {
                  borderColor: errors.review ? '#ef5350' : 'rgba(255, 255, 255, 0.5)',
                }
              }
            }}
          />
          {errors.review && (
            <FormHelperText error sx={{ mt: 1, color: '#ef5350' }}>
              {errors.review}
            </FormHelperText>
          )}
          <FormHelperText sx={{ color: 'rgba(255, 255, 255, 0.7)', mt: 1 }}>
            请畅所欲言, 默认模板可以按需修改
            <br />
            测评内容理想上应当富有事实且描述全面，比如一门课讲得好但考试很难, 二者都写入测评更有利于同学们做出全面的选择和判断
          </FormHelperText>
        </Box>

        <Box sx={{ mb: 4 }}>
          <Typography variant="body1" sx={{ mb: 1, fontWeight: 'medium' }}>
            成绩
          </Typography>
          <TextField
            fullWidth
            variant="outlined"
            placeholder="可选填，例如：100、90+、A、优秀、退课等"
            value={grade}
            onChange={(e) => setGrade(e.target.value)}
            sx={{
              '& .MuiOutlinedInput-root': {
                bgcolor: 'rgba(255, 255, 255, 0.1)',
                color: 'white',
                '& fieldset': {
                  borderColor: 'transparent',
                },
                '&:hover fieldset': {
                  borderColor: 'rgba(255, 255, 255, 0.3)',
                },
                '&.Mui-focused fieldset': {
                  borderColor: 'rgba(255, 255, 255, 0.5)',
                }
              },
              '& .MuiFormHelperText-root': {
                color: 'rgba(255, 255, 255, 0.7)'
              }
            }}
          />
          <FormHelperText sx={{ color: 'rgba(255, 255, 255, 0.7)', mt: 1 }}>
            可选，分享你的成绩有助于同学们进行更全面的判断
          </FormHelperText>
        </Box>

        <Divider sx={{ bgcolor: 'rgba(255, 255, 255, 0.1)', my: 3 }} />

        <Box sx={{ textAlign: 'center' }}>
          <Button
            variant="contained"
            size="large"
            onClick={handleSubmit}
            disabled={submitting}
            sx={{
              bgcolor: '#3498db',
              color: 'white',
              px: 6,
              py: 1.5,
              fontSize: '1.1rem',
              borderRadius: 2,
              '&:hover': {
                bgcolor: '#2980b9'
              },
              '&.Mui-disabled': {
                bgcolor: 'rgba(52, 152, 219, 0.6)',
                color: 'rgba(255, 255, 255, 0.7)'
              }
            }}
          >
            {submitting ? '提交中...' : '提交测评'}
          </Button>
          <Typography variant="caption" display="block" sx={{ mt: 2, color: 'rgba(255, 255, 255, 0.7)' }}>
            提交测评表示您同意我们的使用测评的内容，并且了解本站的相关立场。
          </Typography>
        </Box>
      </Paper>

      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={() => setSnackbarOpen(false)}
      >
        <Alert onClose={() => setSnackbarOpen(false)} severity={snackbarSeverity}>
          {snackbarMessage}
        </Alert>
      </Snackbar>
    </Container>
  );
}

export default ReviewPost; 