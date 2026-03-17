const renderOption = (props, option) => (
  <li {...props}>
    <span style={{ 
      display: 'inline-block', 
      padding: '2px 6px', 
      borderRadius: '4px', 
      fontSize: '12px', 
      marginRight: '8px',
      backgroundColor: option.isGraduate ? '#ffd6e7' : '#e6f7ff',
      color: option.isGraduate ? '#eb2f96' : '#1890ff'
    }}>
      {option.isGraduate ? '研' : '本'}
    </span>
    {option.name}
  </li>
); 