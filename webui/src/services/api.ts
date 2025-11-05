import axios from 'axios';

export const api = axios.create({
  baseURL: '/api',
});

export const setAuthToken = (token: string | null) => {
  if (token) {
    api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  } else {
    delete api.defaults.headers.common['Authorization'];
  }
};

// Initialize the token from localStorage on startup
const token = localStorage.getItem('token');
if (token) {
  setAuthToken(token);
}

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

export const uploadFile = async (file: File) => {
  const formData = new FormData();
  formData.append('file', file);

  const response = await api.post('/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
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