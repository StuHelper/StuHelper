import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { 
  Box, 
  List, 
  ListItem, 
  ListItemText, 
  Typography, 
  Collapse, 
  CircularProgress,
  Chip
} from '@mui/material';
import { ExpandLess, ExpandMore } from '@mui/icons-material';
import { Link, useLocation, useParams } from 'react-router-dom';
import courseService from '../services/courseService';
import cacheService from '../services/cacheService';

const CourseListSidebar = ({ expandAll, isInSidebar = true }) => {
  const [departments, setDepartments] = useState({});
  const [loading, setLoading] = useState(true);
  const [expandedDepts, setExpandedDepts] = useState(new Set());
  const [error, setError] = useState(null);
  const requestSentRef = useRef(false);
  const location = useLocation();
  const { id: courseId } = useParams();
  
  // 获取当前选中的课程ID和所属院系
  const [selectedCourseId, setSelectedCourseId] = useState(null);
  const [selectedDept, setSelectedDept] = useState(null);

  // 使用 useCallback 优化函数
  const fetchCourses = useCallback(async () => {
    // 避免重复请求
    if (requestSentRef.current) return;
    requestSentRef.current = true;
    
    try {
      setLoading(true);
      setError(null);
      
      // 检查是否是页面刷新或首次加载
      const isRefresh = cacheService.isPageRefresh();
      
      // 如果是刷新或首次加载，强制从API获取
      const courses = await courseService.getAllCourses({ forceRefresh: isRefresh });
      
      if (courses.length === 0) {
        setError('没有获取到任何课程数据');
        return;
      }

      // 按院系分组课程
      const groupedCourses = courses.reduce((acc, course) => {
        const dept = course.department || '其他';
        if (!acc[dept]) {
          acc[dept] = [];
        }
        acc[dept].push(course);
        return acc;
      }, {});

      // 对每个院系内的课程按名称排序
      Object.keys(groupedCourses).forEach(dept => {
        groupedCourses[dept].sort((a, b) => 
          isInSidebar 
            ? a.code.localeCompare(b.code) // 侧边栏按代码排序
            : a.name.localeCompare(b.name) // 课程列表页按名称排序
        );
      });

      setDepartments(groupedCourses);
      
      // 如果有courseId，立即更新选中状态
      if (courseId) {
        updateSelectedState(courseId, groupedCourses);
      }
    } catch (error) {
      console.error('获取课程列表失败:', error);
      setError('获取课程列表失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }, [courseId, isInSidebar]);

  // 新增：更新选中状态的函数
  const updateSelectedState = useCallback((id, depts) => {
    if (!id || !depts) return;
    
    setSelectedCourseId(id);
    
    // 查找课程所属院系
    Object.entries(depts).forEach(([dept, courses]) => {
      const found = courses.find(course => course.id.toString() === id.toString());
      if (found) {
        setSelectedDept(dept);
        setExpandedDepts(new Set([dept]));
      }
    });
  }, []);

  // 当路由参数变化时更新选中状态
  useEffect(() => {
    if (courseId) {
      updateSelectedState(courseId, departments);
    } else {
      // 如果没有courseId，清除选中状态
      setSelectedCourseId(null);
      setSelectedDept(null);
      if (isInSidebar) {
        setExpandedDepts(new Set());
      }
    }
  }, [courseId, departments, updateSelectedState, isInSidebar]);

  // 初始加载时执行一次fetchCourses
  useEffect(() => {
    fetchCourses();
  }, [fetchCourses]);

  // 响应展开/收起全部（仅在课程列表页面使用）
  useEffect(() => {
    if (!isInSidebar && expandAll !== undefined) {
      if (expandAll) {
        // 展开全部
        const allDepts = Object.keys(departments);
        setExpandedDepts(new Set(allDepts));
      } else {
        // 收起全部
        setExpandedDepts(new Set());
      }
    }
  }, [expandAll, departments, isInSidebar]);

  // 使用 useCallback 优化事件处理函数
  const handleDeptClick = useCallback((dept) => {
    if (isInSidebar) {
      // 侧边栏模式：一次只能展开一个院系
      setExpandedDepts(prevDepts => {
        const newDepts = new Set();
        if (!prevDepts.has(dept)) {
          newDepts.add(dept);
        }
        return newDepts;
      });
    } else {
      // 课程列表页模式：可以同时展开多个院系
      setExpandedDepts(prevDepts => {
        const newDepts = new Set(prevDepts);
        if (prevDepts.has(dept)) {
          newDepts.delete(dept);
        } else {
          newDepts.add(dept);
        }
        return newDepts;
      });
    }
  }, [isInSidebar]);

  // 使用 useMemo 优化排序后的院系列表
  const sortedDepartments = useMemo(() => {
    return Object.entries(departments).sort((a, b) => a[0].localeCompare(b[0]));
  }, [departments]);

  // 自定义样式
  const styles = {
    listContainer: {
      width: '100%',
      maxWidth: isInSidebar ? '100%' : '1000px',
      bgcolor: 'background.paper',
      color: 'white',
      p: isInSidebar ? 1 : 2,
      overflow: 'auto'
    },
    departmentItem: {
      width: '100%',
      bgcolor: '#2c3e50',
      color: 'white',
      p: 2,
      display: 'flex',
      alignItems: 'center',
      borderRadius: '4px',
      mb: 1,
      transition: 'background-color 0.2s ease',
      '&:hover': {
        bgcolor: '#3a546e',
      }
    },
    selectedDepartmentItem: {
      bgcolor: '#3498db',
      '&:hover': {
        bgcolor: '#2980b9',
      },
      boxShadow: '0 0 8px rgba(52, 152, 219, 0.7)'
    },
    departmentText: {
      fontWeight: 'bold',
      fontSize: '0.9rem',
      color: 'rgba(255, 255, 255, 0.9)'
    },
    courseItem: {
      p: 1.5,
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      borderRadius: '4px',
      mb: 1,
      bgcolor: '#3a546e',
      transition: 'background-color 0.2s ease, transform 0.2s ease',
      '&:hover': {
        bgcolor: '#445c79',
        transform: 'translateX(5px)'
      }
    },
    selectedCourseItem: {
      bgcolor: '#2ecc71 !important',
      '&:hover': {
        bgcolor: '#27ae60 !important',
      },
      boxShadow: '0 2px 8px rgba(46, 204, 113, 0.5)',
      transform: 'translateX(5px)',
      borderLeft: '4px solid #27ae60'
    },
    courseText: {
      color: 'rgba(255, 255, 255, 0.8)',
      maxWidth: '80%'
    },
    courseCount: {
      color: 'rgba(255, 255, 255, 0.5)',
      ml: 1,
      fontSize: '0.75rem'
    },
    courseContainer: {
      display: 'flex',
      flexDirection: 'column',
      width: '100%',
      p: 1,
      mb: 2
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 2, color: 'error.main' }}>
        <Typography>
          {error}
        </Typography>
      </Box>
    );
  }

  if (sortedDepartments.length === 0) {
    return (
      <Box sx={{ p: 2, color: 'text.secondary', display: 'flex', justifyContent: 'center' }}>
        <Typography>
          没有找到任何课程数据
        </Typography>
      </Box>
    );
  }

  return (
    <Box 
      sx={{ 
        width: '100%',
        height: '100%',
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        pt: isInSidebar ? 0.5 : 1
      }}
    >
      <List 
        component="nav" 
        aria-label="课程列表"
        sx={styles.listContainer}
        dense={isInSidebar}
      >
        {sortedDepartments.map(([dept, courses]) => {
          const isDeptSelected = dept === selectedDept;
          
          return (
            <Box key={dept} sx={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', mb: isInSidebar ? 0.2 : 2 }}>
              <Box 
                onClick={() => handleDeptClick(dept)}
                sx={{
                  ...styles.departmentItem,
                  ...(isDeptSelected ? styles.selectedDepartmentItem : {}),
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  cursor: 'pointer'
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  <Typography 
                    variant="body1"
                    sx={{
                      ...styles.departmentText,
                      color: isDeptSelected ? 'white' : 'rgba(255, 255, 255, 0.9)',
                      fontWeight: isDeptSelected ? 'bold' : 'bold',
                    }}
                  >
                    {dept}
                  </Typography>
                  <Typography variant="caption" sx={styles.courseCount}>
                    {`${courses.length} 门课程`}
                  </Typography>
                </Box>
                {expandedDepts.has(dept) ? 
                  <ExpandLess sx={{ color: 'rgba(255, 255, 255, 0.7)' }} /> : 
                  <ExpandMore sx={{ color: 'rgba(255, 255, 255, 0.7)' }} />
                }
              </Box>
              <Collapse in={expandedDepts.has(dept)} timeout="auto" unmountOnExit sx={{ width: '100%' }}>
                {isInSidebar ? (
                  <List component="div" disablePadding>
                    {courses.map((course) => {
                      const isCourseSelected = course.id.toString() === selectedCourseId?.toString();
                      
                      return (
                        <ListItem 
                          key={course.id} 
                          button
                          component={Link}
                          to={`/courses/view/${course.id}`}
                          state={{ fromSidebar: true, course: course }}
                          sx={{
                            ...styles.courseItem,
                            ...(isCourseSelected ? styles.selectedCourseItem : {}),
                            transition: 'all 0.3s ease',
                            '&:hover': {
                              bgcolor: isCourseSelected ? '#27ae60 !important' : '#445c79',
                              transform: 'translateX(5px)'
                            }
                          }}
                        >
                          <ListItemText
                            primary={
                              <Typography 
                                variant="body2" 
                                noWrap
                                sx={{ 
                                  fontWeight: isCourseSelected ? 'bold' : 'normal',
                                  color: isCourseSelected ? 'white' : 'rgba(255, 255, 255, 0.8)',
                                  transition: 'all 0.3s ease'
                                }}
                              >
                                {course.name}
                              </Typography>
                            }
                          />
                        </ListItem>
                      );
                    })}
                  </List>
                ) : (
                  <Box sx={styles.courseContainer}>
                    {courses.map((course) => (
                      <Box 
                        key={course.id}
                        component={Link}
                        to={`/courses/view/${course.id}`}
                        state={{ fromSidebar: true, course: course }}
                        sx={{
                          ...styles.courseItem,
                          textDecoration: 'none',
                          color: 'inherit'
                        }}
                      >
                        <Box sx={{ display: 'flex', alignItems: 'center', flex: 1, overflow: 'hidden' }}>
                          <Typography variant="body2" noWrap sx={styles.courseText}>
                            {course.name}
                          </Typography>
                        </Box>
                        <Typography 
                          variant="caption" 
                          sx={styles.courseCount}
                        >
                          {course.reviewCount || 0} 条评价
                        </Typography>
                      </Box>
                    ))}
                  </Box>
                )}
              </Collapse>
            </Box>
          );
        })}
      </List>
    </Box>
  );
};

export default CourseListSidebar; 