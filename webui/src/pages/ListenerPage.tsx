import { useState, useEffect } from 'react';
import { getListeners, createListener, deleteListener } from '../services/api';

interface Listener {
  ID: number;
  Name: string;
  Type: string;
  Config: string;
  CreatedAt: string;
}

const ListenerPage = () => {
  const [listeners, setListeners] = useState<Listener[]>([]);
  const [error, setError] = useState('');

  // Form state
  const [name, setName] = useState('http-listener-1');
  const [type, setType] = useState('http');
  const [config, setConfig] = useState('{"port":8888}');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const fetchListeners = async () => {
    try {
      const data = await getListeners();
      setListeners(data || []);
    } catch (err) {
      setError('Failed to fetch listeners.');
      console.error(err);
    }
  };

  useEffect(() => {
    fetchListeners();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError('');
    try {
      await createListener(name, type, config);
      fetchListeners(); // Refresh the list
      // Clear form
      setName('');
      setType('');
      setConfig('');
    } catch (err) {
      setError('Failed to create listener.');
      console.error(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async (listenerName: string) => {
    if (window.confirm(`Are you sure you want to delete listener '${listenerName}'?`)) {
      try {
        await deleteListener(listenerName);
        fetchListeners(); // Refresh the list
      } catch (err) {
        setError('Failed to delete listener.');
        console.error(err);
      }
    }
  };

  return (
    <div>
      <h2 className="mb-4">Listener Management</h2>
      
      <div className="card bg-dark border-secondary text-light mb-4">
        <div className="card-header">Create New Listener</div>
        <div className="card-body">
          <form onSubmit={handleCreate}>
            {error && <div className="alert alert-danger">{error}</div>}
            <div className="row">
              <div className="col-md-4 mb-3">
                <label htmlFor="name" className="form-label">Name</label>
                <input type="text" id="name" className="form-control bg-dark text-light border-secondary" value={name} onChange={e => setName(e.target.value)} required />
              </div>
              <div className="col-md-4 mb-3">
                <label htmlFor="type" className="form-label">Type</label>
                <input type="text" id="type" className="form-control bg-dark text-light border-secondary" value={type} onChange={e => setType(e.target.value)} required />
              </div>
              <div className="col-md-4 mb-3">
                <label htmlFor="config" className="form-label">Config (JSON String)</label>
                <input type="text" id="config" className="form-control bg-dark text-light border-secondary" value={config} onChange={e => setConfig(e.target.value)} />
              </div>
            </div>
            <button type="submit" className="btn btn-primary" disabled={isSubmitting}>{isSubmitting ? 'Creating...' : 'Create Listener'}</button>
          </form>
        </div>
      </div>

      <h3 className="mb-3">Existing Listeners</h3>
      <div className="table-responsive">
        <table className="table table-dark table-hover table-sm">
          <thead>
            <tr>
              <th>Name</th>
              <th>Type</th>
              <th>Config</th>
              <th>Created At</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {listeners.map(listener => (
              <tr key={listener.ID}>
                <td><code>{listener.Name}</code></td>
                <td>{listener.Type}</td>
                <td><code>{listener.Config}</code></td>
                <td>{new Date(listener.CreatedAt).toLocaleString()}</td>
                <td>
                  <button className="btn btn-sm btn-danger" onClick={() => handleDelete(listener.Name)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default ListenerPage;
