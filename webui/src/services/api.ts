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

// 设置认证token
export const setAuthToken = (token: string | null) => {
  if (token) {
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  } else {
    delete api.defaults.headers.common['Authorization'];
  }
};

// Initial setup when the module loads
const initialBaseURL = getBaseURL();
setBaseURL(initialBaseURL);

const initialToken = localStorage.getItem('token');
if (initialToken) {
  setAuthToken(initialToken);
}


export const getBeacons = async (page: number, limit: number, search: string, status: string) => {
  const response = await api.get('/beacons', {
    params: {
      page,
      limit,
      search,
      status,
    },
  });
  if (response.data.success) {
    return {
      beacons: response.data.data,
      total: response.data.meta?.total || 0
    };
  } else {
    throw new Error(response.data.error.message);
  }
};

export const getBeacon = async (beaconId: string) => {
  const response = await api.get(`/beacons/${beaconId}`);
  if (response.data.success) {
    return response.data.data;
  } else {
    throw new Error(response.data.error.message);
  }
};

export const createTask = async (beaconId: string, command: string, args: string) => {
  const response = await api.post(`/beacons/${beaconId}/tasks`, {
    command: command,
    arguments: args,
  });
  if (response.data.success) {
    return response.data.data;
  } else {
    throw new Error(response.data.error.message);
  }
};

export const getTask = async (taskId: string) => {
  const response = await api.get(`/tasks/${taskId}`);
  if (response.data.success) {
    return response.data.data;
  } else {
    throw new Error(response.data.error.message);
  }
};

export const getTasksForBeacon = async (beaconId: string) => {
  const response = await api.get(`/beacons/${beaconId}/tasks`);
  if (response.data.success) {
    return response.data.data;
  } else {
    throw new Error(response.data.error.message);
  }
};

// --- Chunked File Upload ---

export const uploadInit = async (filename: string) => {
  const response = await api.post('/upload/init', { filename });
  if (response.data.success) {
    return response.data.data;
  } else {
    throw new Error(response.data.error.message || 'Failed to initialize upload');
  }
};

export const uploadChunk = async (uploadId: string, chunkNumber: number, chunk: Blob) => {
  const response = await api.post('/upload/chunk', chunk, {
    headers: {
      'Content-Type': 'application/octet-stream',
      'X-Upload-ID': uploadId,
      'X-Chunk-Number': chunkNumber.toString(),
    },
  });
  if (response.status === 200) {
    return;
  } else {
    throw new Error(response.data.error.message || 'Failed to upload chunk');
  }
};

export const uploadComplete = async (uploadId: string, filename: string) => {
  const response = await api.post('/upload/complete', { upload_id: uploadId, filename });
  if (response.data.success) {
    return response.data.data;
  } else {
    throw new Error(response.data.error.message || 'Failed to complete upload');
  }
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
  if (response.data.success) {
    return response.data.data;
  } else {
    throw new Error(response.data.error.message);
  }
};

export const createListener = async (name: string, type: string, config: string) => {
  const response = await api.post('/listeners', { name, type, config });
  if (response.data.success) {
    return response.data.data;
  } else {
    throw new Error(response.data.error.message);
  }
};

export const deleteListener = async (name: string) => {
  const response = await api.delete(`/listeners/${name}`);
  if (response.status === 204) {
    return;
  } else {
    throw new Error(response.data.error.message);
  }
};

// Beacon Management
export const deleteBeacon = async (beaconId: string) => {
  const response = await api.delete(`/beacons/${beaconId}`);
  if (response.status === 204) {
    return;
  } else {
    throw new Error(response.data.error.message);
  }
};

