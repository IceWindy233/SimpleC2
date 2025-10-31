import { Outlet, Link } from "react-router-dom";
import { useAuth } from "./services/AuthContext";

function App() {
  const { logout } = useAuth();

  return (
    <div className="d-flex flex-column vh-100">
      {/* Navbar placeholder */}
      <header className="p-3 bg-dark text-white border-bottom d-flex justify-content-between align-items-center">
        <h5 className="mb-0">SimpleC2</h5>
        <button className="btn btn-outline-light btn-sm" onClick={logout}>Logout</button>
      </header>

      <div className="d-flex flex-grow-1">
        {/* Sidebar placeholder */}
        <aside className="p-3 bg-dark border-end" style={{ width: '280px' }}>
          <ul className="nav nav-pills flex-column">
            <li className="nav-item">
              <Link to="/" className="nav-link text-white">Beacons</Link>
            </li>
            <li className="nav-item">
              <Link to="/listeners" className="nav-link text-white">Listeners</Link>
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
