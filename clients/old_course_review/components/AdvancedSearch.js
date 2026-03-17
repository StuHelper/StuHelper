import React, { useState } from 'react';
import { Card, Link } from 'antd';

const AdvancedSearch = () => {
  const [courses, setCourses] = useState([]);

  return (
    <div>
      {courses.map(course => (
        <Card
          key={course.id}
          hoverable
          style={{ marginBottom: 16 }}
          actions={[
            <Link to={`/courses/view/${course.id}`}>查看详情</Link>
          ]}
        >
          <Card.Meta
            title={
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <span style={{ 
                  display: 'inline-block', 
                  padding: '2px 6px', 
                  borderRadius: '4px', 
                  fontSize: '12px', 
                  marginRight: '8px',
                  backgroundColor: course.isGraduate ? '#ffd6e7' : '#e6f7ff',
                  color: course.isGraduate ? '#eb2f96' : '#1890ff'
                }}>
                  {course.isGraduate ? '研' : '本'}
                </span>
                {course.name}
              </div>
            }
            description={
              <div>
                <p>{course.teacher} · {course.department}</p>
                <p>
                  <span style={{ color: '#1890ff' }}>{course.reviewCount}条测评</span>
                </p>
              </div>
            }
          />
        </Card>
      ))}
    </div>
  );
};

export default AdvancedSearch; 