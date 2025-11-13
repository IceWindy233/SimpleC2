import { useState, useEffect } from 'react';
import { useParams, useNavigate } from "react-router-dom";
import { getBeacon, getTasksForBeacon, deleteBeacon } from '../services/api';
import { useWebSocket } from '../contexts/WebSocketContext';
import TaskingTerminal from '../components/TaskingTerminal';
import FileBrowser from '../components/FileBrowser';
import type { Beacon, Task } from '../types';

const BeaconDetailPage = () => {
  const { beaconId } = useParams<{ beaconId: string }>();
  const navigate = useNavigate();
  const [beacon, setBeacon] = useState<Beacon | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState('terminal');
  const [deleting, setDeleting] = useState(false);
  const { lastMessage } = useWebSocket();

  // Initial fetch for beacon details and tasks
  useEffect(() => {
    if (!beaconId) return;

    const fetchBeaconData = async () => {
      try {
        // Fetch beacon details and tasks in parallel
        const [beaconData, tasksData] = await Promise.all([
          getBeacon(beaconId),
          getTasksForBeacon(beaconId),
        ]);
        
        setBeacon(beaconData);
        // Sort tasks by creation date, newest first
        const sortedTasks = tasksData.sort((a: Task, b: Task) => 
          new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime()
        );
        setTasks(sortedTasks);

      } catch (err) {
        setError('Failed to fetch beacon data.');
        console.error(err);
      }
    };

    fetchBeaconData();
  }, [beaconId]);

  // WebSocket message handling for real-time updates
  useEffect(() => {
    if (lastMessage) {
      try {
        const event = JSON.parse(lastMessage.data);
        
        // Check if the event is relevant for this beacon
        const eventBeaconId = event.payload.beacon_id || event.payload.BeaconID;
        if (eventBeaconId !== beaconId) {
          return; // Ignore events for other beacons
        }

        if (event.type === 'TASK_OUTPUT') {
          const updatedTask = event.payload as Task;
          setTasks(prevTasks =>
            prevTasks.map(t => (t.TaskID === updatedTask.TaskID ? updatedTask : t))
          );
        
        } else if (event.type === 'BEACON_CHECKIN') {
          const { last_seen } = event.payload;
          setBeacon(prevBeacon => prevBeacon ? { ...prevBeacon, LastSeen: last_seen } : null);

        } else if (event.type === 'BEACON_METADATA_UPDATED') {
          const updatedBeacon = event.payload as Beacon;
          setBeacon(updatedBeacon);
        }
      } catch (e) {
        console.error("Failed to parse WebSocket message", e);
      }
    }
  }, [lastMessage, beaconId]);

  const handleNewTask = (newTask: Task) => {
    // Add the new task with a "dispatched" status.
    // The actual result will come in via WebSocket.
    setTasks(prevTasks => [newTask, ...prevTasks]);
  };

  const handleDeleteBeacon = async () => {
    if (!beacon || !beaconId) return;

    const confirmed = window.confirm(
      `确定要删除Beacon ${beacon.BeaconID} 吗？\n\n这个操作将会：\n• 软删除Beacon记录\n• 通知Beacon在下次签到时退出\n• 无法撤销此操作`
    );

    if (!confirmed) return;

    try {
      setDeleting(true);
      await deleteBeacon(beaconId);
      alert('Beacon已成功删除，将在下次签到时退出');
      navigate('/'); // 重定向到Beacons列表页面
    } catch (err) {
      setError('删除Beacon失败');
      console.error('Failed to delete beacon:', err);
    } finally {
      setDeleting(false);
    }
  };

  if (error) {
    return <div className="alert alert-danger">{error}</div>;
  }

  if (!beacon) {
    return <div>Loading beacon details...</div>;
  }

  return (
    <div>
      <h2 className="mb-3">Beacon Interaction: <code>{beacon.BeaconID}</code></h2>
      
      <div className="card bg-dark border-secondary text-light mb-4">
        <div className="card-header d-flex justify-content-between align-items-center">
          <span>Beacon Details</span>
          <button
            className="btn btn-sm btn-danger"
            onClick={handleDeleteBeacon}
            disabled={deleting}
            title="删除Beacon"
          >
            {deleting ? (
              <>
                <span className="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
                删除中...
              </>
            ) : (
              <>
                <i className="bi bi-trash me-1"></i>
                删除Beacon
              </>
            )}
          </button>
        </div>
        <div className="card-body">
          <div className="row">
            <div className="col-md-6">
              <p><strong>Hostname:</strong> {beacon.Hostname}</p>
              <p><strong>Username:</strong> {beacon.Username}</p>
              <p><strong>Operating System:</strong> {beacon.OS} ({beacon.Arch})</p>
            </div>
            <div className="col-md-6">
              <p><strong>Last Seen:</strong> {new Date(beacon.LastSeen).toLocaleString()}</p>
              <p><strong>First Seen:</strong> {new Date(beacon.FirstSeen).toLocaleString()}</p>
              <p><strong>Listener:</strong> {beacon.Listener}</p>
              <p><strong>Sleep:</strong> {beacon.Sleep} seconds</p>
            </div>
          </div>
        </div>
      </div>

      <ul className="nav nav-tabs">
        <li className="nav-item">
          <button className={`nav-link ${activeTab === 'terminal' ? 'active bg-dark text-light' : 'text-secondary'}`} onClick={() => setActiveTab('terminal')}>Terminal</button>
        </li>
        <li className="nav-item">
          <button className={`nav-link ${activeTab === 'filebrowser' ? 'active bg-dark text-light' : 'text-secondary'}`} onClick={() => setActiveTab('filebrowser')}>File Browser</button>
        </li>
      </ul>

      <div className="tab-content">
        <div className={`tab-pane fade ${activeTab === 'terminal' ? 'show active' : ''}`}>
          {beaconId && <TaskingTerminal beaconId={beaconId} tasks={tasks} onNewTask={handleNewTask} />}
        </div>
        <div className={`tab-pane fade ${activeTab === 'filebrowser' ? 'show active' : ''}`}>
          {beacon && <FileBrowser beaconId={beacon.BeaconID} />}
        </div>
      </div>
    </div>
  );
};

export default BeaconDetailPage;
