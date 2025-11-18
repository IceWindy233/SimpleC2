import { Outlet, Link, useNavigate } from "react-router-dom";
import { useAuth } from "./services/AuthContext";
import { useWebSocket } from "./contexts/WebSocketContext";
import { useEffect } from "react";

function App() {
  const { logout, isAuthenticated } = useAuth();
  const { disconnect } = useWebSocket();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/login');
    }
  }, [isAuthenticated, navigate]);

  const handleLogout = () => {
    disconnect(); // Disconnect WebSocket first
    logout();     // Then clear auth state
  };

  return (
    <div className="d-flex flex-column vh-100">
      {/* Navbar placeholder */}
      <header className="p-3 bg-dark text-white border-bottom d-flex justify-content-between align-items-center">
        <h5 className="mb-0">SimpleC2</h5>
        <button className="btn btn-outline-light btn-sm" onClick={handleLogout}>Logout</button>
      </header>

      <div className="d-flex flex-grow-1">
        {/* Sidebar placeholder */}
        <aside className="p-3 bg-dark border-end" style={{ width: '280px' }}>
          <ul className="nav nav-pills flex-column">
            <li className="nav-item">
              <Link to="/" className="nav-link text-white">
                <i className="bi bi-activity me-2"></i>Beacons
              </Link>
            </li>
            <li className="nav-item">
              <Link to="/listeners" className="nav-link text-white">
                <i className="bi bi-wifi me-2"></i>Listeners
              </Link>
            </li>
          </ul>
        </aside>

        {/* Main content area */}
        <main className="p-4 flex-grow-1">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

export default App;
