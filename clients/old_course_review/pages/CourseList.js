import React, { useState, useEffect, useCallback } from 'react';
import { 
  Box, 
  Typography, 
  IconButton, 
  Tooltip,
  Tabs,
  Tab,
  Container,
  Collapse,
  CircularProgress
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { Link } from 'react-router-dom';
import courseService from '../services/courseService';
import cacheService from '../services/cacheService';
import { setDocumentTitle } from '../utils/titleUtils';

// 学院组件
const DepartmentCard = ({ department, courses, expanded, onToggle }) => {
  return (
    <Box 
      sx={{ 
        width: '100%',
        maxWidth: '100%',
        bgcolor: '#34495e',
        borderRadius: '8px',
        overflow: 'hidden',
        mb: 2,
        boxShadow: '0 4px 6px rgba(0,0,0,0.3)',
      }}
    >
      <Box 
        onClick={onToggle}
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          p: 2,
          cursor: 'pointer',
          bgcolor: '#2c3e50',
          '&:hover': {
            bgcolor: '#3a546e',
          }
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <Typography 
            variant="body1"
            sx={{
              fontWeight: 'bold',
              color: 'rgba(255, 255, 255, 0.9)',
            }}
          >
            {department}
          </Typography>
          <Typography 
            variant="caption" 
            sx={{
              color: 'rgba(255, 255, 255, 0.5)',
              fontSize: '0.75rem',
              ml: 1,
            }}
          >
            {`${courses.length} 门课程`}
          </Typography>
        </Box>
        {expanded ? 
          <ExpandLessIcon sx={{ color: 'rgba(255, 255, 255, 0.7)' }} /> : 
          <ExpandMoreIcon sx={{ color: 'rgba(255, 255, 255, 0.7)' }} />
        }
      </Box>
      <Collapse in={expanded} timeout="auto" unmountOnExit>
        <Box sx={{ p: 1 }}>
          {courses.map((course) => (
            <Box 
              key={course.id}
              component={Link}
              to={`/courses/view/${course.id}`}
              state={{ fromSidebar: true, course: course }}
              sx={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                p: 1.5,
                borderRadius: '4px',
                mb: 1,
                bgcolor: '#3a546e',
                textDecoration: 'none',
                color: 'inherit',
                transition: 'background-color 0.2s ease, transform 0.2s ease',
                '&:hover': {
                  bgcolor: '#445c79',
                  transform: 'translateX(5px)'
                }
              }}
            >
              <Typography 
                variant="body2" 
                noWrap 
                sx={{ 
                  color: 'rgba(255, 255, 255, 0.8)',
                  maxWidth: '80%'
                }}
              >
                {course.name}
              </Typography>
              <Typography 
                variant="caption" 
                sx={{
                  color: 'rgba(255, 255, 255, 0.5)',
                  fontSize: '0.75rem',
                }}
              >
                {course.reviewCount || 0} 条评价
              </Typography>
            </Box>
          ))}
        </Box>
      </Collapse>
    </Box>
  );
};

const CourseList = () => {
  const [departments, setDepartments] = useState({});
  const [expandedDepts, setExpandedDepts] = useState(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // 获取课程数据
  const fetchCourses = useCallback(async () => {
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
        groupedCourses[dept].sort((a, b) => a.name.localeCompare(b.name));
      });

      setDepartments(groupedCourses);
    } catch (error) {
      console.error('获取课程列表失败:', error);
      setError('获取课程列表失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // 设置页面标题
    setDocumentTitle('课程列表');
    
    fetchCourses();
  }, [fetchCourses]);

  const handleExpandAll = () => {
    const allDepts = Object.keys(departments);
    setExpandedDepts(new Set(allDepts));
  };

  const handleCollapseAll = () => {
    setExpandedDepts(new Set());
  };

  const handleToggleDept = (dept) => {
    setExpandedDepts(prevDepts => {
      const newDepts = new Set(prevDepts);
      if (newDepts.has(dept)) {
        newDepts.delete(dept);
      } else {
        newDepts.add(dept);
      }
      return newDepts;
    });
  };

  // 排序院系
  const sortedDepartments = Object.entries(departments).sort((a, b) => a[0].localeCompare(b[0]));

  return (
    <Box 
      sx={{ 
        display: 'flex', 
        flexDirection: 'column',
        alignItems: 'center',
        minHeight: 'calc(100vh - 64px)',
        bgcolor: '#34495e',
        width: '100%',
        overflow: 'hidden'
      }}
    >
      <Box 
        sx={{ 
          width: '100%',
          maxWidth: '1200px',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          py: 2,
          bgcolor: 'transparent',
        }}
      >
        <Box 
          sx={{ 
            width: '100%',
            display: 'flex',
            justifyContent: 'flex-end',
            mb: 2
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <Tooltip title="展开全部">
              <IconButton 
                onClick={handleExpandAll}
                size="small"
                sx={{ color: '#3498db' }}
              >
                <ExpandMoreIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title="收起全部">
              <IconButton 
                onClick={handleCollapseAll}
                size="small"
                sx={{ color: '#3498db' }}
              >
                <ExpandLessIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        {loading ? (
          <Box display="flex" justifyContent="center" alignItems="center" minHeight="300px">
            <CircularProgress />
          </Box>
        ) : error ? (
          <Box display="flex" justifyContent="center" alignItems="center" minHeight="300px" p={2}>
            <Typography color="error" align="center">
              {error}
            </Typography>
          </Box>
        ) : (
          <Box 
            sx={{ 
              width: '100%',
              maxWidth: '1200px',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
            }}
          >
            {sortedDepartments.map(([dept, courses]) => (
              <DepartmentCard 
                key={dept}
                department={dept}
                courses={courses}
                expanded={expandedDepts.has(dept)}
                onToggle={() => handleToggleDept(dept)}
              />
            ))}
          </Box>
        )}
      </Box>
    </Box>
  );
};

export default CourseList; 