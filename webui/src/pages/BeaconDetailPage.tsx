import { useState, useEffect } from 'react';
import { useParams, useNavigate } from "react-router-dom";
import { getBeacon, getTask, deleteBeacon } from '../services/api';
import TaskingTerminal from '../components/TaskingTerminal';
import FileBrowser from '../components/FileBrowser';

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
  FirstSeen: string;
  Listener: string;
  Sleep: number; // New field for sleep interval
}

// Define the type for a single task object
interface Task {
  TaskID: string;
  Command: string;
  Arguments: string;
  Status: string;
  Output: string;
  CreatedAt: string;
}

const BeaconDetailPage = () => {
  const { beaconId } = useParams<{ beaconId: string }>();
  const navigate = useNavigate();
  const [beacon, setBeacon] = useState<Beacon | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState('terminal');
  const [deleting, setDeleting] = useState(false);

    // Fetch main beacon details and set up refresh

    useEffect(() => {

      if (!beaconId) return;

  

      const fetchBeaconDetails = async () => {

        try {

          const data = await getBeacon(beaconId);

          setBeacon(data);

        } catch (err) {

          setError('Failed to fetch beacon details.');

          console.error(err);

        }

      };

  

      fetchBeaconDetails(); // Fetch immediately on mount

      const intervalId = setInterval(fetchBeaconDetails, 5000); // Refresh every 5 seconds

  

      return () => clearInterval(intervalId); // Cleanup on unmount

    }, [beaconId]);

  const pollForResult = (taskId: string) => {
    const startTime = Date.now();
    const timeout = 60000; // 60 seconds timeout

    const intervalId = setInterval(async () => {
      // Check for timeout
      if (Date.now() - startTime > timeout) {
        clearInterval(intervalId);
        setTasks(prevTasks =>
          prevTasks.map(t =>
            t.TaskID === taskId ? { ...t, Status: "Timeout" } : t
          )
        );
        console.warn(`Polling for task ${taskId} timed out.`);
        return;
      }

      try {
        const updatedTask = await getTask(taskId);
        if (updatedTask.Status === 'completed' || updatedTask.Status === 'error') {
          setTasks(prevTasks => 
            prevTasks.map(t => t.TaskID === taskId ? updatedTask : t)
          );
          clearInterval(intervalId);
        }
      } catch (error) {
        console.error(`Failed to poll for task ${taskId}:`, error);
        clearInterval(intervalId); // Stop polling on error
      }
    }, 3000); // Poll every 3 seconds
  };

  const handleNewTask = (newTask: Task) => {
    setTasks(prevTasks => [newTask, ...prevTasks]);
    pollForResult(newTask.TaskID);
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
          {beacon && <FileBrowser beaconId={beacon.BeaconID} os={beacon.OS} />}
        </div>
      </div>
    </div>
  );
};

export default BeaconDetailPage;
