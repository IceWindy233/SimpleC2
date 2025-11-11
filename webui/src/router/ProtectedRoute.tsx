import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../services/AuthContext";
import { WebSocketProvider } from "../contexts/WebSocketContext";

const ProtectedRoute = () => {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    // If not authenticated, redirect to the login page
    return <Navigate to="/login" />;
  }

  // If authenticated, wrap the child routes in the WebSocketProvider
  return (
    <WebSocketProvider>
      <Outlet />
    </WebSocketProvider>
  );
};

export default ProtectedRoute;
