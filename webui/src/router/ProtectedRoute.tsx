import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../services/AuthContext";

const ProtectedRoute = () => {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    // If not authenticated, redirect to the login page
    return <Navigate to="/login" />;
  }

  // If authenticated, render the child routes
  return <Outlet />;
};

export default ProtectedRoute;
