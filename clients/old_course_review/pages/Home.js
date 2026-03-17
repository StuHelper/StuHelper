import React, { useState, useEffect, useRef } from 'react';
import { 
  Box, 
  Typography, 
  Button, 
  TextField, 
  InputAdornment, 
  IconButton, 
  Stack,
  Paper,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Popper,
  Fade,
  Chip,
  Grid,
  CircularProgress,
  Snackbar,
  Alert,
  Link
} from '@mui/material';
import { useNavigate, Link as RouterLink } from 'react-router-dom';
import SearchIcon from '@mui/icons-material/Search';
import RateReviewIcon from '@mui/icons-material/RateReview';
import MenuBookIcon from '@mui/icons-material/MenuBook';
import courseService from '../services/courseService';
import reviewService from '../services/reviewService';
import cacheService from '../services/cacheService';
import { setDocumentTitle } from '../utils/titleUtils';

function Home() {
  const navigate = useNavigate();
  const [searchTerm, setSearchTerm] = useState('');
  const [anchorEl, setAnchorEl] = useState(null);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [filteredCourses, setFilteredCourses] = useState([]);
  const [allCourses, setAllCourses] = useState([]);
  const [loading, setLoading] = useState(true);
  const [reviewCount, setReviewCount] = useState(0);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [snackbarMessage, setSnackbarMessage] = useState('');
  const [snackbarSeverity, setSnackbarSeverity] = useState('info');

  // 显示提示消息
  const showSnackbar = (message, severity = 'info') => {
    setSnackbarMessage(message);
    setSnackbarSeverity(severity);
    setSnackbarOpen(true);
  };

  // 关闭提示消息
  const handleSnackbarClose = () => {
    setSnackbarOpen(false);
  };

  // 获取所有课程
  useEffect(() => {
    // 设置页面标题 - 首页不需要前缀
    setDocumentTitle('');
    
    const fetchCourses = async () => {
      try {
        setLoading(true);
        
        // 检查是否是页面刷新或首次加载
        const isRefresh = cacheService.isPageRefresh();
        
        // 如果是刷新或首次加载，强制从API获取
        const courses = await courseService.getAllCourses({ forceRefresh: isRefresh });
        
        setAllCourses(courses);
        // 默认显示按评价数排序的课程
        setFilteredCourses(courses.sort((a, b) => (b.reviewCount || 0) - (a.reviewCount || 0)).slice(0, 10));
        setLoading(false);
      } catch (error) {
        console.error('获取课程列表失败:', error);
        showSnackbar('获取课程列表失败，请稍后重试', 'error');
        setLoading(false);
      }
    };

    // 获取评价总数
    const fetchReviewCount = async () => {
      try {
        const count = await courseService.getReviewCount();
        setReviewCount(count);
      } catch (error) {
        console.error('获取评价总数失败:', error);
      }
    };

    fetchCourses();
    fetchReviewCount();
  }, []);

  // 搜索词变化时更新过滤结果
  useEffect(() => {
    const searchCourses = async () => {
      try {
        setLoading(true);
        
        // 如果搜索词为空，显示热门课程
        if (!searchTerm.trim()) {
          setFilteredCourses(
            allCourses
              .sort((a, b) => (b.reviewCount || 0) - (a.reviewCount || 0))
              .slice(0, 10)
          );
          setLoading(false);
          return;
        }
        
        // 本地搜索实现
        const searchTermLower = searchTerm.toLowerCase().trim();
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
        
        setLoading(false);
      } catch (err) {
        console.error('搜索课程失败:', err);
        setFilteredCourses([]);
        setLoading(false);
      }
    };

    searchCourses();
  }, [searchTerm, allCourses]);

  const handleSearchFocus = (event) => {
    setAnchorEl(event.currentTarget);
  };

  const handleSearchBlur = () => {
    // 不再关闭候选框，让它保持展开状态
  };

  const handleKeyDown = (event) => {
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
          navigate(`/courses/view/${filteredCourses[selectedIndex].id}`);
        }
        break;
      default:
        break;
    }
  };

  const handleCourseClick = (courseId) => {
    navigate(`/courses/view/${courseId}`);
  };

  // 判断是否应该显示候选框
  const shouldShowPopper = Boolean(anchorEl) && searchTerm.trim().length > 0;

  return (
    <>
      <Box sx={{ mb: 4 }}>
        <Typography variant="h3" component="h1" sx={{ mb: 2, fontWeight: 'bold' }}>
          评课社区@BUAA
        </Typography>
        <Typography variant="subtitle1" sx={{ opacity: 0.8 }}>
          来自同学们的{reviewCount}条测评，帮你找到最适合的课程
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', mt: 1, justifyContent: 'center' }}>
          <Typography variant="subtitle1" sx={{ color: '#2ecc71' }}>
            课程数据已更新至2024-2025学年春季学期
          </Typography>
        </Box>
      </Box>

      <Box 
        sx={{ 
          width: '100%',
          maxWidth: 800,
          mb: 6,
          position: 'relative'
        }}
      >
        <TextField
          fullWidth
          variant="outlined"
          placeholder="搜索课程名称、拼音或首字母，如：高等数学、gaodengshuxue、gdsx"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          onFocus={handleSearchFocus}
          onBlur={handleSearchBlur}
          onKeyDown={handleKeyDown}
          sx={{
            '& .MuiOutlinedInput-root': {
              borderRadius: 2,
              backgroundColor: 'white',
              boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
            },
            '& .MuiInputBase-input': {
              color: '#000000',
            }
          }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: 'rgba(0, 0, 0, 0.54)' }} />
              </InputAdornment>
            ),
            endAdornment: loading ? (
              <InputAdornment position="end">
                <CircularProgress size={20} />
              </InputAdornment>
            ) : searchTerm ? (
              <InputAdornment position="end">
                <IconButton
                  size="small"
                  onClick={() => setSearchTerm('')}
                >
                  <Typography variant="caption" color="text.secondary">
                    清除
                  </Typography>
                </IconButton>
              </InputAdornment>
            ) : null
          }}
        />

        <Popper 
          open={shouldShowPopper}
          anchorEl={anchorEl}
          placement="bottom-start"
          transition
          sx={{ width: 'auto' }}
          style={{ width: anchorEl ? anchorEl.clientWidth : 'auto', maxWidth: '800px', zIndex: 1300 }}
        >
          {({ TransitionProps }) => (
            <Fade {...TransitionProps} timeout={350}>
              <Paper 
                elevation={3} 
                sx={{ 
                  mt: 0.5, 
                  maxHeight: 400, 
                  overflow: 'auto',
                  border: '1px solid #e0e0e0',
                  width: '100%'
                }}
              >
                {filteredCourses.length > 0 ? (
                  <List dense>
                    {filteredCourses.map((course, index) => (
                      <ListItem 
                        key={course.id} 
                        disablePadding
                        selected={index === selectedIndex}
                      >
                        <ListItemButton 
                          onClick={() => handleCourseClick(course.id)}
                          sx={{ 
                            py: 1.2,
                            backgroundColor: index === selectedIndex ? 'rgba(25, 118, 210, 0.08)' : 'transparent',
                            '&:hover': {
                              backgroundColor: 'rgba(25, 118, 210, 0.12)'
                            }
                          }}
                        >
                          <ListItemText 
                            primary={
                              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
                                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                                  <Typography variant="body1" component="span" sx={{ fontWeight: 500 }}>
                                    {course.name}
                                  </Typography>
                                  <Typography variant="body2" color="text.secondary" sx={{ ml: 1 }}>
                                    {course.department || '未知院系'}
                                  </Typography>
                                </Box>
                                {course.reviewCount > 0 && (
                                  <Chip 
                                    size="small" 
                                    label={`${course.reviewCount}条评价`} 
                                    color="primary" 
                                    variant="outlined"
                                    sx={{ height: 20, '& .MuiChip-label': { px: 0.8, py: 0 } }} 
                                  />
                                )}
                              </Box>
                            }
                            secondary={null}
                          />
                        </ListItemButton>
                      </ListItem>
                    ))}
                  </List>
                ) : (
                  <Box sx={{ p: 2, textAlign: 'center' }}>
                    <Typography variant="body1" sx={{ mb: 1 }}>
                      搜索的课程暂未被收录，
                      <Link 
                        component={RouterLink} 
                        to="/about" 
                        sx={{ color: '#3498db', textDecoration: 'none', '&:hover': { textDecoration: 'underline' } }}
                      >
                        联系我们
                      </Link>
                      反馈。
                    </Typography>
                    <Typography variant="body1">
                      或者试试
                      <Link 
                        component={RouterLink} 
                        to="/search" 
                        sx={{ color: '#3498db', textDecoration: 'none', '&:hover': { textDecoration: 'underline' } }}
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

      <Grid container spacing={3} sx={{ mb: 6 }}>
        <Grid item xs={12} md={6}>
          <Paper 
            elevation={3} 
            sx={{ 
              p: 3, 
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              bgcolor: 'background.paper',
              transition: 'transform 0.3s, box-shadow 0.3s',
              '&:hover': {
                transform: 'translateY(-5px)',
                boxShadow: 8
              }
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
              <MenuBookIcon sx={{ fontSize: 28, mr: 1, color: 'primary.main' }} />
              <Typography variant="h5" component="h2" sx={{ fontWeight: 'bold' }}>
                浏览课程
              </Typography>
            </Box>
            <Typography variant="body1" sx={{ mb: 3, flex: 1 }}>
              按开课院系浏览所有课程，查看详细的课程信息和学生评价。
            </Typography>
            <Button 
              variant="contained" 
              color="primary" 
              size="large"
              onClick={() => navigate('/courses/list')}
              sx={{ 
                borderRadius: 2,
                textTransform: 'none',
                fontWeight: 'bold'
              }}
            >
              查看所有课程
            </Button>
          </Paper>
        </Grid>
        
        <Grid item xs={12} md={6}>
          <Paper 
            elevation={3} 
            sx={{ 
              p: 3, 
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              bgcolor: 'background.paper',
              transition: 'transform 0.3s, box-shadow 0.3s',
              '&:hover': {
                transform: 'translateY(-5px)',
                boxShadow: 8
              }
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
              <RateReviewIcon sx={{ fontSize: 28, mr: 1, color: 'secondary.main' }} />
              <Typography variant="h5" component="h2" sx={{ fontWeight: 'bold' }}>
                发布评价
              </Typography>
            </Box>
            <Typography variant="body1" sx={{ mb: 3, flex: 1 }}>
              分享你的课程体验，帮助其他同学做出更好的选择。
            </Typography>
            <Button 
              variant="contained" 
              color="secondary" 
              size="large"
              onClick={() => navigate('/reviews/post')}
              sx={{ 
                borderRadius: 2,
                textTransform: 'none',
                fontWeight: 'bold'
              }}
            >
              写一条评价
            </Button>
          </Paper>
        </Grid>
      </Grid>

      <Box sx={{ mb: 4 }}>
        <Typography variant="h5" component="h2" sx={{ mb: 3, fontWeight: 'bold' }}>
          热门课程
        </Typography>
        
        <Grid container spacing={2}>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', width: '100%', py: 4 }}>
              <CircularProgress />
            </Box>
          ) : (
            allCourses
              .sort((a, b) => (b.reviewCount || 0) - (a.reviewCount || 0))
              .slice(0, 6)
              .map(course => (
                <Grid item xs={12} sm={6} md={4} key={course.id}>
                  <Paper 
                    elevation={2}
                    sx={{ 
                      p: 2,
                      cursor: 'pointer',
                      transition: 'transform 0.2s, box-shadow 0.2s',
                      '&:hover': {
                        transform: 'translateY(-3px)',
                        boxShadow: 4
                      }
                    }}
                    onClick={() => navigate(`/courses/view/${course.id}`)}
                  >
                    <Typography variant="h6" noWrap sx={{ mb: 1 }}>
                      {course.name}
                    </Typography>
                    <Box sx={{ 
                      display: 'flex', 
                      alignItems: 'center',
                      justifyContent: 'space-between'
                    }}>
                      <Chip 
                        size="small" 
                        label={course.department}
                        sx={{
                          backgroundColor: '#3498db',
                          color: 'white',
                          fontWeight: 'bold',
                          height: '20px'
                        }}
                      />
                      <Typography 
                        variant="body2" 
                        sx={{ 
                          color: '#3498db',
                          fontWeight: 'bold'
                        }}
                      >
                        {course.reviewCount || 0} 条测评
                      </Typography>
                    </Box>
                  </Paper>
                </Grid>
              ))
          )}
        </Grid>
      </Box>

      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={handleSnackbarClose}
      >
        <Alert onClose={handleSnackbarClose} severity={snackbarSeverity}>
          {snackbarMessage}
        </Alert>
      </Snackbar>

      <style jsx global>{`
        @keyframes spin {
          0% { transform: rotate(0deg); }
          100% { transform: rotate(360deg); }
        }
      `}</style>
    </>
  );
}

export default Home; 