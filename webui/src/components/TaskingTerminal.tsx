import { useState } from 'react';
import { createTask, downloadLootFile } from '../services/api';
import type { Task } from '../types';

interface TaskingTerminalProps {
  beaconId: string;
  tasks: Task[];
  onNewTask: (task: Task) => void;
}

const TaskingTerminal = ({ beaconId, tasks, onNewTask }: TaskingTerminalProps) => {
  const [commandInput, setCommandInput] = useState('shell whoami');
  const [sleepInput, setSleepInput] = useState('10'); // Default sleep to 10 seconds
  const [isLoading, setIsLoading] = useState(false);

  const handleTaskSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!commandInput.trim()) return;

    setIsLoading(true);
    const [command, ...args] = commandInput.trim().split(/\s+/);
    const argumentsStr = args.join(' ');

    try {
      const newTask = await createTask(beaconId, command, argumentsStr);
      onNewTask(newTask); // Notify parent component
      setCommandInput('');
    } catch (error) {
      console.error("Failed to create task:", error);
      // You might want to show an error message to the user here
    } finally {
      setIsLoading(false);
    }
  };

  const handleSleepSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!sleepInput.trim()) return;

    setIsLoading(true);
    try {
      const newTask = await createTask(beaconId, 'sleep', sleepInput);
      onNewTask(newTask);
    } catch (error) {
      console.error("Failed to create sleep task:", error);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="card bg-dark border-secondary text-light mt-n1" style={{borderTopLeftRadius: 0, borderTopRightRadius: 0}}>
      <div className="card-body">
        <form onSubmit={handleTaskSubmit} className="mb-3">
          <div className="input-group">
            <span className="input-group-text bg-dark text-light border-secondary">$</span>
            <input
              type="text"
              className="form-control bg-dark text-light border-secondary"
              placeholder="e.g., shell whoami"
              value={commandInput}
              onChange={(e) => setCommandInput(e.target.value)}
              disabled={isLoading}
            />
            <button className="btn btn-primary" type="submit" disabled={isLoading}>
              {isLoading ? 'Sending...' : 'Send Task'}
            </button>
          </div>
        </form>

        <form onSubmit={handleSleepSubmit}>
          <div className="input-group">
            <span className="input-group-text bg-dark text-light border-secondary">Sleep (seconds)</span>
            <input
              type="number"
              className="form-control bg-dark text-light border-secondary"
              placeholder="e.g., 30"
              value={sleepInput}
              onChange={(e) => setSleepInput(e.target.value)}
              disabled={isLoading}
              min="1"
            />
            <button className="btn btn-info" type="submit" disabled={isLoading}>
              {isLoading ? 'Setting...' : 'Set Sleep'}
            </button>
          </div>
        </form>
      </div>

      <div className="card-footer" style={{ minHeight: '300px', maxHeight: '500px', overflowY: 'auto' }}>
        {tasks.map(task => (
          <div key={task.TaskID} className="mb-3">
            <p className="mb-0 text-muted">Tasked at {new Date(task.CreatedAt).toLocaleString()}</p>
            <p className="mb-1"><strong>$ {task.Command} {task.Arguments}</strong> (Status: {task.Status})</p>
            <pre className="bg-black p-2 rounded text-light" style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
              <code>
                {task.Status === 'completed' && task.Command === 'upload' ? (
                  <button 
                    className="btn btn-sm btn-success"
                    onClick={async () => {
                      try {
                        const filename = task.Output; // The output for an upload task is now just the filename
                        const blob = await downloadLootFile(filename);
                        
                        const url = window.URL.createObjectURL(blob);
                        const a = document.createElement('a');
                        a.href = url;
                        a.download = task.Arguments; // Use original filename as download name
                        document.body.appendChild(a);
                        a.click();
                        window.URL.revokeObjectURL(url);
                        a.remove();
                      } catch (err) {
                        console.error("Failed to download loot file:", err);
                      }
                    }}
                  >
                    Download Uploaded File
                  </button>
                ) : (task.Status === 'completed' ? task.Output : 'Waiting for output...')}
              </code>
            </pre>
          </div>
        ))}
      </div>
    </div>
  );
};

export default TaskingTerminal;
