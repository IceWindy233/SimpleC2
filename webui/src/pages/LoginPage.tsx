import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../services/AuthContext';
import { setBaseURL, getBaseURL } from '../services/api';

const LoginPage = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [teamServerURL, setTeamServerURL] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const navigate = useNavigate();
  const { login } = useAuth();

  // 加载保存的TeamServer URL
  useEffect(() => {
    const savedURL = getBaseURL();
    // 移除协议部分，只显示主机名和端口
    const displayURL = savedURL.replace(/^https?:\/\//, '');
    setTeamServerURL(displayURL);
  }, []);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      // 确保URL包含协议
      const fullURL = teamServerURL.startsWith('http')
        ? teamServerURL
        : `http://${teamServerURL}`;

      // 设置API基础URL
      setBaseURL(fullURL);

      await login(username, password);
      navigate('/'); // Redirect to main page on successful login
    } catch (err) {
      setError('Login failed. Please check your username, password, and TeamServer URL.');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="d-flex align-items-center justify-content-center vh-100">
      <div className="card bg-dark text-light p-4" style={{ width: '400px' }}>
        <div className="card-body">
          <h3 className="card-title text-center mb-4">SimpleC2 Login</h3>
          <form onSubmit={handleLogin}>
            {error && <div className="alert alert-danger">{error}</div>}
            <div className="mb-3">
              <label htmlFor="teamserver" className="form-label">TeamServer URL</label>
              <input
                type="text"
                className="form-control bg-secondary text-light border-secondary"
                id="teamserver"
                value={teamServerURL}
                onChange={(e) => setTeamServerURL(e.target.value)}
                placeholder="localhost:8080"
                required
              />
            </div>
            <div className="mb-3">
              <label htmlFor="username" className="form-label">Username</label>
              <input
                type="text"
                className="form-control bg-secondary text-light border-secondary"
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="mb-3">
              <label htmlFor="password" className="form-label">Password</label>
              <input
                type="password"
                className="form-control bg-secondary text-light border-secondary"
                id="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <div className="d-grid">
              <button type="submit" className="btn btn-primary" disabled={loading}>
                {loading ? 'Logging in...' : 'Login'}
              </button>
            </div>
          </form>
          <div className="text-center mt-3">
             <p className="text-muted small">Enter any username and the pre-shared operator password.</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default LoginPage;
