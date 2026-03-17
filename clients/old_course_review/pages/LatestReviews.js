import React, { useState, useEffect } from 'react';
import { 
  Box, 
  Typography, 
  Container, 
  Card,
  CardContent,
  Divider,
  IconButton,
  Chip,
  CircularProgress,
  Alert,
  Link,
  Tooltip
} from '@mui/material';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import ThumbDownIcon from '@mui/icons-material/ThumbDown';
import SentimentVeryDissatisfiedIcon from '@mui/icons-material/SentimentVeryDissatisfied';
import SentimentDissatisfiedIcon from '@mui/icons-material/SentimentDissatisfied';
import SentimentNeutralIcon from '@mui/icons-material/SentimentNeutral';
import SentimentSatisfiedIcon from '@mui/icons-material/SentimentSatisfied';
import SentimentVerySatisfiedIcon from '@mui/icons-material/SentimentVerySatisfied';
import WarningIcon from '@mui/icons-material/Warning';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import InfoIcon from '@mui/icons-material/Info';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { getLatestReviews, voteReview } from '../services/reviewService';
import { getBrowserFingerprint } from '../utils/fingerprintUtils';
import { setDocumentTitle } from '../utils/titleUtils';

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

function LatestReviews() {
  const navigate = useNavigate();
  const [reviews, setReviews] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    // 设置页面标题
    setDocumentTitle('最新测评');
    
    const fetchLatestReviews = async () => {
      try {
        setLoading(true);
        const { reviews: latestReviews, error: reviewError } = await getLatestReviews(20);
        
        if (reviewError) {
          setError(reviewError);
        } else {
          setReviews(latestReviews);
        }
      } catch (err) {
        console.error('获取最新测评失败:', err);
        setError('获取最新测评失败，请稍后重试');
      } finally {
        setLoading(false);
      }
    };

    fetchLatestReviews();
  }, []);

  if (loading) {
    return (
      <Container maxWidth="md" sx={{ py: 4, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress />
      </Container>
    );
  }

  if (error) {
    return (
      <Container maxWidth="md" sx={{ py: 4 }}>
        <Alert severity="error" sx={{ mb: 3 }}>{error}</Alert>
        <Typography variant="h4" component="h1" sx={{ mb: 4, fontWeight: 'bold' }}>
          最新测评
        </Typography>
      </Container>
    );
  }

  if (reviews.length === 0) {
    return (
      <Container maxWidth="md" sx={{ py: 4 }}>
        <Typography variant="h4" component="h1" sx={{ mb: 4, fontWeight: 'bold' }}>
          最新测评
        </Typography>
        <Alert severity="info">暂无测评数据</Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="md" sx={{ py: 4 }}>
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
          最新测评
        </Typography>
      </Box>

      {reviews.map((review) => (
        <ReviewCard key={review.id} review={review} />
      ))}
    </Container>
  );
}

export default LatestReviews; 