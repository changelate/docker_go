import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { Table } from 'antd';
import 'antd/dist/reset.css';

interface ContainerStatus {
  ip: string;
  ping_time: number;
  last_success: string;
}

const App: React.FC = () => {
  const [data, setData] = useState<ContainerStatus[]>([]);

  const fetchData = async () => {
    try {
      // URL backend задается через переменную окружения REACT_APP_BACKEND_URL
      const backendURL = process.env.REACT_APP_BACKEND_URL || 'http://localhost:8080';
      const response = await axios.get<ContainerStatus[]>(`${backendURL}/status`);
      setData(response.data);
    } catch (error) {
      console.error('Error fetching data: ', error);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  const columns = [
    {
      title: 'IP адрес',
      dataIndex: 'ip',
      key: 'ip'
    },
    {
      title: 'Время пинга (мс)',
      dataIndex: 'ping_time',
      key: 'ping_time'
    },
    {
      title: 'Дата последней успешной попытки',
      dataIndex: 'last_success',
      key: 'last_success',
      render: (text: string) => new Date(text).toLocaleString()
    }
  ];

  return (
    <div style={{ padding: 20 }}>
      <h1>Статус контейнеров</h1>
      <Table dataSource={data} columns={columns} rowKey="ip" />
    </div>
  );
};

export default App;
