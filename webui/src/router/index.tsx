import { createBrowserRouter } from "react-router-dom";
import App from "../App";
import LoginPage from "../pages/LoginPage";
import ProtectedRoute from "./ProtectedRoute";
import DashboardPage from "../pages/DashboardPage";
import ListenerPage from "../pages/ListenerPage";
import BeaconDetailPage from "../pages/BeaconDetailPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <ProtectedRoute />,
    children: [
      {
        path: "/",
        element: <App />,
        children: [
          {
            index: true,
            element: <DashboardPage />,
          },
          {
            path: "/listeners",
            element: <ListenerPage />,
          },
          {
            path: "/beacons/:beaconId",
            element: <BeaconDetailPage />,
          },
        ],
      },
    ],
  },
  {
    path: "/login",
    element: <LoginPage />,
  },
]);
