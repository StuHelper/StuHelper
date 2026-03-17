<Menu.Item key={course.id}>
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
</Menu.Item> 