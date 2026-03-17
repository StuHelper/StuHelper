import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import './index.css';
// 导入安全工具
import { applyAllSecurityMeasures } from './utils/security';
// 导入增强版安全工具
import { applyEnhancedSecurity } from './utils/enhancedSecurity';

// 应用安全措施
if (process.env.NODE_ENV === 'production' || process.env.REACT_APP_ENV === 'production') {
  // 在生产环境使用增强版安全
  applyEnhancedSecurity();
} else {
  // 在开发环境使用基本安全
  applyAllSecurityMeasures();
}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>
); 