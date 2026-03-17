import React, { useState } from 'react';
import { Link as RouterLink, useLocation } from 'react-router-dom';
import {
  AppBar,
  Toolbar,
  Typography,
  Container,
  Box,
  Link,
  Stack,
  Fade,
  IconButton,
  Tooltip,
  Snackbar,
  Alert
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import CourseListSidebar from './CourseListSidebar';
import cacheService from '../services/cacheService';

function Layout({ children }) {
  const location = useLocation();
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  
  const isViewingCourse = location.pathname.includes('/courses/view/');
  const isListPage = location.pathname === '/courses/list';
  const isHomePage = location.pathname === '/';
  const isReviewPostPage = location.pathname === '/reviews/post';
  const isLatestReviewsPage = location.pathname === '/reviews/latest';
  const isSearchPage = location.pathname === '/search';
  const isAboutPage = location.pathname === '/about';
  const showSidebar = isViewingCourse || isListPage;

  const navLinks = [
    { text: '课程列表', path: '/courses/list' },
    { text: '最新', path: '/reviews/latest' },
    { text: '搜索', path: '/search' },
    { text: '发布测评', path: '/reviews/post' },
    { text: '关于', path: '/about' }
  ];
  
  // 手动刷新缓存
  const handleRefreshCache = () => {
    cacheService.forceRefreshCache();
    setSnackbarOpen(true);
    // 刷新页面以获取最新数据
    window.location.reload();
  };
  
  // 关闭提示
  const handleCloseSnackbar = () => {
    setSnackbarOpen(false);
  };

  return (
    <Box sx={{ 
      display: 'flex', 
      flexDirection: 'column', 
      minHeight: '100vh',
      bgcolor: isHomePage ? '#2c3e50' : 'background.default',
      paddingTop: '64px'
    }}>
      <AppBar 
        position="fixed" 
        sx={{ 
          bgcolor: isHomePage ? '#2c3e50' : 'background.default',
          borderBottom: 1,
          borderColor: 'divider',
          backdropFilter: 'blur(8px)',
          background: isHomePage ? 
            'linear-gradient(180deg, rgba(44, 62, 80, 0.95) 0%, rgba(44, 62, 80, 0.95) 100%)' : 
            'linear-gradient(180deg, rgba(44, 62, 80, 0.95) 0%, rgba(44, 62, 80, 0.95) 100%)',
        }}
        elevation={0}
      >
        <Toolbar sx={{ gap: 2 }}>
          <Link
            component={RouterLink}
            to="/"
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1,
              color: 'white',
              textDecoration: 'none',
              '&:hover': { opacity: 0.8 }
            }}
          >
            <Typography variant="h6" component="span">
              课程测评
            </Typography>
          </Link>
          <Stack direction="row" spacing={3} sx={{ ml: 2 }}>
            {navLinks.map((link) => (
              <Link
                key={link.path}
                component={RouterLink}
                to={link.path}
                sx={{
                  color: 'white',
                  textDecoration: 'none',
                  fontSize: '0.9rem',
                  opacity: location.pathname === link.path ? 1 : 0.8,
                  position: 'relative',
                  '&:hover': { 
                    opacity: 1,
                  },
                  '&::after': location.pathname === link.path ? {
                    content: '""',
                    position: 'absolute',
                    bottom: -2,
                    left: 0,
                    width: '100%',
                    height: '2px',
                    backgroundColor: '#3498db',
                    borderRadius: '2px'
                  } : {}
                }}
              >
                {link.text}
              </Link>
            ))}
          </Stack>
          
          <Box sx={{ flexGrow: 1 }} />
          
          <Tooltip title="刷新数据">
            <IconButton 
              color="inherit" 
              onClick={handleRefreshCache}
              sx={{ opacity: 0.8, '&:hover': { opacity: 1 } }}
            >
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Toolbar>
      </AppBar>
      
      <Snackbar
        open={snackbarOpen}
        autoHideDuration={3000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
      >
        <Alert onClose={handleCloseSnackbar} severity="success" sx={{ width: '100%' }}>
          正在刷新数据...
        </Alert>
      </Snackbar>

      {isHomePage || isReviewPostPage || isLatestReviewsPage || isSearchPage || isAboutPage ? (
        <Container 
          component="main" 
          sx={{ 
            mt: isHomePage ? 8 : 4,
            mb: 4, 
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: isHomePage ? 'center' : 'stretch',
            color: 'white',
            textAlign: isHomePage ? 'center' : 'left'
          }}
        >
          {children}
        </Container>
      ) : (
        <Box sx={{ display: 'flex', flex: 1, position: 'relative', height: 'calc(100vh - 64px)', overflow: 'hidden' }}>
          <Box
            sx={{
              width: isViewingCourse ? '30%' : '100%',
              position: 'absolute',
              left: 0,
              top: 0,
              bottom: 0,
              transition: 'width 0.4s ease-in-out',
              borderRight: isViewingCourse ? 1 : 0,
              borderColor: 'divider',
              bgcolor: 'background.paper',
              display: 'flex',
              justifyContent: isListPage ? 'center' : 'flex-start',
              overflow: 'auto',
              zIndex: 10,
              boxShadow: isViewingCourse ? '2px 0 10px rgba(0, 0, 0, 0.1)' : 'none',
              transform: isViewingCourse ? 'translateX(0)' : 'translateX(0)',
            }}
          >
            {isListPage ? (
              <Box sx={{ width: '100%', maxWidth: '1000px' }}>
                {children}
              </Box>
            ) : (
              <CourseListSidebar />
            )}
          </Box>
          {isViewingCourse && (
            <Fade in={true} timeout={500}>
              <Box
                sx={{
                  width: '100%',
                  paddingLeft: '30%',
                  bgcolor: '#2c3e50',
                  transition: 'all 0.4s ease-in-out',
                  height: '100%',
                  overflow: 'auto',
                }}
              >
                {children}
              </Box>
            </Fade>
          )}
        </Box>
      )}

      <Box
        component="footer"
        sx={{
          py: 2,
          px: 2,
          mt: 'auto',
          bgcolor: isHomePage ? 'transparent' : 'background.paper',
          borderTop: isHomePage ? 0 : 1,
          borderColor: 'divider',
          color: isHomePage ? 'white' : 'text.secondary'
        }}
      >
        <Container maxWidth="sm">
          <Typography variant="body2" align="center" sx={{ opacity: 0.7 }}>
            © {new Date().getFullYear()} 评课社区@BUAA
          </Typography>
          <Typography variant="caption" align="center" display="block" sx={{ mt: 1, opacity: 0.5 }}>
            本网站仅供北航学子浏览和发布课程评价，与任何官方机构或组织无关，内容真实性由发布者负责
          </Typography>
        </Container>
      </Box>
    </Box>
  );
}

export default Layout; 