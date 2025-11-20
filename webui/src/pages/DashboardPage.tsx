import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { getBeacons, deleteBeacon } from '../services/api';
import { useWebSocket } from '../contexts/WebSocketContext';
import { useAuth } from '../services/AuthContext';
import type { Beacon } from '../types';

const DashboardPage = () => {
  const [beacons, setBeacons] = useState<Beacon[]>([]);
  const [error, setError] = useState('');
  const [deletingIds, setDeletingIds] = useState<string[]>([]);
  const { lastMessage } = useWebSocket();
  const { isAuthenticated } = useAuth();
  const [page, setPage] = useState(1);
  const [limit] = useState(10);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [total, setTotal] = useState(0);

  // Initial fetch for beacons
  useEffect(() => {
    const fetchBeacons = async () => {
      try {
        const data = await getBeacons(page, limit, search, status);
        setBeacons(data.beacons || []); // Ensure data is not null/undefined
        setTotal(data.total || 0);
      } catch (err) {
        setError('Failed to fetch beacons.');
        console.error(err);
      }
    };

    if (isAuthenticated) {
      fetchBeacons();
    }
  }, [isAuthenticated, page, limit, search, status]);

  // WebSocket message handling for real-time updates
  useEffect(() => {
    if (lastMessage) {
      const messages = lastMessage.data.split('\n').filter((msg: string) => msg.trim() !== '');
      messages.forEach((message: string) => {
        try {
          const event = JSON.parse(message);
          if (event.type === 'BEACON_NEW') {
            const newBeacon = event.payload as Beacon;
            setBeacons(prevBeacons => {
              // Avoid adding duplicates
              if (prevBeacons.some(b => b.BeaconID === newBeacon.BeaconID)) {
                return prevBeacons;
              }
              return [...prevBeacons, newBeacon];
            });
          } else if (event.type === 'BEACON_CHECKIN') {
            const { beacon_id, last_seen } = event.payload;
            setBeacons(prevBeacons =>
              prevBeacons.map(b =>
                b.BeaconID === beacon_id
                  ? { ...b, LastSeen: last_seen }
                  : b
              )
            );
          }
        } catch (e) {
          console.error("Failed to parse WebSocket message", e);
        }
      });
    }
  }, [lastMessage]);


  const handleDelete = async (beaconId: string) => {
    const confirmed = window.confirm(
      `确定要删除Beacon ${beaconId} 吗？\n\n这个操作将会：\n• 软删除Beacon记录\n• 通知Beacon在下次签到时退出\n• 无法撤销此操作`
    );

    if (!confirmed) return;

    try {
      // Add to deleting state to disable button
      setDeletingIds(prev => [...prev, beaconId]);
      await deleteBeacon(beaconId);
      // Remove from UI immediately for better UX
      setBeacons(prevBeacons => prevBeacons.filter(b => b.BeaconID !== beaconId));
    } catch (err) {
      setError('删除Beacon失败');
      console.error('Failed to delete beacon:', err);
    } finally {
      // Remove from deleting state
      setDeletingIds(prev => prev.filter(id => id !== beaconId));
    }
  };

  const isBeaconActive = (lastSeen: string) => {
    const lastSeenTime = new Date(lastSeen).getTime();
    const now = new Date().getTime();
    // Consider beacon active if last seen within the last 30 seconds
    return (now - lastSeenTime) < 30000;
  };

  return (
    <div>
      <h2 className="mb-4">Beacons</h2>
      <div className="d-flex justify-content-between mb-3">
        <div className="d-flex">
          <input type="text" className="form-control me-2" placeholder="Search..." value={search} onChange={e => setSearch(e.target.value)} />
          <select className="form-select me-2" value={status} onChange={e => setStatus(e.target.value)}>
            <option value="">All Statuses</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>
        <div className="d-flex align-items-center">
          <span className="me-3">Total: {total}</span>
          <button className="btn btn-sm btn-outline-secondary me-2" onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}>
            &lt;
          </button>
          <span className="me-2">Page {page}</span>
          <button className="btn btn-sm btn-outline-secondary" onClick={() => setPage(p => p + 1)} disabled={page * limit >= total}>
            &gt;
          </button>
        </div>
      </div>
      {error && <div className="alert alert-danger">{error}</div>}
      <div className="table-responsive">
        <table className="table table-dark table-hover table-sm">
          <thead>
            <tr>
              <th>Status</th>
              <th>Beacon ID</th>
              <th>OS</th>
              <th>Hostname</th>
              <th>Username</th>
              <th>Internal IP</th>
              <th>Last Seen</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {beacons.map((beacon) => (
              <tr key={beacon.ID}>
                <td>
                  <span
                    className={`badge ${isBeaconActive(beacon.LastSeen) ? 'bg-success' : 'bg-danger'}`}>
                    {isBeaconActive(beacon.LastSeen) ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td>
                  <Link to={`/beacons/${beacon.BeaconID}`}>
                    <code>{beacon.BeaconID}</code>
                  </Link>
                </td>
                <td>{beacon.OS}/{beacon.Arch}</td>
                <td>{beacon.Hostname}</td>
                <td>{beacon.Username}</td>
                <td>{beacon.InternalIP}</td>
                <td>{new Date(beacon.LastSeen).toLocaleString()}</td>
                <td>
                  <button
                    className="btn btn-sm btn-outline-danger"
                    onClick={() => handleDelete(beacon.BeaconID)}
                    disabled={deletingIds.includes(beacon.BeaconID)}
                    title="删除Beacon"
                  >
                    {deletingIds.includes(beacon.BeaconID) ? (
                      <>
                        <span className="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
                        删除中...
                      </>
                    ) : (
                      "🗑️"
                    )}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default DashboardPage;
