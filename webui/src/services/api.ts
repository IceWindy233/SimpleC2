import axios from 'axios';

const apiClient = axios.create({
  baseURL: 'http://localhost:8080', // Our TeamServer API base URL
});

// Add a request interceptor to include the token in all requests
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('jwt_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

export const login = async (username, password) => {
  const response = await apiClient.post('/api/auth/login', {
    username,
    password,
  });
  return response.data; // This will return { token: "..." }
};

export const getBeacons = async () => {
  const response = await apiClient.get('/api/beacons');
  return response.data.data; // The API returns { data: [...] }
};

export const getBeacon = async (beaconId: string) => {
  const response = await apiClient.get(`/api/beacons/${beaconId}`);
  return response.data.data;
};

export const createTask = async (beaconId: string, command: string, args: string) => {
  const response = await apiClient.post(`/api/beacons/${beaconId}/tasks`, {
    command: command,
    arguments: args,
  });
  return response.data.data;
};

export const getTask = async (taskId: string) => {
  const response = await apiClient.get(`/api/tasks/${taskId}`);
  return response.data.data;
};

export const uploadFile = async (file: File) => {
  const formData = new FormData();
  formData.append('file', file);

  const response = await apiClient.post('/api/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
  return response.data; // Returns { filepath: "..." }
};

export const downloadLootFile = async (filename: string) => {
  const response = await apiClient.get(`/api/loot/${filename}`, {
    responseType: 'blob',
  });
  return response.data;
};

// Listener Management
export const getListeners = async () => {
  const response = await apiClient.get('/api/listeners');
  return response.data.data;
};

export const createListener = async (name: string, type: string, config: string) => {
  const response = await apiClient.post('/api/listeners', { name, type, config });
  return response.data.data;
};

export const deleteListener = async (name: string) => {
  await apiClient.delete(`/api/listeners/${name}`);
};

// Beacon Management
export const deleteBeacon = async (beaconId: string) => {
  await apiClient.delete(`/api/beacons/${beaconId}`);
};

// We will add other API calls here later

export default apiClient;
