import axios from 'axios';

// 默认API地址
const DEFAULT_API_URL = 'http://localhost:8080';

// API客户端实例
export const api = axios.create({
  baseURL: '/api',
});

// 获取保存的API基础URL
export const getBaseURL = (): string => {
  return localStorage.getItem('api_base_url') || DEFAULT_API_URL;
};

// 设置API基础URL
export const setBaseURL = (url: string): void => {
  localStorage.setItem('api_base_url', url);
  // 更新api实例的baseURL
  api.defaults.baseURL = `${url}/api`;
};

// 初始化API配置
const initializeAPI = () => {
  const baseURL = getBaseURL();
  setBaseURL(baseURL);

  // 从localStorage初始化token
  const token = localStorage.getItem('token');
  if (token) {
    setAuthToken(token);
  }
};

// 设置认证token
export const setAuthToken = (token: string | null) => {
  if (token) {
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  } else {
    delete api.defaults.headers.common['Authorization'];
  }
};

// 初始化API
initializeAPI();

export const getBeacons = async () => {
  const response = await api.get('/beacons');
  return response.data.data;
};

export const getBeacon = async (beaconId: string) => {
  const response = await api.get(`/beacons/${beaconId}`);
  return response.data.data;
};

export const createTask = async (beaconId: string, command: string, args: string) => {
  const response = await api.post(`/beacons/${beaconId}/tasks`, {
    command: command,
    arguments: args,
  });
  return response.data.data;
};

export const getTask = async (taskId: string) => {
  const response = await api.get(`/tasks/${taskId}`);
  return response.data.data;
};

export const getTasksForBeacon = async (beaconId: string) => {
  const response = await api.get(`/beacons/${beaconId}/tasks`);
  return response.data.data;
};

// --- Chunked File Upload ---

export const uploadInit = async (filename: string) => {
  const response = await api.post('/upload/init', { filename });
  return response.data;
};

export const uploadChunk = async (uploadId: string, chunkNumber: number, chunk: Blob) => {
  const response = await api.post('/upload/chunk', chunk, {
    headers: {
      'Content-Type': 'application/octet-stream',
      'X-Upload-ID': uploadId,
      'X-Chunk-Number': chunkNumber.toString(),
    },
  });
  return response.data;
};

export const uploadComplete = async (uploadId: string, filename: string) => {
  const response = await api.post('/upload/complete', { upload_id: uploadId, filename });
  return response.data;
};


export const downloadLootFile = async (filename: string) => {
  const response = await api.get(`/loot/${filename}`, {
    responseType: 'blob',
  });
  return response.data;
};

// Listener Management
export const getListeners = async () => {
  const response = await api.get('/listeners');
  return response.data.data;
};

export const createListener = async (name: string, type: string, config: string) => {
  const response = await api.post('/listeners', { name, type, config });
  return response.data.data;
};

export const deleteListener = async (name: string) => {
  await api.delete(`/listeners/${name}`);
};

// Beacon Management
export const deleteBeacon = async (beaconId: string) => {
  await api.delete(`/beacons/${beaconId}`);
};

