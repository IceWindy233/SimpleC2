import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { getBeacons, deleteBeacon } from '../services/api';

// Define the type for a single beacon object based on our API response
interface Beacon {
  ID: number;
  BeaconID: string;
  OS: string;
  Arch: string;
  Hostname: string;
  Username: string;
  InternalIP: string;
  LastSeen: string;
  Status: string;
}

const DashboardPage = () => {
  const [beacons, setBeacons] = useState<Beacon[]>([]);
  const [error, setError] = useState('');
  const [deletingIds, setDeletingIds] = useState<string[]>([]);

  const fetchBeacons = async () => {
    try {
      const data = await getBeacons();
      setBeacons(data || []); // Ensure data is not null/undefined
    } catch (err) {
      setError('Failed to fetch beacons.');
      console.error(err);
    }
  };

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

  useEffect(() => {
    fetchBeacons(); // Fetch immediately on component mount
    const intervalId = setInterval(fetchBeacons, 5000); // Refresh every 5 seconds

    return () => clearInterval(intervalId); // Cleanup interval on component unmount
  }, []);

  const isBeaconActive = (lastSeen: string) => {
    const lastSeenTime = new Date(lastSeen).getTime();
    const now = new Date().getTime();
    // Consider beacon active if last seen within the last 30 seconds
    return (now - lastSeenTime) < 30000;
  };

  return (
    <div>
      <h2 className="mb-4">Beacons</h2>
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
