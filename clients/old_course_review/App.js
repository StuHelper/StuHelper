import React, { useEffect } from 'react';
import { Routes, Route, useLocation } from 'react-router-dom';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import Layout from './components/Layout';
import Home from './pages/Home';
import CourseList from './pages/CourseList';
import CourseDetail from './pages/CourseDetail';
import ReviewPost from './pages/ReviewPost';
import LatestReviews from './pages/LatestReviews';
import Search from './pages/Search';
import About from './pages/About';
import cacheService from './services/cacheService';
import { setDocumentTitle, PAGE_TITLES } from './utils/titleUtils';

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#3498db',
    },
    secondary: {
      main: '#2ecc71',
    },
    background: {
      default: '#2c3e50',
      paper: '#34495e',
    },
    text: {
      primary: '#ffffff',
      secondary: 'rgba(255, 255, 255, 0.7)',
    },
    divider: 'rgba(255, 255, 255, 0.12)',
  },
  typography: {
    fontFamily: [
      '-apple-system',
      'BlinkMacSystemFont',
      '"Segoe UI"',
      'Roboto',
      '"Helvetica Neue"',
      'Arial',
      'sans-serif',
    ].join(','),
    h1: {
      fontWeight: 700,
    },
    h2: {
      fontWeight: 700,
    },
    h3: {
      fontWeight: 600,
    },
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          borderRadius: 8,
        },
      },
    },
    MuiTextField: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            '& fieldset': {
              borderColor: 'rgba(255, 255, 255, 0.23)',
            },
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          backgroundColor: '#34495e',
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: '#34495e',
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          backgroundColor: 'rgba(52, 152, 219, 0.2)',
        },
      },
    },
  },
});

function App() {
  const location = useLocation();
  
  // 监听路由变化设置页面标题
  useEffect(() => {
    // 根据路径设置页面标题
    const path = location.pathname;
    
    if (path === '/') {
      setDocumentTitle(PAGE_TITLES.HOME);
    } else if (path === '/courses/list') {
      setDocumentTitle(PAGE_TITLES.COURSE_LIST);
    } else if (path.startsWith('/courses/view/')) {
      // 对于课程详情页，标题将在组件内部设置，因为需要课程名
    } else if (path === '/reviews/latest') {
      setDocumentTitle(PAGE_TITLES.LATEST_REVIEWS);
    } else if (path === '/reviews/post') {
      setDocumentTitle(PAGE_TITLES.REVIEW_POST);
    } else if (path === '/search') {
      setDocumentTitle(PAGE_TITLES.SEARCH);
    } else if (path === '/about') {
      setDocumentTitle(PAGE_TITLES.ABOUT);
    }
  }, [location]);

  // 检测页面刷新并清除缓存
  useEffect(() => {
    // 检查是否是页面刷新
    if (cacheService.isPageRefresh()) {
      console.log('App组件检测到页面刷新，确保使用最新数据');
      // 注意：现在isPageRefresh只在真正的页面刷新或手动触发时返回true
    } else {
      console.log('App组件正常加载，继续使用缓存数据');
    }
  }, []);

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Layout>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/courses/list" element={<CourseList />} />
          <Route path="/courses/view/:id" element={<CourseDetail />} />
          <Route path="/reviews/post" element={<ReviewPost />} />
          <Route path="/reviews/latest" element={<LatestReviews />} />
          <Route path="/search" element={<Search />} />
          <Route path="/about" element={<About />} />
        </Routes>
      </Layout>
    </ThemeProvider>
  );
}

export default App; 